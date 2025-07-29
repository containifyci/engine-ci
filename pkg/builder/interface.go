package builder

import (
	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

// LanguageBuilder defines the unified interface for all language-specific container builders.
// This interface captures the common operations that all language builders must implement,
// enabling consistent behavior across Go, Maven, Python, and other language implementations.
type LanguageBuilder interface {
	// Core lifecycle operations
	Name() string                        // Returns the language/builder name (e.g., "golang", "maven", "python")
	IsAsync() bool                       // Indicates if this builder supports asynchronous operations
	Pull() error                         // Pulls required base images for this language
	Build() error                        // Executes the main build process
	Run() error                          // Runs the complete build pipeline (Pull + BuildIntermediateImage + Build)
	Images() []string                    // Returns list of Docker images used by this builder

	// Production and deployment operations
	Prod() error                         // Creates production-ready container image
	
	// Language-specific image management
	BuildIntermediateImage() error       // Builds the intermediate language-specific image
	IntermediateImage() string           // Returns the intermediate image name/tag
	
	// Configuration and script generation
	BuildScript() string                 // Generates the build script for the container
	CacheFolder() string                 // Returns the language-specific cache directory path
}

// AsyncLanguageBuilder extends LanguageBuilder for builders that support asynchronous operations.
// Builders implementing this interface can be executed concurrently with other build steps.
type AsyncLanguageBuilder interface {
	LanguageBuilder
	
	// Additional async-specific operations can be added here in the future
	// For example: Cancel(), Status(), Progress(), etc.
}

// LintableBuilder defines the interface for language builders that support linting operations.
// This is implemented by languages like Go that have integrated linting tools.
type LintableBuilder interface {
	LanguageBuilder
	
	// Lint operations
	Lint() error                         // Executes linting for the language
	LintImage() string                   // Returns the linting tool image name
}

// BuildFactory creates language-specific builders from a container.Build configuration.
// This factory pattern allows for centralized builder creation and configuration injection.
type BuildFactory interface {
	// CreateBuilder creates a new LanguageBuilder instance for the specified build type
	CreateBuilder(build container.Build) (LanguageBuilder, error)
	
	// CreateLinter creates a build.Build instance for linting operations (if supported)
	CreateLinter(build container.Build) (build.Build, error)
	
	// CreateProd creates a build.Build instance for production image creation
	CreateProd(build container.Build) (build.Build, error)
	
	// SupportedTypes returns the list of container.BuildType values this factory supports
	SupportedTypes() []container.BuildType
}

// BuildConfiguration encapsulates common configuration used across all language builders.
// This struct will be extended in future phases to support centralized configuration.
type BuildConfiguration struct {
	// Container and platform configuration
	Platform    types.Platform
	Platforms   []*types.PlatformSpec
	Registry    string
	Environment container.EnvType
	Verbose     bool
	
	// Application configuration
	App      string
	File     string
	Folder   string
	Image    string
	ImageTag string
	Tags     []string
	
	// Custom configuration map for language-specific settings
	Custom   container.Custom
}

// BaseLanguageBuilder provides common functionality that can be embedded by concrete implementations.
// This reduces code duplication and ensures consistent behavior across all language builders.
type BaseLanguageBuilder struct {
	*container.Container
	Config BuildConfiguration
}

// GetBuild returns the underlying container.Build for compatibility with existing code.
// This method provides access to the container build configuration.
func (b *BaseLanguageBuilder) GetBuild() *container.Build {
	return b.Container.Build
}

// ApplyContainerOptions applies common container configuration options.
// This method encapsulates shared logic for setting up container environments.
func (b *BaseLanguageBuilder) ApplyContainerOptions(opts *types.ContainerConfig) {
	// Common container setup that's shared across all language builders
	if opts.WorkingDir == "" {
		opts.WorkingDir = "/src"
	}
	
	// Apply verbose flag if set
	if b.Config.Verbose && len(opts.Cmd) > 0 && opts.Cmd[0] == "sh" {
		opts.Cmd = append(opts.Cmd, "-v")
	}
}