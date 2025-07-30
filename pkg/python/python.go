package python

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/filesystem"
	"github.com/containifyci/engine-ci/pkg/language"
	"github.com/containifyci/engine-ci/pkg/network"
)

//go:embed Dockerfile.*
var f embed.FS

// PythonContainer implements the LanguageBuilder interface for Python builds
type PythonContainer struct {
	*language.BaseLanguageBuilder
	Platform types.Platform
	App      string
	File     string
	Folder   string
	Image    string
	ImageTag string
}

func New(build container.Build) *PythonContainer {
	// Create configuration for Python
	cfg := &config.LanguageConfig{
		BaseImage:     "python:3.11-slim-bookworm",
		CacheLocation: "/root/.cache/pip",
		WorkingDir:    "/src",
		Environment: map[string]string{
			"_PIP_USE_IMPORTLIB_METADATA": "0",
			"UV_CACHE_DIR":                "/root/.cache/pip",
		},
	}

	baseBuilder := language.NewBaseLanguageBuilder("python", cfg, container.New(build), nil)

	return &PythonContainer{
		BaseLanguageBuilder: baseBuilder,
		App:                build.App,
		Image:              build.Image,
		Folder:             build.Folder,
		ImageTag:           build.ImageTag,
		Platform:           build.Platform,
	}
}

func CacheFolder() (string, error) {
	pipCache := os.Getenv("PIP_CACHE_DIR")
	if pipCache == "" {
		pipCache = os.TempDir() + ".pip"
		slog.Info("Python_HOME not set, using default", "pipCache", pipCache)
		err := filesystem.DirectoryExists(pipCache)
		if err != nil {
			return "", language.NewCacheError("create_cache_folder", "python", err).WithPath(pipCache)
		}
	}
	return pipCache, nil
}

func (c *PythonContainer) Pull() error {
	return c.GetContainer().Pull(c.BaseImage())
}

func (c *PythonContainer) Images() []string {
	pythonImage, err := c.PythonImage()
	if err != nil {
		slog.Error("Failed to get Python image", "error", err)
		return []string{c.BaseImage()}
	}
	return []string{pythonImage, c.BaseImage()}
}

// TODO: provide a shorter checksum
func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (c *PythonContainer) PythonImage() (string, error) {
	dockerFile, err := f.ReadFile("Dockerfile.python")
	if err != nil {
		return "", language.NewBuildError("read_dockerfile", "python", err)
	}
	tag := ComputeChecksum(dockerFile)
	return utils.ImageURI(c.GetContainer().GetBuild().ContainifyRegistry, "python-3.11-slim-bookworm", tag), nil
}

func (c *PythonContainer) BuildPythonImage() error {
	dockerFile, err := f.ReadFile("Dockerfile.python")
	if err != nil {
		return language.NewBuildError("read_dockerfile", "python", err)
	}
	
	tmpl, err := template.New("Dockerfile.python").Parse(string(dockerFile))
	if err != nil {
		return language.NewBuildError("parse_dockerfile_template", "python", err)
	}

	var buf bytes.Buffer

	installUv := "RUN pip3 --no-cache install uv"

	// Podman can't run uv installed with x86_64.manylinux packages
	if c.GetContainer().GetBuild().Runtime == utils.Podman {
		installUv = `
RUN pip3 install --force-reinstall --platform musllinux_1_1_x86_64 --upgrade --only-binary=:all: --target /tmp/uv uv && \
	mv /tmp/uv/bin/uv /usr/local/bin && \
	rm -rf /tmp/uv
`
	}

	err = tmpl.Execute(&buf, map[string]string{"INSTALL_UV": installUv})
	if err != nil {
		return language.NewBuildError("render_dockerfile_template", "python", err)
	}
	
	image, err := c.PythonImage()
	if err != nil {
		return err
	}

	platforms := types.GetPlatforms(c.GetContainer().GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	return c.GetContainer().BuildIntermidiateContainer(image, dockerFile, platforms...)
}

func (c *PythonContainer) Address() *network.Address {
	return &network.Address{Host: "localhost"}
}

func (c *PythonContainer) Build() (string, error) {
	imageTag, err := c.PythonImage()
	if err != nil {
		return "", err
	}

	ssh, err := network.SSHForward(*c.GetContainer().GetBuild())
	if err != nil {
		return "", language.NewBuildError("ssh_forward", "python", err)
	}

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	
	// Use configuration for environment variables
	cfg := c.GetConfig()
	for key, value := range cfg.Environment {
		opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", key, value))
	}

	opts.Platform = types.AutoPlatform
	opts.WorkingDir = cfg.WorkingDir

	dir, _ := filepath.Abs(".")

	cacheFolder, err := CacheFolder()
	if err != nil {
		return "", err
	}

	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: cfg.WorkingDir,
		},
		{
			Type:   "bind",
			Source: cacheFolder,
			Target: cfg.CacheLocation,
		},
	}

	opts = ssh.Apply(&opts)
	opts.Script = c.BuildScript()

	err = c.GetContainer().BuildingContainer(opts)
	if err != nil {
		return "", language.NewBuildError("building_container", "python", err)
	}

	imageId, err := c.GetContainer().Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from container", "CMD [\"python\", \"/src/run.py\"]")
	if err != nil {
		return "", language.NewBuildError("commit_container", "python", err)
	}

	return imageId, nil
}

func (c *PythonContainer) BuildScript() string {
	// Create a temporary script in-memory
	verbose := c.GetContainer().GetBuild().Verbose
	return Script(NewBuildScript(verbose))
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
	pythonImage, err := container.PythonImage()
	if err != nil {
		slog.Error("Failed to get Python image for production build", "error", err)
		pythonImage = container.BaseImage() // fallback to base image
	}
	
	return PythonBuild{
		rf: func() error {
			return container.Prod()
		},
		name:   "python-prod",
		images: []string{pythonImage},
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

	err := c.GetContainer().Create(opts)
	if err != nil {
		return language.NewContainerError("create", err)
	}

	err = c.GetContainer().Start()
	if err != nil {
		return language.NewContainerError("start", err)
	}

	err = c.GetContainer().CopyDirectoryTo(c.Folder, "/app")
	if err != nil {
		return language.NewContainerError("copy_directory", err)
	}

	err = c.GetContainer().Exec([]string{"pip", "install", "--no-cache", "/app/wheels/*"}...)
	if err != nil {
		return language.NewContainerError("install_wheels", err)
	}

	imageId, err := c.GetContainer().Commit(opts.Image, "Created from container", "CMD [\"python\", \"/app/run.py\"]", "WORKDIR /app")
	if err != nil {
		return language.NewBuildError("commit_production_container", "python", err)
	}

	err = c.GetContainer().Stop()
	if err != nil {
		return language.NewContainerError("stop", err)
	}

	imageUri := utils.ImageURI(c.GetContainer().GetBuild().Registry, c.Image, c.ImageTag)
	err = c.GetContainer().Push(imageId, imageUri, container.PushOption{Remove: false})
	if err != nil {
		return language.NewContainerError("push_image", err)
	}

	return nil
}

func (c *PythonContainer) Run() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull base images", "error", err)
		return err
	}

	err = c.BuildPythonImage()
	if err != nil {
		slog.Error("Failed to build python image", "error", err)
		return err
	}

	imageID, err := c.Build()
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		return err
	}
	
	slog.Info("Container created", "containerId", c.GetContainer().ID)

	err = c.GetContainer().Tag(imageID, fmt.Sprintf("%s:%s", c.Image, c.ImageTag))
	if err != nil {
		slog.Error("Failed to tag image", "error", err)
		return err
	}
	return nil
}

// BuildImage implements the LanguageBuilder interface
func (c *PythonContainer) BuildImage(ctx context.Context) (string, error) {
	return c.Build()
}

// Execute implements the BuildStep interface
func (c *PythonContainer) Execute(ctx context.Context) error {
	return c.Run()
}

// Validate implements the BuildStep interface  
func (c *PythonContainer) Validate(ctx context.Context) error {
	// Validate that required files exist
	if _, err := os.Stat("requirements.txt"); os.IsNotExist(err) {
		return language.NewValidationError("requirements.txt", nil, "requirements.txt file is required for Python builds")
	}
	return nil
}
