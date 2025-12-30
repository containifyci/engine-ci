package maven

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"

	"github.com/containifyci/engine-ci/pkg/filesystem"
	"github.com/containifyci/engine-ci/pkg/network"
	u "github.com/containifyci/engine-ci/pkg/utils"
)

const (
	ProdImage     = "registry.access.redhat.com/ubi8/openjdk-17:latest"
	CacheLocation = "/root/.m2/"
)

//go:embed Dockerfile.*
var f embed.FS

type MavenContainer struct {
	Platform types.Platform
	*container.Container
	App      string
	File     string
	Folder   string
	Image    string
	ImageTag string
}

// Matches implements the Build interface - Debian variant runs when from=debian
func Matches(build container.Build) bool {
	return build.BuildType == container.Maven
}

func New() build.BuildStepv3 {
	return build.Stepper{
		BuildType_: container.Maven,
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  Images,
		Name_:     "maven",
		Alias_:    "build",
		Async_:    false,
	}
}

func new(build container.Build) *MavenContainer {
	return &MavenContainer{
		App:       build.App,
		Container: container.New(build),
		Image:     build.Image,
		Folder:    build.Folder,
		ImageTag:  build.ImageTag,
		Platform:  build.Platform,
	}
}

func CacheFolder() string {
	mvnHome := u.GetEnvs([]string{"MAVEN_HOME", "CONTAINIFYCI_CACHE"}, "build")
	if mvnHome == "" {
		usr, err := user.Current()
		if err != nil {
			slog.Error("Failed to get current user", "error", err)
			os.Exit(1)
		}
		mvnHome = fmt.Sprintf("%s%s%s", usr.HomeDir, string(os.PathSeparator), ".m2")
		slog.Info("MAVEN_HOME not set, using default", "mavenHome", mvnHome)
		err = filesystem.DirectoryExists(mvnHome)
		if err != nil {
			slog.Error("Failed to create cache folder", "error", err)
			os.Exit(1)
		}
	}
	return mvnHome
}

func (c *MavenContainer) Pull() error {
	return c.Container.Pull(ProdImage)
}

func Images(build container.Build) []string {
	return []string{MavenImage(build), ProdImage}
}

// TODO: provide a shorter checksum
func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func MavenImage(build container.Build) string {
	dockerFile, err := f.ReadFile("Dockerfile.maven")
	if err != nil {
		slog.Error("Failed to read Dockerfile.maven", "error", err)
		os.Exit(1)
	}
	tag := ComputeChecksum(dockerFile)
	return utils.ImageURI(build.ContainifyRegistry, "maven-3-eclipse-temurin-17-alpine", tag)
	// return fmt.Sprintf("%s/%s/%s:%s", container.GetBuild().Registry, "containifyci", "maven-3-eclipse-temurin-17-alpine", tag)
}

func (c *MavenContainer) BuildMavenImage() error {
	image := MavenImage(*c.GetBuild())
	dockerFile, err := f.ReadFile("Dockerfile.maven")
	if err != nil {
		slog.Error("Failed to read Dockerfile.maven", "error", err)
		os.Exit(1)
	}

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	err = c.BuildIntermidiateContainer(image, dockerFile, platforms...)
	if err != nil {
		slog.Error("Failed to build maven image", "error", err)
		os.Exit(1)
	}
	return nil
}

func (c *MavenContainer) Address() *network.Address {
	return &network.Address{Host: "localhost"}
}

func (c *MavenContainer) Build() error {
	imageTag := MavenImage(*c.GetBuild())

	ssh, err := network.SSHForward(*c.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	opts.Env = append(opts.Env, []string{
		"MAVEN_OPTS=-Xms512m -Xmx512m -XX:MaxDirectMemorySize=512m",
	}...)

	// On MacOS, we need to set a special docker host so that the testcontainers can access the host
	if c.Platform.Host.OS == "darwin" {
		opts.Env = append(opts.Env, []string{
			fmt.Sprintf("TC_HOST=%s", c.Address().ForContainerDefault(c.GetBuild())),
			fmt.Sprintf("TESTCONTAINERS_HOST_OVERRIDE=%s", c.Address().ForContainerDefault(c.GetBuild())),
		}...)
	}

	opts.WorkingDir = "/src"

	dir, _ := filepath.Abs(".")

	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/src",
		},
		{
			Type:   "bind",
			Source: CacheFolder(),
			Target: CacheLocation,
		},
	}
	opts.Memory = int64(4073741824)
	opts.CPU = uint64(2048)

	opts = ssh.Apply(&opts)
	opts = utils.ApplySocket(c.GetBuild().Runtime, &opts)

	if c.GetBuild().Runtime == utils.Podman {
		//https://stackoverflow.com/questions/71549856/testcontainers-with-podman-in-java-tests
		opts.Env = append(opts.Env, []string{
			"DOCKER_HOST=unix://var/run/podman.sock",
			"TESTCONTAINERS_RYUK_DISABLED=true",
			//TODO identify if we need privileged mode or not
		}...)
	}

	if privilged := u.GetEnv("CONTAINER_PRIVILGED", "build"); privilged == "false" {
		opts.Env = append(opts.Env,
			"TESTCONTAINERS_RYUK_DISABLED=true",
			"TESTCONTAINERS_RYUK_PRIVILEGED=false",
		)
	}

	opts.Script = c.BuildScript()

	err = c.BuildingContainer(opts)
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		return fmt.Errorf("failed to build container: %w", err)
	}

	return err
}

func (c *MavenContainer) BuildScript() string {
	// Create a temporary script in-memory
	return Script(NewBuildScript(c.Verbose))
}

func NewProd() build.BuildStepv3 {
	return build.Stepper{
		BuildType_: container.Maven,
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Prod()
		},
		ImagesFn: build.StepperImages(ProdImage),
		Name_:    "maven-prod",
		Alias_:   "push",
		Async_:   false,
	}
}

func (c *MavenContainer) Prod() error {
	opts := types.ContainerConfig{}
	opts.Image = ProdImage
	opts.Env = []string{
		"JAVA_OPTS=-javaagent:/deployments/dd-java-agent.jar -Dquarkus.http.host=0.0.0.0 -Djava.util.logging.manager=org.jboss.logmanager.LogManager",
		"JAVA_APP_JAR=/deployments/quarkus-run.jar",
	}
	opts.Platform = types.AutoPlatform
	opts.Cmd = []string{"sleep", "300"}
	opts.User = "185"
	opts.WorkingDir = "/src"

	err := c.Create(opts)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Start()
	if err != nil {
		slog.Error("Failed to start container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Exec("curl", "-Lo", "/deployments/dd-java-agent.jar", "https://dtdg.co/latest-java-tracer")
	if err != nil {
		slog.Error("Failed to execute command: %s", "error", err)
		os.Exit(1)
	}

	err = c.CopyDirectoryTo(c.Folder, "/deployments")
	if err != nil {
		slog.Error("Failed to copy directory to container: %s", "error", err)
		os.Exit(1)
	}

	imageId, err := c.Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from container", "CMD [\"/usr/local/s2i/run\"]", "USER 185")
	if err != nil {
		slog.Error("Failed to commit container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Stop()
	if err != nil {
		slog.Error("Failed to stop container: %s", "error", err)
		os.Exit(1)
	}

	imageUri := utils.ImageURI(c.GetBuild().Registry, c.Image, c.ImageTag)
	err = c.Push(imageId, imageUri)
	if err != nil {
		slog.Error("Failed to push image: %s", "error", err)
		os.Exit(1)
	}

	return err
}

func (c *MavenContainer) Run() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull base images: %s", "error", err)
		return err
	}

	err = c.BuildMavenImage()
	if err != nil {
		slog.Error("Failed to build go image: %s", "error", err)
		return err
	}

	err = c.Build()
	slog.Info("Container created", "containerId", c.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		return err
	}
	return nil
}
