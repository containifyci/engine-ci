package alpine

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
	"github.com/containifyci/engine-ci/pkg/golang/buildscript"
	"github.com/containifyci/engine-ci/pkg/network"
	"github.com/containifyci/engine-ci/protos2"

	u "github.com/containifyci/engine-ci/pkg/utils"
)

const (
	DEFAULT_GO = "1.25.3"
	PROJ_MOUNT = "/src"
	OUT_DIR    = "/out/"
)

//go:embed Dockerfile*
var f embed.FS

type GoContainer struct {
	*container.Container
	ContainerFiles map[string]*protos2.ContainerFile
	App            string
	File           u.SrcFile
	Folder         string
	Image          string
	ImageTag       string
	Platforms      []*types.PlatformSpec
	Tags           []string
}

func New() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  GoImages,
		Name_:     "golang",
		Async_:    false,
	}
}

func new(build container.Build) *GoContainer {
	platforms := []*types.PlatformSpec{build.Platform.Container}
	if !build.Platform.Same() {
		slog.Debug("Different platform detected", "host", build.Platform.Host, "container", build.Platform.Container)
		platforms = []*types.PlatformSpec{types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/arm64")}
	}
	return &GoContainer{
		App:            build.App,
		Container:      container.New(build),
		ContainerFiles: build.ContainerFiles,
		Image:          build.Image,
		ImageTag:       build.ImageTag,
		// TODO: only build multiple platforms when buildenv and localenv are running on different platforms
		// FIX: linux-arm64 go build is needed when building contains on MacOS M1/M2
		Platforms: platforms,
		// Platforms: []*types.PlatformSpec{types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/arm64")},
		File:   u.SrcFile(build.File),
		Folder: build.Folder,
		Tags:   build.Custom["tags"],
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
	imageTag := fmt.Sprintf("golang:%s-alpine", DEFAULT_GO)
	return c.Container.Pull(imageTag, "alpine:latest")
}

func NewLinter() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Lint()
		},
		MatchedFn: Matches,
		Name_:     "golangci-lint",
		Async_:    true, // Linter runs async
	}
}
func (c *GoContainer) Lint() error {
	image := GoImage(*c.GetBuild())

	ssh, err := network.SSHForward(*c.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = image
	opts.Env = append(opts.Env, []string{
		"GOMODCACHE=/go/pkg/",
		"GOCACHE=/go/pkg/build-cache",
		"GOLANGCI_LINT_CACHE=/go/pkg/lint-cache",
	}...)
	opts.Cmd = []string{"sh", "/tmp/script.sh"}
	// opts.User = "golangci-lint"
	if c.Verbose {
		opts.Cmd = append(opts.Cmd, "-v")
	}
	// opts.Platform = "auto"
	opts.WorkingDir = "/src"

	dir, _ := filepath.Abs(".")
	if c.Folder != "" {
		dir, _ = filepath.Abs(c.Folder)
	}
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

	err = c.Create(opts)
	slog.Info("Container created", "containerId", c.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	script := NewGolangCiLint().LintScript(c.Tags, c.Folder)
	err = c.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Start()
	if err != nil {
		slog.Error("Failed to start container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Wait()
	if err != nil {
		slog.Error("Failed to wait for container: %s", "error", err)
		// GIVE time to receive all logs
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	return err
}

func dockerFile(build container.Build) (*protos2.ContainerFile, error) {
	if v, ok := build.ContainerFiles["build"]; ok {
		return v, nil
	}

	dockerFileName := "Dockerfile_go"
	typ := build.CustomString("go_type")
	name := fmt.Sprintf("golang-%s-alpine", DEFAULT_GO)
	if typ != "" {
		dockerFileName = fmt.Sprintf("Dockerfile_%s_go", typ)
		name = fmt.Sprintf("golang-%s-alpine-%s", DEFAULT_GO, typ)
	}
	dockerFile, err := f.ReadFile(dockerFileName)
	if err != nil {
		slog.Error("Failed to read Dockerfile.go", "error", err)
		os.Exit(1)
	}
	return &protos2.ContainerFile{
		Name:    name,
		Content: string(dockerFile),
	}, nil
}

func GoImages(build container.Build) []string {
	image := fmt.Sprintf("golang:%s-alpine", DEFAULT_GO)
	return []string{image, "alpine:latest", GoImage(build)}
}

func GoImage(build container.Build) string {
	dockerFile, err := dockerFile(build)
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}
	tag := container.ComputeChecksum([]byte(dockerFile.Content))
	// image := fmt.Sprintf("golang-%s-alpine", DEFAULT_GO)
	image := dockerFile.Name
	return utils.ImageURI(build.ContainifyRegistry, image, tag)
}

func (c *GoContainer) BuildGoImage() error {
	image := GoImage(*c.GetBuild())

	dockerFile, err := dockerFile(*c.GetBuild())
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)
	return c.BuildIntermidiateContainer(image, []byte(dockerFile.Content), platforms...)
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

	// dir, _ := filepath.Abs(c.Folder)
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
	buildScript := c.BuildScript()
	opts.Script = buildScript.String()

	if len(buildScript.Artifacts) > 0 {
		f, err := os.OpenFile("artifacts.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			slog.Error("Failed to create artifacts file", "error", err)
			os.Exit(1)
		}
		defer f.Close()
		// Write each artifact on a new line
		_, err = f.WriteString(strings.Join(buildScript.Artifacts, "\n") + "\n")

		// err = os.WriteFile("artifacts.txt", []byte(strings.Join(buildScript.Artifacts, "\n")), 0644)
		if err != nil {
			slog.Error("Failed to write content to artifacts file", "error", err)
			os.Exit(1)
		}
	}

	err = c.BuildingContainer(opts)
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		os.Exit(1)
	}

	return err
}

func (c *GoContainer) BuildScript() *buildscript.BuildScript {
	// Create a temporary script in-memory
	nocoverage := c.GetBuild().Custom.Bool("nocoverage", false)
	coverageMode := buildscript.CoverageMode(c.GetBuild().Custom.String("coverage_mode"))
	return buildscript.NewBuildScript(c.App, c.File.Container(), c.Folder, c.Tags, c.Verbose, nocoverage, coverageMode, c.Platforms...)
}

func Matches(build container.Build) bool {
	// Only match golang builds
	if build.BuildType != container.GoLang {
		return false
	}

	if from, ok := build.Custom["from"]; ok && len(from) > 0 {
		return from[0] == "alpine"
	}

	return true
}

func NewProd() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Prod()
		},
		Name_:     "golang-prod",
		ImagesFn:  build.StepperImages("alpine"),
		Async_:    false,
		MatchedFn: Matches,
	}
}

func (c *GoContainer) Prod() error {
	if c.GetBuild().Env == container.LocalEnv {
		slog.Info("Skip building prod image in local environment")
		return nil
	}
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

	err = c.Exec("addgroup", "-g", "11211", "app")
	if err != nil {
		slog.Error("Failed to execute command: %s", "error", err)
		os.Exit(1)
	}

	err = c.Exec("adduser", "-D", "-u", "1121", "-G", "app", "app")
	if err != nil {
		slog.Error("Failed to execute command", "error", err)
		os.Exit(1)
	}

	containerInfo, err := c.Inspect()
	if err != nil {
		slog.Error("Failed to inspect container", "error", err)
		os.Exit(1)
	}

	slog.Info("Container info", "name", containerInfo.Name, "image", containerInfo.Image, "arch", containerInfo.Platform.Container.Architecture, "os", containerInfo.Platform.Container.OS, "varian", containerInfo.Platform.Container.Variant)

	err = c.CopyFileTo(fmt.Sprintf("%s/%s-%s-%s", c.Folder, c.App, containerInfo.Platform.Container.OS, containerInfo.Platform.Container.Architecture), fmt.Sprintf("/app/%s", c.App))
	if err != nil {
		slog.Error("Failed to copy file to container", "error", err)
		os.Exit(1)
	}

	imageId, err := c.Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from container", fmt.Sprintf("CMD [\"/app/%s\"]", c.App), "USER app", "WORKDIR /app")
	if err != nil {
		slog.Error("Failed to commit container", "error", err)
		os.Exit(1)
	}

	err = c.Stop()
	if err != nil {
		slog.Error("Failed to stop container: %s", "error", err)
		os.Exit(1)
	}

	push := c.GetBuild().Custom.Bool("push", true)
	if !push {
		slog.Info("Skip pushing image")
		return nil
	}

	imageUri := utils.ImageURI(c.GetBuild().Registry, c.Image, c.ImageTag)
	err = c.Push(imageId, imageUri, container.PushOption{Remove: false})
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
	slog.Info("Container created", "containerId", c.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		return err
	}
	return nil
}
