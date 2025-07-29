package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/containifyci/engine-ci/pkg/container"
)

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, "1.0", config.Version)
	assert.Equal(t, "1.24.2", config.Language.Go.Version)
	assert.Equal(t, "golangci/golangci-lint:v2.1.2", config.Language.Go.LintImage)
	assert.Equal(t, "registry.access.redhat.com/ubi8/openjdk-17:latest", config.Language.Maven.ProdImage)
	assert.Equal(t, "python:3.11-slim-bookworm", config.Language.Python.BaseImage)
	assert.Equal(t, 30*time.Second, config.Container.Timeouts.ContainerStart)
	assert.Equal(t, 10*time.Second, config.Container.Timeouts.ContainerStop)
	assert.True(t, config.Cache.Enabled)
	assert.True(t, config.Security.UserManagement.CreateNonRootUser)
}

func TestGetEnvironmentDefaults(t *testing.T) {
	tests := []struct {
		name        string
		env         container.EnvType
		expectLevel string
		expectPull  string
	}{
		{
			name:        "local environment",
			env:         container.LocalEnv,
			expectLevel: "debug",
			expectPull:  "never",
		},
		{
			name:        "build environment", 
			env:         container.BuildEnv,
			expectLevel: "info",
			expectPull:  "if_not_present",
		},
		{
			name:        "production environment",
			env:         container.ProdEnv,
			expectLevel: "warn",
			expectPull:  "always",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetEnvironmentDefaults(tt.env)
			assert.Equal(t, tt.env, config.Environment.Type)
			assert.Equal(t, tt.expectLevel, config.Logging.Level)
			assert.Equal(t, tt.expectPull, config.Container.Images.PullPolicy)
		})
	}
}

func TestLoadConfigFromYAML(t *testing.T) {
	// Create a temporary YAML config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")
	
	yamlContent := `
version: "1.1"
language:
  go:
    version: "1.25.0"
    lint_image: "golangci/golangci-lint:v2.2.0"
    test_timeout: "5m"
container:
  timeouts:
    container_start: "45s"
    build: "2h"
cache:
  enabled: false
`
	
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)
	
	// Load configuration from file
	config, err := LoadConfigFromFile(configFile)
	require.NoError(t, err)
	
	assert.Equal(t, "1.1", config.Version)
	assert.Equal(t, "1.25.0", config.Language.Go.Version)
	assert.Equal(t, "golangci/golangci-lint:v2.2.0", config.Language.Go.LintImage)
	assert.Equal(t, 5*time.Minute, config.Language.Go.TestTimeout)
	assert.Equal(t, 45*time.Second, config.Container.Timeouts.ContainerStart)
	assert.Equal(t, 2*time.Hour, config.Container.Timeouts.Build)
	assert.False(t, config.Cache.Enabled)
}

func TestLoadConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	envVars := map[string]string{
		"ENGINE_CI_LANGUAGE_GO_VERSION":     "1.26.0",
		"ENGINE_CI_LANGUAGE_GO_LINT_IMAGE":  "golangci/golangci-lint:v2.3.0",
		"ENGINE_CI_CONTAINER_REGISTRY":      "my-registry.com",
		"ENGINE_CI_CACHE_ENABLED":           "false",
		"ENGINE_CI_LOGGING_LEVEL":           "debug",
	}
	
	// Set environment variables
	for key, value := range envVars {
		os.Setenv(key, value)
	}
	
	// Clean up environment variables after test
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()
	
	// Load configuration
	config := GetDefaultConfig()
	err := LoadFromEnvironmentVariables(config)
	require.NoError(t, err)
	
	assert.Equal(t, "1.26.0", config.Language.Go.Version)
	assert.Equal(t, "golangci/golangci-lint:v2.3.0", config.Language.Go.LintImage)
	assert.Equal(t, "my-registry.com", config.Container.Registry)
	assert.False(t, config.Cache.Enabled)
	assert.Equal(t, "debug", config.Logging.Level)
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name:      "valid default config",
			config:    GetDefaultConfig(),
			expectErr: false,
		},
		{
			name: "invalid go version",
			config: func() *Config {
				config := GetDefaultConfig()
				config.Language.Go.Version = "invalid-version"
				return config
			}(),
			expectErr: true,
		},
		{
			name: "empty required field",
			config: func() *Config {
				config := GetDefaultConfig()
				config.Language.Go.ProjectMount = ""
				return config
			}(),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuilderFactory(t *testing.T) {
	config := GetDefaultConfig()
	factory := NewBuilderFactory(config)
	
	assert.NotNil(t, factory)
	assert.Equal(t, config, factory.GetConfig())
	
	// Test creating a Go builder
	build := container.Build{
		BuildType: container.GoLang,
		App:       "test-app",
	}
	
	builder, err := factory.CreateBuilderWithConfig(build)
	require.NoError(t, err)
	assert.NotNil(t, builder)
	
	// Test that builder has the configuration
	assert.Equal(t, config, builder.GetConfig())
	
	// Test validation
	err = builder.ValidateConfig()
	assert.NoError(t, err)
}

func TestConfigurableGoBuilder(t *testing.T) {
	config := GetDefaultConfig()
	build := container.Build{
		BuildType: container.GoLang,
		App:       "test-app",
	}
	
	builder := &ConfigurableGoBuilder{
		build:    build,
		config:   config,
		goConfig: config.Language.Go,
	}
	
	assert.Equal(t, "1.24.2", builder.GetGoVersion())
	assert.Equal(t, "golangci/golangci-lint:v2.1.2", builder.GetLintImage())
	assert.Equal(t, "/src", builder.GetProjectMount())
	assert.Equal(t, "/out/", builder.GetOutputDir())
	assert.Equal(t, "2m0s", builder.GetTestTimeout())
	assert.Equal(t, "text", builder.GetCoverageMode())
	
	// Test configuration update
	newConfig := GetDefaultConfig()
	newConfig.Language.Go.Version = "1.25.0"
	newConfig.Language.Go.TestTimeout = 5 * time.Minute
	
	err := builder.SetConfig(newConfig)
	require.NoError(t, err)
	
	assert.Equal(t, "1.25.0", builder.GetGoVersion())
	assert.Equal(t, "5m0s", builder.GetTestTimeout())
}

func TestMergeWithDefaults(t *testing.T) {
	// Create a partial configuration
	partialConfig := &Config{
		Version: "1.5",
		Language: LanguageConfig{
			Go: GoConfig{
				Version:   "1.25.0",
				LintImage: "custom/lint:latest",
			},
		},
	}
	
	// Merge with defaults
	mergedConfig := MergeWithDefaults(partialConfig)
	
	// Should have custom values
	assert.Equal(t, "1.5", mergedConfig.Version)
	assert.Equal(t, "1.25.0", mergedConfig.Language.Go.Version)
	assert.Equal(t, "custom/lint:latest", mergedConfig.Language.Go.LintImage)
	
	// Should have default values for unspecified fields
	assert.Equal(t, "registry.access.redhat.com/ubi8/openjdk-17:latest", mergedConfig.Language.Maven.ProdImage)
	assert.True(t, mergedConfig.Cache.Enabled)
}

func TestEnvironmentVariableValidation(t *testing.T) {
	// Set invalid environment variables
	invalidEnvVars := map[string]string{
		"ENGINE_CI_LANGUAGE_GO_TEST_TIMEOUT": "invalid-duration",
		"ENGINE_CI_CACHE_ENABLED":            "not-a-boolean",
		"ENGINE_CI_LANGUAGE_GO_VERSION":      "not-a-version",
	}
	
	for key, value := range invalidEnvVars {
		os.Setenv(key, value)
	}
	
	defer func() {
		for key := range invalidEnvVars {
			os.Unsetenv(key)
		}
	}()
	
	issues := ValidateEnvironmentVariables()
	assert.NotEmpty(t, issues)
	assert.Contains(t, issues[0], "invalid duration format")
	assert.Contains(t, issues[1], "invalid boolean format")
	assert.Contains(t, issues[2], "invalid version format")
}

func TestConfigValueGetSet(t *testing.T) {
	config := GetDefaultConfig()
	
	// Test getting values
	value, err := GetConfigValue(config, "language.go.version")
	require.NoError(t, err)
	assert.Equal(t, "1.24.2", value)
	
	value, err = GetConfigValue(config, "container.timeouts.build")
	require.NoError(t, err)
	assert.Equal(t, 1*time.Hour, value)
	
	// Test setting values
	err = SetConfigValue(config, "language.go.version", "1.25.0")
	require.NoError(t, err)
	
	newValue, err := GetConfigValue(config, "language.go.version")
	require.NoError(t, err)
	assert.Equal(t, "1.25.0", newValue)
}

func TestGlobalConfig(t *testing.T) {
	// Test getting global config
	globalConfig := GetGlobalConfig()
	assert.NotNil(t, globalConfig)
	
	// Test setting global config
	newConfig := GetDefaultConfig()
	newConfig.Version = "2.0"
	
	SetGlobalConfig(newConfig)
	
	retrievedConfig := GetGlobalConfig()
	assert.Equal(t, "2.0", retrievedConfig.Version)
}

// Benchmark tests for performance validation
func BenchmarkLoadDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetDefaultConfig()
	}
}

func BenchmarkValidateConfig(b *testing.B) {
	config := GetDefaultConfig()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_ = ValidateConfig(config)
	}
}

func BenchmarkGlobalConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetGlobalConfig()
	}
}

// Helper functions for testing
func createTestConfig() *Config {
	return &Config{
		Version: "test",
		Language: LanguageConfig{
			Go: GoConfig{
				Version:      "1.24.2",
				LintImage:    "test/lint:latest",
				TestTimeout:  2 * time.Minute,
				BuildTimeout: 10 * time.Minute,
				ProjectMount: "/src",
				OutputDir:    "/out",
			},
		},
		Container: ContainerConfig{
			Timeouts: TimeoutConfig{
				ContainerStart: 30 * time.Second,
				ContainerStop:  10 * time.Second,
				Build:          1 * time.Hour,
				Test:           2 * time.Minute,
			},
		},
		Cache: CacheConfig{
			Enabled: true,
		},
	}
}