package zig

//go:generate go run ../../tools/dockerfile-metadata/ -input Dockerfile.zig -output docker_metadata_gen.go -package zig

import (
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

	u "github.com/containifyci/engine-ci/pkg/utils"
)

const (
	BaseImage     = "alpine:latest"
	CacheLocation = "/root/.cache/zig"
)

type ZigContainer struct {
	Platform types.Platform
	*container.Container
	App      string
	File     string
	Folder   string
	Image    string
	ImageTag string
	Secret   map[string]string
	Optimize string
	Target   string
}

// Matches implements the Build interface - runs when buildtype is Zig
func Matches(build container.Build) bool {
	return build.BuildType == container.Zig
}

func New() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  Images,
		Name_:     "zig",
		Async_:    false,
	}
}

func new(build container.Build) *ZigContainer {
	return &ZigContainer{
		App:       build.App,
		Container: container.New(build),
		Image:     build.Image,
		Folder:    build.Folder,
		File:      build.File,
		ImageTag:  build.ImageTag,
		Platform:  build.Platform,
		Secret:    build.Secret,
		Optimize:  build.Custom.String("optimize"),
		Target:    build.Custom.String("target"),
	}
}

func CacheFolder() string {
	zigCache := u.GetEnvs([]string{"ZIG_GLOBAL_CACHE_DIR", "CONTAINIFYCI_CACHE"}, "build")
	if zigCache == "" {
		zigCache = filepath.Join(os.TempDir(), ".zig-cache")
		slog.Info("ZIG_GLOBAL_CACHE_DIR not set, using default", "zigCache", zigCache)
		err := filesystem.DirectoryExists(zigCache)
		if err != nil {
			slog.Error("Failed to create cache folder", "error", err)
			os.Exit(1)
		}
	}
	return zigCache
}

func (c *ZigContainer) Pull() error {
	return c.Container.Pull(BaseImage)
}

func Images(build container.Build) []string {
	return []string{ZigImage(build), BaseImage}
}

func ZigImage(build container.Build) string {
	image := fmt.Sprintf("zig-%s", ImageVersion)
	return utils.ImageURI(build.ContainifyRegistry, image, DockerfileChecksum)
}

func (c *ZigContainer) BuildZigImage() error {
	image := ZigImage(*c.GetBuild())

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	return c.BuildIntermidiateContainer(image, ([]byte)(DockerfileContent), platforms...)
}

func (c *ZigContainer) Address() *network.Address {
	return &network.Address{Host: "localhost"}
}

func (c *ZigContainer) Build() (string, error) {
	imageTag := ZigImage(*c.GetBuild())

	ssh, err := network.SSHForward(*c.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	opts.Env = append(opts.Env, []string{
		fmt.Sprintf("ZIG_GLOBAL_CACHE_DIR=%s", CacheLocation),
	}...)

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

	// Determine the CMD based on the app name or default binary
	appCmd := c.App
	if appCmd == "" {
		appCmd = "app"
	}

	imageId, err := c.Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from Zig build", fmt.Sprintf("CMD [\"/src/zig-out/bin/%s\"]", appCmd))
	if err != nil {
		slog.Error("Failed to commit container", "error", err)
		os.Exit(1)
	}

	return imageId, err
}

func (c *ZigContainer) BuildScript() *BuildScript {
	return NewBuildScript(c.Folder, c.Optimize, c.Target, c.Verbose, CacheLocation)
}

func NewProd() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Prod()
		},
		ImagesFn:  Images,
		Name_:     "zig-prod",
		MatchedFn: Matches,
		Async_:    false,
	}
}

func (c *ZigContainer) Prod() error {
	opts := types.ContainerConfig{}
	opts.Image = BaseImage
	opts.Env = []string{}
	opts.Platform = types.AutoPlatform
	opts.Cmd = []string{"sleep", "300"}
	opts.WorkingDir = "/app"

	opts.Secrets = c.Secret

	err := c.Create(opts)
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		os.Exit(1)
	}

	err = c.Start()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Exec("mkdir", "-p", "/app/bin")
	if err != nil {
		slog.Error("Failed to create directory in container", "error", err)
		os.Exit(1)
	}

	// Copy built binaries from zig-out/bin to /app/bin
	zigOutBin := filepath.Join("zig-out", "bin")
	err = c.CopyDirectoryTo(zigOutBin+"/", "/app/bin")
	if err != nil {
		slog.Error("Failed to copy directory to container", "error", err)
		os.Exit(1)
	}

	// Determine the default binary name
	appCmd := c.App
	if appCmd == "" {
		appCmd = "app"
	}

	imageId, err := c.Commit(
		fmt.Sprintf("%s:%s", c.Image, c.ImageTag),
		"Created from Zig production build",
		fmt.Sprintf("CMD [\"/app/bin/%s\"]", appCmd),
		"WORKDIR /app",
	)
	if err != nil {
		slog.Error("Failed to commit container", "error", err)
		os.Exit(1)
	}

	err = c.Stop()
	if err != nil {
		slog.Error("Failed to stop container", "error", err)
		os.Exit(1)
	}

	imageUri := utils.ImageURI(c.GetBuild().Registry, c.Image, c.ImageTag)
	err = c.Push(imageId, imageUri, container.PushOption{Remove: false})
	if err != nil {
		slog.Error("Failed to push image", "error", err)
		os.Exit(1)
	}

	return err
}

func (c *ZigContainer) Run() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull base images", "error", err)
		return err
	}

	err = c.BuildZigImage()
	if err != nil {
		slog.Error("Failed to build Zig image", "error", err)
		return err
	}

	imageID, err := c.Build()
	slog.Info("Container created", "containerId", c.ID)
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		return err
	}

	if c.Image == "" {
		slog.Debug("No image name provided, skipping tagging")
		return nil
	}

	err = c.Tag(imageID, fmt.Sprintf("%s:%s", c.Image, c.ImageTag))
	if err != nil {
		slog.Error("Failed to tag image", "error", err)
		return err
	}
	return nil
}
