package language

import (
	"testing"
	"time"

	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestBaseLanguageBuilder(t *testing.T) {
	// Create a test language configuration
	cfg := &config.LanguageConfig{
		BaseImage:     "test:latest",
		CacheLocation: "/test/cache",
		WorkingDir:    "/test/src",
		BuildTimeout:  10 * time.Minute,
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		Enabled: true,
	}
	
	// Create base language builder (passing nil for container and cache manager for unit test)
	builder := NewBaseLanguageBuilder("test-lang", cfg, nil, nil)
	
	t.Run("basic properties", func(t *testing.T) {
		assert.Equal(t, "test-lang", builder.Name())
		assert.Equal(t, "test:latest", builder.BaseImage())
		assert.False(t, builder.IsAsync()) // Default implementation
	})
	
	t.Run("configuration access", func(t *testing.T) {
		config := builder.GetConfig()
		assert.NotNil(t, config)
		assert.Equal(t, "test:latest", config.BaseImage)
		assert.Equal(t, "/test/cache", config.CacheLocation)
		assert.Equal(t, "/test/src", config.WorkingDir)
		assert.Equal(t, "test_value", config.Environment["TEST_VAR"])
	})
	
	t.Run("image tag computation", func(t *testing.T) {
		testData := []byte("test dockerfile content")
		tag1 := builder.ComputeImageTag(testData)
		tag2 := builder.ComputeImageTag(testData)
		
		// Same input should produce same tag
		assert.Equal(t, tag1, tag2)
		assert.NotEmpty(t, tag1)
		assert.Len(t, tag1, 64) // SHA256 hex string length
		
		// Different input should produce different tag
		differentData := []byte("different dockerfile content")
		tag3 := builder.ComputeImageTag(differentData)
		assert.NotEqual(t, tag1, tag3)
	})
	
	t.Run("environment variables", func(t *testing.T) {
		env := builder.DefaultEnvironment()
		assert.Contains(t, env, "TEST_VAR=test_value")
	})
	
	t.Run("build timeout", func(t *testing.T) {
		timeout := builder.BuildTimeout()
		assert.Equal(t, 10*time.Minute, timeout)
	})
	
	t.Run("cache location", func(t *testing.T) {
		cacheLocation := builder.CacheLocation()
		assert.Equal(t, "/test/cache", cacheLocation)
	})
}

func TestBaseLanguageBuilderValidation(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		cfg := &config.LanguageConfig{
			BaseImage:     "valid:latest",
			CacheLocation: "/valid/cache",
			WorkingDir:    "/valid/src",
			BuildTimeout:  5 * time.Minute,
			Environment:   make(map[string]string),
			Enabled:       true,
		}
		
		builder := NewBaseLanguageBuilder("valid-lang", cfg, nil, nil)
		
		// Basic validation should pass for valid configuration
		err := builder.Validate()
		assert.NoError(t, err)
	})
	
	t.Run("validation with missing fields", func(t *testing.T) {
		cfg := &config.LanguageConfig{
			BaseImage:     "", // Missing required field
			CacheLocation: "/cache",
			BuildTimeout:  5 * time.Minute,
			Environment:   make(map[string]string),
			Enabled:       true,
		}
		
		builder := NewBaseLanguageBuilder("invalid-lang", cfg, nil, nil)
		
		err := builder.Validate()
		assert.Error(t, err)
		
		// Should be a ValidationError
		var validationErr *ValidationError
		assert.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "base_image", validationErr.Field)
	})
}

func TestBuilderStep(t *testing.T) {
	cfg := &config.LanguageConfig{
		BaseImage:     "step:latest",
		CacheLocation: "/step/cache",
		BuildTimeout:  5 * time.Minute,
		Environment:   make(map[string]string),
		Enabled:       true,
	}
	
	builder := NewBaseLanguageBuilder("step-lang", cfg, nil, nil)
	
	t.Run("build step adapter", func(t *testing.T) {
		// Test BuildStep interface through adapter
		stepAdapter := builder.CreateBuildStepAdapter()
		
		assert.Equal(t, "step-lang-build", stepAdapter.Name())
		assert.False(t, stepAdapter.IsAsync())
		assert.Equal(t, 5*time.Minute, stepAdapter.Timeout())
		
		// Dependencies should be empty by default
		deps := stepAdapter.Dependencies()
		assert.Empty(t, deps)
	})
	
	t.Run("validation", func(t *testing.T) {
		stepAdapter := builder.CreateBuildStepAdapter()
		
		// Validate should work without container dependency
		err := stepAdapter.Validate()
		assert.NoError(t, err)
	})
	
	// Note: Execute test skipped because it requires a real container
	// In real usage, BaseLanguageBuilder would be initialized with a proper container
}

func TestBackwardCompatibility(t *testing.T) {
	// This test ensures that BaseLanguageBuilder maintains compatibility
	// with existing language package patterns
	
	t.Run("configuration patterns", func(t *testing.T) {
		// Simulate how existing language packages create configuration
		cfg := &config.LanguageConfig{
			BaseImage:     "python:3.11-slim-bookworm",
			CacheLocation: "/root/.cache/pip",
			WorkingDir:    "/src",
			Environment: map[string]string{
				"_PIP_USE_IMPORTLIB_METADATA": "0",
				"UV_CACHE_DIR":                "/root/.cache/pip",
			},
			BuildTimeout: 20 * time.Minute,
			Enabled:      true,
		}
		
		builder := NewBaseLanguageBuilder("python", cfg, nil, nil)
		
		// These access patterns should work as they did before
		assert.Equal(t, "python", builder.Name())
		assert.Equal(t, "python:3.11-slim-bookworm", builder.BaseImage())
		assert.Equal(t, "/root/.cache/pip", builder.CacheLocation())
		
		// Environment variables should be accessible
		env := builder.DefaultEnvironment()
		found := false
		for _, envVar := range env {
			if envVar == "_PIP_USE_IMPORTLIB_METADATA=0" {
				found = true
				break
			}
		}
		assert.True(t, found, "Environment variables should be accessible")
	})
	
	t.Run("interface compliance", func(t *testing.T) {
		cfg := &config.LanguageConfig{
			BaseImage:     "test:latest",
			CacheLocation: "/cache",
			BuildTimeout:  10 * time.Minute,
			Environment:   make(map[string]string),
			Enabled:       true,
		}
		
		builder := NewBaseLanguageBuilder("test", cfg, nil, nil)
		
		// Should implement LanguageBuilder interface methods
		assert.NotEmpty(t, builder.Name())
		assert.NotEmpty(t, builder.BaseImage())
		assert.NotEmpty(t, builder.CacheLocation())
		assert.Greater(t, builder.BuildTimeout(), time.Duration(0))
		
		// Should implement BuildStep interface methods through adapter
		stepAdapter := builder.CreateBuildStepAdapter()
		assert.Equal(t, "test-build", stepAdapter.Name()) // Adapter appends "-build"
		assert.NotNil(t, stepAdapter.Dependencies())        // Should return slice (even if empty)
		assert.GreaterOrEqual(t, stepAdapter.Timeout(), time.Duration(0)) // Should have timeout
	})
}