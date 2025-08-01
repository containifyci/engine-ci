package language

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"os/exec"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/golang/buildscript"
)

const (
	DEFAULT_GO = "1.24.2"
)

// GolangStrategyConfig defines the configuration for different golang strategies
type GolangStrategyConfig struct {
	ImageSuffix  string // e.g., "-alpine", "", "-cgo"
	DockerFile   string // e.g., "Dockerfilego"
	GoVersion    string // e.g., "1.24.2"
}

// golangStrategy provides a unified implementation for all golang language strategies
// This eliminates ~100 lines of duplicated code per golang package
type golangStrategy struct {
	config    GolangStrategyConfig
	embedFS   embed.FS
	platforms []*types.PlatformSpec
	build     container.Build
}

// NewAlpineStrategy creates a golang strategy for Alpine builds
func NewAlpineStrategy(build container.Build, embedFS embed.FS, platforms []*types.PlatformSpec) LanguageStrategy {
	return &golangStrategy{
		config: GolangStrategyConfig{
			ImageSuffix: "-alpine",
			DockerFile:  "Dockerfilego",
			GoVersion:   DEFAULT_GO,
		},
		build:     build,
		embedFS:   embedFS,
		platforms: platforms,
	}
}

// NewDebianStrategy creates a golang strategy for Debian builds
func NewDebianStrategy(build container.Build, embedFS embed.FS, platforms []*types.PlatformSpec) LanguageStrategy {
	return &golangStrategy{
		config: GolangStrategyConfig{
			ImageSuffix: "",
			DockerFile:  "Dockerfilego",
			GoVersion:   DEFAULT_GO,
		},
		build:     build,
		embedFS:   embedFS,
		platforms: platforms,
	}
}

// NewDebianCGOStrategy creates a golang strategy for Debian CGO builds
func NewDebianCGOStrategy(build container.Build, embedFS embed.FS, platforms []*types.PlatformSpec) LanguageStrategy {
	return &golangStrategy{
		config: GolangStrategyConfig{
			ImageSuffix: "-cgo",
			DockerFile:  "Dockerfilego",
			GoVersion:   DEFAULT_GO,
		},
		build:     build,
		embedFS:   embedFS,
		platforms: platforms,
	}
}

// GetIntermediateImage returns the Go-specific intermediate container image
// Unified implementation for all golang variants with configurable image naming
func (s *golangStrategy) GetIntermediateImage(ctx context.Context) (string, error) {
	dockerFile, err := s.embedFS.ReadFile(s.config.DockerFile)
	if err != nil {
		return "", NewBuildError("read_dockerfile", "golang", err)
	}

	// Compute deterministic tag from dockerfile content
	hash := sha256.Sum256(dockerFile)
	tag := fmt.Sprintf("%x", hash[:8])
	image := fmt.Sprintf("golang-%s%s", s.config.GoVersion, s.config.ImageSuffix)
	return utils.ImageURI(s.build.ContainifyRegistry, image, tag), nil
}

// GenerateBuildScript returns the Go-specific build script
// Unified implementation with common file path adjustment logic
func (s *golangStrategy) GenerateBuildScript() string {
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

// GetAdditionalImages returns additional images needed for Go builds
// All golang variants use the same additional images
func (s *golangStrategy) GetAdditionalImages() []string {
	return []string{"alpine:latest"}
}

// ShouldCommitResult determines if the build result should be committed
// All golang variants need to commit results to create optimized final images
func (s *golangStrategy) ShouldCommitResult() bool {
	return true
}

// GetCommitCommand returns the commit command for Go builds
// Unified implementation across all golang variants
func (s *golangStrategy) GetCommitCommand() string {
	return fmt.Sprintf(`CMD ["/app/%s"]`, s.build.App)
}

// GetIntermediateImageDockerfile returns the dockerfile content for building the intermediate image
// Unified implementation using configurable dockerfile name
func (s *golangStrategy) GetIntermediateImageDockerfile(ctx context.Context) ([]byte, error) {
	return s.embedFS.ReadFile(s.config.DockerFile)
}

// GetIntermediateImagePlatforms returns the platforms for the intermediate image build
// Unified platform conversion logic for all golang variants
func (s *golangStrategy) GetIntermediateImagePlatforms() []*types.PlatformSpec {
	// Convert platform specs to container-compatible platforms (darwin -> linux conversion)
	var containerPlatforms []*types.PlatformSpec
	for _, platform := range s.platforms {
		// Use the same conversion logic as the original code
		containerPlatform := types.GetImagePlatform(platform)
		containerPlatforms = append(containerPlatforms, containerPlatform)
	}
	return containerPlatforms
}

// GetCacheDirectory returns the Go module cache directory using 'go env GOMODCACHE'
// Unified cache resolution for all golang variants (alpine, debian, debiancgo)
func (s *golangStrategy) GetCacheDirectory() (string, error) {
	// Execute 'go env GOMODCACHE' to get the proper Go module cache location
	cmd := exec.Command("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err != nil {
		return "", NewCacheError("get_gomodcache", "golang", err)
	}

	// Clean and return the cache directory path
	gomodcache := strings.TrimSpace(string(output))
	return gomodcache, nil
}