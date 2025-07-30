package builder

import (
	"fmt"
	"sync"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/builder/common"
	"github.com/containifyci/engine-ci/pkg/container"
)

// StandardBuildFactory implements BuildFactory for creating language-specific builders.
// This factory will be extended in future phases to support different language implementations.
type StandardBuildFactory struct {
	// Future: registry of builder implementations
	// builders map[container.BuildType]BuilderConstructor
}

// NewStandardBuildFactory creates a new factory instance.
func NewStandardBuildFactory() *StandardBuildFactory {
	return &StandardBuildFactory{}
}

// CreateBuilder creates a new LanguageBuilder for the specified build configuration.
// This method will be extended to support different language implementations.
func (f *StandardBuildFactory) CreateBuilder(build container.Build) (LanguageBuilder, error) {
	// For now, return an error indicating that specific implementations are needed
	// In the next phase, this will be replaced with actual builder creation logic
	return nil, fmt.Errorf("builder creation not yet implemented for build type: %s", build.BuildType)
}

// CreateLinter creates a build.Build instance for linting operations.
// This provides a standard way to create linting builds across all languages.
func (f *StandardBuildFactory) CreateLinter(build container.Build) (build.Build, error) {
	defaults, exists := common.GetLanguageDefaults(build.BuildType)
	if !exists {
		return nil, fmt.Errorf("no defaults found for build type: %s", build.BuildType)
	}

	// Create a standard linter build
	linterBuild := common.NewLanguageBuild(
		func() error {
			// This will be implemented by specific language builders
			return fmt.Errorf("lint operation not implemented for: %s", defaults.Language)
		},
		fmt.Sprintf("%s-lint", defaults.Language),
		[]string{defaults.LintImage},
		false, // Linting is typically synchronous
	)

	return linterBuild, nil
}

// CreateProd creates a build.Build instance for production image creation.
// This provides a standard way to create production builds across all languages.
func (f *StandardBuildFactory) CreateProd(build container.Build) (build.Build, error) {
	defaults, exists := common.GetLanguageDefaults(build.BuildType)
	if !exists {
		return nil, fmt.Errorf("no defaults found for build type: %s", build.BuildType)
	}

	// Create a standard production build
	prodBuild := common.NewLanguageBuild(
		func() error {
			// This will be implemented by specific language builders
			return fmt.Errorf("prod operation not implemented for: %s", defaults.Language)
		},
		fmt.Sprintf("%s-prod", defaults.Language),
		[]string{defaults.BaseImage},
		false, // Production builds are typically synchronous
	)

	return prodBuild, nil
}

// SupportedTypes returns the list of container.BuildType values this factory supports.
func (f *StandardBuildFactory) SupportedTypes() []container.BuildType {
	types := make([]container.BuildType, 0, len(common.LanguageDefaultsRegistry))
	for buildType := range common.LanguageDefaultsRegistry {
		types = append(types, buildType)
	}
	return types
}

// BuilderRegistration represents a registered builder implementation.
// This will be used in future phases for dynamic builder registration.
type BuilderRegistration struct {
	BuildType   container.BuildType
	Name        string
	Constructor BuilderConstructor
	Features    BuilderFeatures
}

// BuilderConstructor is a function type for creating new builder instances.
type BuilderConstructor func(container.Build) (LanguageBuilder, error)

// BuilderFeatures describes the capabilities of a specific builder implementation.
type BuilderFeatures struct {
	RequiredFiles      []string
	OptionalFiles      []string
	SupportsLinting    bool
	SupportsProduction bool
	SupportsAsync      bool
	SupportsMultiStage bool
}

// BuilderRegistry will be used in future phases to register and manage builder implementations.
// This enables a plugin-like architecture for adding new language support.
type BuilderRegistry struct {
	builders map[container.BuildType]*BuilderRegistration
	mu       sync.RWMutex
}

// NewBuilderRegistry creates a new registry for managing builder implementations.
func NewBuilderRegistry() *BuilderRegistry {
	return &BuilderRegistry{
		builders: make(map[container.BuildType]*BuilderRegistration),
	}
}

// Register adds a new builder implementation to the registry.
func (r *BuilderRegistry) Register(registration *BuilderRegistration) error {
	if registration == nil {
		return fmt.Errorf("registration cannot be nil")
	}

	if registration.Constructor == nil {
		return fmt.Errorf("constructor cannot be nil for build type: %s", registration.BuildType)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.builders[registration.BuildType] = registration
	return nil
}

// Get retrieves a builder registration for the specified build type.
func (r *BuilderRegistry) Get(buildType container.BuildType) (*BuilderRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	registration, exists := r.builders[buildType]
	return registration, exists
}

// List returns all registered build types.
func (r *BuilderRegistry) List() []container.BuildType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]container.BuildType, 0, len(r.builders))
	for buildType := range r.builders {
		types = append(types, buildType)
	}
	return types
}

// CreateBuilder creates a new builder instance using the registered constructor.
func (r *BuilderRegistry) CreateBuilder(buildType container.BuildType, build container.Build) (LanguageBuilder, error) {
	r.mu.RLock()
	registration, exists := r.builders[buildType]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no builder registered for build type: %s", buildType)
	}

	return registration.Constructor(build)
}

// Global registry instance that will be populated by language packages
var DefaultRegistry = NewBuilderRegistry()

// RegisterBuilder is a convenience function for registering builders with the default registry.
func RegisterBuilder(registration *BuilderRegistration) error {
	return DefaultRegistry.Register(registration)
}

// GetBuilder is a convenience function for retrieving builders from the default registry.
func GetBuilder(buildType container.BuildType) (*BuilderRegistration, bool) {
	return DefaultRegistry.Get(buildType)
}

// CreateLanguageBuilder creates a new language builder using the default registry.
func CreateLanguageBuilder(buildType container.BuildType, build container.Build) (LanguageBuilder, error) {
	return DefaultRegistry.CreateBuilder(buildType, build)
}
