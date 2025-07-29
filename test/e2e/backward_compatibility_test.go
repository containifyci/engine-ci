package e2e

import (
	"testing"
	"time"

	"github.com/containifyci/engine-ci/pkg/builder"
	"github.com/containifyci/engine-ci/pkg/builder/common"
	"github.com/containifyci/engine-ci/pkg/config"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackwardCompatibility tests that existing APIs continue to work unchanged
func TestBackwardCompatibility(t *testing.T) {
	t.Run("ExistingAPIs", func(t *testing.T) {
		testExistingAPIs(t)
	})

	t.Run("ContainerWorkflows", func(t *testing.T) {
		testContainerWorkflows(t)
	})

	t.Run("ConfigurationMethods", func(t *testing.T) {
		testConfigurationMethods(t)
	})

	t.Run("BuilderFunctions", func(t *testing.T) {
		testBuilderFunctions(t)
	})

	t.Run("LegacyIntegration", func(t *testing.T) {
		testLegacyIntegration(t)
	})
}

func testExistingAPIs(t *testing.T) {
	// Test that all existing golang package functions still work
	
	build := createCompatibilityTestBuild()
	
	t.Run("LegacyGoBuilderFunctions", func(t *testing.T) {
		// Test golang.New() - should create alpine variant
		alpineBuilder, err := golang.New(build)
		require.NoError(t, err)
		assert.NotNil(t, alpineBuilder)
		assert.Equal(t, golang.VariantAlpine, alpineBuilder.Variant)
		assert.Equal(t, "golang-alpine", alpineBuilder.Name())
		
		// Test golang.NewDebian() - should create debian variant
		debianBuilder, err := golang.NewDebian(build)
		require.NoError(t, err)
		assert.NotNil(t, debianBuilder)
		assert.Equal(t, golang.VariantDebian, debianBuilder.Variant)
		assert.Equal(t, "golang-debian", debianBuilder.Name())
		
		// Test golang.NewCGO() - should create debian CGO variant
		cgoBuilder, err := golang.NewCGO(build)
		require.NoError(t, err)
		assert.NotNil(t, cgoBuilder)
		assert.Equal(t, golang.VariantDebianCGO, cgoBuilder.Variant)
		assert.Equal(t, "golang-debiancgo", cgoBuilder.Name())
		
		// All builders should implement the same interfaces
		var _ builder.LanguageBuilder = alpineBuilder
		var _ builder.LintableBuilder = alpineBuilder
		var _ builder.LanguageBuilder = debianBuilder
		var _ builder.LintableBuilder = debianBuilder
		var _ builder.LanguageBuilder = cgoBuilder
		var _ builder.LintableBuilder = cgoBuilder
	})

	t.Run("LegacyUtilityFunctions", func(t *testing.T) {
		// Test golang.NewLinter() - should create linter build
		linter := golang.NewLinter(build)
		assert.NotNil(t, linter)
		assert.Equal(t, "golangci-lint", linter.Name())
		assert.Contains(t, linter.Images(), "golangci/golangci-lint:v2.1.2")
		assert.False(t, linter.IsAsync())
		
		// Test golang.NewProd() - should create production build
		prod := golang.NewProd(build)
		assert.NotNil(t, prod)
		assert.Equal(t, "golang-prod", prod.Name())
		assert.False(t, prod.IsAsync())
		
		// Test golang.LintImage() - should return lint image
		lintImage := golang.LintImage()
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", lintImage)
		
		// Test golang.CacheFolder() - should return cache folder
		cacheFolder := golang.CacheFolder()
		assert.NotEmpty(t, cacheFolder)
	})

	t.Run("BuildInterfaceCompatibility", func(t *testing.T) {
		// Test that legacy builders implement build.Build interface correctly
		linter := golang.NewLinter(build)
		prod := golang.NewProd(build)
		
		// Both should have expected methods
		assert.Equal(t, "golangci-lint", linter.Name())
		assert.Equal(t, "golang-prod", prod.Name())
		
		linterImages := linter.Images()
		prodImages := prod.Images()
		
		assert.NotEmpty(t, linterImages)
		assert.NotEmpty(t, prodImages)
		
		assert.False(t, linter.IsAsync())
		assert.False(t, prod.IsAsync())
		
		// Test that Run() method exists and doesn't panic
		// (actual execution would require Docker environment)
		assert.NotPanics(t, func() {
			// These would fail in test environment without Docker, but shouldn't panic
			_ = linter.Run()
			_ = prod.Run()
		})
	})
}

func testContainerWorkflows(t *testing.T) {
	// Test that existing container build workflows still work
	
	t.Run("ContainerBuildIntegration", func(t *testing.T) {
		build := createCompatibilityTestBuild()
		
		// Create builder using legacy function
		builder, err := golang.New(build)
		require.NoError(t, err)
		
		// Test integration with container.Build
		containerBuild := builder.GetBuild()
		assert.NotNil(t, containerBuild)
		assert.Equal(t, build.App, containerBuild.App)
		assert.Equal(t, build.BuildType, containerBuild.BuildType)
		assert.Equal(t, build.File, containerBuild.File)
		assert.Equal(t, build.Folder, containerBuild.Folder)
		assert.Equal(t, build.Env, containerBuild.Env)
		assert.Equal(t, build.Verbose, containerBuild.Verbose)
		
		// Test that builder works with BaseBuilder functionality
		assert.NotNil(t, builder.BaseBuilder)
		
		// Test container configuration
		opts := &types.ContainerConfig{}
		builder.ApplyContainerOptions(opts)
		
		assert.Equal(t, "/src", opts.WorkingDir)
		
		// Test verbose mode
		build.Verbose = true
		builder, err = golang.New(build)
		require.NoError(t, err)
		
		opts = &types.ContainerConfig{Cmd: []string{"sh", "test.sh"}}
		builder.ApplyContainerOptions(opts)
		assert.Contains(t, opts.Cmd, "-v")
	})

	t.Run("BuildScriptGeneration", func(t *testing.T) {
		build := createCompatibilityTestBuild()
		build.Custom = container.Custom{
			"tags":       []string{"legacy", "test"},
			"nocoverage": true,
		}
		
		builder, err := golang.New(build)
		require.NoError(t, err)
		
		// Test build script generation
		script := builder.BuildScript()
		assert.NotEmpty(t, script)
		assert.Contains(t, script, build.App)
		
		// Test that custom options are handled
		assert.NotEmpty(t, script) // Script should be generated successfully
	})

	t.Run("ImageManagement", func(t *testing.T) {
		build := createCompatibilityTestBuild()
		
		// Test all variants
		variants := []struct {
			builderFunc func(container.Build) (*golang.GoBuilder, error)
			variant     golang.GoVariant
			baseImage   string
		}{
			{golang.New, golang.VariantAlpine, "golang:1.24.2-alpine"},
			{golang.NewDebian, golang.VariantDebian, "golang:1.24.2"},
			{golang.NewCGO, golang.VariantDebianCGO, "golang:1.24.2"},
		}
		
		for _, v := range variants {
			builder, err := v.builderFunc(build)
			require.NoError(t, err)
			
			// Test image methods
			images := builder.Images()
			assert.Contains(t, images, v.baseImage)
			assert.Contains(t, images, "alpine:latest")
			assert.Len(t, images, 3) // base image, alpine, intermediate
			
			// Test intermediate image
			intermediate := builder.IntermediateImage()
			assert.NotEmpty(t, intermediate)
			assert.Contains(t, intermediate, "golang-")
			
			// Test lint image
			lintImage := builder.LintImage()
			assert.Equal(t, "golangci/golangci-lint:v2.1.2", lintImage)
			
			// Test cache folder
			cacheFolder := builder.CacheFolder()
			assert.NotEmpty(t, cacheFolder)
		}
	})
}

func testConfigurationMethods(t *testing.T) {
	// Test that existing configuration methods continue to work
	
	t.Run("DefaultConfiguration", func(t *testing.T) {
		// Test config.GetDefaultConfig()
		cfg := config.GetDefaultConfig()
		assert.NotNil(t, cfg)
		assert.Equal(t, "1.0", cfg.Version)
		assert.Equal(t, "1.24.2", cfg.Language.Go.Version)
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", cfg.Language.Go.LintImage)
		assert.True(t, cfg.Cache.Enabled)
		assert.True(t, cfg.Security.UserManagement.CreateNonRootUser)
	})

	t.Run("EnvironmentDefaults", func(t *testing.T) {
		// Test config.GetEnvironmentDefaults()
		environments := []container.EnvType{
			container.LocalEnv,
			container.BuildEnv,
			container.ProdEnv,
		}
		
		expectedLogLevels := map[container.EnvType]string{
			container.LocalEnv: "debug",
			container.BuildEnv: "info",
			container.ProdEnv:  "warn",
		}
		
		expectedPullPolicies := map[container.EnvType]string{
			container.LocalEnv: "never",
			container.BuildEnv: "if_not_present",
			container.ProdEnv:  "always",
		}
		
		for _, env := range environments {
			cfg := config.GetEnvironmentDefaults(env)
			assert.NotNil(t, cfg)
			assert.Equal(t, env, cfg.Environment.Type)
			assert.Equal(t, expectedLogLevels[env], cfg.Logging.Level)
			assert.Equal(t, expectedPullPolicies[env], cfg.Container.Images.PullPolicy)
		}
	})

	t.Run("ConfigurationValueAccess", func(t *testing.T) {
		cfg := config.GetDefaultConfig()
		
		// Test config.GetConfigValue()
		value, err := config.GetConfigValue(cfg, "language.go.version")
		require.NoError(t, err)
		assert.Equal(t, "1.24.2", value)
		
		value, err = config.GetConfigValue(cfg, "container.timeouts.build")
		require.NoError(t, err)
		assert.Equal(t, 1*time.Hour, value)
		
		value, err = config.GetConfigValue(cfg, "cache.enabled")
		require.NoError(t, err)
		assert.Equal(t, true, value)
		
		// Test config.SetConfigValue()
		err = config.SetConfigValue(cfg, "language.go.version", "1.25.0")
		require.NoError(t, err)
		
		newValue, err := config.GetConfigValue(cfg, "language.go.version")
		require.NoError(t, err)
		assert.Equal(t, "1.25.0", newValue)
		
		// Test invalid paths
		_, err = config.GetConfigValue(cfg, "invalid.path")
		assert.Error(t, err)
		
		err = config.SetConfigValue(cfg, "invalid.path", "value")
		assert.Error(t, err)
	})

	t.Run("GlobalConfiguration", func(t *testing.T) {
		// Test config.GetGlobalConfig() and config.SetGlobalConfig()
		
		originalConfig := config.GetGlobalConfig()
		assert.NotNil(t, originalConfig)
		
		// Create new configuration
		newConfig := config.GetDefaultConfig()
		newConfig.Version = "compatibility-test"
		
		// Set global configuration
		config.SetGlobalConfig(newConfig)
		
		// Verify it was set
		retrievedConfig := config.GetGlobalConfig()
		assert.Equal(t, "compatibility-test", retrievedConfig.Version)
		
		// Restore original configuration
		config.SetGlobalConfig(originalConfig)
	})

	t.Run("ConfigurationMerging", func(t *testing.T) {
		// Test config.MergeWithDefaults()
		
		partialConfig := &config.Config{
			Version: "merge-test",
			Language: config.LanguageConfig{
				Go: config.GoConfig{
					Version:   "1.26.0",
					LintImage: "custom/lint:latest",
				},
			},
			Cache: config.CacheConfig{
				Enabled: false,
			},
		}
		
		mergedConfig := config.MergeWithDefaults(partialConfig)
		
		// Should have custom values
		assert.Equal(t, "merge-test", mergedConfig.Version)
		assert.Equal(t, "1.26.0", mergedConfig.Language.Go.Version)
		assert.Equal(t, "custom/lint:latest", mergedConfig.Language.Go.LintImage)
		assert.False(t, mergedConfig.Cache.Enabled)
		
		// Should have defaults for unspecified values
		assert.Equal(t, 2*time.Minute, mergedConfig.Language.Go.TestTimeout)
		assert.Equal(t, "registry.access.redhat.com/ubi8/openjdk-17:latest", mergedConfig.Language.Maven.ProdImage)
		assert.True(t, mergedConfig.Security.UserManagement.CreateNonRootUser)
	})
}

func testBuilderFunctions(t *testing.T) {
	// Test that existing builder functions and patterns continue to work
	
	t.Run("BuilderDefaults", func(t *testing.T) {
		// Test common.GetLanguageDefaults()
		goDefaults, exists := common.GetLanguageDefaults(container.GoLang)
		assert.True(t, exists)
		assert.Equal(t, "golang", goDefaults.Language)
		assert.Equal(t, container.GoLang, goDefaults.BuildType)
		assert.Equal(t, "golang:1.24.2-alpine", goDefaults.BaseImage)
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", goDefaults.LintImage)
		assert.Equal(t, "/src", goDefaults.SourceMount)
		assert.Equal(t, "/go/pkg", goDefaults.CacheMount)
		assert.Contains(t, goDefaults.RequiredFiles, "go.mod")
		
		mavenDefaults, exists := common.GetLanguageDefaults(container.Maven)
		assert.True(t, exists)
		assert.Equal(t, "maven", mavenDefaults.Language)
		assert.Equal(t, container.Maven, mavenDefaults.BuildType)
		
		pythonDefaults, exists := common.GetLanguageDefaults(container.Python)
		assert.True(t, exists)
		assert.Equal(t, "python", pythonDefaults.Language)
		assert.Equal(t, container.Python, pythonDefaults.BuildType)
	})

	t.Run("LanguageBuildUtility", func(t *testing.T) {
		// Test common.NewLanguageBuild()
		executed := false
		runFunc := func() error {
			executed = true
			return nil
		}
		
		langBuild := common.NewLanguageBuild(
			runFunc,
			"compatibility-test",
			[]string{"test:latest"},
			false,
		)
		
		assert.Equal(t, "compatibility-test", langBuild.Name())
		assert.Equal(t, []string{"test:latest"}, langBuild.Images())
		assert.False(t, langBuild.IsAsync())
		
		err := langBuild.Run()
		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("StandardBuildFactory", func(t *testing.T) {
		// Test builder.NewStandardBuildFactory()
		factory := builder.NewStandardBuildFactory()
		assert.NotNil(t, factory)
		
		// Test supported types
		supportedTypes := factory.SupportedTypes()
		assert.Contains(t, supportedTypes, container.GoLang)
		assert.Contains(t, supportedTypes, container.Maven)
		assert.Contains(t, supportedTypes, container.Python)
		
		build := createCompatibilityTestBuild()
		
		// Test CreateLinter (should work)
		linter, err := factory.CreateLinter(build)
		assert.NoError(t, err)
		assert.NotNil(t, linter)
		assert.Equal(t, "golang-lint", linter.Name())
		
		// Test CreateProd (should work)
		prod, err := factory.CreateProd(build)
		assert.NoError(t, err)
		assert.NotNil(t, prod)
		assert.Equal(t, "golang-prod", prod.Name())
		
		// Test CreateBuilder (should return error - not implemented yet)
		_, err = factory.CreateBuilder(build)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "builder creation not yet implemented")
	})

	t.Run("BuilderRegistry", func(t *testing.T) {
		// Test builder.NewBuilderRegistry()
		registry := builder.NewBuilderRegistry()
		assert.NotNil(t, registry)
		
		// Test registration
		registration := &builder.BuilderRegistration{
			BuildType: container.GoLang,
			Name:      "compatibility-test-builder",
			Constructor: func(build container.Build) (builder.LanguageBuilder, error) {
				return golang.New(build)
			},
			Features: builder.BuilderFeatures{
				SupportsLinting:    true,
				SupportsProduction: true,
				RequiredFiles:      []string{"go.mod"},
			},
		}
		
		err := registry.Register(registration)
		assert.NoError(t, err)
		
		// Test retrieval
		retrieved, exists := registry.Get(container.GoLang)
		assert.True(t, exists)
		assert.Equal(t, "compatibility-test-builder", retrieved.Name)
		
		// Test creation
		build := createCompatibilityTestBuild()
		langBuilder, err := registry.CreateBuilder(container.GoLang, build)
		assert.NoError(t, err)
		assert.NotNil(t, langBuilder)
		
		// Should be a GoBuilder
		goBuilder, ok := langBuilder.(*golang.GoBuilder)
		assert.True(t, ok)
		assert.Equal(t, golang.VariantAlpine, goBuilder.Variant)
	})
}

func testLegacyIntegration(t *testing.T) {
	// Test complete integration scenarios using legacy APIs
	
	t.Run("LegacyToNewArchitecture", func(t *testing.T) {
		// Test that legacy builders work with new architecture components
		
		build := createCompatibilityTestBuild()
		
		// Create builder using legacy function
		legacyBuilder, err := golang.New(build)
		require.NoError(t, err)
		
		// Should implement new interfaces
		var _ builder.LanguageBuilder = legacyBuilder
		var _ builder.LintableBuilder = legacyBuilder
		
		// Should work with new configuration system
		assert.NotNil(t, legacyBuilder.Config)
		assert.Equal(t, "1.0", legacyBuilder.Config.Version)
		
		// Should have BaseBuilder functionality
		assert.NotNil(t, legacyBuilder.BaseBuilder)
		retrievedBuild := legacyBuilder.GetBuild()
		assert.Equal(t, build.App, retrievedBuild.App)
		
		// Should work with new defaults system
		assert.NotNil(t, legacyBuilder.Defaults)
		assert.Equal(t, "golang", legacyBuilder.Defaults.Language)
	})

	t.Run("MixedOldAndNewUsage", func(t *testing.T) {
		// Test using both old and new APIs in the same workflow
		
		build := createCompatibilityTestBuild()
		
		// Create builder using legacy function
		legacyBuilder, err := golang.New(build)
		require.NoError(t, err)
		
		// Create factory using new system
		cfg := config.GetDefaultConfig()
		factory := config.NewBuilderFactory(cfg)
		
		newBuilder, err := factory.CreateBuilderWithConfig(build)
		require.NoError(t, err)
		
		// Both should work and be compatible
		assert.Equal(t, legacyBuilder.Name(), "golang-alpine")
		assert.NotNil(t, newBuilder)
		
		// Both should implement same interfaces
		var _ builder.LanguageBuilder = legacyBuilder
		var _ builder.LanguageBuilder = newBuilder
		
		// Both should produce similar results
		legacyImages := legacyBuilder.Images()
		assert.Contains(t, legacyImages, "golang:1.24.2-alpine")
		
		legacyScript := legacyBuilder.BuildScript()
		assert.NotEmpty(t, legacyScript)
		assert.Contains(t, legacyScript, build.App)
	})

	t.Run("BackwardCompatibleConfiguration", func(t *testing.T) {
		// Test that new builders work with legacy configuration approaches
		
		build := createCompatibilityTestBuild()
		
		// Create using legacy approach
		legacyBuilder, err := golang.New(build)
		require.NoError(t, err)
		
		// Should have access to configuration
		assert.NotNil(t, legacyBuilder.Config)
		
		// Should work with both old and new configuration methods
		cfg := config.GetDefaultConfig()
		
		// Legacy builder should be able to use new configuration
		assert.Equal(t, cfg.Language.Go.Version, legacyBuilder.Config.Language.Go.Version)
		assert.Equal(t, cfg.Language.Go.LintImage, legacyBuilder.Config.Language.Go.LintImage)
	})

	t.Run("LegacyBuildWorkflows", func(t *testing.T) {
		// Test complete legacy build workflows
		
		build := createCompatibilityTestBuild()
		build.Custom = container.Custom{
			"tags":          []string{"legacy", "compatibility"},
			"nocoverage":    true,
			"coverage_mode": "binary",
		}
		
		// Test alpine workflow
		alpineBuilder, err := golang.New(build)
		require.NoError(t, err)
		testLegacyWorkflow(t, alpineBuilder, "alpine")
		
		// Test debian workflow  
		debianBuilder, err := golang.NewDebian(build)
		require.NoError(t, err)
		testLegacyWorkflow(t, debianBuilder, "debian")
		
		// Test CGO workflow
		cgoBuilder, err := golang.NewCGO(build)
		require.NoError(t, err)
		testLegacyWorkflow(t, cgoBuilder, "debiancgo")
		
		// Test utility builds
		linter := golang.NewLinter(build)
		assert.NotNil(t, linter)
		assert.Equal(t, "golangci-lint", linter.Name())
		
		prod := golang.NewProd(build)
		assert.NotNil(t, prod)
		assert.Equal(t, "golang-prod", prod.Name())
	})
}

// testLegacyWorkflow tests a complete legacy workflow for a specific variant
func testLegacyWorkflow(t *testing.T, builder *golang.GoBuilder, variant string) {
	t.Helper()
	
	// Test basic properties
	assert.Contains(t, builder.Name(), variant)
	assert.False(t, builder.IsAsync())
	
	// Test image management
	images := builder.Images()
	assert.NotEmpty(t, images)
	assert.Contains(t, images, "alpine:latest")
	
	// Test intermediate image
	intermediate := builder.IntermediateImage()
	assert.NotEmpty(t, intermediate)
	assert.Contains(t, intermediate, "golang-")
	
	// Test build script
	script := builder.BuildScript()
	assert.NotEmpty(t, script)
	assert.Contains(t, script, "compatibility-test-app")
	
	// Test cache folder
	cache := builder.CacheFolder()
	assert.NotEmpty(t, cache)
	
	// Test lint image
	lintImg := builder.LintImage()
	assert.Equal(t, "golangci/golangci-lint:v2.1.2", lintImg)
	
	// Test interfaces
	var _ builder.LanguageBuilder = builder
	var _ builder.LintableBuilder = builder
	
	// Test that it has new architecture components
	assert.NotNil(t, builder.BaseBuilder)
	assert.NotNil(t, builder.Config)
	assert.NotNil(t, builder.Defaults)
}

// createCompatibilityTestBuild creates a test build for compatibility testing
func createCompatibilityTestBuild() container.Build {
	return container.Build{
		BuildType: container.GoLang,
		App:       "compatibility-test-app",
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

// TestBackwardCompatibilityPerformance tests that legacy APIs maintain performance
func TestBackwardCompatibilityPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("LegacyBuilderCreationPerformance", func(t *testing.T) {
		build := createCompatibilityTestBuild()
		
		// Test legacy function performance
		start := time.Now()
		for i := 0; i < 100; i++ {
			builder, err := golang.New(build)
			require.NoError(t, err)
			_ = builder
		}
		legacyDuration := time.Since(start)
		
		// Test new function performance for comparison
		start = time.Now()
		for i := 0; i < 100; i++ {
			builder, err := golang.NewGoBuilder(build, golang.VariantAlpine)
			require.NoError(t, err)
			_ = builder
		}
		newDuration := time.Since(start)
		
		// Legacy should be similar performance (within 50% of new)
		assert.Less(t, legacyDuration, newDuration*3/2, 
			"Legacy builders should maintain reasonable performance")
		
		t.Logf("Legacy: %v, New: %v (ratio: %.2f)", 
			legacyDuration, newDuration, float64(legacyDuration)/float64(newDuration))
	})

	t.Run("LegacyMethodCallPerformance", func(t *testing.T) {
		build := createCompatibilityTestBuild()
		legacyBuilder, err := golang.New(build)
		require.NoError(t, err)
		
		start := time.Now()
		for i := 0; i < 1000; i++ {
			_ = legacyBuilder.Name()
			_ = legacyBuilder.IsAsync()
			_ = legacyBuilder.Images()
			_ = legacyBuilder.LintImage()
			_ = legacyBuilder.CacheFolder()
		}
		duration := time.Since(start)
		
		// Should be very fast
		assert.Less(t, duration, 10*time.Millisecond, 
			"Legacy method calls should be fast")
		
		t.Logf("1000 legacy method calls took: %v", duration)
	})
}

// TestBackwardCompatibilityConcurrency tests concurrent access to legacy APIs
func TestBackwardCompatibilityConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	t.Run("ConcurrentLegacyBuilderCreation", func(t *testing.T) {
		build := createCompatibilityTestBuild()
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				
				for j := 0; j < 10; j++ {
					// Test all legacy functions
					alpineBuilder, err := golang.New(build)
					assert.NoError(t, err)
					assert.NotNil(t, alpineBuilder)
					
					debianBuilder, err := golang.NewDebian(build)
					assert.NoError(t, err)
					assert.NotNil(t, debianBuilder)
					
					cgoBuilder, err := golang.NewCGO(build)
					assert.NoError(t, err)
					assert.NotNil(t, cgoBuilder)
					
					linter := golang.NewLinter(build)
					assert.NotNil(t, linter)
					
					prod := golang.NewProd(build)
					assert.NotNil(t, prod)
					
					// Test utility functions
					_ = golang.LintImage()
					_ = golang.CacheFolder()
				}
			}()
		}
		
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("MixedLegacyAndNewConcurrency", func(t *testing.T) {
		build := createCompatibilityTestBuild()
		cfg := config.GetDefaultConfig()
		factory := config.NewBuilderFactory(cfg)
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()
				
				if idx%2 == 0 {
					// Use legacy APIs
					builder, err := golang.New(build)
					assert.NoError(t, err)
					assert.NotNil(t, builder)
					
					_ = builder.Name()
					_ = builder.Images()
				} else {
					// Use new APIs
					builder, err := factory.CreateBuilderWithConfig(build)
					assert.NoError(t, err)
					assert.NotNil(t, builder)
					
					_ = builder.GetGoVersion()
					_ = builder.GetLintImage()
				}
			}(i)
		}
		
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}