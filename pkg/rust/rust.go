package rust

//go:generate go run ../../tools/dockerfile-metadata/ -input Dockerfile.rust -output docker_metadata_gen.go -package rust

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
	CacheLocation = "/root/.cargo"
)

type RustContainer struct {
	*container.Container
	Secret    map[string]string
	App       string
	File      string
	Folder    string
	Image     string
	ImageTag  string
	Profile   string
	Target    string
	Features  []string
	Platforms []*types.PlatformSpec
}

// Matches implements the Build interface - runs when buildtype is Rust
func Matches(build container.Build) bool {
	return build.BuildType == container.Rust
}

func New() build.BuildStepv3 {
	return build.Stepper{
		BuildType_: container.Rust,
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  Images,
		Name_:     "rust",
		Alias_:    "build",
		Async_:    false,
	}
}

func new(build container.Build) *RustContainer {
	platforms := []*types.PlatformSpec{build.Platform.Container}

	target := build.Custom.String("target")
	profile := build.Custom.String("profile")
	if profile == "" {
		profile = "release"
	}

	// Parse features from custom config
	featuresStr := build.Custom.String("features")
	var features []string
	if featuresStr != "" {
		features = []string{featuresStr}
	}

	if !build.Platform.Same() {
		slog.Debug("Different platform detected", "host", build.Platform.Host, "container", build.Platform.Container)
		platforms = []*types.PlatformSpec{types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/arm64")}
	}

	return &RustContainer{
		App:       build.App,
		Container: container.New(build),
		Image:     build.Image,
		Folder:    build.Folder,
		File:      build.File,
		ImageTag:  build.ImageTag,
		Platforms: platforms,
		Secret:    build.Secret,
		Profile:   profile,
		Target:    target,
		Features:  features,
	}
}

func CacheFolder() string {
	cargoHome := u.GetEnvs([]string{"CARGO_HOME", "CONTAINIFYCI_CACHE"}, "build")
	if cargoHome == "" {
		cargoHome = filepath.Join(os.TempDir(), ".cargo")
		slog.Info("CARGO_HOME not set, using default", "cargoHome", cargoHome)
		err := filesystem.DirectoryExists(cargoHome)
		if err != nil {
			slog.Error("Failed to create cache folder", "error", err)
			os.Exit(1)
		}
	}
	return cargoHome
}

func (c *RustContainer) Pull() error {
	return c.Container.Pull(BaseImage)
}

func Images(build container.Build) []string {
	return []string{RustImage(build), BaseImage}
}

func RustImage(build container.Build) string {
	image := fmt.Sprintf("rust-%s", ImageVersion)
	return utils.ImageURI(build.ContainifyRegistry, image, DockerfileChecksum)
}

func (c *RustContainer) BuildRustImage() error {
	image := RustImage(*c.GetBuild())

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	return c.BuildIntermidiateContainer(image, ([]byte)(DockerfileContent), platforms...)
}

func (c *RustContainer) Address() *network.Address {
	return &network.Address{Host: "localhost"}
}

func (c *RustContainer) Build() (string, error) {
	imageTag := RustImage(*c.GetBuild())

	ssh, err := network.SSHForward(*c.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	opts.Env = append(opts.Env, []string{
		fmt.Sprintf("CARGO_HOME=%s", CacheLocation),
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
		return "", fmt.Errorf("failed to build container: %w", err)
	}

	if c.Image == "" {
		slog.Debug("No image name provided, skipping commit")
		return "", nil
	}

	// Determine the binary path based on profile
	binaryPath := "debug"
	if c.Profile == "release" {
		binaryPath = "release"
	}

	// Determine the CMD based on the app name or default binary
	appCmd := c.App
	if appCmd == "" {
		appCmd = "app"
	}

	imageId, err := c.Commit(
		fmt.Sprintf("%s:%s", c.Image, c.ImageTag),
		"Created from Rust build",
		fmt.Sprintf("CMD [\"/src/target/%s/%s\"]", binaryPath, appCmd),
	)
	if err != nil {
		slog.Error("Failed to commit container", "error", err)
		os.Exit(1)
	}

	return imageId, err
}

func (c *RustContainer) BuildScript() *BuildScript {
	return NewBuildScript(c.Folder, c.Profile, c.Target, c.Features, c.Verbose, CacheLocation, c.Platforms)
}

func NewProd() build.BuildStepv3 {
	return build.Stepper{
		BuildType_: container.Rust,
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Prod()
		},
		ImagesFn:  Images,
		Name_:     "rust-prod",
		Alias_:    "push",
		MatchedFn: Matches,
		Async_:    false,
	}
}

func (c *RustContainer) Prod() error {
	if c.Image == "" {
		slog.Info("Skip No image specified to push")
		return nil
	}
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

	// Determine the binary path based on profile
	binaryPath := "debug"
	if c.Profile == "release" {
		binaryPath = "release"
	}

	// Copy built binaries from target/{profile} to /app/bin
	targetBin := filepath.Join("target", binaryPath)
	err = c.CopyDirectoryTo(targetBin+"/", "/app/bin")
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
		"Created from Rust production build",
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

func (c *RustContainer) Run() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull base images", "error", err)
		return err
	}

	err = c.BuildRustImage()
	if err != nil {
		slog.Error("Failed to build Rust image", "error", err)
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
