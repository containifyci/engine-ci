package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containifyci/engine-ci/pkg/builder"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildWorkflows tests complete end-to-end build workflows
func TestBuildWorkflows(t *testing.T) {
	// Skip E2E tests in unit test environments
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	t.Run("CompleteGolangWorkflow", func(t *testing.T) {
		testCompleteGolangWorkflow(t)
	})

	t.Run("ConfiguredBuildWorkflow", func(t *testing.T) {
		testConfiguredBuildWorkflow(t)
	})

	t.Run("MultiVariantWorkflow", func(t *testing.T) {
		testMultiVariantWorkflow(t)
	})

	t.Run("FactoryBasedWorkflow", func(t *testing.T) {
		testFactoryBasedWorkflow(t)
	})
}

func testCompleteGolangWorkflow(t *testing.T) {
	// Test a complete Go build workflow using the new architecture
	
	// Create a test project structure
	tempDir := t.TempDir()
	testProject := setupTestGoProject(t, tempDir)
	
	// Create build configuration
	build := container.Build{
		BuildType: container.GoLang,
		App:       "test-app",
		File:      "main.go",
		Folder:    testProject,
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
		Verbose: true,
		Custom:  make(container.Custom),
	}
	
	// Create Go builder
	builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
	require.NoError(t, err)
	assert.NotNil(t, builder)
	
	// Test builder properties
	assert.Equal(t, "golang-alpine", builder.Name())
	assert.False(t, builder.IsAsync())
	
	// Test image management
	images := builder.Images()
	assert.NotEmpty(t, images)
	assert.Contains(t, images, "golang:1.24.2-alpine")
	assert.Contains(t, images, "alpine:latest")
	
	// Test intermediate image generation
	intermediateImg := builder.IntermediateImage()
	assert.NotEmpty(t, intermediateImg)
	assert.Contains(t, intermediateImg, "golang-")
	
	// Test build script generation
	buildScript := builder.BuildScript()
	assert.NotEmpty(t, buildScript)
	assert.Contains(t, buildScript, "test-app")
	
	// Test cache folder
	cacheFolder := builder.CacheFolder()
	assert.NotEmpty(t, cacheFolder)
	
	// Test lint image
	lintImg := builder.LintImage()
	assert.Equal(t, "golangci/golangci-lint:v2.1.2", lintImg)
	
	t.Logf("Complete workflow test passed - Builder: %s, Images: %v, Cache: %s", 
		builder.Name(), images, cacheFolder)
}

func testConfiguredBuildWorkflow(t *testing.T) {
	// Test build workflow with custom configuration
	
	// Create custom configuration
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "e2e-config.yaml")
	
	yamlContent := `
version: "e2e-test"
language:
  go:
    version: "1.25.0"
    lint_image: "golangci/golangci-lint:v2.2.0"
    test_timeout: "5m"
    build_timeout: "2h"
    coverage_mode: "binary"
    project_mount: "/workspace"
    output_dir: "/artifacts"
    build_tags: ["e2e", "integration"]
container:
  registry: "test-registry.io"
  timeouts:
    container_start: "2m"
    build: "3h"
cache:
  enabled: true
  directories:
    go: "/test-cache/go"
logging:
  level: "debug"
  format: "json"
`
	
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)
	
	// Load configuration
	cfg, err := config.LoadConfigFromFile(configFile)
	require.NoError(t, err)
	
	// Verify configuration was loaded
	assert.Equal(t, "e2e-test", cfg.Version)
	assert.Equal(t, "1.25.0", cfg.Language.Go.Version)
	assert.Equal(t, "golangci/golangci-lint:v2.2.0", cfg.Language.Go.LintImage)
	assert.Equal(t, 5*time.Minute, cfg.Language.Go.TestTimeout)
	
	// Create builder factory with configuration
	factory := config.NewBuilderFactory(cfg)
	assert.NotNil(t, factory)
	
	// Setup test project
	testProject := setupTestGoProject(t, t.TempDir())
	
	// Create build
	build := container.Build{
		BuildType: container.GoLang,
		App:       "configured-app",
		File:      "main.go",
		Folder:    testProject,
		Env:       container.BuildEnv,
		Verbose:   true,
		Custom: container.Custom{
			"tags": []string{"e2e", "configured"},
		},
	}
	
	// Create configured builder
	builder, err := factory.CreateBuilderWithConfig(build)
	require.NoError(t, err)
	assert.NotNil(t, builder)
	
	// Verify builder has custom configuration
	assert.Equal(t, cfg, builder.GetConfig())
	assert.Equal(t, "1.25.0", builder.GetGoVersion())
	assert.Equal(t, "golangci/golangci-lint:v2.2.0", builder.GetLintImage())
	assert.Equal(t, "/workspace", builder.GetProjectMount())
	assert.Equal(t, "/artifacts", builder.GetOutputDir())
	assert.Equal(t, "5m0s", builder.GetTestTimeout())
	assert.Equal(t, "binary", builder.GetCoverageMode())
	
	// Test configuration validation
	err = builder.ValidateConfig()
	assert.NoError(t, err)
	
	t.Logf("Configured workflow test passed - Version: %s, Go: %s, Registry: %s", 
		cfg.Version, cfg.Language.Go.Version, cfg.Container.Registry)
}

func testMultiVariantWorkflow(t *testing.T) {
	// Test workflows with different Go variants
	
	variants := []struct {
		variant      golang.GoVariant
		expectedName string
		baseImage    string
	}{
		{
			variant:      golang.VariantAlpine,
			expectedName: "golang-alpine",
			baseImage:    "golang:1.24.2-alpine",
		},
		{
			variant:      golang.VariantDebian,
			expectedName: "golang-debian",
			baseImage:    "golang:1.24.2",
		},
		{
			variant:      golang.VariantDebianCGO,
			expectedName: "golang-debiancgo",
			baseImage:    "golang:1.24.2",
		},
	}
	
	testProject := setupTestGoProject(t, t.TempDir())
	
	for _, v := range variants {
		t.Run(string(v.variant), func(t *testing.T) {
			// Create build for this variant
			build := container.Build{
				BuildType: container.GoLang,
				App:       "multi-variant-app",
				File:      "main.go",
				Folder:    testProject,
				Env:       container.BuildEnv,
				Custom: container.Custom{
					"variant": string(v.variant),
				},
			}
			
			// Create builder for this variant
			builder, err := golang.NewGoBuilder(build, v.variant)
			require.NoError(t, err)
			assert.NotNil(t, builder)
			
			// Verify variant-specific properties
			assert.Equal(t, v.variant, builder.Variant)
			assert.Equal(t, v.expectedName, builder.Name())
			
			images := builder.Images()
			assert.Contains(t, images, v.baseImage)
			
			// All variants should implement the same interfaces
			var _ builder.LanguageBuilder = builder
			var _ builder.LintableBuilder = builder
			
			// Test build script generation for each variant
			buildScript := builder.BuildScript()
			assert.NotEmpty(t, buildScript)
			assert.Contains(t, buildScript, "multi-variant-app")
			
			// Test intermediate image naming
			intermediateImg := builder.IntermediateImage()
			assert.NotEmpty(t, intermediateImg)
			assert.Contains(t, intermediateImg, "golang-")
			
			t.Logf("Variant %s workflow passed - Name: %s, BaseImage: %s", 
				v.variant, v.expectedName, v.baseImage)
		})
	}
}

func testFactoryBasedWorkflow(t *testing.T) {
	// Test workflow using the factory pattern with builder registration
	
	// Create Go builder factory
	goFactory, err := golang.NewGoBuilderFactory()
	require.NoError(t, err)
	
	// Register Go builder with global registry
	registration := &builder.BuilderRegistration{
		BuildType: container.GoLang,
		Name:      "e2e-golang-builder",
		Constructor: func(build container.Build) (builder.LanguageBuilder, error) {
			return goFactory.CreateBuilder(build)
		},
		Features: builder.BuilderFeatures{
			SupportsLinting:    true,
			SupportsProduction: true,
			SupportsAsync:      false,
			RequiredFiles:      []string{"go.mod"},
		},
	}
	
	err = builder.RegisterBuilder(registration)
	require.NoError(t, err)
	
	// Verify registration
	retrieved, exists := builder.GetBuilder(container.GoLang)
	assert.True(t, exists)
	assert.Equal(t, "e2e-golang-builder", retrieved.Name)
	assert.True(t, retrieved.Features.SupportsLinting)
	assert.True(t, retrieved.Features.SupportsProduction)
	
	// Setup test project
	testProject := setupTestGoProject(t, t.TempDir())
	
	// Create build using factory
	build := container.Build{
		BuildType: container.GoLang,
		App:       "factory-app",
		File:      "main.go",
		Folder:    testProject,
		Env:       container.BuildEnv,
		Custom: container.Custom{
			"factory_test": true,
		},
	}
	
	// Create builder through global registry
	langBuilder, err := builder.CreateLanguageBuilder(container.GoLang, build)
	require.NoError(t, err)
	assert.NotNil(t, langBuilder)
	
	// Verify it's a Go builder
	goBuilder, ok := langBuilder.(*golang.GoBuilder)
	assert.True(t, ok, "Should return GoBuilder instance")
	assert.Equal(t, golang.VariantAlpine, goBuilder.Variant) // Default variant
	
	// Test factory-created builder functionality
	assert.Equal(t, "golang-alpine", langBuilder.Name())
	assert.False(t, langBuilder.IsAsync())
	
	images := langBuilder.Images()
	assert.NotEmpty(t, images)
	assert.Contains(t, images, "golang:1.24.2-alpine")
	
	// Test linter creation through factory
	linter, err := goFactory.CreateLinter(build)
	require.NoError(t, err)
	assert.NotNil(t, linter)
	assert.Equal(t, "golangci-lint", linter.Name())
	
	// Test production builder creation through factory
	prod, err := goFactory.CreateProd(build)
	require.NoError(t, err)
	assert.NotNil(t, prod)
	assert.Equal(t, "golang-prod", prod.Name())
	
	t.Logf("Factory workflow test passed - Builder: %s, Linter: %s, Prod: %s", 
		langBuilder.Name(), linter.Name(), prod.Name())
}

// setupTestGoProject creates a minimal Go project structure for testing
func setupTestGoProject(t *testing.T, dir string) string {
	t.Helper()
	
	// Create go.mod
	goMod := `module test-app

go 1.24

require (
	github.com/stretchr/testify v1.8.4
)
`
	err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)
	require.NoError(t, err)
	
	// Create main.go
	mainGo := `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello from test-app!")
	
	if len(os.Args) > 1 {
		fmt.Printf("Args: %v\n", os.Args[1:])
	}
}
`
	err = os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644)
	require.NoError(t, err)
	
	// Create a simple test file
	mainTest := `package main

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	// Simple test to verify testify works
	assert.True(t, true, "This should always pass")
	assert.Equal(t, "hello", "hello", "Strings should match")
}

func TestArgs(t *testing.T) {
	// Test that we can import and use testing
	assert.NotNil(t, t, "Test context should not be nil")
}
`
	err = os.WriteFile(filepath.Join(dir, "main_test.go"), []byte(mainTest), 0644)
	require.NoError(t, err)
	
	// Create go.sum (empty is fine for this test)
	err = os.WriteFile(filepath.Join(dir, "go.sum"), []byte(""), 0644)
	require.NoError(t, err)
	
	t.Logf("Created test Go project in: %s", dir)
	return dir
}

// TestWorkflowIntegrationWithEnvironment tests workflows with environment configuration
func TestWorkflowIntegrationWithEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping environment integration tests in short mode")
	}

	t.Run("EnvironmentOverrides", func(t *testing.T) {
		// Set environment variables for testing
		envVars := map[string]string{
			"ENGINE_CI_LANGUAGE_GO_VERSION":       "1.26.0",
			"ENGINE_CI_LANGUAGE_GO_LINT_IMAGE":    "custom/golangci-lint:e2e",
			"ENGINE_CI_LANGUAGE_GO_TEST_TIMEOUT":  "10m",
			"ENGINE_CI_LANGUAGE_GO_PROJECT_MOUNT": "/e2e/src",
			"ENGINE_CI_LANGUAGE_GO_OUTPUT_DIR":    "/e2e/out",
			"ENGINE_CI_CONTAINER_REGISTRY":        "e2e-registry.io",
			"ENGINE_CI_CACHE_ENABLED":             "false",
			"ENGINE_CI_LOGGING_LEVEL":             "debug",
		}
		
		for key, value := range envVars {
			os.Setenv(key, value)
		}
		defer func() {
			for key := range envVars {
				os.Unsetenv(key)
			}
		}()
		
		// Load configuration from environment
		cfg := config.GetDefaultConfig()
		err := config.LoadFromEnvironmentVariables(cfg)
		require.NoError(t, err)
		
		// Verify environment variables were applied
		assert.Equal(t, "1.26.0", cfg.Language.Go.Version)
		assert.Equal(t, "custom/golangci-lint:e2e", cfg.Language.Go.LintImage)
		assert.Equal(t, 10*time.Minute, cfg.Language.Go.TestTimeout)
		assert.Equal(t, "/e2e/src", cfg.Language.Go.ProjectMount)
		assert.Equal(t, "/e2e/out", cfg.Language.Go.OutputDir)
		assert.Equal(t, "e2e-registry.io", cfg.Container.Registry)
		assert.False(t, cfg.Cache.Enabled)
		assert.Equal(t, "debug", cfg.Logging.Level)
		
		// Create builder with environment configuration
		factory := config.NewBuilderFactory(cfg)
		testProject := setupTestGoProject(t, t.TempDir())
		
		build := container.Build{
			BuildType: container.GoLang,
			App:       "env-app",
			File:      "main.go",
			Folder:    testProject,
			Env:       container.BuildEnv,
		}
		
		builder, err := factory.CreateBuilderWithConfig(build)
		require.NoError(t, err)
		
		// Verify builder uses environment configuration
		assert.Equal(t, "1.26.0", builder.GetGoVersion())
		assert.Equal(t, "custom/golangci-lint:e2e", builder.GetLintImage())
		assert.Equal(t, "/e2e/src", builder.GetProjectMount())
		assert.Equal(t, "/e2e/out", builder.GetOutputDir())
		assert.Equal(t, "10m0s", builder.GetTestTimeout())
		
		t.Logf("Environment integration test passed - Go: %s, Registry: %s", 
			cfg.Language.Go.Version, cfg.Container.Registry)
	})

	t.Run("MultiEnvironmentProfiles", func(t *testing.T) {
		// Test different environment profiles
		
		environments := []container.EnvType{
			container.LocalEnv,
			container.BuildEnv,
			container.ProdEnv,
		}
		
		for _, env := range environments {
			t.Run(string(env), func(t *testing.T) {
				// Get environment-specific defaults
				cfg := config.GetEnvironmentDefaults(env)
				assert.Equal(t, env, cfg.Environment.Type)
				
				// Verify environment-specific settings
				switch env {
				case container.LocalEnv:
					assert.Equal(t, "debug", cfg.Logging.Level)
					assert.Equal(t, "never", cfg.Container.Images.PullPolicy)
				case container.BuildEnv:
					assert.Equal(t, "info", cfg.Logging.Level)
					assert.Equal(t, "if_not_present", cfg.Container.Images.PullPolicy)
				case container.ProdEnv:
					assert.Equal(t, "warn", cfg.Logging.Level)
					assert.Equal(t, "always", cfg.Container.Images.PullPolicy)
				}
				
				// Create builder with environment-specific configuration
				factory := config.NewBuilderFactory(cfg)
				testProject := setupTestGoProject(t, t.TempDir())
				
				build := container.Build{
					BuildType: container.GoLang,
					App:       "profile-app",
					File:      "main.go",
					Folder:    testProject,
					Env:       env,
				}
				
				builder, err := factory.CreateBuilderWithConfig(build)
				require.NoError(t, err)
				assert.NotNil(t, builder)
				
				// Verify builder configuration matches environment
				builderConfig := builder.GetConfig()
				assert.Equal(t, env, builderConfig.Environment.Type)
				
				t.Logf("Environment profile %s test passed - LogLevel: %s, PullPolicy: %s", 
					env, cfg.Logging.Level, cfg.Container.Images.PullPolicy)
			})
		}
	})
}

// TestWorkflowErrorHandling tests error handling in complete workflows
func TestWorkflowErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling tests in short mode")
	}

	t.Run("InvalidConfiguration", func(t *testing.T) {
		// Test workflow with invalid configuration
		cfg := config.GetDefaultConfig()
		
		// Make configuration invalid
		cfg.Language.Go.Version = "" // Required field
		cfg.Language.Go.ProjectMount = "" // Required field
		
		err := config.ValidateConfig(cfg)
		assert.Error(t, err, "Should reject invalid configuration")
		
		// Factory should still be created (validation happens later)
		factory := config.NewBuilderFactory(cfg)
		assert.NotNil(t, factory)
		
		build := container.Build{
			BuildType: container.GoLang,
			App:       "invalid-config-app",
		}
		
		builder, err := factory.CreateBuilderWithConfig(build)
		require.NoError(t, err) // Builder creation succeeds
		
		// But validation should fail
		err = builder.ValidateConfig()
		assert.Error(t, err, "Builder validation should fail with invalid config")
	})

	t.Run("UnsupportedVariant", func(t *testing.T) {
		build := container.Build{
			BuildType: container.GoLang,
			App:       "unsupported-app",
		}
		
		_, err := golang.NewGoBuilder(build, golang.GoVariant("unsupported"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported Go variant")
	})

	t.Run("MissingProjectFiles", func(t *testing.T) {
		// Create empty project directory (no go.mod)
		emptyDir := t.TempDir()
		
		build := container.Build{
			BuildType: container.GoLang,
			App:       "missing-files-app",
			File:      "main.go",
			Folder:    emptyDir,
		}
		
		// Builder creation should succeed (it doesn't validate project structure)
		builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		
		// But actual build operations would fail in a real environment
		// (This would be tested in integration tests with actual Docker)
		
		t.Logf("Missing files test passed - Builder created but would fail in real build")
	})
}

// TestWorkflowPerformance tests performance characteristics of complete workflows
func TestWorkflowPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("WorkflowCreationPerformance", func(t *testing.T) {
		testProject := setupTestGoProject(t, t.TempDir())
		
		// Test how quickly we can create and configure build workflows
		start := time.Now()
		
		for i := 0; i < 50; i++ {
			// Create configuration
			cfg := config.GetDefaultConfig()
			factory := config.NewBuilderFactory(cfg)
			
			// Create build
			build := container.Build{
				BuildType: container.GoLang,
				App:       "perf-app",
				File:      "main.go",
				Folder:    testProject,
				Env:       container.LocalEnv,
			}
			
			// Create builder
			builder, err := factory.CreateBuilderWithConfig(build)
			require.NoError(t, err)
			
			// Perform basic operations
			_ = builder.GetGoVersion()
			_ = builder.GetLintImage()
			_ = builder.GetProjectMount()
			_ = builder.GetOutputDir()
		}
		
		duration := time.Since(start)
		
		// Should be able to create 50 workflows in under 100ms
		assert.Less(t, duration, 100*time.Millisecond, "Workflow creation should be fast")
		
		t.Logf("Created 50 workflows in %v (avg: %v per workflow)", 
			duration, duration/50)
	})

	t.Run("ConcurrentWorkflows", func(t *testing.T) {
		testProject := setupTestGoProject(t, t.TempDir())
		
		// Test concurrent workflow creation
		done := make(chan bool, 10)
		
		start := time.Now()
		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()
				
				for j := 0; j < 10; j++ {
					cfg := config.GetDefaultConfig()
					factory := config.NewBuilderFactory(cfg)
					
					build := container.Build{
						BuildType: container.GoLang,
						App:       "concurrent-app",
						File:      "main.go",
						Folder:    testProject,
						Env:       container.LocalEnv,
					}
					
					builder, err := factory.CreateBuilderWithConfig(build)
					assert.NoError(t, err)
					assert.NotNil(t, builder)
					
					// Test operations
					_ = builder.GetGoVersion()
					_ = builder.ValidateConfig()
				}
			}(i)
		}
		
		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
		
		duration := time.Since(start)
		
		// 10 goroutines * 10 workflows = 100 total workflows
		assert.Less(t, duration, 200*time.Millisecond, "Concurrent workflows should be fast")
		
		t.Logf("Created 100 concurrent workflows in %v", duration)
	})
}