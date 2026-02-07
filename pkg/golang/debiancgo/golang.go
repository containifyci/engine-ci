package debiancgo

//go:generate go run ../../../tools/dockerfile-metadata/ -input Dockerfilego -output docker_metadata_gen.go -package debiancgo

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/golang/buildscript"
	"github.com/containifyci/engine-ci/pkg/network"

	u "github.com/containifyci/engine-ci/pkg/utils"
)

const (
	PROJ_MOUNT = "/src"
	OUT_DIR    = "/out/"
)

// Dockerfile content is now available via generated constants in docker_metadata_gen.go
// No longer need embed.FS for Dockerfile parsing

type GoContainer struct {
	//TODO add option to fail on linter or not
	*container.Container
	App       string
	File      u.SrcFile
	Folder    string
	Image     string
	ImageTag  string
	Platforms []*types.PlatformSpec
	Tags      []string
}

// Matches implements the Build interface - CGO variant runs when from=debiancgo
func Matches(build container.Build) bool {
	if build.BuildType != container.GoLang {
		return false
	}
	if from, ok := build.Custom["from"]; ok && len(from) > 0 {
		return from[0] == "debiancgo"
	}
	return false
}

func New() build.BuildStepv3 {
	return build.Stepper{
		BuildType_: container.GoLang,
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  Images,
		Name_:     "golang",
		Alias_:    "build",
		Async_:    false,
	}
}

func new(build container.Build) *GoContainer {
	platforms := []*types.PlatformSpec{build.Platform.Container}
	if !build.Platform.Same() {
		slog.Debug("Different platform detected", "host", build.Platform.Host, "container", build.Platform.Container)
		platforms = []*types.PlatformSpec{types.ParsePlatform("darwin/arm64"), types.ParsePlatform("darwin/amd64"), types.ParsePlatform("linux/arm64")}
	}
	return &GoContainer{
		App:       build.App,
		Container: container.New(build),
		Image:     build.Image,
		ImageTag:  build.ImageTag,
		Platforms: platforms,
		File:      u.NewSrcFile(build.Folder, build.File),
		Folder:    build.Folder,
		Tags:      build.Custom["tags"],
	}
}

func CacheFolder() string {
	// Command to get the GOMODCACHE location
	cmd := exec.Command("go", "env", "GOMODCACHE")

	// Run the command and capture its output
	output, err := cmd.Output()
	if err != nil {
		slog.Error("Failed to execute command: %s", "error", err)
		os.Exit(1)
	}

	// Print the GOMODCACHE location
	gomodcache := strings.Trim(string(output), "\n")
	slog.Debug("GOMODCACHE location", "path", gomodcache)
	return gomodcache
}

func (c *GoContainer) Pull() error {
	imageTag := fmt.Sprintf("golang:%s", ImageVersion)
	return c.Container.Pull(imageTag, "alpine:latest")
}

func GoImage(build container.Build) string {
	image := fmt.Sprintf("golang-%s-cgo", ImageVersion)
	return utils.ImageURI(build.ContainifyRegistry, image, DockerfileChecksum)
}

func Images(build container.Build) []string {
	image := fmt.Sprintf("golang:%s", ImageVersion)
	return []string{image, "alpine:latest", GoImage(build)}
}

func (c *GoContainer) BuildGoImage() error {
	image := GoImage(*c.GetBuild())

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)
	return c.BuildIntermidiateContainer(image, []byte(DockerfileContent), platforms...)
}

func (c *GoContainer) Build() error {
	imageTag := GoImage(*c.GetBuild())

	ssh, err := network.SSHForward(*c.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	opts.Env = append(opts.Env, []string{
		"GOMODCACHE=/go/pkg/",
		"GOCACHE=/go/pkg/build-cache",
	}...)
	opts.WorkingDir = "/src"

	c.Apply(&opts)

	dir, _ := filepath.Abs(".")

	cache := CacheFolder()
	if cache == "" {
		cache, _ = filepath.Abs(".tmp/go")
		err := os.MkdirAll(".tmp/go", os.ModePerm)
		if err != nil {
			slog.Error("Failed to create cache folder: %s", "error", err)
			os.Exit(1)
		}
	}

	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/src",
		},
		{
			Type:   "bind",
			Source: cache,
			Target: "/go/pkg",
		},
	}

	opts = ssh.Apply(&opts)
	opts.Script = c.BuildScript()

	err = c.BuildingContainer(opts)
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		return fmt.Errorf("failed to build container: %w", err)
	}

	return err
}

func (c *GoContainer) BuildScript() string {
	// Create a temporary script in-memory
	platforms := c.Platforms
	if c.GetBuild().Custom.Strings("platforms") != nil {
		platforms = types.ParsePlatforms(c.GetBuild().Custom.Strings("platforms")...)
	}
	nocoverage := c.GetBuild().Custom.Bool("nocoverage", false)
	coverageMode := buildscript.CoverageMode(c.GetBuild().Custom.String("coverage_mode"))
	generateMode := c.GetBuild().Custom.String("generate")
	if generateMode == "" {
		generateMode = "auto"
	}
	return buildscript.NewCGOBuildScript(c.App, c.File.Container(), c.Folder, c.Tags, c.Container.Verbose, nocoverage, coverageMode, generateMode, platforms...).String()
}

func (c *GoContainer) Run() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull base images: %s", "error", err)
		return err
	}

	err = c.BuildGoImage()
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
