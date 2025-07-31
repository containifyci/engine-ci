# New Architecture Design - Maintainability Enhancement

## Overview
This document outlines the new architecture designed to eliminate code duplication, centralize configuration, and improve maintainability for the engine-ci project.

## Core Principles

1. **Single Responsibility**: Each component has one clear purpose
2. **Don't Repeat Yourself**: Common patterns are abstracted into reusable components
3. **Configuration as Code**: All settings centralized and validated
4. **Dependency Injection**: Components receive dependencies through interfaces
5. **Error Propagation**: Errors bubble up rather than causing os.Exit()

## New Interface Hierarchy

### 1. LanguageBuilder Interface
```go
// LanguageBuilder defines the contract for all language-specific builders
type LanguageBuilder interface {
    // Core identification
    Name() string
    IsAsync() bool
    
    // Image management
    BaseImage() string
    Images() []string
    BuildImage() (string, error)
    
    // Container operations
    Pull() error
    Build() (string, error)
    
    // Configuration
    CacheLocation() string
    DefaultEnvironment() []string
    
    // Build script generation
    BuildScript() string
    
    // Lifecycle management
    PreBuild() error
    PostBuild() error
    
    // Validation
    Validate() error
}
```

### 2. BuildStep Interface
```go
// BuildStep represents a single step in the build pipeline
type BuildStep interface {
    Name() string
    Execute(ctx context.Context) error
    Dependencies() []string
    IsAsync() bool
    Timeout() time.Duration
    Validate() error
}
```

### 3. ConfigProvider Interface
```go
// ConfigProvider manages configuration for different components
type ConfigProvider interface {
    Get(key string) (interface{}, error)
    GetString(key string) (string, error)
    GetInt(key string) (int, error)
    GetBool(key string) (bool, error)
    GetDuration(key string) (time.Duration, error)
    Validate() error
}
```

### 4. CacheManager Interface
```go
// CacheManager handles caching strategies for different languages
type CacheManager interface {
    GetCacheDir(language string) (string, error)
    EnsureCacheDir(language string) error
    CleanCache(language string) error
    CacheKey(language string, dependencies []string) string
}
```

## New Package Structure

```
pkg/
├── build/
│   ├── interfaces.go         # Core interfaces (LanguageBuilder, BuildStep)
│   ├── orchestrator.go       # Enhanced build orchestration
│   ├── validation.go         # Build validation logic
│   ├── pipeline.go           # Pipeline management
│   └── context.go            # Build context management
├── config/
│   ├── config.go             # Centralized configuration
│   ├── provider.go           # Configuration provider implementation
│   ├── validation.go         # Configuration validation
│   ├── defaults.go           # Default values and constants
│   └── environment.go        # Environment variable handling
├── language/
│   ├── base.go               # BaseLanguageBuilder implementation
│   ├── cache.go              # Common caching strategies
│   ├── container.go          # Common container operations
│   ├── script.go             # Build script utilities
│   ├── errors.go             # Standardized error handling
│   └── validation.go         # Common validation logic
├── cache/
│   ├── manager.go            # Cache manager implementation
│   ├── strategies.go         # Different caching strategies
│   └── cleanup.go            # Cache cleanup utilities
└── [existing packages - updated to implement new interfaces]
```

## Configuration Schema

### Centralized Configuration Structure
```go
type Config struct {
    // Language-specific settings
    Languages map[string]LanguageConfig `yaml:"languages" validate:"required"`
    
    // Container settings
    Container ContainerConfig `yaml:"container" validate:"required"`
    
    // Cache settings
    Cache CacheConfig `yaml:"cache" validate:"required"`
    
    // Build settings
    Build BuildConfig `yaml:"build" validate:"required"`
    
    // Registry settings
    Registry RegistryConfig `yaml:"registry" validate:"required"`
}

type LanguageConfig struct {
    BaseImage     string            `yaml:"base_image" validate:"required"`
    CacheLocation string            `yaml:"cache_location" validate:"required"`
    Environment   map[string]string `yaml:"environment"`
    BuildTimeout  time.Duration     `yaml:"build_timeout" validate:"min=1m,max=1h"`
    CustomArgs    []string          `yaml:"custom_args"`
}

type ContainerConfig struct {
    Runtime        string        `yaml:"runtime" validate:"oneof=docker podman"`
    PullTimeout    time.Duration `yaml:"pull_timeout" validate:"min=30s,max=10m"`
    BuildTimeout   time.Duration `yaml:"build_timeout" validate:"min=1m,max=2h"`
    DefaultUser    string        `yaml:"default_user"`
    NetworkMode    string        `yaml:"network_mode"`
    PlatformConfig PlatformConfig `yaml:"platform"`
}

type CacheConfig struct {
    BaseDir    string            `yaml:"base_dir" validate:"required"`
    MaxSize    string            `yaml:"max_size" validate:"required"`
    TTL        time.Duration     `yaml:"ttl" validate:"min=1h"`
    Strategies map[string]string `yaml:"strategies"`
}

type BuildConfig struct {
    Parallel      int           `yaml:"parallel" validate:"min=1,max=10"`
    DefaultTarget string        `yaml:"default_target"`
    Timeout       time.Duration `yaml:"timeout" validate:"min=1m,max=4h"`
    FailFast      bool          `yaml:"fail_fast"`
}
```

## Base Implementation Pattern

### BaseLanguageBuilder
```go
type BaseLanguageBuilder struct {
    name      string
    config    *LanguageConfig
    container *container.Container
    cache     CacheManager
    logger    *slog.Logger
}

func NewBaseLanguageBuilder(name string, config *LanguageConfig, 
                           container *container.Container, 
                           cache CacheManager) *BaseLanguageBuilder {
    return &BaseLanguageBuilder{
        name:      name,
        config:    config,
        container: container,
        cache:     cache,
        logger:    slog.With("component", "language-builder", "language", name),
    }
}

// Common implementations that can be used by all languages
func (b *BaseLanguageBuilder) Name() string {
    return b.name
}

func (b *BaseLanguageBuilder) IsAsync() bool {
    return false // Most languages are synchronous by default
}

func (b *BaseLanguageBuilder) BaseImage() string {
    return b.config.BaseImage
}

func (b *BaseLanguageBuilder) CacheLocation() string {
    return b.config.CacheLocation
}

func (b *BaseLanguageBuilder) DefaultEnvironment() []string {
    env := make([]string, 0, len(b.config.Environment))
    for k, v := range b.config.Environment {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    return env
}

func (b *BaseLanguageBuilder) PreBuild() error {
    // Common pre-build operations
    if err := b.cache.EnsureCacheDir(b.name); err != nil {
        return fmt.Errorf("failed to ensure cache directory: %w", err)
    }
    return nil
}

func (b *BaseLanguageBuilder) PostBuild() error {
    // Common post-build operations (cleanup, validation, etc.)
    return nil
}

func (b *BaseLanguageBuilder) Validate() error {
    // Common validation logic
    if b.name == "" {
        return errors.New("language name cannot be empty")
    }
    if b.config.BaseImage == "" {
        return errors.New("base image must be specified")
    }
    return nil
}

// Methods that must be implemented by specific languages
func (b *BaseLanguageBuilder) BuildScript() string {
    panic("BuildScript must be implemented by specific language builders")
}

func (b *BaseLanguageBuilder) Build() (string, error) {
    panic("Build must be implemented by specific language builders")
}
```

## Migration Strategy

### Phase 1: Infrastructure (Non-Breaking)
1. Create new packages (`config`, `language`, `cache`) alongside existing code
2. Implement interfaces and base classes
3. Add configuration management
4. Add comprehensive tests

### Phase 2: Language Package Updates (Breaking Changes)
1. Update one language package at a time
2. Maintain backward compatibility where possible
3. Use adapter pattern for existing consumers
4. Provide migration utilities

### Phase 3: Integration and Cleanup
1. Update build orchestration to use new interfaces
2. Remove duplicated code
3. Update CLI commands and documentation
4. Performance testing and optimization

## Error Handling Strategy

### Eliminate os.Exit() Pattern
Replace:
```go
if err != nil {
    slog.Error("Failed to build container", "error", err)
    os.Exit(1)
}
```

With:
```go
if err != nil {
    return fmt.Errorf("failed to build container: %w", err)
}
```

### Structured Error Types
```go
type BuildError struct {
    Operation string
    Language  string
    Cause     error
}

func (e *BuildError) Error() string {
    return fmt.Sprintf("build failed [%s:%s]: %v", e.Language, e.Operation, e.Cause)
}

func (e *BuildError) Unwrap() error {
    return e.Cause
}
```

## Configuration Management

### Default Configuration
```yaml
# Default engine-ci.yaml
languages:
  golang:
    base_image: "golang:1.24.2-alpine"
    cache_location: "/go/pkg/mod"
    build_timeout: "30m"
    environment:
      CGO_ENABLED: "0"
      GOOS: "linux"
      
  python:
    base_image: "python:3.11-slim-bookworm"
    cache_location: "/root/.cache/pip"
    build_timeout: "20m"
    environment:
      _PIP_USE_IMPORTLIB_METADATA: "0"
      UV_CACHE_DIR: "/root/.cache/pip"
      
  maven:
    base_image: "registry.access.redhat.com/ubi8/openjdk-17:latest"
    cache_location: "/root/.m2"
    build_timeout: "45m"

container:
  runtime: "docker"
  pull_timeout: "5m"
  build_timeout: "1h"
  default_user: "root"

cache:
  base_dir: "${HOME}/.containifyci/cache"
  max_size: "10GB"
  ttl: "24h"

build:
  parallel: 3
  timeout: "2h"
  fail_fast: true
```

## Benefits of New Architecture

1. **Reduced Duplication**: 70%+ code reduction through shared base classes
2. **Centralized Configuration**: Single source of truth for all settings
3. **Better Error Handling**: Proper error propagation and structured error types
4. **Improved Testing**: Interfaces enable better mocking and testing
5. **Enhanced Maintainability**: Clear separation of concerns and responsibilities
6. **Better Performance**: Caching strategies and parallel execution
7. **Developer Experience**: Clear interfaces and comprehensive documentation

## Implementation Priority

1. **High Priority**: Core interfaces and base implementations
2. **Medium Priority**: Configuration management and caching
3. **Low Priority**: Advanced features and optimizations

This architecture provides a solid foundation for long-term maintainability while preserving existing functionality.