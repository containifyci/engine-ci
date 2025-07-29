package golang

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/containifyci/engine-ci/pkg/builder"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoBuilderIntegration tests the complete Go builder integration with the new architecture
func TestGoBuilderIntegration(t *testing.T) {
	t.Run("VariantSupport", func(t *testing.T) {
		testVariantSupport(t)
	})

	t.Run("BuildOperations", func(t *testing.T) {
		testBuildOperations(t)
	})

	t.Run("ConfigurationIntegration", func(t *testing.T) {
		testConfigurationIntegration(t)
	})

	t.Run("ContainerIntegration", func(t *testing.T) {
		testContainerIntegration(t)
	})

	t.Run("FactoryIntegration", func(t *testing.T) {
		testFactoryIntegration(t)
	})

	t.Run("BackwardCompatibility", func(t *testing.T) {
		testBackwardCompatibility(t)
	})
}

func testVariantSupport(t *testing.T) {
	variants := []struct {
		variant      GoVariant
		expectedName string
		baseImage    string
	}{
		{
			variant:      VariantAlpine,
			expectedName: "golang-alpine",
			baseImage:    "golang:1.24.2-alpine",
		},
		{
			variant:      VariantDebian,
			expectedName: "golang-debian",
			baseImage:    "golang:1.24.2",
		},
		{
			variant:      VariantDebianCGO,
			expectedName: "golang-debiancgo",
			baseImage:    "golang:1.24.2",
		},
	}

	for _, v := range variants {
		t.Run(string(v.variant), func(t *testing.T) {
			build := createTestBuild()
			
			builder, err := NewGoBuilder(build, v.variant)
			require.NoError(t, err)
			assert.NotNil(t, builder)

			// Test variant-specific properties
			assert.Equal(t, v.variant, builder.Variant)
			assert.Equal(t, v.expectedName, builder.Name())
			
			// Test that builder implements required interfaces
			var _ builder.LanguageBuilder = builder
			var _ builder.LintableBuilder = builder
			
			// Test IsAsync returns false (Go builds are synchronous)
			assert.False(t, builder.IsAsync())
			
			// Test Images method returns expected images
			images := builder.Images()
			assert.Contains(t, images, v.baseImage)
			assert.Contains(t, images, "alpine:latest")
			assert.Len(t, images, 3) // base image, alpine, intermediate image
			
			// Test intermediate image naming
			intermediateImg := builder.IntermediateImage()
			assert.NotEmpty(t, intermediateImg)
			assert.Contains(t, intermediateImg, "golang-")
			
			// Test cache folder
			cacheFolder := builder.CacheFolder()
			assert.NotEmpty(t, cacheFolder)
			
			// Test lint image
			lintImg := builder.LintImage()
			assert.Equal(t, "golangci/golangci-lint:v2.1.2", lintImg)
		})
	}
}

func testBuildOperations(t *testing.T) {
	build := createTestBuild()
	builder, err := NewGoBuilder(build, VariantAlpine)
	require.NoError(t, err)

	t.Run("BuildScript", func(t *testing.T) {
		// Test build script generation
		script := builder.BuildScript()
		assert.NotEmpty(t, script)
		
		// Script should contain the app name
		assert.Contains(t, script, build.App)
		
		// Test with custom build tags
		build.Custom = container.Custom{
			"tags": []string{"integration", "e2e"},
		}
		builder, err = NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		script = builder.BuildScript()
		assert.NotEmpty(t, script)
		// Build script should handle tags (exact content depends on buildscript package)
	})

	t.Run("CacheFolderDetection", func(t *testing.T) {
		cacheFolder := builder.CacheFolder()
		assert.NotEmpty(t, cacheFolder)
		
		// Should be a valid path
		assert.True(t, filepath.IsAbs(cacheFolder) || strings.HasPrefix(cacheFolder, "."))
	})

	t.Run("IntermediateImageGeneration", func(t *testing.T) {
		img1 := builder.IntermediateImage()
		img2 := builder.IntermediateImage()
		
		// Should be consistent
		assert.Equal(t, img1, img2)
		
		// Should contain version and variant info
		assert.Contains(t, img1, "golang")
		
		// Should be a valid image name format
		parts := strings.Split(img1, ":")
		assert.GreaterOrEqual(t, len(parts), 2, "Should have registry/name:tag format")
	})

	// Note: We can't test actual Build(), Pull(), Prod() operations without a real Docker environment
	// These would be covered by E2E tests in a CI environment
}

func testConfigurationIntegration(t *testing.T) {
	t.Run("DefaultConfiguration", func(t *testing.T) {
		build := createTestBuild()
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		// Should have default configuration
		assert.NotNil(t, builder.Config)
		assert.Equal(t, "1.0", builder.Config.Version)
		assert.Equal(t, "1.24.2", builder.Config.Language.Go.Version)
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", builder.Config.Language.Go.LintImage)
	})

	t.Run("ConfigurationOverrides", func(t *testing.T) {
		// Test configuration via environment variables
		envVars := map[string]string{
			"ENGINE_CI_LANGUAGE_GO_VERSION":    "1.25.0",
			"ENGINE_CI_LANGUAGE_GO_LINT_IMAGE": "custom/golangci-lint:latest",
		}
		
		for key, value := range envVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range envVars {
				os.Unsetenv(key)
			}
		}()
		
		// Create builder - configuration should be loaded from environment
		build := createTestBuild()
		
		// Manually load config with environment variables for testing
		cfg := config.GetDefaultConfig()
		err := config.LoadFromEnvironmentVariables(cfg)
		require.NoError(t, err)
		
		// Verify environment variables were applied
		assert.Equal(t, "1.25.0", cfg.Language.Go.Version)
		assert.Equal(t, "custom/golangci-lint:latest", cfg.Language.Go.LintImage)
	})

	t.Run("ConfigurationValidation", func(t *testing.T) {
		// Test that builder works with valid configuration
		cfg := config.GetDefaultConfig()
		err := config.ValidateConfig(cfg)
		assert.NoError(t, err)
		
		build := createTestBuild()
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		assert.NotNil(t, builder)
	})

	t.Run("VariantSpecificConfiguration", func(t *testing.T) {
		variants := []GoVariant{VariantAlpine, VariantDebian, VariantDebianCGO}
		
		for _, variant := range variants {
			t.Run(string(variant), func(t *testing.T) {
				build := createTestBuild()
				builder, err := NewGoBuilder(build, variant)
				require.NoError(t, err)
				
				// Each variant should have appropriate base image
				images := builder.Images()
				switch variant {
				case VariantAlpine:
					assert.Contains(t, images, "golang:1.24.2-alpine")
				case VariantDebian, VariantDebianCGO:
					assert.Contains(t, images, "golang:1.24.2")
				}
				
				// Intermediate image should reflect variant
				intermediate := builder.IntermediateImage()
				assert.Contains(t, intermediate, "golang-")
			})
		}
	})
}

func testContainerIntegration(t *testing.T) {
	t.Run("BaseBuilderIntegration", func(t *testing.T) {
		build := createTestBuild()
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		// Should have BaseBuilder functionality
		assert.NotNil(t, builder.BaseBuilder)
		
		// Should be able to get underlying build
		retrievedBuild := builder.GetBuild()
		assert.NotNil(t, retrievedBuild)
		assert.Equal(t, build.App, retrievedBuild.App)
		assert.Equal(t, build.BuildType, retrievedBuild.BuildType)
		
		// Should have defaults available
		assert.NotNil(t, builder.Defaults)
		assert.Equal(t, "golang", builder.Defaults.Language)
		assert.Equal(t, container.GoLang, builder.Defaults.BuildType)
	})

	t.Run("ContainerConfiguration", func(t *testing.T) {
		build := createTestBuild()
		build.Verbose = true
		
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		// Test container options application
		opts := &types.ContainerConfig{}
		builder.ApplyContainerOptions(opts)
		
		assert.Equal(t, "/src", opts.WorkingDir)
		
		// Test verbose mode
		opts.Cmd = []string{"sh", "test.sh"}
		builder.ApplyContainerOptions(opts)
		assert.Contains(t, opts.Cmd, "-v")
	})

	t.Run("BuildConfigurationFields", func(t *testing.T) {
		build := createTestBuild()
		build.Registry = "custom-registry.io"
		build.Image = "test-image"
		build.ImageTag = "v1.0.0"
		build.Custom = container.Custom{
			"custom_flag": "value",
		}
		
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		// Configuration should reflect build settings
		config := builder.Config
		assert.Equal(t, build.Platform, config.Platform)
		assert.Equal(t, build.Env, config.Environment)
		assert.Equal(t, build.Verbose, config.Verbose)
		assert.Equal(t, build.App, config.App)
		assert.Equal(t, build.File, config.File)
		assert.Equal(t, build.Folder, config.Folder)
		assert.Equal(t, build.Image, config.Image)
		assert.Equal(t, build.ImageTag, config.ImageTag)
		assert.Equal(t, build.Custom, config.Custom)
	})
}

func testFactoryIntegration(t *testing.T) {
	t.Run("GoBuilderFactory", func(t *testing.T) {
		factory, err := NewGoBuilderFactory()
		require.NoError(t, err)
		assert.NotNil(t, factory)
		
		// Test supported types
		supportedTypes := factory.SupportedTypes()
		assert.Contains(t, supportedTypes, container.GoLang)
		assert.Len(t, supportedTypes, 1)
		
		build := createTestBuild()
		
		// Test builder creation
		langBuilder, err := factory.CreateBuilder(build)
		require.NoError(t, err)
		assert.NotNil(t, langBuilder)
		
		goBuilder, ok := langBuilder.(*GoBuilder)
		assert.True(t, ok, "Should return GoBuilder instance")
		assert.Equal(t, VariantAlpine, goBuilder.Variant) // Default variant
		
		// Test linter creation
		linter, err := factory.CreateLinter(build)
		require.NoError(t, err)
		assert.NotNil(t, linter)
		assert.Equal(t, "golangci-lint", linter.Name())
		
		// Test prod creation
		prod, err := factory.CreateProd(build)
		require.NoError(t, err)
		assert.NotNil(t, prod)
		assert.Equal(t, "golang-prod", prod.Name())
	})

	t.Run("FactoryRegistration", func(t *testing.T) {
		// Test that GoBuilder can be registered with the global registry
		factory, err := NewGoBuilderFactory()
		require.NoError(t, err)
		
		registration := &builder.BuilderRegistration{
			BuildType: container.GoLang,
			Name:      "golang-builder",
			Constructor: func(build container.Build) (builder.LanguageBuilder, error) {
				return factory.CreateBuilder(build)
			},
			Features: builder.BuilderFeatures{
				SupportsLinting:    true,
				SupportsProduction: true,
				SupportsAsync:      false,
				RequiredFiles:      []string{"go.mod"},
			},
		}
		
		err = builder.RegisterBuilder(registration)
		assert.NoError(t, err)
		
		// Verify registration
		retrieved, exists := builder.GetBuilder(container.GoLang)
		assert.True(t, exists)
		assert.Equal(t, "golang-builder", retrieved.Name)
		assert.True(t, retrieved.Features.SupportsLinting)
		assert.True(t, retrieved.Features.SupportsProduction)
		assert.False(t, retrieved.Features.SupportsAsync)
		
		// Test creation through registry
		build := createTestBuild()
		langBuilder, err := builder.CreateLanguageBuilder(container.GoLang, build)
		assert.NoError(t, err)
		assert.NotNil(t, langBuilder)
	})
}

func testBackwardCompatibility(t *testing.T) {
	t.Run("LegacyFunctions", func(t *testing.T) {
		build := createTestBuild()
		
		// Test legacy New functions
		alpineBuilder, err := New(build)
		require.NoError(t, err)
		assert.NotNil(t, alpineBuilder)
		assert.Equal(t, VariantAlpine, alpineBuilder.Variant)
		
		debianBuilder, err := NewDebian(build)
		require.NoError(t, err)
		assert.NotNil(t, debianBuilder)
		assert.Equal(t, VariantDebian, debianBuilder.Variant)
		
		cgoBuilder, err := NewCGO(build)
		require.NoError(t, err)
		assert.NotNil(t, cgoBuilder)
		assert.Equal(t, VariantDebianCGO, cgoBuilder.Variant)
		
		// Test legacy utility functions
		linter := NewLinter(build)
		assert.NotNil(t, linter)
		assert.Equal(t, "golangci-lint", linter.Name())
		
		prod := NewProd(build)
		assert.NotNil(t, prod)
		assert.Equal(t, "golang-prod", prod.Name())
		
		// Test legacy image and cache functions
		lintImage := LintImage()
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", lintImage)
		
		cacheFolder := CacheFolder()
		assert.NotEmpty(t, cacheFolder)
	})

	t.Run("ExistingBuildWorkflows", func(t *testing.T) {
		// Test that existing build workflows still work
		build := createTestBuild()
		
		// Using legacy function
		builder, err := New(build)
		require.NoError(t, err)
		
		// Should implement all required interfaces
		var _ builder.LanguageBuilder = builder
		var _ builder.LintableBuilder = builder
		
		// Should have all required methods
		assert.Equal(t, "golang-alpine", builder.Name())
		assert.False(t, builder.IsAsync())
		
		images := builder.Images()
		assert.NotEmpty(t, images)
		
		intermediate := builder.IntermediateImage()
		assert.NotEmpty(t, intermediate)
		
		script := builder.BuildScript()
		assert.NotEmpty(t, script)
		
		cache := builder.CacheFolder()
		assert.NotEmpty(t, cache)
		
		lintImg := builder.LintImage()
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", lintImg)
		
		// Note: Pull(), Build(), Prod(), Lint() would need Docker environment to test
	})

	t.Run("ConfigurationCompatibility", func(t *testing.T) {
		// Test that builders work with both old and new configuration approaches
		
		// Legacy approach - using defaults
		build := createTestBuild()
		builder, err := New(build)
		require.NoError(t, err)
		
		// Should have configuration available
		assert.NotNil(t, builder.Config)
		
		// New approach - explicit configuration
		cfg := config.GetDefaultConfig()
		factory := config.NewBuilderFactory(cfg)
		
		configBuilder, err := factory.CreateBuilderWithConfig(build)
		require.NoError(t, err)
		assert.NotNil(t, configBuilder)
		
		// Both should work and have similar capabilities
		assert.Equal(t, cfg, configBuilder.GetConfig())
	})

	t.Run("FilePathCompatibility", func(t *testing.T) {
		// Test that Docker file embeds still work
		build := createTestBuild()
		
		variants := []GoVariant{VariantAlpine, VariantDebian, VariantDebianCGO}
		
		for _, variant := range variants {
			t.Run(string(variant), func(t *testing.T) {
				builder, err := NewGoBuilder(build, variant)
				require.NoError(t, err)
				
				// Should be able to generate intermediate image
				// (this tests that embedded Dockerfiles are accessible)
				intermediate := builder.IntermediateImage()
				assert.NotEmpty(t, intermediate)
				
				// Should contain expected variant information
				assert.Contains(t, intermediate, "golang-")
			})
		}
	})
}

// TestGoBuilderEdgeCases tests edge cases and error conditions
func TestGoBuilderEdgeCases(t *testing.T) {
	t.Run("InvalidVariant", func(t *testing.T) {
		build := createTestBuild()
		
		_, err := NewGoBuilder(build, GoVariant("invalid"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported Go variant")
	})

	t.Run("EmptyBuildConfiguration", func(t *testing.T) {
		build := container.Build{} // Empty build
		
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err) // Should not fail
		assert.NotNil(t, builder)
		
		// Should still have reasonable defaults
		assert.Equal(t, "golang-alpine", builder.Name())
		assert.False(t, builder.IsAsync())
	})

	t.Run("CustomBuildFlags", func(t *testing.T) {
		build := createTestBuild()
		build.Custom = container.Custom{
			"tags":          []string{"custom", "integration"},
			"nocoverage":    true,
			"coverage_mode": "binary",
			"platforms":     []string{"linux/amd64", "linux/arm64"},
		}
		
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		// Build script should handle custom flags
		script := builder.BuildScript()
		assert.NotEmpty(t, script)
		// Specific content depends on buildscript package implementation
	})

	t.Run("BuilderState", func(t *testing.T) {
		build := createTestBuild()
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		// Multiple calls to same methods should be consistent
		img1 := builder.IntermediateImage()
		img2 := builder.IntermediateImage()
		assert.Equal(t, img1, img2)
		
		cache1 := builder.CacheFolder()
		cache2 := builder.CacheFolder()
		assert.Equal(t, cache1, cache2)
		
		name1 := builder.Name()
		name2 := builder.Name()
		assert.Equal(t, name1, name2)
	})
}

// TestGoBuilderPerformance tests performance characteristics
func TestGoBuilderPerformance(t *testing.T) {
	t.Run("BuilderCreationPerformance", func(t *testing.T) {
		build := createTestBuild()
		
		start := time.Now()
		for i := 0; i < 100; i++ {
			builder, err := NewGoBuilder(build, VariantAlpine)
			require.NoError(t, err)
			_ = builder
		}
		duration := time.Since(start)
		
		// Should be fast - less than 100ms for 100 creations
		assert.Less(t, duration, 100*time.Millisecond, "Builder creation should be fast")
	})

	t.Run("MethodCallPerformance", func(t *testing.T) {
		build := createTestBuild()
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		start := time.Now()
		for i := 0; i < 1000; i++ {
			_ = builder.Name()
			_ = builder.IsAsync()
			_ = builder.Images()
			_ = builder.CacheFolder()
			_ = builder.LintImage()
		}
		duration := time.Since(start)
		
		// Should be very fast - less than 10ms for 1000 calls
		assert.Less(t, duration, 10*time.Millisecond, "Method calls should be fast")
	})

	t.Run("IntermediateImageCaching", func(t *testing.T) {
		build := createTestBuild()
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		
		// First call might be slower (computing checksum)
		start := time.Now()
		img1 := builder.IntermediateImage()
		firstCall := time.Since(start)
		
		// Subsequent calls should be cached and faster
		start = time.Now()
		for i := 0; i < 100; i++ {
			img := builder.IntermediateImage()
			assert.Equal(t, img1, img)
		}
		subsequentCalls := time.Since(start)
		
		t.Logf("First call: %v, Subsequent 100 calls: %v", firstCall, subsequentCalls)
		
		// Subsequent calls should be much faster due to caching
		assert.Less(t, subsequentCalls, firstCall/2, "Intermediate image should be cached")
	})

	t.Run("FactoryPerformance", func(t *testing.T) {
		factory, err := NewGoBuilderFactory()
		require.NoError(t, err)
		
		build := createTestBuild()
		
		start := time.Now()
		for i := 0; i < 100; i++ {
			builder, err := factory.CreateBuilder(build)
			require.NoError(t, err)
			_ = builder
		}
		duration := time.Since(start)
		
		// Factory creation should be fast
		assert.Less(t, duration, 50*time.Millisecond, "Factory builder creation should be fast")
	})
}

// TestGoBuilderConcurrency tests concurrent access
func TestGoBuilderConcurrency(t *testing.T) {
	t.Run("ConcurrentBuilderCreation", func(t *testing.T) {
		build := createTestBuild()
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				
				builder, err := NewGoBuilder(build, VariantAlpine)
				assert.NoError(t, err)
				assert.NotNil(t, builder)
				
				// Test various methods
				_ = builder.Name()
				_ = builder.Images()
				_ = builder.IntermediateImage()
				_ = builder.CacheFolder()
			}()
		}
		
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("ConcurrentFactoryAccess", func(t *testing.T) {
		factory, err := NewGoBuilderFactory()
		require.NoError(t, err)
		
		build := createTestBuild()
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				
				builder, err := factory.CreateBuilder(build)
				assert.NoError(t, err)
				assert.NotNil(t, builder)
				
				linter, err := factory.CreateLinter(build)
				assert.NoError(t, err)
				assert.NotNil(t, linter)
				
				prod, err := factory.CreateProd(build)
				assert.NoError(t, err)
				assert.NotNil(t, prod)
			}()
		}
		
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// createTestBuild creates a test container.Build with all required fields
func createTestBuild() container.Build {
	return container.Build{
		BuildType: container.GoLang,
		App:       "test-app",
		File:      "main.go",
		Folder:    "./",
		Platform: types.Platform{
			Host: &types.PlatformSpec{
				OS:           "darwin",
				Architecture: "arm64",
			},
			Container: &types.PlatformSpec{
				OS:           "linux",
				Architecture: "amd64",
			},
		},
		Env:     container.LocalEnv,
		Verbose: false,
		Custom:  make(container.Custom),
	}
}

// TestGoBuilderIntegrationWithRealConfig tests integration with actual configuration loading
func TestGoBuilderIntegrationWithRealConfig(t *testing.T) {
	// This test verifies that the Go builder works correctly with the full configuration system
	
	t.Run("ConfigurationFromFile", func(t *testing.T) {
		// Create a temporary configuration file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "go-config.yaml")
		
		yamlContent := `
version: "test-version"
language:
  go:
    version: "1.26.0"
    lint_image: "custom/golangci-lint:v3.0"
    test_timeout: "10m"
    build_timeout: "3h"
    coverage_mode: "binary"
    project_mount: "/custom/src"
    output_dir: "/custom/out"
    mod_cache: "/custom/mod"
    build_tags: ["integration", "e2e"]
    variants:
      alpine:
        base_image: "golang:1.26.0-alpine"
        cgo_enabled: false
      debian:
        base_image: "golang:1.26.0"
        cgo_enabled: false
      debian_cgo:
        base_image: "golang:1.26.0"
        cgo_enabled: true
container:
  registry: "test-registry.io"
  timeouts:
    build: "4h"
    container_start: "2m"
cache:
  enabled: true
  directories:
    go: "/custom/go-cache"
`
		
		err := os.WriteFile(configFile, []byte(yamlContent), 0644)
		require.NoError(t, err)
		
		// Load configuration from file
		cfg, err := config.LoadConfigFromFile(configFile)
		require.NoError(t, err)
		
		// Verify configuration was loaded correctly
		assert.Equal(t, "test-version", cfg.Version)
		assert.Equal(t, "1.26.0", cfg.Language.Go.Version)
		assert.Equal(t, "custom/golangci-lint:v3.0", cfg.Language.Go.LintImage)
		assert.Equal(t, 10*time.Minute, cfg.Language.Go.TestTimeout)
		assert.Equal(t, 3*time.Hour, cfg.Language.Go.BuildTimeout)
		assert.Equal(t, "binary", cfg.Language.Go.CoverageMode)
		assert.Equal(t, "/custom/src", cfg.Language.Go.ProjectMount)
		assert.Equal(t, "/custom/out", cfg.Language.Go.OutputDir)
		assert.Equal(t, "/custom/mod", cfg.Language.Go.ModCache)
		
		// Create builder factory with configuration
		factory := config.NewBuilderFactory(cfg)
		assert.NotNil(t, factory)
		assert.Equal(t, cfg, factory.GetConfig())
		
		// Create a configurable Go builder
		build := createTestBuild()
		builder, err := factory.CreateBuilderWithConfig(build)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		
		// Verify builder has the configuration
		assert.Equal(t, cfg, builder.GetConfig())
		
		// Test configuration validation
		err = builder.ValidateConfig()
		assert.NoError(t, err)
		
		// Test configuration accessor methods
		assert.Equal(t, "1.26.0", builder.GetGoVersion())
		assert.Equal(t, "custom/golangci-lint:v3.0", builder.GetLintImage())
		assert.Equal(t, "/custom/src", builder.GetProjectMount())
		assert.Equal(t, "/custom/out", builder.GetOutputDir())
		assert.Equal(t, "10m0s", builder.GetTestTimeout())
		assert.Equal(t, "binary", builder.GetCoverageMode())
	})

	t.Run("ConfigurationFromEnvironment", func(t *testing.T) {
		// Set environment variables
		envVars := map[string]string{
			"ENGINE_CI_LANGUAGE_GO_VERSION":       "1.27.0",
			"ENGINE_CI_LANGUAGE_GO_LINT_IMAGE":    "env/golangci-lint:latest",
			"ENGINE_CI_LANGUAGE_GO_TEST_TIMEOUT":  "15m",
			"ENGINE_CI_LANGUAGE_GO_PROJECT_MOUNT": "/env/src",
			"ENGINE_CI_LANGUAGE_GO_OUTPUT_DIR":    "/env/out",
			"ENGINE_CI_LANGUAGE_GO_COVERAGE_MODE": "binary",
			"ENGINE_CI_CONTAINER_REGISTRY":        "env-registry.io",
			"ENGINE_CI_CACHE_ENABLED":             "false",
		}
		
		for key, value := range envVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range envVars {
				os.Unsetenv(key)
			}
		}()
		
		// Load configuration with environment variables
		cfg := config.GetDefaultConfig()
		err := config.LoadFromEnvironmentVariables(cfg)
		require.NoError(t, err)
		
		// Verify environment variables were applied
		assert.Equal(t, "1.27.0", cfg.Language.Go.Version)
		assert.Equal(t, "env/golangci-lint:latest", cfg.Language.Go.LintImage)
		assert.Equal(t, 15*time.Minute, cfg.Language.Go.TestTimeout)
		assert.Equal(t, "/env/src", cfg.Language.Go.ProjectMount)
		assert.Equal(t, "/env/out", cfg.Language.Go.OutputDir)
		assert.Equal(t, "binary", cfg.Language.Go.CoverageMode)
		assert.Equal(t, "env-registry.io", cfg.Container.Registry)
		assert.False(t, cfg.Cache.Enabled)
		
		// Create builder with environment-configured settings
		factory := config.NewBuilderFactory(cfg)
		build := createTestBuild()
		
		builder, err := factory.CreateBuilderWithConfig(build)
		require.NoError(t, err)
		
		// Verify builder reflects environment configuration
		assert.Equal(t, "1.27.0", builder.GetGoVersion())
		assert.Equal(t, "env/golangci-lint:latest", builder.GetLintImage())
		assert.Equal(t, "/env/src", builder.GetProjectMount())
		assert.Equal(t, "/env/out", builder.GetOutputDir())
		assert.Equal(t, "15m0s", builder.GetTestTimeout())
		assert.Equal(t, "binary", builder.GetCoverageMode())
	})

	t.Run("ConfigurationHierarchy", func(t *testing.T) {
		// Test complete configuration hierarchy: defaults -> file -> env
		
		// 1. Create config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "hierarchy-config.yaml")
		
		yamlContent := `
language:
  go:
    version: "1.28.0"
    lint_image: "file/golangci-lint:latest"
    test_timeout: "8m"
    project_mount: "/file/src"
`
		
		err := os.WriteFile(configFile, []byte(yamlContent), 0644)
		require.NoError(t, err)
		
		// 2. Set environment variables (should override file)
		envVars := map[string]string{
			"ENGINE_CI_LANGUAGE_GO_VERSION":    "1.29.0", // Override file
			"ENGINE_CI_LANGUAGE_GO_OUTPUT_DIR": "/env/out", // Not in file, override default
		}
		
		for key, value := range envVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range envVars {
				os.Unsetenv(key)
			}
		}()
		
		// 3. Load configuration: defaults -> file -> env
		cfg, err := config.LoadConfigFromFile(configFile)
		require.NoError(t, err)
		
		err = config.LoadFromEnvironmentVariables(cfg)
		require.NoError(t, err)
		
		// 4. Verify hierarchy
		assert.Equal(t, "1.29.0", cfg.Language.Go.Version) // From env (highest priority)
		assert.Equal(t, "file/golangci-lint:latest", cfg.Language.Go.LintImage) // From file
		assert.Equal(t, 8*time.Minute, cfg.Language.Go.TestTimeout) // From file
		assert.Equal(t, "/file/src", cfg.Language.Go.ProjectMount) // From file
		assert.Equal(t, "/env/out", cfg.Language.Go.OutputDir) // From env
		assert.Equal(t, "text", cfg.Language.Go.CoverageMode) // Default (not overridden)
		
		// 5. Create builder and verify it uses the hierarchical configuration
		factory := config.NewBuilderFactory(cfg)
		build := createTestBuild()
		
		builder, err := factory.CreateBuilderWithConfig(build)
		require.NoError(t, err)
		
		assert.Equal(t, "1.29.0", builder.GetGoVersion())
		assert.Equal(t, "file/golangci-lint:latest", builder.GetLintImage())
		assert.Equal(t, "/file/src", builder.GetProjectMount())
		assert.Equal(t, "/env/out", builder.GetOutputDir())
		assert.Equal(t, "8m0s", builder.GetTestTimeout())
		assert.Equal(t, "text", builder.GetCoverageMode())
	})
}