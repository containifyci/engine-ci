# Configuration System Migration Guide

This guide explains how to migrate existing hardcoded values in engine-ci to use the new centralized configuration system.

## Overview

The new configuration system replaces hardcoded constants throughout the codebase with configurable parameters that can be set via:
1. CLI flags (highest priority)
2. Environment variables
3. Configuration files (YAML/JSON)
4. Default values (lowest priority)

## Migration Steps

### Step 1: Replace Hardcoded Constants

**Before (golang packages):**
```go
const (
    DEFAULT_GO = "1.24.2"
    LINT_IMAGE = "golangci/golangci-lint:v2.1.2" 
    PROJ_MOUNT = "/src"
    OUT_DIR    = "/out/"
)

func (c *GoContainer) Pull() error {
    imageTag := fmt.Sprintf("golang:%s", DEFAULT_GO)
    return c.Container.Pull(imageTag, "alpine:latest")
}
```

**After:**
```go
func (c *ConfigurableGoBuilder) Pull() error {
    config := c.GetConfig()
    imageTag := fmt.Sprintf("golang:%s", config.Language.Go.Version)
    return c.Container.Pull(imageTag, "alpine:latest")
}
```

### Step 2: Update Container Creation

**Before:**
```go
func New(build container.Build) *GoContainer {
    return &GoContainer{
        Container: container.New(build),
    }
}
```

**After:**
```go
func NewWithConfig(build container.Build, config *config.Config) *ConfigurableGoBuilder {
    factory := config.NewBuilderFactory(config)
    builder, err := factory.CreateBuilderWithConfig(build)
    if err != nil {
        // Handle error
    }
    return builder.(*ConfigurableGoBuilder)
}
```

### Step 3: Update CLI Commands

**Before:**
```go
var buildCmd = &cobra.Command{
    Use: "build",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Hardcoded values used throughout
        return runBuild(args)
    },
}
```

**After:**
```go
var buildCmd = &cobra.Command{
    Use: "build",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Load configuration with CLI flag overrides
        cfg := config.GetGlobalConfig()
        if err := config.ApplyFlagsToConfig(cmd, cfg); err != nil {
            return err
        }
        return runBuildWithConfig(cfg, args)
    },
}

func init() {
    cfg := config.GetDefaultConfig()
    config.AddConfigFlags(buildCmd, cfg)
}
```

### Step 4: Update Timeout Usage

**Before:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
err := c.client().StartContainer(ctx, c.ID)
```

**After:**
```go
config := c.GetConfig()
timeout := config.Container.Timeouts.ContainerStart
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
err := c.client().StartContainer(ctx, c.ID)
```

### Step 5: Update Cache Paths

**Before:**
```go
folder, err := filepath.Abs(filepath.Join(dir, "/.trivy/cache"))
```

**After:**
```go
config := c.GetConfig()
folder := config.Cache.Directories.Trivy
```

## Configuration Examples

### YAML Configuration File (engine-ci.yaml)
```yaml
version: "1.0"

language:
  go:
    version: "1.25.0"
    lint_image: "golangci/golangci-lint:v2.2.0"
    test_timeout: "5m"
    coverage_mode: "binary"
    build_tags: ["integration"]
  
  maven:
    prod_image: "registry.access.redhat.com/ubi8/openjdk-21:latest"
    java_version: "21"
    cache_location: "/opt/maven/.m2"
  
  python:
    base_image: "python:3.12-slim"
    version: "3.12"
    uv_enabled: true

container:
  registry: "my-registry.com"
  timeouts:
    container_start: "60s"
    build: "2h"
    test: "30m"
  resources:
    memory_limit: "4GB"
    cpu_limit: "4"

cache:
  enabled: true
  cleanup_policy: "7d"
  directories:
    go: "/opt/cache/go"
    maven: "/opt/cache/maven"
    python: "/opt/cache/pip"

security:
  user_management:
    create_non_root_user: true
    uid: "1000"
    gid: "1000"
```

### Environment Variables
```bash
# Go Configuration
export ENGINE_CI_LANGUAGE_GO_VERSION=1.25.0
export ENGINE_CI_LANGUAGE_GO_LINT_IMAGE=golangci/golangci-lint:v2.2.0

# Container Configuration
export ENGINE_CI_CONTAINER_REGISTRY=my-registry.com
export ENGINE_CI_CONTAINER_TIMEOUTS_BUILD=2h

# Cache Configuration
export ENGINE_CI_CACHE_ENABLED=true
export ENGINE_CI_CACHE_DIRECTORIES_GO=/opt/cache/go
```

### CLI Flags
```bash
# Override specific configuration values
engine-ci build --go-version=1.25.0 --build-timeout=2h --verbose

# Use custom configuration file
engine-ci build --config=./custom-config.yaml

# Production environment settings
engine-ci build --log-level=warn --create-non-root-user
```

## Builder Factory Usage

### Creating Language-Specific Builders

```go
// Get global configuration
config := config.GetGlobalConfig()

// Create factory
factory := config.NewBuilderFactory(config)

// Create Go builder
build := container.Build{
    BuildType: container.GoLang,
    App:       "my-app",
}

goBuilder, err := factory.CreateBuilderWithConfig(build)
if err != nil {
    return err
}

// Use builder with configuration
version := goBuilder.(*config.ConfigurableGoBuilder).GetGoVersion()
lintImage := goBuilder.(*config.ConfigurableGoBuilder).GetLintImage()
```

### Updating Configuration at Runtime

```go
// Load new configuration
newConfig, err := config.LoadConfigFromFile("updated-config.yaml")
if err != nil {
    return err
}

// Update factory
factory.SetConfig(newConfig)

// All new builders will use the updated configuration
```

## Testing with Configuration

### Unit Tests
```go
func TestGoBuilderWithConfig(t *testing.T) {
    // Create test configuration
    testConfig := &config.Config{
        Language: config.LanguageConfig{
            Go: config.GoConfig{
                Version:   "1.24.2",
                LintImage: "test/lint:latest",
            },
        },
    }
    
    // Create builder with test config
    factory := config.NewBuilderFactory(testConfig)
    build := container.Build{BuildType: container.GoLang}
    
    builder, err := factory.CreateBuilderWithConfig(build)
    require.NoError(t, err)
    
    goBuilder := builder.(*config.ConfigurableGoBuilder)
    assert.Equal(t, "1.24.2", goBuilder.GetGoVersion())
    assert.Equal(t, "test/lint:latest", goBuilder.GetLintImage())
}
```

### Integration Tests
```go
func TestConfigurationPrecedence(t *testing.T) {
    // Test that CLI flags override config file values
    tempDir := t.TempDir()
    configFile := filepath.Join(tempDir, "test.yaml")
    
    // Create config file with base values
    yamlContent := `
language:
  go:
    version: "1.24.0"
`
    err := os.WriteFile(configFile, []byte(yamlContent), 0644)
    require.NoError(t, err)
    
    // Load config from file
    config, err := config.LoadConfigFromFile(configFile)
    require.NoError(t, err)
    assert.Equal(t, "1.24.0", config.Language.Go.Version)
    
    // Simulate CLI flag override
    config.Language.Go.Version = "1.25.0"
    assert.Equal(t, "1.25.0", config.Language.Go.Version)
}
```

## Validation and Error Handling

### Configuration Validation
```go
// Validate configuration before use
config, err := config.LoadConfig()
if err != nil {
    return fmt.Errorf("failed to load configuration: %w", err)
}

if err := config.ValidateConfig(config); err != nil {
    return fmt.Errorf("invalid configuration: %w", err)
}

// Environment-specific validation
if err := config.ValidateForEnvironment(config, container.ProdEnv); err != nil {
    return fmt.Errorf("production validation failed: %w", err)
}
```

### Error Messages
The validation system provides clear, actionable error messages:
```
configuration validation failed:
  - Language.Go.Version is required
  - Language.Go.LintImage must be a valid Docker image name (got: invalid-image-name)
  - Container.Volumes.SourceMount must be an absolute path (got: relative/path)
```

## Migration Checklist

### For Each Package:
- [ ] Identify all hardcoded constants
- [ ] Replace constants with configuration access
- [ ] Update constructors to accept configuration
- [ ] Add configuration validation for package-specific requirements
- [ ] Update unit tests to use configuration
- [ ] Update integration tests to verify configuration precedence

### For CLI Commands:
- [ ] Add configuration flags using `config.AddConfigFlags()`
- [ ] Apply flag values using `config.ApplyFlagsToConfig()`
- [ ] Load configuration at command startup
- [ ] Validate configuration for command context
- [ ] Update help text to reference configuration options

### For Container Operations:
- [ ] Replace hardcoded timeouts with configured values
- [ ] Use configured image names and tags
- [ ] Apply configured resource limits
- [ ] Use configured mount paths and volumes
- [ ] Apply configured security settings (user management)

## Backward Compatibility

The configuration system maintains backward compatibility by:

1. **Default Values**: All defaults match existing hardcoded values
2. **Gradual Migration**: Existing code continues to work while being migrated
3. **Environment Variables**: Existing environment variables are still supported
4. **CLI Flags**: New flags supplement existing ones without breaking changes

## Performance Considerations

- Configuration loading is cached and thread-safe
- Global configuration access is optimized for concurrent use
- Configuration validation is performed once at startup
- Builder factories cache configuration for performance

## Troubleshooting

### Common Issues

1. **Configuration Not Loading**
   - Check file path and permissions
   - Verify YAML/JSON syntax
   - Check environment variable names

2. **Validation Errors**
   - Review error messages for specific field issues
   - Check required fields are provided
   - Verify format requirements (semantic versions, paths, etc.)

3. **Environment Variable Issues**
   - Ensure correct naming convention: `ENGINE_CI_SECTION_FIELD`
   - Check for typos in variable names
   - Verify boolean and duration formats

4. **CLI Flag Conflicts**
   - Check for duplicate flag definitions
   - Verify flag names don't conflict with existing flags
   - Ensure proper flag types (string, bool, duration)

For additional help, run:
```bash
engine-ci config validate    # Validate current configuration
engine-ci config show       # Display current configuration
engine-ci config env         # List all environment variables
```