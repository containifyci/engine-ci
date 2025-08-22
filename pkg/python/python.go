package python

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/filesystem"
	"github.com/containifyci/engine-ci/pkg/network"
)

const (
	BaseImage     = "python:3.11-slim-bookworm"
	CacheLocation = "/root/.cache/pip"
)

//go:embed Dockerfile.*
var f embed.FS

type PythonContainer struct {
	Platform types.Platform
	*container.Container
	App      string
	File     string
	Folder   string
	Image    string
	ImageTag string
}

func New(build container.Build) *PythonContainer {
	return &PythonContainer{
		App:       build.App,
		Container: container.New(build),
		Image:     build.Image,
		Folder:    build.Folder,
		ImageTag:  build.ImageTag,
		Platform:  build.Platform,
	}
}

func (c *PythonContainer) IsAsync() bool {
	return false
}

func (c *PythonContainer) Name() string {
	return "python"
}

func CacheFolder() string {
	pipCache := os.Getenv("PIP_CACHE_DIR")
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

func (c *PythonContainer) Images() []string {
	return []string{c.PythonImage(), BaseImage}
}

// TODO: provide a shorter checksum
func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (c *PythonContainer) PythonImage() string {
	dockerFile, err := f.ReadFile("Dockerfile.python")
	if err != nil {
		slog.Error("Failed to read Dockerfile.Python", "error", err)
		os.Exit(1)
	}
	tag := ComputeChecksum(dockerFile)
	return utils.ImageURI(c.GetBuild().ContainifyRegistry, "python-3.11-slim-bookworm", tag)

	// return fmt.Sprintf("%s/%s/%s:%s", container.GetBuild().Registry, "containifyci", "python-3.11-slim-bookworm", tag)
}

func (c *PythonContainer) BuildPythonImage() error {
	dockerFile, err := f.ReadFile("Dockerfile.python")
	if err != nil {
		slog.Error("Failed to read Dockerfile.Python", "error", err)
		os.Exit(1)
	}
	tmpl, err := template.New("Dockerfile.python").Parse(string(dockerFile))
	if err != nil {
		slog.Error("Failed to parse Dockerfile.Python", "error", err)
		os.Exit(1)
	}

	var buf bytes.Buffer

	installUv := "RUN pip3 --no-cache install uv"

	// Podman can't run uv installed with x86_64.manylinux packages
	if c.GetBuild().Runtime == utils.Podman {
		installUv = `
RUN pip3 install --force-reinstall --platform musllinux_1_1_x86_64 --upgrade --only-binary=:all: --target /tmp/uv uv && \
	mv /tmp/uv/bin/uv /usr/local/bin && \
	rm -rf /tmp/uv
`
	}

	err = tmpl.Execute(&buf, map[string]string{"INSTALL_UV": installUv})
	if err != nil {
		slog.Error("Failed to render Dockerfile.Python", "error", err)
		os.Exit(1)
	}
	image := c.PythonImage()

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	return c.BuildIntermidiateContainer(image, dockerFile, platforms...)
}

func (c *PythonContainer) Address() *network.Address {
	return &network.Address{Host: "localhost"}
}

func (c *PythonContainer) Build() (string, error) {
	imageTag := c.PythonImage()

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

	opts.Platform = types.AutoPlatform
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
	opts.Script = c.BuildScript()

	err = c.BuildingContainer(opts)
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		os.Exit(1)
	}

	imageId, err := c.Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from container", "CMD [\"python\", \"/src/main.py\"]") /*, "USER worker")*/
	if err != nil {
		slog.Error("Failed to commit container: %s", "error", err)
		os.Exit(1)
	}

	return imageId, err
}

func (c *PythonContainer) BuildScript() string {
	// Create a temporary script in-memory
	return Script(NewBuildScript(c.Verbose))
}

type PythonBuild struct {
	rf     build.RunFunc
	name   string
	images []string
	async  bool
}

func (g PythonBuild) Run() error       { return g.rf() }
func (g PythonBuild) Name() string     { return g.name }
func (g PythonBuild) Images() []string { return g.images }
func (g PythonBuild) IsAsync() bool    { return g.async }

func NewProd(build container.Build) build.Build {
	container := New(build)
	return PythonBuild{
		rf: func() error {
			return container.Prod()
		},
		name:   "python-prod",
		images: []string{container.PythonImage()},
		async:  false,
	}
}

func (c *PythonContainer) Prod() error {
	opts := types.ContainerConfig{}
	opts.Image = fmt.Sprintf("%s:%s", c.Image, c.ImageTag)
	opts.Env = []string{}
	opts.Platform = types.AutoPlatform
	opts.Cmd = []string{"sleep", "300"}
	// opts.User = "185"

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

	err = c.CopyDirectoryTo(c.Folder, "/app")
	if err != nil {
		slog.Error("Failed to copy directory to container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Exec([]string{"pip", "install", "--no-cache", "/app/wheels/*"}...)
	if err != nil {
		slog.Error("Failed to install wheels: %s", "error", err)
		os.Exit(1)
	}

	imageId, err := c.Commit(opts.Image, "Created from container", "CMD [\"python\", \"/app/main.py\"]", "WORKDIR /app") /*, "USER 185")*/
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

	err = c.Tag(imageID, fmt.Sprintf("%s:%s", c.Image, c.ImageTag))
	if err != nil {
		slog.Error("Failed to tag image: %s", "error", err)
		return err
	}
	return nil
}
