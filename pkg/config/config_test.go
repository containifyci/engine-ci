package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	// Verify default values are set
	assert.NotNil(t, config)
	assert.NotNil(t, config.Languages)
	assert.NotNil(t, config.Container)
	assert.NotNil(t, config.Cache)
	assert.NotNil(t, config.Build)
	assert.NotNil(t, config.Registry)
	
	// Check specific language configurations
	pythonConfig, exists := config.Languages["python"]
	require.True(t, exists, "Python configuration should exist in defaults")
	assert.Equal(t, "python:3.11-slim-bookworm", pythonConfig.BaseImage)
	assert.Equal(t, "/root/.cache/pip", pythonConfig.CacheLocation)
	assert.True(t, pythonConfig.Enabled)
	
	golangConfig, exists := config.Languages["golang"]
	require.True(t, exists, "Golang configuration should exist in defaults")
	assert.Equal(t, "golang:1.24.2-alpine", golangConfig.BaseImage)
	assert.Equal(t, "/go/pkg/mod", golangConfig.CacheLocation)
	assert.True(t, golangConfig.Enabled)
}

func TestLanguageConfigDefaults(t *testing.T) {
	config := DefaultConfig()
	
	for name, langConfig := range config.Languages {
		t.Run(name, func(t *testing.T) {
			// Every language should have required fields
			assert.NotEmpty(t, langConfig.BaseImage, "BaseImage should not be empty")
			assert.NotEmpty(t, langConfig.CacheLocation, "CacheLocation should not be empty")
			assert.Greater(t, langConfig.BuildTimeout, time.Duration(0), "BuildTimeout should be positive")
			
			// Environment should be initialized (can be empty map)
			assert.NotNil(t, langConfig.Environment, "Environment should be initialized")
		})
	}
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		config := DefaultConfig()
		
		err := config.Validate()
		assert.NoError(t, err, "Default configuration should be valid")
	})
	
	t.Run("missing required fields", func(t *testing.T) {
		config := &Config{
			Languages: map[string]*LanguageConfig{
				"test": {
					BaseImage:     "", // Required field missing
					CacheLocation: "/cache",
					BuildTimeout:  10 * time.Minute,
					Environment:   make(map[string]string),
					Enabled:       true,
				},
			},
		}
		
		err := config.Validate()
		assert.Error(t, err, "Configuration with missing required fields should fail validation")
		assert.Contains(t, err.Error(), "base_image", "Error should mention missing BaseImage field")
	})
}

func TestProvider(t *testing.T) {
	config := DefaultConfig()
	provider := NewProvider(config)
	
	t.Run("type-safe access", func(t *testing.T) {
		// Test string access
		baseImage, err := provider.GetString("languages.python.base_image")
		assert.NoError(t, err)
		assert.Equal(t, "python:3.11-slim-bookworm", baseImage)
		
		// Test bool access
		enabled, err := provider.GetBool("languages.python.enabled")
		assert.NoError(t, err)
		assert.True(t, enabled)
		
		// Test duration access
		timeout, err := provider.GetDuration("languages.python.build_timeout")
		assert.NoError(t, err)
		assert.Equal(t, 20*time.Minute, timeout)
	})
	
	t.Run("access with defaults", func(t *testing.T) {
		// Test existing key
		result := provider.GetStringWithDefault("languages.python.base_image", "default")
		assert.Equal(t, "python:3.11-slim-bookworm", result)
		
		// Test non-existing key
		result = provider.GetStringWithDefault("nonexistent.key", "default-value")
		assert.Equal(t, "default-value", result)
	})
	
	t.Run("has key check", func(t *testing.T) {
		assert.True(t, provider.Has("languages.python.base_image"))
		assert.False(t, provider.Has("nonexistent.key"))
	})
}

func TestConfigFileOperations(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")
	
	configContent := `
languages:
  python:
    base_image: "python:3.12-slim"
    cache_location: "/root/.cache/pip"
    build_timeout: "15m"
    enabled: true
    environment:
      PYTHONPATH: "/app"
container:
  runtime: "docker"
  pull_timeout: "5m"
  build_timeout: "1h"
cache:
  base_dir: "/tmp/cache"
  max_size: "1GB"
  ttl: "24h"
  enabled: true
build:
  parallel: 4
  timeout: "2h"
  retry_count: 2
  retry_delay: "10s"
  fail_fast: true
registry:
  default: "docker.io"
  timeout: "2m"
`
	
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	
	t.Run("load from file", func(t *testing.T) {
		config, err := LoadConfig(configPath)
		require.NoError(t, err)
		
		// Verify loaded values
		pythonConfig := config.Languages["python"]
		require.NotNil(t, pythonConfig)
		assert.Equal(t, "python:3.12-slim", pythonConfig.BaseImage)
		assert.Equal(t, 15*time.Minute, pythonConfig.BuildTimeout)
		assert.Equal(t, "/app", pythonConfig.Environment["PYTHONPATH"])
		
		// Verify other sections
		assert.Equal(t, "docker", config.Container.Runtime)
		assert.Equal(t, 4, config.Build.Parallel)
		assert.True(t, config.Cache.Enabled)
	})
	
	t.Run("save to file", func(t *testing.T) {
		config := DefaultConfig()
		config.Languages["python"].BaseImage = "python:3.12-custom"
		
		err := config.SaveConfig(configPath)
		require.NoError(t, err)
		
		// Load it back and verify
		loadedConfig, err := LoadConfig(configPath)
		require.NoError(t, err)
		
		assert.Equal(t, "python:3.12-custom", loadedConfig.Languages["python"].BaseImage)
	})
}

func TestBackwardCompatibility(t *testing.T) {
	// This test ensures that the new configuration system maintains
	// backward compatibility with existing code patterns
	
	t.Run("language config access patterns", func(t *testing.T) {
		config := DefaultConfig()
		
		// These patterns should continue to work as before
		pythonConfig := config.Languages["python"]
		assert.NotNil(t, pythonConfig)
		assert.NotEmpty(t, pythonConfig.BaseImage)
		assert.NotEmpty(t, pythonConfig.CacheLocation)
		assert.NotNil(t, pythonConfig.Environment)
		
		// Environment variable access should work
		for key, value := range pythonConfig.Environment {
			assert.NotEmpty(t, key, "Environment variable key should not be empty")
			assert.NotEmpty(t, value, "Environment variable value should not be empty")
		}
	})
	
	t.Run("configuration modification", func(t *testing.T) {
		config := DefaultConfig()
		
		// Code should be able to modify configuration as before
		config.Languages["python"].BaseImage = "python:3.10-slim"
		config.Languages["python"].Environment["NEW_VAR"] = "test_value"
		
		assert.Equal(t, "python:3.10-slim", config.Languages["python"].BaseImage)
		assert.Equal(t, "test_value", config.Languages["python"].Environment["NEW_VAR"])
	})
}