package debian

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/golang/buildscript"
	"github.com/containifyci/engine-ci/pkg/language"
)

const (
	DEFAULT_GO = "1.24.2"
	PROJ_MOUNT = "/src"
	LINT_IMAGE = "golangci/golangci-lint:v2.1.2"
	OUT_DIR    = "/out/"
)

//go:embed Dockerfile*
var f embed.FS

// GoContainer implements the LanguageBuilder interface for Go builds using Debian base image
type GoContainer struct {
	orchestrator *language.ContainerBuildOrchestrator
	App          string
	File         string
	Folder       string
	Image        string
	ImageTag     string
	Platforms    []*types.PlatformSpec
	Tags         []string
}

func New(build container.Build) *GoContainer {
	// Create configuration for Golang Debian
	cfg := &config.LanguageConfig{
		BaseImage:     fmt.Sprintf("golang:%s", DEFAULT_GO),
		CacheLocation: "/go/pkg",
		WorkingDir:    "/src",
		BuildTimeout:  30 * time.Minute,
		Environment: map[string]string{
			"GOMODCACHE": "/go/pkg/",
			"GOCACHE":    "/go/pkg/build-cache",
		},
		Enabled: true,
	}

	baseBuilder := language.NewBaseLanguageBuilder("golang-debian", cfg, container.New(build), nil)

	platforms := []*types.PlatformSpec{build.Platform.Container}
	if !build.Platform.Same() {
		slog.Info("Different platform detected", "host", build.Platform.Host, "container", build.Platform.Container)
		platforms = []*types.PlatformSpec{types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/arm64")}
	}

	// Create Go Debian strategy locally to avoid import cycles
	strategy := newGoDebianStrategy(build, f, platforms)

	// Create orchestrator with strategy and base builder
	orchestrator := language.NewContainerBuildOrchestrator(strategy, baseBuilder)

	return &GoContainer{
		orchestrator: orchestrator,
		App:          build.App,
		Image:        build.Image,
		ImageTag:     build.ImageTag,
		Platforms:    platforms,
		File:         build.File,
		Folder:       build.Folder,
		Tags:         build.Custom["tags"],
	}
}

// goDebianStrategy implements the LanguageStrategy interface for Go Debian builds
type goDebianStrategy struct {
	embedFS   embed.FS
	platforms []*types.PlatformSpec
	build     container.Build
}

// newGoDebianStrategy creates a local Go Debian strategy to avoid import cycles
func newGoDebianStrategy(build container.Build, embedFS embed.FS, platforms []*types.PlatformSpec) language.LanguageStrategy {
	return &goDebianStrategy{
		build:     build,
		embedFS:   embedFS,
		platforms: platforms,
	}
}

// GetIntermediateImage returns the Go-specific intermediate container image (equivalent to GoImage())
func (s *goDebianStrategy) GetIntermediateImage(ctx context.Context) (string, error) {
	dockerFile, err := s.embedFS.ReadFile("Dockerfilego")
	if err != nil {
		return "", language.NewBuildError("read_dockerfile", "golang", err)
	}

	// Compute deterministic tag from dockerfile content (same logic as BaseLanguageBuilder.ComputeImageTag)
	hash := sha256.Sum256(dockerFile)
	tag := fmt.Sprintf("%x", hash[:8])
	image := fmt.Sprintf("golang-%s", DEFAULT_GO)
	return utils.ImageURI(s.build.ContainifyRegistry, image, tag), nil
}

// GenerateBuildScript returns the Go-specific build script
func (s *goDebianStrategy) GenerateBuildScript() string {
	// Extract build configuration
	nocoverage := s.build.Custom.Bool("nocoverage")
	coverageMode := buildscript.CoverageMode(s.build.Custom.String("coverage_mode"))
	tags := s.build.Custom["tags"]

	// Adjust file path for container volume mounting
	// When a specific folder is mounted, the file path should be relative to that folder
	adjustedFile := s.build.File
	if s.build.Folder != "" {
		// Handle both /src/folder/file.go and folder/file.go patterns
		expectedPath := "/src/" + s.build.Folder + "/"
		if strings.HasPrefix(s.build.File, expectedPath) {
			// Remove the /src/folder/ prefix since the folder is mounted as root
			adjustedFile = strings.TrimPrefix(s.build.File, expectedPath)
		} else if strings.HasPrefix(s.build.File, s.build.Folder+"/") {
			// Handle folder/file.go pattern (without /src/ prefix)
			adjustedFile = strings.TrimPrefix(s.build.File, s.build.Folder+"/")
		}
	}

	return buildscript.NewBuildScript(
		s.build.App,
		adjustedFile,
		s.build.Folder,
		tags,
		s.build.Verbose,
		nocoverage,
		coverageMode,
		s.platforms...,
	).String()
}

// GetAdditionalImages returns additional images needed for Go Debian builds
func (s *goDebianStrategy) GetAdditionalImages() []string {
	return []string{"alpine:latest"}
}

// ShouldCommitResult determines if the build result should be committed
func (s *goDebianStrategy) ShouldCommitResult() bool {
	return true // Go builds need to commit results to create optimized final images
}

// GetCommitCommand returns the commit command (not used since ShouldCommitResult is false)
func (s *goDebianStrategy) GetCommitCommand() string {
	return fmt.Sprintf(
		`--change 'ENTRYPOINT ["/app/%s"]' --change 'WORKDIR /app' --change 'USER app'`,
		s.build.App,
	)
}

// GetIntermediateImageDockerfile returns the dockerfile content for building the intermediate image
func (s *goDebianStrategy) GetIntermediateImageDockerfile(ctx context.Context) ([]byte, error) {
	return s.embedFS.ReadFile("Dockerfilego")
}

// GetIntermediateImagePlatforms returns the platforms for the intermediate image build
func (s *goDebianStrategy) GetIntermediateImagePlatforms() []*types.PlatformSpec {
	// Convert platform specs to container-compatible platforms (darwin -> linux conversion)
	var containerPlatforms []*types.PlatformSpec
	for _, platform := range s.platforms {
		// Use the same conversion logic as the original code
		containerPlatform := types.GetImagePlatform(platform)
		containerPlatforms = append(containerPlatforms, containerPlatform)
	}
	return containerPlatforms
}

// IsAsync returns whether this container runs asynchronously
func (c *GoContainer) IsAsync() bool {
	return c.orchestrator.GetBaseBuilder().IsAsync()
}

// Name returns the name of this language builder
func (c *GoContainer) Name() string {
	return c.orchestrator.GetBaseBuilder().Name()
}

// GetBaseBuilder returns the base language builder for compatibility
func (c *GoContainer) GetBaseBuilder() *language.BaseLanguageBuilder {
	return c.orchestrator.GetBaseBuilder()
}

// GetContainer returns the container for compatibility with existing methods
func (c *GoContainer) GetContainer() *container.Container {
	return c.orchestrator.GetBaseBuilder().GetContainer()
}

// GetLogger returns the logger for compatibility
func (c *GoContainer) GetLogger() *slog.Logger {
	return c.orchestrator.GetBaseBuilder().GetLogger()
}

// GetConfig returns the configuration for compatibility
func (c *GoContainer) GetConfig() *config.LanguageConfig {
	return c.orchestrator.GetBaseBuilder().GetConfig()
}

// BaseImage returns the base image for compatibility
func (c *GoContainer) BaseImage() string {
	return c.orchestrator.GetBaseBuilder().BaseImage()
}

// ComputeImageTag computes image tag for compatibility
func (c *GoContainer) ComputeImageTag(content []byte) string {
	return c.orchestrator.GetBaseBuilder().ComputeImageTag(content)
}

// PreBuild executes pre-build operations
func (c *GoContainer) PreBuild() error {
	return c.orchestrator.GetBaseBuilder().PreBuild()
}

// PostBuild executes post-build operations
func (c *GoContainer) PostBuild() error {
	return c.orchestrator.GetBaseBuilder().PostBuild()
}

func CacheFolder() (string, error) {
	// Command to get the GOMODCACHE location
	cmd := exec.Command("go", "env", "GOMODCACHE")

	// Run the command and capture its output
	output, err := cmd.Output()
	if err != nil {
		return "", language.NewCacheError("get_gomodcache", "golang", err)
	}

	// Print the GOMODCACHE location
	gomodcache := strings.Trim(string(output), "\n")
	slog.Debug("GOMODCACHE location", "path", gomodcache)
	return gomodcache, nil
}

func (c *GoContainer) Pull() error {
	return c.orchestrator.Pull()
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

func (c *GoContainer) GoImage() (string, error) {
	dockerFile, err := f.ReadFile("Dockerfilego")
	if err != nil {
		return "", language.NewBuildError("read_dockerfile", "golang", err)
	}
	tag := c.ComputeImageTag(dockerFile)
	image := fmt.Sprintf("golang-%s", DEFAULT_GO)
	return utils.ImageURI(c.GetContainer().GetBuild().ContainifyRegistry, image, tag), nil
}

func (c *GoContainer) Images() []string {
	return c.orchestrator.Images()
}

func (c *GoContainer) BuildGoImage() error {
	image, err := c.GoImage()
	if err != nil {
		return err
	}

	dockerFile, err := f.ReadFile("Dockerfilego")
	if err != nil {
		return language.NewBuildError("read_dockerfile", "golang", err)
	}

	platforms := types.GetPlatforms(c.GetContainer().GetBuild().Platform)
	c.GetLogger().Info("Building intermediate image", "image", image, "platforms", platforms)
	return c.GetContainer().BuildIntermidiateContainer(image, dockerFile, platforms...)
}

func (c *GoContainer) Build() (string, error) {
	return c.orchestrator.Build(context.Background())
}

func (c *GoContainer) BuildScript() string {
	// Create a temporary script in-memory
	nocoverage := c.GetContainer().GetBuild().Custom.Bool("nocoverage")
	coverageMode := buildscript.CoverageMode(c.GetContainer().GetBuild().Custom.String("coverage_mode"))
	return buildscript.NewBuildScript(c.App, c.File, c.Folder, c.Tags, c.GetContainer().Verbose, nocoverage, coverageMode, c.Platforms...).String()
}

// BuildImage implements the LanguageBuilder interface
func (c *GoContainer) BuildImage() (string, error) {
	return c.Build()
}

func NewProd(build container.Build) build.Build {
	container := New(build)
	return GoBuild{
		rf: func() error {
			return container.Prod()
		},
		name: "golang-prod",
		// images: []string{"alpine"},
	}
}

func (c *GoContainer) Prod() error {
	build := c.GetContainer().GetBuild()
	
	if build.Env == container.LocalEnv {
		c.GetLogger().Info("Skip building prod image in local environment")
		return nil
	}
	if c.Image == "" {
		c.GetLogger().Info("Skip No image specified to push")
		return nil
	}
	imageTag := "alpine"

	opts := types.ContainerConfig{}
	opts.Image = imageTag
	opts.Env = []string{}
	opts.Cmd = []string{"sleep", "300"}
	opts.Platform = types.AutoPlatform
	opts.WorkingDir = "/src"

	err := c.GetContainer().Create(opts)
	if err != nil {
		return language.NewContainerError("create_prod_container", err)
	}

	err = c.GetContainer().Start()
	if err != nil {
		return language.NewContainerError("start_prod_container", err)
	}

	err = c.GetContainer().Exec("addgroup", "-g", "11211", "app")
	if err != nil {
		return language.NewContainerError("create_app_group", err)
	}

	err = c.GetContainer().Exec("adduser", "-D", "-u", "1121", "-G", "app", "app")
	if err != nil {
		return language.NewContainerError("create_app_user", err)
	}

	containerInfo, err := c.GetContainer().Inspect()
	if err != nil {
		return language.NewContainerError("inspect_prod_container", err)
	}

	c.GetLogger().Info("Container info", "name", containerInfo.Name, "image", containerInfo.Image, "arch", containerInfo.Platform.Container.Architecture, "os", containerInfo.Platform.Container.OS, "variant", containerInfo.Platform.Container.Variant)

	err = c.GetContainer().CopyFileTo(fmt.Sprintf("%s/%s-%s-%s", c.Folder, c.App, containerInfo.Platform.Container.OS, containerInfo.Platform.Container.Architecture), fmt.Sprintf("/app/%s", c.App))
	if err != nil {
		return language.NewContainerError("copy_binary", err)
	}

	imageId, err := c.GetContainer().Commit(fmt.Sprintf("%s:%s", c.Image, c.ImageTag), "Created from container", fmt.Sprintf("CMD [\"/app/%s\"]", c.App), "USER app", "WORKDIR /app")
	if err != nil {
		return language.NewBuildError("commit_prod_image", "golang", err)
	}

	err = c.GetContainer().Stop()
	if err != nil {
		return language.NewContainerError("stop_prod_container", err)
	}

	imageUri := utils.ImageURI(build.Registry, c.Image, c.ImageTag)
	err = c.GetContainer().Push(imageId, imageUri, container.PushOption{Remove: false})
	if err != nil {
		return language.NewContainerError("push_prod_image", err)
	}

	return nil
}

func (c *GoContainer) Run() error {
	// Execute pre-build operations
	if err := c.PreBuild(); err != nil {
		return err
	}

	// Pull base images
	if err := c.Pull(); err != nil {
		c.GetLogger().Error("Failed to pull base images", "error", err)
		return err
	}

	// Build Go-specific intermediate image
	if err := c.BuildGoImage(); err != nil {
		c.GetLogger().Error("Failed to build go image", "error", err)
		return err
	}

	// Execute main build
	_, err := c.Build()
	if err != nil {
		c.GetLogger().Error("Failed to build container", "error", err)
		return err
	}
	
	c.GetLogger().Info("Container created", "containerId", c.GetContainer().ID)

	// Execute post-build operations
	if err := c.PostBuild(); err != nil {
		return err
	}
	
	return nil
}
