package config

import (
	"fmt"
	"sync"

	"github.com/containifyci/engine-ci/pkg/container"
)

// ConfigurableBuilder defines the interface for builders that can be configured.
// This interface extends the basic LanguageBuilder with configuration support.
type ConfigurableBuilder interface {
	// SetConfig applies configuration to the builder
	SetConfig(config *Config) error

	// GetConfig returns the current configuration
	GetConfig() *Config

	// ValidateConfig validates the configuration for this builder
	ValidateConfig() error
}

// BuilderFactory creates language-specific builders with configuration injection.
// This factory pattern allows for centralized builder creation and configuration management.
type BuilderFactory struct {
	config      *Config
	configMutex sync.RWMutex
}

// NewBuilderFactory creates a new builder factory with the specified configuration.
func NewBuilderFactory(config *Config) *BuilderFactory {
	if config == nil {
		config = GetDefaultConfig()
	}

	return &BuilderFactory{
		config: config,
	}
}

// GetGlobalBuilderFactory returns a global builder factory instance.
// The factory uses the global configuration and is thread-safe.
func GetGlobalBuilderFactory() *BuilderFactory {
	return NewBuilderFactory(GetGlobalConfig())
}

// CreateBuilderWithConfig creates a builder for the specified build type with configuration.
func (bf *BuilderFactory) CreateBuilderWithConfig(build container.Build) (ConfigurableBuilder, error) {
	bf.configMutex.RLock()
	config := bf.config
	bf.configMutex.RUnlock()

	switch build.BuildType {
	case container.GoLang:
		return bf.createGoBuilder(build, config)
	case container.Maven:
		return bf.createMavenBuilder(build, config)
	case container.Python:
		return bf.createPythonBuilder(build, config)
	case container.Generic:
		return bf.createProtobufBuilder(build, config)
	default:
		return nil, fmt.Errorf("unsupported build type: %s", build.BuildType)
	}
}

// SetConfig updates the factory's configuration.
// This affects all builders created after this call.
func (bf *BuilderFactory) SetConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if err := ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	bf.configMutex.Lock()
	bf.config = config
	bf.configMutex.Unlock()

	return nil
}

// GetConfig returns the current factory configuration.
func (bf *BuilderFactory) GetConfig() *Config {
	bf.configMutex.RLock()
	defer bf.configMutex.RUnlock()
	return bf.config
}

// CreateBuildConfigFromContainer creates a container.Build with configuration overrides.
// This function bridges the old container.Build structure with the new configuration system.
func (bf *BuilderFactory) CreateBuildConfigFromContainer(build container.Build) container.Build {
	bf.configMutex.RLock()
	config := bf.config
	bf.configMutex.RUnlock()

	// Apply configuration overrides to the container build
	newBuild := build

	// Apply container configuration
	if config.Container.Registry != "" {
		newBuild.Registry = config.Container.Registry
	}

	// Apply environment-specific overrides
	if config.Environment.Type != "" {
		newBuild.Env = config.Environment.Type
	}

	// Apply runtime configuration
	switch config.Container.Runtime.Type {
	case "docker":
		newBuild.Runtime = "docker"
	case "podman":
		newBuild.Runtime = "podman"
	}

	return newBuild
}

// createGoBuilder creates a Go builder with configuration.
func (bf *BuilderFactory) createGoBuilder(build container.Build, config *Config) (ConfigurableBuilder, error) {
	// Create Go-specific configuration from global config
	goConfig := config.Language.Go

	// Apply Go configuration to build
	newBuild := bf.CreateBuildConfigFromContainer(build)

	// Create a configurable Go builder wrapper
	builder := &ConfigurableGoBuilder{
		build:    newBuild,
		config:   config,
		goConfig: goConfig,
	}

	return builder, nil
}

// createMavenBuilder creates a Maven builder with configuration.
func (bf *BuilderFactory) createMavenBuilder(build container.Build, config *Config) (ConfigurableBuilder, error) {
	mavenConfig := config.Language.Maven
	newBuild := bf.CreateBuildConfigFromContainer(build)

	builder := &ConfigurableMavenBuilder{
		build:       newBuild,
		config:      config,
		mavenConfig: mavenConfig,
	}

	return builder, nil
}

// createPythonBuilder creates a Python builder with configuration.
func (bf *BuilderFactory) createPythonBuilder(build container.Build, config *Config) (ConfigurableBuilder, error) {
	pythonConfig := config.Language.Python
	newBuild := bf.CreateBuildConfigFromContainer(build)

	builder := &ConfigurablePythonBuilder{
		build:        newBuild,
		config:       config,
		pythonConfig: pythonConfig,
	}

	return builder, nil
}

// createProtobufBuilder creates a Protobuf builder with configuration.
func (bf *BuilderFactory) createProtobufBuilder(build container.Build, config *Config) (ConfigurableBuilder, error) {
	protobufConfig := config.Language.Protobuf
	newBuild := bf.CreateBuildConfigFromContainer(build)

	builder := &ConfigurableProtobufBuilder{
		build:          newBuild,
		config:         config,
		protobufConfig: protobufConfig,
	}

	return builder, nil
}

// ConfigurableGoBuilder wraps Go builders with configuration support.
type ConfigurableGoBuilder struct {
	config   *Config
	build    container.Build
	goConfig GoConfig
	mutex    sync.RWMutex
}

// SetConfig applies configuration to the Go builder.
func (gb *ConfigurableGoBuilder) SetConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	gb.mutex.Lock()
	gb.config = config
	gb.goConfig = config.Language.Go
	gb.mutex.Unlock()

	return nil
}

// GetConfig returns the current configuration.
func (gb *ConfigurableGoBuilder) GetConfig() *Config {
	gb.mutex.RLock()
	defer gb.mutex.RUnlock()
	return gb.config
}

// ValidateConfig validates the Go-specific configuration.
func (gb *ConfigurableGoBuilder) ValidateConfig() error {
	gb.mutex.RLock()
	goConfig := gb.goConfig
	gb.mutex.RUnlock()

	return ValidatePartialConfig(&goConfig)
}

// GetGoVersion returns the configured Go version.
func (gb *ConfigurableGoBuilder) GetGoVersion() string {
	gb.mutex.RLock()
	defer gb.mutex.RUnlock()
	return gb.goConfig.Version
}

// GetLintImage returns the configured Go lint image.
func (gb *ConfigurableGoBuilder) GetLintImage() string {
	gb.mutex.RLock()
	defer gb.mutex.RUnlock()
	return gb.goConfig.LintImage
}

// GetTestTimeout returns the configured test timeout.
func (gb *ConfigurableGoBuilder) GetTestTimeout() string {
	gb.mutex.RLock()
	defer gb.mutex.RUnlock()
	return gb.goConfig.TestTimeout.String()
}

// GetProjectMount returns the configured project mount path.
func (gb *ConfigurableGoBuilder) GetProjectMount() string {
	gb.mutex.RLock()
	defer gb.mutex.RUnlock()
	return gb.goConfig.ProjectMount
}

// GetOutputDir returns the configured output directory.
func (gb *ConfigurableGoBuilder) GetOutputDir() string {
	gb.mutex.RLock()
	defer gb.mutex.RUnlock()
	return gb.goConfig.OutputDir
}

// GetBuildTags returns the configured build tags.
func (gb *ConfigurableGoBuilder) GetBuildTags() []string {
	gb.mutex.RLock()
	defer gb.mutex.RUnlock()
	return gb.goConfig.BuildTags
}

// GetCoverageMode returns the configured coverage mode.
func (gb *ConfigurableGoBuilder) GetCoverageMode() string {
	gb.mutex.RLock()
	defer gb.mutex.RUnlock()
	return gb.goConfig.CoverageMode
}

// ConfigurableMavenBuilder wraps Maven builders with configuration support.
type ConfigurableMavenBuilder struct {
	mavenConfig MavenConfig
	config      *Config
	build       container.Build
	mutex       sync.RWMutex
}

// SetConfig applies configuration to the Maven builder.
func (mb *ConfigurableMavenBuilder) SetConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	mb.mutex.Lock()
	mb.config = config
	mb.mavenConfig = config.Language.Maven
	mb.mutex.Unlock()

	return nil
}

// GetConfig returns the current configuration.
func (mb *ConfigurableMavenBuilder) GetConfig() *Config {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return mb.config
}

// ValidateConfig validates the Maven-specific configuration.
func (mb *ConfigurableMavenBuilder) ValidateConfig() error {
	mb.mutex.RLock()
	mavenConfig := mb.mavenConfig
	mb.mutex.RUnlock()

	return ValidatePartialConfig(&mavenConfig)
}

// GetProdImage returns the configured Maven production image.
func (mb *ConfigurableMavenBuilder) GetProdImage() string {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return mb.mavenConfig.ProdImage
}

// GetCacheLocation returns the configured Maven cache location.
func (mb *ConfigurableMavenBuilder) GetCacheLocation() string {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return mb.mavenConfig.CacheLocation
}

// GetJavaVersion returns the configured Java version.
func (mb *ConfigurableMavenBuilder) GetJavaVersion() string {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return mb.mavenConfig.JavaVersion
}

// GetJavaOpts returns the configured Java options.
func (mb *ConfigurableMavenBuilder) GetJavaOpts() string {
	mb.mutex.RLock()
	defer mb.mutex.RUnlock()
	return mb.mavenConfig.JavaOpts
}

// ConfigurablePythonBuilder wraps Python builders with configuration support.
type ConfigurablePythonBuilder struct {
	config       *Config
	build        container.Build
	pythonConfig PythonConfig
	mutex        sync.RWMutex
}

// SetConfig applies configuration to the Python builder.
func (pb *ConfigurablePythonBuilder) SetConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	pb.mutex.Lock()
	pb.config = config
	pb.pythonConfig = config.Language.Python
	pb.mutex.Unlock()

	return nil
}

// GetConfig returns the current configuration.
func (pb *ConfigurablePythonBuilder) GetConfig() *Config {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.config
}

// ValidateConfig validates the Python-specific configuration.
func (pb *ConfigurablePythonBuilder) ValidateConfig() error {
	pb.mutex.RLock()
	pythonConfig := pb.pythonConfig
	pb.mutex.RUnlock()

	return ValidatePartialConfig(&pythonConfig)
}

// GetBaseImage returns the configured Python base image.
func (pb *ConfigurablePythonBuilder) GetBaseImage() string {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.pythonConfig.BaseImage
}

// GetCacheLocation returns the configured Python cache location.
func (pb *ConfigurablePythonBuilder) GetCacheLocation() string {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.pythonConfig.CacheLocation
}

// IsUVEnabled returns whether UV package manager is enabled.
func (pb *ConfigurablePythonBuilder) IsUVEnabled() bool {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.pythonConfig.UVEnabled
}

// GetUVCacheDir returns the configured UV cache directory.
func (pb *ConfigurablePythonBuilder) GetUVCacheDir() string {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.pythonConfig.UVCacheDir
}

// ConfigurableProtobufBuilder wraps Protobuf builders with configuration support.
type ConfigurableProtobufBuilder struct {
	protobufConfig ProtobufConfig
	config         *Config
	build          container.Build
	mutex          sync.RWMutex
}

// SetConfig applies configuration to the Protobuf builder.
func (pb *ConfigurableProtobufBuilder) SetConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	pb.mutex.Lock()
	pb.config = config
	pb.protobufConfig = config.Language.Protobuf
	pb.mutex.Unlock()

	return nil
}

// GetConfig returns the current configuration.
func (pb *ConfigurableProtobufBuilder) GetConfig() *Config {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.config
}

// ValidateConfig validates the Protobuf-specific configuration.
func (pb *ConfigurableProtobufBuilder) ValidateConfig() error {
	pb.mutex.RLock()
	protobufConfig := pb.protobufConfig
	pb.mutex.RUnlock()

	return ValidatePartialConfig(&protobufConfig)
}

// GetBaseImage returns the configured Protobuf base image.
func (pb *ConfigurableProtobufBuilder) GetBaseImage() string {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.protobufConfig.BaseImage
}

// GetScriptPath returns the configured script path.
func (pb *ConfigurableProtobufBuilder) GetScriptPath() string {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.protobufConfig.ScriptPath
}

// GetOutputDir returns the configured output directory.
func (pb *ConfigurableProtobufBuilder) GetOutputDir() string {
	pb.mutex.RLock()
	defer pb.mutex.RUnlock()
	return pb.protobufConfig.OutputDir
}

// ConfigAwareBaseBuilder provides a base implementation for configuration-aware builders.
// This can be embedded by language-specific builders to provide common configuration functionality.
type ConfigAwareBaseBuilder struct {
	config *Config
	mutex  sync.RWMutex
}

// NewConfigAwareBaseBuilder creates a new configuration-aware base builder.
func NewConfigAwareBaseBuilder(config *Config) *ConfigAwareBaseBuilder {
	if config == nil {
		config = GetDefaultConfig()
	}

	return &ConfigAwareBaseBuilder{
		config: config,
	}
}

// SetConfig sets the configuration for the base builder.
func (cb *ConfigAwareBaseBuilder) SetConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	cb.mutex.Lock()
	cb.config = config
	cb.mutex.Unlock()

	return nil
}

// GetConfig returns the current configuration.
func (cb *ConfigAwareBaseBuilder) GetConfig() *Config {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.config
}

// GetContainerTimeouts returns configured container timeouts.
func (cb *ConfigAwareBaseBuilder) GetContainerTimeouts() TimeoutConfig {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.config.Container.Timeouts
}

// GetVolumeConfig returns configured volume settings.
func (cb *ConfigAwareBaseBuilder) GetVolumeConfig() VolumeConfig {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.config.Container.Volumes
}

// GetSecurityConfig returns configured security settings.
func (cb *ConfigAwareBaseBuilder) GetSecurityConfig() SecurityConfig {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.config.Security
}

// GetCacheConfig returns configured cache settings.
func (cb *ConfigAwareBaseBuilder) GetCacheConfig() CacheConfig {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.config.Cache
}

// IsEnvironment checks if the current environment matches the specified type.
func (cb *ConfigAwareBaseBuilder) IsEnvironment(env container.EnvType) bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.config.Environment.Type == env
}

// GetEnvironmentProfile returns the configuration profile for the current environment.
func (cb *ConfigAwareBaseBuilder) GetEnvironmentProfile() EnvironmentProfile {
	cb.mutex.RLock()
	config := cb.config
	cb.mutex.RUnlock()

	switch config.Environment.Type {
	case container.LocalEnv:
		return config.Environment.Profiles.Local
	case container.BuildEnv:
		return config.Environment.Profiles.Build
	case container.ProdEnv:
		return config.Environment.Profiles.Production
	default:
		return config.Environment.Profiles.Build // Default fallback
	}
}

// ValidateConfig validates the base configuration.
func (cb *ConfigAwareBaseBuilder) ValidateConfig() error {
	cb.mutex.RLock()
	config := cb.config
	cb.mutex.RUnlock()

	return ValidateConfig(config)
}

// ConfigUpdateCallback defines a function type for configuration update notifications.
type ConfigUpdateCallback func(oldConfig, newConfig *Config) error

// BuilderFactoryWithCallbacks extends BuilderFactory with configuration change notifications.
type BuilderFactoryWithCallbacks struct {
	*BuilderFactory
	callbacks     []ConfigUpdateCallback
	callbackMutex sync.RWMutex
}

// NewBuilderFactoryWithCallbacks creates a new builder factory with callback support.
func NewBuilderFactoryWithCallbacks(config *Config) *BuilderFactoryWithCallbacks {
	return &BuilderFactoryWithCallbacks{
		BuilderFactory: NewBuilderFactory(config),
		callbacks:      make([]ConfigUpdateCallback, 0),
	}
}

// AddConfigUpdateCallback adds a callback function that will be called when configuration changes.
func (bf *BuilderFactoryWithCallbacks) AddConfigUpdateCallback(callback ConfigUpdateCallback) {
	bf.callbackMutex.Lock()
	bf.callbacks = append(bf.callbacks, callback)
	bf.callbackMutex.Unlock()
}

// SetConfig updates the configuration and notifies all registered callbacks.
func (bf *BuilderFactoryWithCallbacks) SetConfig(newConfig *Config) error {
	bf.configMutex.Lock()
	oldConfig := bf.config
	bf.configMutex.Unlock()

	// Validate the new configuration first
	if err := ValidateConfig(newConfig); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Notify callbacks before the change
	bf.callbackMutex.RLock()
	callbacks := make([]ConfigUpdateCallback, len(bf.callbacks))
	copy(callbacks, bf.callbacks)
	bf.callbackMutex.RUnlock()

	for _, callback := range callbacks {
		if err := callback(oldConfig, newConfig); err != nil {
			return fmt.Errorf("configuration update callback failed: %w", err)
		}
	}

	// Apply the configuration change
	return bf.BuilderFactory.SetConfig(newConfig)
}
