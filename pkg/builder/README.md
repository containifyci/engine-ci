# Language Builder Interface

This package provides a unified interface for all language-specific container builders in the engine-ci project. It eliminates massive code duplication across language packages by providing common abstractions, utilities, and base implementations.

## Overview

The current codebase has 90%+ code duplication across language packages:
- `pkg/golang/alpine/golang.go` (~400 lines)
- `pkg/golang/debian/golang.go` (~309 lines)  
- `pkg/maven/maven.go` (~200+ lines)
- `pkg/python/python.go` (similar patterns)

This package solves the duplication problem by providing:

1. **Unified Interface**: `LanguageBuilder` interface that captures all common operations
2. **Common Utilities**: Shared functions for cache management, image building, container setup
3. **Base Implementation**: `BaseBuilder` struct with common functionality that can be embedded
4. **Factory Pattern**: Centralized builder creation and configuration injection
5. **Migration Path**: Backward compatibility and incremental migration support

## Architecture

```
pkg/builder/
├── interface.go      # Core LanguageBuilder interface
├── base.go          # BaseBuilder implementation  
├── factory.go       # Builder factory and registry
├── migration.go     # Migration examples and compatibility
├── common/
│   ├── utils.go     # Shared utility functions
│   └── types.go     # Common types and configurations
└── README.md        # This documentation
```

## Core Interface

```go
type LanguageBuilder interface {
    // Core lifecycle operations
    Name() string                        
    IsAsync() bool                       
    Pull() error                         
    Build() error                        
    Run() error                          
    Images() []string                    

    // Production and deployment operations
    Prod() error                         
    
    // Language-specific image management
    BuildIntermediateImage() error       
    IntermediateImage() string           
    
    // Configuration and script generation
    BuildScript() string                 
    CacheFolder() string                 
}
```

## Key Features

### 1. Extensible Interface Design

The interface supports all existing functionality while being extensible for new features:

- **LintableBuilder**: Extension for languages with linting support
- **AsyncLanguageBuilder**: Extension for asynchronous operations
- **BuildFactory**: Plugin-like architecture for new language support

### 2. Common Utilities

Shared functions eliminate duplication:

```go
// Checksum computation (used by all builders)
func ComputeChecksum(data []byte) string

// Image URI generation from Dockerfile
func ImageURIFromDockerfile(fs embed.FS, dockerfilePath, baseName, registry string) string

// Cache folder resolution with fallbacks
func CacheFolderFromEnv(envVars []string, defaultSubDir string) string

// Container volume setup
func SetupCommonVolumes(sourceDir, cacheDir, sourceMountPath, cacheMountPath string) []types.Volume
```

### 3. Base Implementation

`BaseBuilder` provides common functionality that can be embedded:

```go
type BaseBuilder struct {
    *container.Container
    Config     BuildConfiguration
    Defaults   common.LanguageDefaults
    // ... common fields
}

// Common methods implemented
func (b *BaseBuilder) Name() string
func (b *BaseBuilder) IsAsync() bool  
func (b *BaseBuilder) Images() []string
func (b *BaseBuilder) CacheFolder() string
func (b *BaseBuilder) SetupContainerEnvironment(opts *types.ContainerConfig)
func (b *BaseBuilder) ExecuteBuildContainer(imageTag, script string) error
// ... more common functionality
```

### 4. Configuration System

Centralized configuration management:

```go
type BuildConfiguration struct {
    Platform    types.Platform
    Registry    string
    Environment container.EnvType
    Verbose     bool
    App         string
    File        string
    Folder      string
    Image       string
    ImageTag    string
    Tags        []string
    Custom      container.Custom
}
```

### 5. Language Defaults

Predefined configurations for each language:

```go
func GetGoDefaults() LanguageDefaults {
    return LanguageDefaults{
        Language:        "golang",
        BuildType:       container.GoLang,
        BaseImage:       "golang:1.24.2-alpine",
        LintImage:       "golangci/golangci-lint:v2.1.2",
        SourceMount:     "/src",
        CacheMount:      "/go/pkg",
        DefaultEnv: map[string]string{
            "GOMODCACHE": "/go/pkg/",
            "GOCACHE":    "/go/pkg/build-cache",
        },
        RequiredFiles: []string{"go.mod"},
    }
}
```

## Usage Examples

### Creating a New Language Builder

```go
// 1. Define your builder struct
type GoBuilder struct {
    *BaseBuilder
    goVersion   string
    lintImage   string
    dockerFiles embed.FS
}

// 2. Implement required methods
func (g *GoBuilder) BuildScript() string {
    return buildscript.NewBuildScript(...).String()
}

func (g *GoBuilder) IntermediateImage() string {
    return common.ImageURIFromDockerfile(
        g.dockerFiles, "Dockerfilego", 
        fmt.Sprintf("golang-%s-alpine", g.goVersion),
        g.Config.Registry,
    )
}

// 3. Register with factory
func init() {
    RegisterBuilder(&BuilderRegistration{
        BuildType: container.GoLang,
        Name:      "golang",
        Constructor: func(build container.Build) (LanguageBuilder, error) {
            return NewGoBuilder(build, dockerFiles), nil
        },
        Features: BuilderFeatures{
            SupportsLinting:    true,
            SupportsProduction: true,
        },
    })
}
```

### Using the Factory

```go
// Create a builder for a specific language  
builder, err := CreateLanguageBuilder(container.GoLang, build)
if err != nil {
    return err
}

// Use the builder
err = builder.Run()
```

### Backward Compatibility

```go
// Legacy wrapper for existing code
type LegacyGoContainer struct {
    builder LanguageBuilder
    *container.Container
    // ... legacy fields
}

func NewLegacyGoContainer(build container.Build) *LegacyGoContainer {
    builder := NewGoBuilder(build, dockerFiles)
    return &LegacyGoContainer{
        builder:   builder,
        Container: container.New(build),
        // ... legacy field initialization
    }
}

// Delegate to new builder
func (l *LegacyGoContainer) Build() error {
    return l.builder.Build()
}
```

## Migration Strategy

### Phase 1.1: Interface Creation ✅

- [x] Create LanguageBuilder interface
- [x] Create common utilities and types  
- [x] Create BaseBuilder implementation
- [x] Create factory pattern
- [x] Create migration examples

### Phase 1.2: Language Implementation (Next)

- [ ] Create language-specific builder implementations
- [ ] Register builders with factory
- [ ] Add configuration injection system
- [ ] Create unit tests for each builder

### Phase 1.3: Package Migration (Future)

- [ ] Update existing packages to use new builders
- [ ] Maintain backward compatibility with legacy wrappers
- [ ] Remove duplicate code incrementally
- [ ] Update integration tests

### Phase 1.4: Legacy Cleanup (Future)

- [ ] Remove compatibility wrappers
- [ ] Update all callers to use new interface
- [ ] Remove duplicate implementations  
- [ ] Validate no functionality regression

## Benefits

### Code Reduction
- **Eliminates 90%+ duplication** across language packages
- **Centralizes common patterns** like SSH forwarding, volume mounting, environment setup
- **Reduces maintenance burden** with single implementation of shared functionality

### Extensibility  
- **Plugin architecture** for new language support
- **Interface segregation** allows optional features (linting, async)
- **Configuration injection** prepares for centralized config system

### Maintainability
- **Single source of truth** for common operations
- **Consistent behavior** across all language builders
- **Easier testing** with shared test utilities

### Performance
- **Reuses optimized implementations** like ComputeChecksum with memory tracking
- **Centralizes caching logic** for better cache hit rates
- **Reduces binary size** by eliminating duplicate code

## Integration with Existing Code

The new interface is designed to work with existing systems:

- **container.Build**: Configuration structure remains unchanged
- **build.Build**: Interface compatibility maintained  
- **types.ContainerConfig**: Container configuration unchanged
- **CRI packages**: Container runtime interface unchanged

## Future Enhancements

### Phase 2: Configuration System
- Centralized configuration loading
- Environment-specific overrides
- Validation and defaults management

### Phase 3: Advanced Features
- Pipeline orchestration
- Dependency management between builders
- Parallel build execution
- Build caching and optimization

### Phase 4: Extensibility
- Plugin system for external builders
- Custom build step injection
- Build result processing and artifacts

## Testing Strategy

### Unit Tests
- Interface compliance tests
- Common utility function tests  
- Base builder functionality tests
- Factory registration and creation tests

### Integration Tests
- End-to-end build pipeline tests
- Container creation and execution tests
- Image building and management tests
- Backward compatibility validation tests

### Performance Tests
- Memory usage optimization validation
- Build time comparison with existing implementations
- Cache effectiveness measurements
- Resource utilization monitoring

## Contributing

When adding new language support:

1. **Create builder implementation** using BaseBuilder
2. **Register with factory** in init() function
3. **Add language defaults** to common/types.go
4. **Create unit tests** for your builder
5. **Update documentation** with usage examples

When migrating existing packages:

1. **Create compatibility wrapper** for backward compatibility
2. **Delegate to new builder** incrementally  
3. **Update tests** to use new implementation
4. **Remove legacy code** after migration complete
5. **Validate no functionality regression**