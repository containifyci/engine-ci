package python

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/filesystem"
	"github.com/containifyci/engine-ci/pkg/network"
	"github.com/containifyci/engine-ci/protos2"

	u "github.com/containifyci/engine-ci/pkg/utils"
)

const (
	BaseImage     = "python:3.14-slim-bookworm"
	CacheLocation = "/root/.cache/pip"
)

//go:embed Dockerfile.*
var f embed.FS

type PythonContainer struct {
	Platform types.Platform
	*container.Container
	App          string
	File         string
	Folder       string
	Image        string
	ImageTag     string
	Secret       map[string]string
	PrivateIndex PrivateIndex
}

// Matches implements the Build interface - Debian variant runs when from=debian
func Matches(build container.Build) bool {
	return build.BuildType == container.Python
}

func New() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  Images,
		Name_:     "python",
		Async_:    false,
	}
}

func new(build container.Build) *PythonContainer {
	return &PythonContainer{
		App:          build.App,
		Container:    container.New(build),
		Image:        build.Image,
		Folder:       build.Folder,
		File:         build.File,
		ImageTag:     build.ImageTag,
		Platform:     build.Platform,
		Secret:       build.Secret,
		PrivateIndex: NewPrivateIndex(build.Custom),
	}
}

func CacheFolder() string {
	pipCache := u.GetEnvs([]string{"PIP_CACHE_DIR", "CONTAINIFYCI_CACHE"}, "build")
	if pipCache == "" {
		pipCache = os.TempDir() + ".pip"
		slog.Info("Python_HOME not set, using default", "pipCache", pipCache)
		err := filesystem.DirectoryExists(pipCache)
		if err != nil {
			slog.Error("Failed to create cache folder", "error", err)
			os.Exit(1)
		}
	}
	return pipCache
}

func (c *PythonContainer) Pull() error {
	return c.Container.Pull(BaseImage)
}

func Images(build container.Build) []string {
	return []string{PythonImage(build), BaseImage}
}

func PythonImage(build container.Build) string {
	dockerFile, err := dockerFile(build)
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}
	tag := container.ComputeChecksum([]byte(dockerFile.Content))
	image := dockerFile.Name
	return utils.ImageURI(build.ContainifyRegistry, image, tag)
}

func dockerFile(build container.Build) (*protos2.ContainerFile, error) {
	if v, ok := build.ContainerFiles["build"]; ok {
		return v, nil
	}

	dockerFile, err := f.ReadFile("Dockerfile.python")
	if err != nil {
		slog.Error("Failed to read Dockerfile.Python", "error", err)
		os.Exit(1)
	}
	return &protos2.ContainerFile{
		Name:    "python-3.14-slim-bookworm",
		Content: string(dockerFile),
	}, nil
}

func (c *PythonContainer) BuildPythonImage() error {
	dockerFile, err := dockerFile(*c.GetBuild())
	if err != nil {
		slog.Error("Failed to read Dockerfile.Python", "error", err)
		os.Exit(1)
	}
	image := PythonImage(*c.GetBuild())

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	return c.BuildIntermidiateContainer(image, ([]byte)(dockerFile.Content), platforms...)
}

func (c *PythonContainer) Address() *network.Address {
	return &network.Address{Host: "localhost"}
}

func (c *PythonContainer) Build() (string, error) {
	imageTag := PythonImage(*c.GetBuild())

	ssh, err := network.SSHForward(*c.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	opts.Env = append(opts.Env, []string{
		"_PIP_USE_IMPORTLIB_METADATA=0",
		"UV_CACHE_DIR=/root/.cache/pip",
	}...)

	//TODO this env var should be set in the CI/CD environment because the name of the secret may vary the DATA_UTILS is the name of the dependency
	if c.PrivateIndex.Username() != "" {
		opts.Env = append(opts.Env, c.PrivateIndex.Username())
	}

	opts.Platform = types.AutoPlatform
	opts.Secrets = c.Secret
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

	opts = ssh.Apply(&opts)
	opts.Script = c.BuildScript().Script()

	err = c.BuildingContainer(opts)
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		os.Exit(1)
	}

	if c.Image == "" {
		slog.Debug("No image name provided, skipping commit")
		return "", nil
	}

	imageId, err := c.Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from container", "CMD [\"python\", \"/src/main.py\"]") /*, "USER worker")*/
	if err != nil {
		slog.Error("Failed to commit container: %s", "error", err)
		os.Exit(1)
	}

	return imageId, err
}

func (c *PythonContainer) BuildScript() *BuildScript {
	// Create a temporary script in-memory

	builder := NewBuilder(c.Folder)
	_, err := builder.Analyze()
	if err != nil {
		slog.Error("Failed to analyze python project", "error", err)
		os.Exit(1)
	}
	cmds, err := builder.Build()
	if err != nil {
		slog.Error("Failed to build python commands", "error", err)
		os.Exit(1)
	}
	installCmds, err := builder.Install()
	if err != nil {
		slog.Error("Failed to build python commands", "error", err)
		os.Exit(1)
	}
	return NewBuildScript(c.Folder, c.Verbose, c.PrivateIndex, cmds, installCmds)
}

func NewProd() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Prod()
		},
		ImagesFn:  Images,
		Name_:     "python-prod",
		MatchedFn: Matches,
		Async_:    false,
	}
}

func (c *PythonContainer) Prod() error {
	opts := types.ContainerConfig{}
	// opts.Image = fmt.Sprintf("%s:%s", c.Image, c.ImageTag)
	opts.Image = BaseImage
	opts.Env = []string{}
	opts.Platform = types.AutoPlatform
	opts.Cmd = []string{"sleep", "300"}
	opts.WorkingDir = "/app"
	// opts.User = "185"

	opts.Secrets = c.Secret

	opts.Volumes = []types.Volume{{
		Type:   "bind",
		Source: CacheFolder(),
		Target: CacheLocation,
	}}

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

	err = c.Exec("mkdir", "-p", "/app/dist")
	if err != nil {
		slog.Error("Failed to create directory in container: %s", "error", err)
		os.Exit(1)
	}

	err = c.CopyDirectoryTo(c.Folder+"/dist/", "/app/dist")
	if err != nil {
		slog.Error("Failed to copy directory to container: %s", "error", err)
		os.Exit(1)
	}

	cmds := c.BuildScript().InstallCommands

	for _, cmd := range cmds {
		slog.Info("Running command in container", "cmd", cmd)
		err = c.Exec(cmd...)
		if err != nil {
			slog.Error("Failed to run", "error", err, "cmd", cmd)
			os.Exit(1)
		}
	}

	imageId, err := c.Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from container", "CMD [\"python\", \"-m\", \""+c.File+"\"]", "WORKDIR /app") /*, "USER 185")*/
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
	err = c.Push(imageId, imageUri, container.PushOption{Remove: false})
	if err != nil {
		slog.Error("Failed to push image: %s", "error", err)
		os.Exit(1)
	}

	return err
}

func (c *PythonContainer) Run() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull base images: %s", "error", err)
		return err
	}

	err = c.BuildPythonImage()
	if err != nil {
		slog.Error("Failed to build go image: %s", "error", err)
		return err
	}

	imageID, err := c.Build()
	slog.Info("Container created", "containerId", c.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		return err
	}

	if c.Image == "" {
		slog.Debug("No image name provided, skipping tagging")
		return nil
	}

	err = c.Tag(imageID, fmt.Sprintf("%s:%s", c.Image, c.ImageTag))
	if err != nil {
		slog.Error("Failed to tag image: %s", "error", err)
		return err
	}
	return nil
}
