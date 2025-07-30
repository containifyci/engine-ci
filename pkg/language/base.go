package language

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
)

// BaseLanguageBuilder provides common functionality for all language builders.
// This eliminates code duplication across language packages by implementing
// shared behavior that all language builders need.
type BaseLanguageBuilder struct {
	cache     build.CacheManager
	validator build.Validator
	config    *config.LanguageConfig
	container *container.Container
	logger    *slog.Logger
	name      string
}

// NewBaseLanguageBuilder creates a new base language builder
func NewBaseLanguageBuilder(
	name string,
	config *config.LanguageConfig,
	container *container.Container,
	cache build.CacheManager,
) *BaseLanguageBuilder {
	return &BaseLanguageBuilder{
		name:      name,
		config:    config,
		container: container,
		cache:     cache,
		logger:    slog.With("component", "language-builder", "language", name),
	}
}

// Name returns the name of the language builder
func (b *BaseLanguageBuilder) Name() string {
	return b.name
}

// IsAsync returns whether this builder supports async execution
// Most language builders are synchronous by default
func (b *BaseLanguageBuilder) IsAsync() bool {
	return false
}

// BaseImage returns the base container image for this language
func (b *BaseLanguageBuilder) BaseImage() string {
	return b.config.BaseImage
}

// CacheLocation returns the cache directory path inside the container
func (b *BaseLanguageBuilder) CacheLocation() string {
	return b.config.CacheLocation
}

// DefaultEnvironment returns the default environment variables for this language
func (b *BaseLanguageBuilder) DefaultEnvironment() []string {
	if b.config.Environment == nil {
		return []string{}
	}

	env := make([]string, 0, len(b.config.Environment))
	for key, value := range b.config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

// BuildTimeout returns the maximum build time allowed for this language
func (b *BaseLanguageBuilder) BuildTimeout() time.Duration {
	return b.config.BuildTimeout
}

// ComputeImageTag computes a deterministic tag from dockerfile content
// This replaces the duplicated ComputeChecksum functions across packages
func (b *BaseLanguageBuilder) ComputeImageTag(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Pull pulls the required base images for this language
func (b *BaseLanguageBuilder) Pull() error {
	b.logger.Info("Pulling base image", "image", b.BaseImage())
	return b.container.Pull(b.BaseImage())
}

// PreBuild executes common pre-build operations
func (b *BaseLanguageBuilder) PreBuild() error {
	b.logger.Debug("Executing pre-build operations")

	// Ensure cache directory exists
	if b.cache != nil {
		if err := b.cache.EnsureCacheDir(b.name); err != nil {
			return fmt.Errorf("failed to ensure cache directory for %s: %w", b.name, err)
		}
	}

	// Validate configuration
	if err := b.Validate(); err != nil {
		return fmt.Errorf("validation failed for %s builder: %w", b.name, err)
	}

	return nil
}

// PostBuild executes common post-build operations
func (b *BaseLanguageBuilder) PostBuild() error {
	b.logger.Debug("Executing post-build operations")

	// Cleanup temporary resources
	// This could include cleaning up intermediate containers,
	// temporary files, or other resources

	return nil
}

// Validate validates the builder configuration and dependencies
func (b *BaseLanguageBuilder) Validate() error {
	if b.name == "" {
		return NewValidationError("name", b.name, "language name cannot be empty")
	}

	if b.config == nil {
		return NewValidationError("config", nil, "language configuration is required")
	}

	if b.config.BaseImage == "" {
		return NewValidationError("base_image", b.config.BaseImage, "base image must be specified")
	}

	if b.config.CacheLocation == "" {
		return NewValidationError("cache_location", b.config.CacheLocation, "cache location must be specified")
	}

	if b.config.BuildTimeout <= 0 {
		return NewValidationError("build_timeout", b.config.BuildTimeout, "build timeout must be positive")
	}

	return nil
}

// GetContainer returns the underlying container instance
func (b *BaseLanguageBuilder) GetContainer() *container.Container {
	return b.container
}

// GetConfig returns the language configuration
func (b *BaseLanguageBuilder) GetConfig() *config.LanguageConfig {
	return b.config
}

// GetLogger returns the logger instance
func (b *BaseLanguageBuilder) GetLogger() *slog.Logger {
	return b.logger
}

// GetCacheManager returns the cache manager instance
func (b *BaseLanguageBuilder) GetCacheManager() build.CacheManager {
	return b.cache
}

// Protected methods that can be overridden by specific language implementations

// BuildScript generates the build script for this language
// This method must be implemented by specific language builders
func (b *BaseLanguageBuilder) BuildScript() string {
	panic(fmt.Sprintf("BuildScript method must be implemented by %s language builder", b.name))
}

// Build executes the build process and returns the resulting image ID
// This method must be implemented by specific language builders
func (b *BaseLanguageBuilder) Build() (string, error) {
	panic(fmt.Sprintf("Build method must be implemented by %s language builder", b.name))
}

// BuildImage builds the intermediate language-specific image
// This method must be implemented by specific language builders
func (b *BaseLanguageBuilder) BuildImage() (string, error) {
	panic(fmt.Sprintf("BuildImage method must be implemented by %s language builder", b.name))
}

// Images returns all images required by this builder
// This method must be implemented by specific language builders
func (b *BaseLanguageBuilder) Images() []string {
	panic(fmt.Sprintf("Images method must be implemented by %s language builder", b.name))
}

// SetValidator sets the validator for this builder
func (b *BaseLanguageBuilder) SetValidator(validator build.Validator) {
	b.validator = validator
}

// ValidateWithValidator uses the configured validator if available
func (b *BaseLanguageBuilder) ValidateWithValidator(ctx context.Context) (*build.ValidationResult, error) {
	if b.validator == nil {
		// If no validator is set, just return a simple validation based on Validate()
		err := b.Validate()
		return &build.ValidationResult{
			IsValid:  err == nil,
			Errors:   []error{err},
			Warnings: []string{},
			Context:  map[string]interface{}{"language": b.name},
		}, nil
	}

	return b.validator.ValidateLanguageBuilder(b)
}

// CreateBuildStepAdapter creates a BuildStep adapter for this language builder
// This allows language builders to be used in build pipelines
func (b *BaseLanguageBuilder) CreateBuildStepAdapter() build.BuildStep {
	return &LanguageBuilderStep{
		builder: b,
		name:    fmt.Sprintf("%s-build", b.name),
	}
}

// LanguageBuilderStep adapts a LanguageBuilder to the BuildStep interface
type LanguageBuilderStep struct {
	builder build.LanguageBuilder
	name    string
}

// Name returns the name of this build step
func (s *LanguageBuilderStep) Name() string {
	return s.name
}

// Execute executes the language builder as a build step
func (s *LanguageBuilderStep) Execute(ctx context.Context) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Execute pre-build
	if err := s.builder.PreBuild(); err != nil {
		return fmt.Errorf("pre-build failed: %w", err)
	}

	// Pull images
	if err := s.builder.Pull(); err != nil {
		return fmt.Errorf("failed to pull images: %w", err)
	}

	// Execute build
	_, err := s.builder.Build()
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Execute post-build
	if err := s.builder.PostBuild(); err != nil {
		return fmt.Errorf("post-build failed: %w", err)
	}

	return nil
}

// Dependencies returns the dependencies for this build step
func (s *LanguageBuilderStep) Dependencies() []string {
	// Language builders typically don't have dependencies by default
	// This can be overridden by specific implementations
	return []string{}
}

// IsAsync returns whether this step can run asynchronously
func (s *LanguageBuilderStep) IsAsync() bool {
	return s.builder.IsAsync()
}

// Timeout returns the timeout for this build step
func (s *LanguageBuilderStep) Timeout() time.Duration {
	return s.builder.BuildTimeout()
}

// Validate validates this build step
func (s *LanguageBuilderStep) Validate() error {
	return s.builder.Validate()
}

// Ensure BaseLanguageBuilder implements LanguageBuilder interface
var _ build.LanguageBuilder = (*BaseLanguageBuilder)(nil)

// Ensure LanguageBuilderStep implements BuildStep interface
var _ build.BuildStep = (*LanguageBuilderStep)(nil)
