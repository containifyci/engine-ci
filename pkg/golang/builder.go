package golang

import (
	"embed"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/builder"
	"github.com/containifyci/engine-ci/pkg/builder/common"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/golang/buildscript"
	"github.com/containifyci/engine-ci/pkg/network"
)

//go:embed alpine/Dockerfile* debian/Dockerfile* debiancgo/Dockerfile*
var dockerFiles embed.FS

// GoVariant represents the different Go build variants (alpine, debian, debiancgo)
type GoVariant string

const (
	VariantAlpine    GoVariant = "alpine"
	VariantDebian    GoVariant = "debian"
	VariantDebianCGO GoVariant = "debiancgo"
)

// GoBuilder implements the LanguageBuilder and LintableBuilder interfaces for Go language builds.
// It supports multiple variants (alpine, debian, debiancgo) through configuration.
type GoBuilder struct {
	*builder.BaseBuilder

	// Go-specific configuration
	Config  *config.Config
	Variant GoVariant

	// Cache and intermediate image
	intermediateImage string
	goCache           string
}

// Ensure GoBuilder implements the required interfaces
var _ builder.LanguageBuilder = (*GoBuilder)(nil)
var _ builder.LintableBuilder = (*GoBuilder)(nil)

// NewGoBuilder creates a new Go builder with the specified variant.
func NewGoBuilder(build container.Build, variant GoVariant) (*GoBuilder, error) {
	// For now, just use default config since LoadDefaultConfig doesn't exist yet
	cfg := config.GetDefaultConfig()

	// Get Go-specific defaults from common types
	goDefaults := common.GetGoDefaults()

	// Override base image based on variant using hardcoded values for now
	// TODO: Use configuration once config system is fully integrated
	switch variant {
	case VariantAlpine:
		goDefaults.BaseImage = fmt.Sprintf("golang:%s-alpine", "1.24.2")
	case VariantDebian:
		goDefaults.BaseImage = fmt.Sprintf("golang:%s", "1.24.2")
	case VariantDebianCGO:
		goDefaults.BaseImage = fmt.Sprintf("golang:%s", "1.24.2")
	default:
		return nil, fmt.Errorf("unsupported Go variant: %s", variant)
	}

	// Create base builder
	baseBuilder := builder.NewBaseBuilder(build, goDefaults)

	return &GoBuilder{
		BaseBuilder: baseBuilder,
		Config:      cfg,
		Variant:     variant,
	}, nil
}

// Name returns the builder name with variant information.
func (g *GoBuilder) Name() string {
	return fmt.Sprintf("golang-%s", g.Variant)
}

// CacheFolder returns the Go module cache directory.
func (g *GoBuilder) CacheFolder() string {
	if g.goCache != "" {
		return g.goCache
	}

	// Try to get GOMODCACHE from environment
	cmd := exec.Command("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err != nil {
		slog.Warn("Failed to get GOMODCACHE, using default", "error", err)
		fallbackCache, _ := filepath.Abs(".tmp/go")
		g.goCache = fallbackCache
		return g.goCache
	}

	g.goCache = strings.TrimSpace(string(output))
	slog.Debug("Using GOMODCACHE", "path", g.goCache)
	return g.goCache
}

// Pull pulls the required base images for Go builds.
func (g *GoBuilder) Pull() error {
	var baseImage string
	goVersion := "1.24.2" // TODO: Use g.Config.Language.Go.Version once config is fixed
	
	switch g.Variant {
	case VariantAlpine:
		baseImage = fmt.Sprintf("golang:%s-alpine", goVersion)
	case VariantDebian, VariantDebianCGO:
		baseImage = fmt.Sprintf("golang:%s", goVersion)
	}

	additionalImages := []string{"alpine:latest"}
	return g.DefaultPullImages(append([]string{baseImage}, additionalImages...)...)
}

// IntermediateImage returns the intermediate Go image name with checksum.
func (g *GoBuilder) IntermediateImage() string {
	if g.intermediateImage != "" {
		return g.intermediateImage
	}

	// Get the appropriate Dockerfile based on variant
	var dockerfilePath string
	switch g.Variant {
	case VariantAlpine:
		dockerfilePath = "alpine/Dockerfilego"
	case VariantDebian:
		dockerfilePath = "debian/Dockerfilego"
	case VariantDebianCGO:
		dockerfilePath = "debiancgo/Dockerfilego"
	}

	dockerFile, err := dockerFiles.ReadFile(dockerfilePath)
	if err != nil {
		slog.Error("Failed to read Dockerfile", "path", dockerfilePath, "error", err)
		// Fallback to base image
		return g.Defaults.BaseImage
	}

	// Compute checksum for cache busting
	tag := container.ComputeChecksum(dockerFile)

	// Build image name based on variant
	var imageName string
	goVersion := "1.24.2" // TODO: Use g.Config.Language.Go.Version once config is fixed
	
	switch g.Variant {
	case VariantAlpine:
		imageName = fmt.Sprintf("golang-%s-alpine", goVersion)
	case VariantDebian:
		imageName = fmt.Sprintf("golang-%s", goVersion)
	case VariantDebianCGO:
		imageName = fmt.Sprintf("golang-%s-cgo", goVersion)
	}

	g.intermediateImage = utils.ImageURI(g.GetBuild().ContainifyRegistry, imageName, tag)
	return g.intermediateImage
}

// BuildIntermediateImage builds the Go-specific intermediate image.
func (g *GoBuilder) BuildIntermediateImage() error {
	image := g.IntermediateImage()

	// Get the appropriate Dockerfile based on variant
	var dockerfilePath string
	switch g.Variant {
	case VariantAlpine:
		dockerfilePath = "alpine/Dockerfilego"
	case VariantDebian:
		dockerfilePath = "debian/Dockerfilego"
	case VariantDebianCGO:
		dockerfilePath = "debiancgo/Dockerfilego"
	}

	dockerFile, err := dockerFiles.ReadFile(dockerfilePath)
	if err != nil {
		slog.Error("Failed to read Dockerfile", "path", dockerfilePath, "error", err)
		return fmt.Errorf("failed to read Dockerfile %s: %w", dockerfilePath, err)
	}

	platforms := types.GetPlatforms(g.GetBuild().Platform)
	slog.Info("Building intermediate Go image", "image", image, "platforms", platforms, "variant", g.Variant)

	return g.BuildIntermidiateContainer(image, dockerFile, platforms...)
}

// BuildScript generates the Go build script using the buildscript package.
func (g *GoBuilder) BuildScript() string {
	// Extract build options from container build
	build := g.GetBuild()
	nocoverage := build.Custom.Bool("nocoverage")
	coverageMode := buildscript.CoverageMode(build.Custom.String("coverage_mode"))
	if coverageMode == "" {
		coverageMode = buildscript.CoverageMode("text") // Default coverage mode
	}

	// Determine platforms for build
	var platforms []*types.PlatformSpec
	if customPlatforms := build.Custom.Strings("platforms"); customPlatforms != nil {
		platforms = types.ParsePlatforms(customPlatforms...)
	} else {
		// Use default platforms from the BaseBuilder
		platforms = []*types.PlatformSpec{build.Platform.Container}
	}

	return buildscript.NewBuildScript(
		build.App,
		build.File,
		build.Folder,
		build.Custom.Strings("tags"),
		build.Verbose,
		nocoverage,
		coverageMode,
		platforms...,
	).String()
}

// Build executes the main Go build process.
func (g *GoBuilder) Build() error {
	imageTag := g.IntermediateImage()

	// Setup SSH forwarding
	ssh, err := network.SSHForward(*g.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		return fmt.Errorf("failed to setup SSH forwarding: %w", err)
	}

	// Setup container configuration
	opts := types.ContainerConfig{
		Image:      imageTag,
		WorkingDir: g.Defaults.SourceMount, // Use defaults from BaseBuilder
		Script:     g.BuildScript(),
	}

	// Setup environment variables
	opts.Env = []string{
		fmt.Sprintf("GOMODCACHE=%s", g.Defaults.CacheMount),
		fmt.Sprintf("GOCACHE=%s/build-cache", g.Defaults.CacheMount),
	}

	// Add default Go environment variables
	for key, value := range g.Defaults.DefaultEnv {
		opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Setup volumes
	g.SetupContainerVolumes(&opts)

	// Apply SSH configuration
	opts = ssh.Apply(&opts)

	// Execute the build
	err = g.BuildingContainer(opts)
	if err != nil {
		slog.Error("Failed to build container", "error", err)
		return fmt.Errorf("build failed: %w", err)
	}

	return nil
}

// Prod creates a production-ready container image.
func (g *GoBuilder) Prod() error {
	if g.GetBuild().Env == container.LocalEnv {
		slog.Info("Skip building prod image in local environment")
		return nil
	}

	build := g.GetBuild()
	if build.Image == "" {
		slog.Info("Skip No image specified to push")
		return nil
	}

	// Create production container
	if err := g.CreateProductionContainer("alpine"); err != nil {
		return err
	}

	// Setup non-root user
	if err := g.SetupUserInContainer(); err != nil {
		return err
	}

	// Copy application binary
	if err := g.CopyApplicationToContainer("", "/app"); err != nil {
		return err
	}

	// Commit the image
	imageId, err := g.CommitProductionImage(
		fmt.Sprintf("\"/app/%s\"", build.App),
		"app",
		"/app",
	)
	if err != nil {
		return err
	}

	// Stop container
	if err := g.Stop(); err != nil {
		slog.Error("Failed to stop container", "error", err)
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Push to registry
	imageUri := utils.ImageURI(g.GetBuild().Registry, build.Image, build.ImageTag)
	err = g.Push(imageId, imageUri, container.PushOption{Remove: false})
	if err != nil {
		slog.Error("Failed to push image", "error", err)
		return fmt.Errorf("failed to push image: %w", err)
	}

	return nil
}

// Images returns the list of Docker images used by this builder.
func (g *GoBuilder) Images() []string {
	var baseImage string
	goVersion := "1.24.2" // TODO: Use g.Config.Language.Go.Version once config is fixed
	
	switch g.Variant {
	case VariantAlpine:
		baseImage = fmt.Sprintf("golang:%s-alpine", goVersion)
	case VariantDebian, VariantDebianCGO:
		baseImage = fmt.Sprintf("golang:%s", goVersion)
	}

	return []string{baseImage, "alpine:latest", g.IntermediateImage()}
}

// LintImage returns the golangci-lint image name from configuration.
func (g *GoBuilder) LintImage() string {
	// TODO: Use g.Config.Language.Go.LintImage once config is fixed
	return "golangci/golangci-lint:v2.1.2"
}

// Lint executes golangci-lint for Go code.
func (g *GoBuilder) Lint() error {
	image := g.IntermediateImage()

	// Setup SSH forwarding
	ssh, err := network.SSHForward(*g.GetBuild())
	if err != nil {
		slog.Error("Failed to forward SSH", "error", err)
		return fmt.Errorf("failed to setup SSH forwarding: %w", err)
	}

	// Setup container configuration
	opts := types.ContainerConfig{
		Image:      image,
		WorkingDir: g.Defaults.SourceMount, // Use defaults from BaseBuilder  
		Cmd:        []string{"sh", "/tmp/script.sh"},
	}

	// Setup environment variables
	opts.Env = []string{
		fmt.Sprintf("GOMODCACHE=%s", g.Defaults.CacheMount),
		fmt.Sprintf("GOCACHE=%s/build-cache", g.Defaults.CacheMount),
		fmt.Sprintf("GOLANGCI_LINT_CACHE=%s/lint-cache", g.Defaults.CacheMount),
	}

	// Add verbose flag if needed
	if g.GetBuild().Verbose {
		opts.Cmd = append(opts.Cmd, "-v")
	}

	// Setup volumes
	g.SetupContainerVolumes(&opts)

	// Apply SSH configuration
	opts = ssh.Apply(&opts)

	// Create container
	err = g.Create(opts)
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		return fmt.Errorf("failed to create container: %w", err)
	}

	slog.Info("Container created for linting", "containerId", g.ID)

	// Generate and copy lint script
	script := g.generateLintScript()
	err = g.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy lint script", "error", err)
		return fmt.Errorf("failed to copy lint script: %w", err)
	}

	// Start container
	err = g.Start()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for completion
	err = g.Wait()
	if err != nil {
		slog.Error("Failed to wait for container", "error", err)
		// Give time to receive all logs
		time.Sleep(5 * time.Second)
		return fmt.Errorf("linting failed: %w", err)
	}

	return nil
}

// generateLintScript creates the golangci-lint script.
func (g *GoBuilder) generateLintScript() string {
	build := g.GetBuild()
	tags := ""
	if buildTags := build.Custom.Strings("tags"); len(buildTags) > 0 {
		tags = "--build-tags " + strings.Join(buildTags, ",")
	}

	// Use default timeout for now
	timeout := 5 * time.Minute // TODO: Use g.Config.Language.Go.TestTimeout once config is fixed

	cmd := fmt.Sprintf("golangci-lint -v run %s --timeout=%s", tags, timeout)

	script := fmt.Sprintf(`#!/bin/sh
set -x
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
%s`, cmd)

	return script
}
