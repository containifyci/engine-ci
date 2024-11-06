package debian

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/network"
)

const (
	DEFAULT_GO = "1.23.2"
	PROJ_MOUNT = "/src"
	LINT_IMAGE = "golangci/golangci-lint:v1.61.0"
	OUT_DIR    = "/out/"
)

//go:embed Dockerfile*
var f embed.FS

type GoContainer struct {
	//TODO add option to fail on linter or not
	App       string
	File      string
	Image     string
	ImageTag  string
	Platforms []*types.PlatformSpec
	Tags      []string
	*container.Container
}

func New() *GoContainer {
	platforms := []*types.PlatformSpec{container.GetBuild().Platform.Container}
	if !container.GetBuild().Platform.Same() {
		slog.Info("Different platform detected", "host", container.GetBuild().Platform.Host, "container", container.GetBuild().Platform.Container)
		platforms = []*types.PlatformSpec{types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/arm64")}
	}
	return &GoContainer{
		App:       container.GetBuild().App,
		Container: container.New(container.BuildEnv),
		Image:     container.GetBuild().Image,
		ImageTag:  container.GetBuild().ImageTag,
		// TODO: only build multiple platforms when buildenv and localenv are running on different platforms
		// FIX: linux-arm64 go build is needed when building contains on MacOS M1/M2
		// Platforms: []*types.PlatformSpec{types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/arm64")},
		Platforms: platforms,
		File:      container.GetBuild().File,
		Tags:      container.GetBuild().Custom["tags"],
	}
}

func (c *GoContainer) IsAsync() bool {
	return false
}

func (c *GoContainer) Name() string {
	return "golang"
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
	fmt.Printf("GOMODCACHE location: %s\n", gomodcache)
	return gomodcache
}

func (c *GoContainer) Pull() error {
	imageTag := fmt.Sprintf("golang:%s", DEFAULT_GO)
	return c.Container.Pull(imageTag, "alpine:latest")
}

type GoBuild struct {
	rf     build.RunFunc
	name   string
	images []string
	async  bool
}

func (g GoBuild) Run() error       { return g.rf() }
func (g GoBuild) Name() string     { return g.name }
func (g GoBuild) Images() []string { return g.images }
func (g GoBuild) IsAsync() bool    { return g.async }

func NewLinter() build.Build {
	return GoBuild{
		rf: func() error {
			container := New()
			err := container.Container.Pull(LINT_IMAGE)
			if err != nil {
				slog.Error("Failed to pull image: %s", "error", err, "image", LINT_IMAGE)
				os.Exit(1)
			}

			return container.Lint()
		},
		name:   "golangci-lint",
		images: []string{LINT_IMAGE},
		async:  false,
	}
}

func LintImage() string {
	return LINT_IMAGE
}

func (c *GoContainer) CopyLintScript() error {
	tags := ""
	if len(c.Tags) > 0 {
		// c.Tags = append(c.Tags, "linux")
		tags = "--build-tags " + strings.Join(c.Tags, ",")
	}
	script := fmt.Sprintf(`#!/bin/sh
set -x
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
GOOS=%s GOARCH=%s golangci-lint --out-format colored-line-number -v run %s --timeout=5m
ls -lha /
`, container.GetBuild().Platform.Container.OS, container.GetBuild().Platform.Container.Architecture, tags)
	err := c.Container.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func (c *GoContainer) Lint() error {
	imageTag := LintImage()

	ssh, err := network.SSHForward()
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	opts.Env = append(opts.Env, []string{
		"GOMODCACHE=/go/pkg/",
		"GOCACHE=/go/pkg/build-cache",
		"GOLANGCI_LINT_CACHE=/go/pkg/lint-cache",
	}...)
	opts.Cmd = []string{"sh", "/tmp/script.sh"}
	// opts.User = "golangci-lint"
	if c.Container.Verbose {
		opts.Cmd = append(opts.Cmd, "-v")
	}
	// opts.Platform = "auto"
	opts.WorkingDir = "/src"

	dir, _ := filepath.Abs(".")
	cache := CacheFolder()
	if cache == "" {
		cache, _ = filepath.Abs(".tmp/go")
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

	err = c.Container.Create(opts)
	slog.Info("Container created", "containerId", c.Container.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.CopyLintScript()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Container.Start()
	if err != nil {
		slog.Error("Failed to start container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Container.Wait()
	if err != nil {
		slog.Error("Failed to wait for container: %s", "error", err)
		// GIVE time to receive all logs
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	return err
}

func GoImage() string {
	dockerFile, err := f.ReadFile("Dockerfilego")
	if err != nil {
		slog.Error("Failed to read Dockerfile.go", "error", err)
		os.Exit(1)
	}
	tag := container.ComputeChecksum(dockerFile)
	image := fmt.Sprintf("golang-%s", DEFAULT_GO)
	return utils.ImageURI(container.GetBuild().ContainifyRegistry, image, tag)
}

func (c *GoContainer) Images() []string {
	imageTag := fmt.Sprintf("golang:%s", DEFAULT_GO)

	return []string{imageTag, "alpine:latest", GoImage()}
}

func (c *GoContainer) BuildGoImage() error {
	image := GoImage()

	dockerFile, err := f.ReadFile("Dockerfilego")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}

	platforms := types.GetPlatforms(container.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)
	return c.Container.BuildIntermidiateContainer(image, dockerFile, platforms...)
}

func (c *GoContainer) Build() error {
	imageTag := GoImage()

	ssh, err := network.SSHForward()
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

	c.Container.Apply(&opts)

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

	err = c.Container.BuildingContainer(opts)
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		os.Exit(1)
	}

	return err
}

func (c *GoContainer) BuildScript() string {
	// Create a temporary script in-memory
	return Script(NewBuildScript(c.App, c.File, c.Tags, c.Container.Verbose, c.Platforms...))
}

func NewProd() build.Build {
	container := New()
	return GoBuild{
		rf: func() error {
			return container.Prod()
		},
		name: "golang-prod",
		// images: []string{"alpine"},
	}
}

func (c *GoContainer) Prod() error {
	if c.Image == "" {
		slog.Info("Skip No image specified to push")
		return nil
	}
	imageTag := "alpine"

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	opts.Env = []string{}
	opts.Cmd = []string{"sleep", "300"}
	opts.Platform = types.AutoPlatform
	opts.WorkingDir = "/src"

	err := c.Container.Create(opts)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Container.Start()
	if err != nil {
		slog.Error("Failed to start container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Container.Exec("addgroup", "-g", "11211", "app")
	if err != nil {
		slog.Error("Failed to execute command: %s", "error", err)
		os.Exit(1)
	}

	err = c.Container.Exec("adduser", "-D", "-u", "1121", "-G", "app", "app")
	if err != nil {
		slog.Error("Failed to execute command", "error", err)
		os.Exit(1)
	}

	containerInfo, err := c.Container.Inspect()
	if err != nil {
		slog.Error("Failed to inspect container", "error", err)
		os.Exit(1)
	}

	slog.Info("Container info", "name", containerInfo.Name, "image", containerInfo.Image, "arch", containerInfo.Platform.Container.Architecture, "os", containerInfo.Platform.Container.OS, "varian", containerInfo.Platform.Container.Variant)

	err = c.Container.CopyFileTo(fmt.Sprintf("./%s-%s-%s", c.App, containerInfo.Platform.Container.OS, containerInfo.Platform.Container.Architecture), fmt.Sprintf("/app/%s", c.App))
	if err != nil {
		slog.Error("Failed to copy file to container", "error", err)
		os.Exit(1)
	}

	imageId, err := c.Container.Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from container", fmt.Sprintf("CMD [\"/app/%s\"]", c.App), "USER app", "WORKDIR /app")
	if err != nil {
		slog.Error("Failed to commit container", "error", err)
		os.Exit(1)
	}

	err = c.Container.Stop()
	if err != nil {
		slog.Error("Failed to stop container: %s", "error", err)
		os.Exit(1)
	}

	imageUri := utils.ImageURI(container.GetBuild().Registry, c.Image, c.ImageTag)
	err = c.Container.Push(imageId, imageUri, container.PushOption{Remove: false})
	if err != nil {
		slog.Error("Failed to push image: %s", "error", err)
		os.Exit(1)
	}

	return err
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
	slog.Info("Container created", "containerId", c.Container.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		return err
	}
	return nil
}
