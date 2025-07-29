package builder

import (
	"testing"
	"time"

	"github.com/containifyci/engine-ci/pkg/builder/common"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilderIntegration tests the complete builder system integration
func TestBuilderIntegration(t *testing.T) {
	t.Run("LanguageBuilderInterface", func(t *testing.T) {
		testLanguageBuilderInterface(t)
	})

	t.Run("FactorySystem", func(t *testing.T) {
		testFactorySystem(t)
	})

	t.Run("BaseBuilderFunctionality", func(t *testing.T) {
		testBaseBuilderFunctionality(t)
	})

	t.Run("CommonUtilities", func(t *testing.T) {
		testCommonUtilities(t)
	})

	t.Run("BuilderRegistry", func(t *testing.T) {
		testBuilderRegistry(t)
	})
}

func testLanguageBuilderInterface(t *testing.T) {
	// Test that all supported build types have proper defaults
	supportedTypes := []container.BuildType{
		container.GoLang,
		container.Maven,
		container.Python,
	}

	for _, buildType := range supportedTypes {
		t.Run(string(buildType), func(t *testing.T) {
			defaults, exists := common.GetLanguageDefaults(buildType)
			require.True(t, exists, "Language defaults should exist for %s", buildType)

			// Validate all required fields are present
			assert.NotEmpty(t, defaults.Language, "Language name should not be empty")
			assert.Equal(t, buildType, defaults.BuildType, "BuildType should match")
			assert.NotEmpty(t, defaults.BaseImage, "Base image should not be empty")
			assert.NotEmpty(t, defaults.SourceMount, "Source mount should not be empty")
			assert.NotEmpty(t, defaults.CacheMount, "Cache mount should not be empty")
			assert.NotEmpty(t, defaults.OutputDir, "Output directory should not be empty")
			assert.NotNil(t, defaults.DefaultEnv, "Default environment should not be nil")
			assert.NotEmpty(t, defaults.RequiredFiles, "Required files should not be empty")

			// Validate language-specific configurations
			switch buildType {
			case container.GoLang:
				assert.Equal(t, "golang", defaults.Language)
				assert.Equal(t, "golang:1.24.2-alpine", defaults.BaseImage)
				assert.Equal(t, "golangci/golangci-lint:v2.1.2", defaults.LintImage)
				assert.Contains(t, defaults.RequiredFiles, "go.mod")
				assert.Contains(t, defaults.DefaultEnv, "GOMODCACHE")
				assert.Contains(t, defaults.DefaultEnv, "GOCACHE")

			case container.Maven:
				assert.Equal(t, "maven", defaults.Language)
				assert.Equal(t, "maven:3-eclipse-temurin-17-alpine", defaults.BaseImage)
				assert.Contains(t, defaults.RequiredFiles, "pom.xml")
				assert.Contains(t, defaults.DefaultEnv, "MAVEN_OPTS")

			case container.Python:
				assert.Equal(t, "python", defaults.Language)
				assert.Equal(t, "python:3.11-slim-bookworm", defaults.BaseImage)
				assert.Contains(t, defaults.RequiredFiles, "requirements.txt")
				assert.Contains(t, defaults.DefaultEnv, "UV_CACHE_DIR")
			}
		})
	}
}

func testFactorySystem(t *testing.T) {
	factory := NewStandardBuildFactory()
	require.NotNil(t, factory)

	t.Run("SupportedTypes", func(t *testing.T) {
		supportedTypes := factory.SupportedTypes()
		assert.Contains(t, supportedTypes, container.GoLang)
		assert.Contains(t, supportedTypes, container.Maven)
		assert.Contains(t, supportedTypes, container.Python)
		assert.Len(t, supportedTypes, 3, "Should support exactly 3 build types")
	})

	t.Run("CreateLinter", func(t *testing.T) {
		build := createTestBuild(container.GoLang)
		
		linter, err := factory.CreateLinter(build)
		require.NoError(t, err)
		assert.NotNil(t, linter)
		
		assert.Equal(t, "golang-lint", linter.Name())
		assert.Contains(t, linter.Images(), "golangci/golangci-lint:v2.1.2")
		assert.False(t, linter.IsAsync())
	})

	t.Run("CreateProd", func(t *testing.T) {
		build := createTestBuild(container.GoLang)
		
		prod, err := factory.CreateProd(build)
		require.NoError(t, err)
		assert.NotNil(t, prod)
		
		assert.Equal(t, "golang-prod", prod.Name())
		assert.Contains(t, prod.Images(), "golang:1.24.2-alpine")
		assert.False(t, prod.IsAsync())
	})

	t.Run("CreateBuilder_NotImplemented", func(t *testing.T) {
		build := createTestBuild(container.GoLang)
		
		_, err := factory.CreateBuilder(build)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "builder creation not yet implemented")
	})

	t.Run("UnsupportedBuildType", func(t *testing.T) {
		build := createTestBuild("unsupported")
		
		_, err := factory.CreateLinter(build)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no defaults found for build type")
		
		_, err = factory.CreateProd(build)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no defaults found for build type")
	})
}

func testBaseBuilderFunctionality(t *testing.T) {
	build := createTestBuild(container.GoLang)
	
	// Create a BaseBuilder (using BaseLanguageBuilder for compatibility)
	baseBuilder := &BaseLanguageBuilder{
		Container: &container.Container{Build: &build},
		Config: BuildConfiguration{
			Platform:    build.Platform,
			Environment: build.Env,
			Verbose:     build.Verbose,
			App:         build.App,
			File:        build.File,
			Folder:      build.Folder,
			Custom:      build.Custom,
		},
	}

	t.Run("GetBuild", func(t *testing.T) {
		retrievedBuild := baseBuilder.GetBuild()
		assert.NotNil(t, retrievedBuild)
		assert.Equal(t, build.App, retrievedBuild.App)
		assert.Equal(t, build.BuildType, retrievedBuild.BuildType)
	})

	t.Run("ApplyContainerOptions", func(t *testing.T) {
		opts := &types.ContainerConfig{}
		
		baseBuilder.ApplyContainerOptions(opts)
		
		assert.Equal(t, "/src", opts.WorkingDir, "Should set default working directory")
		
		// Test verbose mode
		baseBuilder.Config.Verbose = true
		opts.Cmd = []string{"sh", "test.sh"}
		baseBuilder.ApplyContainerOptions(opts)
		
		assert.Contains(t, opts.Cmd, "-v", "Should add verbose flag when verbose is enabled")
	})

	t.Run("ConfigurationFields", func(t *testing.T) {
		config := baseBuilder.Config
		
		assert.Equal(t, build.Platform, config.Platform)
		assert.Equal(t, build.Env, config.Environment)
		assert.Equal(t, build.App, config.App)
		assert.Equal(t, build.File, config.File)
		assert.Equal(t, build.Folder, config.Folder)
		assert.Equal(t, build.Custom, config.Custom)
	})
}

func testCommonUtilities(t *testing.T) {
	t.Run("LanguageBuild", func(t *testing.T) {
		executed := false
		runFunc := func() error {
			executed = true
			return nil
		}

		languageBuild := common.NewLanguageBuild(
			runFunc,
			"test-build",
			[]string{"test:latest", "test:v1.0"},
			true, // async
		)

		assert.Equal(t, "test-build", languageBuild.Name())
		assert.Equal(t, []string{"test:latest", "test:v1.0"}, languageBuild.Images())
		assert.True(t, languageBuild.IsAsync())

		err := languageBuild.Run()
		assert.NoError(t, err)
		assert.True(t, executed, "Run function should have been executed")
	})

	t.Run("LanguageDefaultsRegistry", func(t *testing.T) {
		// Test that registry contains all expected build types
		expectedTypes := []container.BuildType{
			container.GoLang,
			container.Maven,
			container.Python,
		}

		for _, buildType := range expectedTypes {
			defaults, exists := common.GetLanguageDefaults(buildType)
			assert.True(t, exists, "Registry should contain defaults for %s", buildType)
			assert.Equal(t, buildType, defaults.BuildType)
		}

		// Test accessing non-existent type
		_, exists := common.GetLanguageDefaults("nonexistent")
		assert.False(t, exists, "Registry should not contain defaults for non-existent type")
	})
}

func testBuilderRegistry(t *testing.T) {
	registry := NewBuilderRegistry()
	require.NotNil(t, registry)

	t.Run("RegisterAndRetrieve", func(t *testing.T) {
		// Create a test registration
		registration := &BuilderRegistration{
			BuildType: container.GoLang,
			Name:      "test-golang-builder",
			Constructor: func(build container.Build) (LanguageBuilder, error) {
				return &mockLanguageBuilder{
					name:   "test-golang-builder",
					images: []string{"golang:test"},
				}, nil
			},
			Features: BuilderFeatures{
				SupportsLinting:    true,
				SupportsProduction: true,
				SupportsAsync:      false,
				RequiredFiles:      []string{"go.mod"},
			},
		}

		// Register the builder
		err := registry.Register(registration)
		assert.NoError(t, err)

		// Retrieve the builder
		retrieved, exists := registry.Get(container.GoLang)
		assert.True(t, exists)
		assert.Equal(t, "test-golang-builder", retrieved.Name)
		assert.True(t, retrieved.Features.SupportsLinting)
		assert.True(t, retrieved.Features.SupportsProduction)
		assert.False(t, retrieved.Features.SupportsAsync)

		// Test listing
		types := registry.List()
		assert.Contains(t, types, container.GoLang)
	})

	t.Run("CreateBuilder", func(t *testing.T) {
		// Register a test builder first
		registration := &BuilderRegistration{
			BuildType: container.GoLang,
			Name:      "test-builder",
			Constructor: func(build container.Build) (LanguageBuilder, error) {
				return &mockLanguageBuilder{
					name:   "test-builder",
					images: []string{"test:latest"},
				}, nil
			},
		}
		
		err := registry.Register(registration)
		require.NoError(t, err)

		// Create builder using registry
		build := createTestBuild(container.GoLang)
		builder, err := registry.CreateBuilder(container.GoLang, build)
		assert.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, "test-builder", builder.Name())
	})

	t.Run("ValidationErrors", func(t *testing.T) {
		// Test nil registration
		err := registry.Register(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "registration cannot be nil")

		// Test nil constructor
		err = registry.Register(&BuilderRegistration{
			BuildType:   container.GoLang,
			Name:        "invalid",
			Constructor: nil,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "constructor cannot be nil")

		// Test creating builder for unregistered type
		_, err = registry.CreateBuilder(container.Maven, createTestBuild(container.Maven))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no builder registered for build type")
	})

	t.Run("DefaultRegistryIntegration", func(t *testing.T) {
		// Test the global registry functions
		registration := &BuilderRegistration{
			BuildType: container.Python,
			Name:      "global-python-builder",
			Constructor: func(build container.Build) (LanguageBuilder, error) {
				return &mockLanguageBuilder{name: "global-python-builder"}, nil
			},
		}

		// Register with global registry
		err := RegisterBuilder(registration)
		assert.NoError(t, err)

		// Retrieve from global registry
		retrieved, exists := GetBuilder(container.Python)
		assert.True(t, exists)
		assert.Equal(t, "global-python-builder", retrieved.Name)

		// Create builder using global function
		build := createTestBuild(container.Python)
		builder, err := CreateLanguageBuilder(container.Python, build)
		assert.NoError(t, err)
		assert.Equal(t, "global-python-builder", builder.Name())
	})
}

// createTestBuild creates a test container.Build with all required fields
func createTestBuild(buildType container.BuildType) container.Build {
	return container.Build{
		BuildType: buildType,
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

// mockLanguageBuilder is a simple mock implementation for testing
type mockLanguageBuilder struct {
	name       string
	images     []string
	isAsync    bool
	runError   error
	buildError error
}

func (m *mockLanguageBuilder) Name() string                        { return m.name }
func (m *mockLanguageBuilder) IsAsync() bool                       { return m.isAsync }
func (m *mockLanguageBuilder) Images() []string                    { return m.images }
func (m *mockLanguageBuilder) Pull() error                         { return m.runError }
func (m *mockLanguageBuilder) Build() error                        { return m.buildError }
func (m *mockLanguageBuilder) Run() error                          { return m.runError }
func (m *mockLanguageBuilder) Prod() error                         { return m.runError }
func (m *mockLanguageBuilder) BuildIntermediateImage() error       { return m.runError }
func (m *mockLanguageBuilder) IntermediateImage() string           { return "intermediate:latest" }
func (m *mockLanguageBuilder) BuildScript() string                 { return "#!/bin/sh\necho 'test'" }
func (m *mockLanguageBuilder) CacheFolder() string                 { return "/tmp/cache" }

// TestBuilderPerformance tests performance characteristics of the builder system
func TestBuilderPerformance(t *testing.T) {
	t.Run("FactoryCreationPerformance", func(t *testing.T) {
		start := time.Now()
		
		// Create factories 1000 times
		for i := 0; i < 1000; i++ {
			factory := NewStandardBuildFactory()
			_ = factory
		}
		
		duration := time.Since(start)
		
		// Should be very fast - less than 10ms for 1000 creations
		assert.Less(t, duration, 10*time.Millisecond, "Factory creation should be fast")
	})

	t.Run("DefaultsLookupPerformance", func(t *testing.T) {
		start := time.Now()
		
		// Look up defaults 10000 times
		for i := 0; i < 10000; i++ {
			_, exists := common.GetLanguageDefaults(container.GoLang)
			assert.True(t, exists)
		}
		
		duration := time.Since(start)
		
		// Should be very fast - less than 10ms for 10000 lookups
		assert.Less(t, duration, 10*time.Millisecond, "Defaults lookup should be fast")
	})

	t.Run("RegistryOperationsPerformance", func(t *testing.T) {
		registry := NewBuilderRegistry()
		
		// Register multiple builders
		start := time.Now()
		for i := 0; i < 100; i++ {
			registration := &BuilderRegistration{
				BuildType: container.BuildType("test-" + string(rune(i))),
				Name:      "test-builder",
				Constructor: func(build container.Build) (LanguageBuilder, error) {
					return &mockLanguageBuilder{}, nil
				},
			}
			err := registry.Register(registration)
			assert.NoError(t, err)
		}
		duration := time.Since(start)
		
		// Should be fast - less than 10ms for 100 registrations
		assert.Less(t, duration, 10*time.Millisecond, "Registry registration should be fast")
	})
}

// TestBuilderConcurrency tests concurrent access to builder system components
func TestBuilderConcurrency(t *testing.T) {
	t.Run("ConcurrentFactoryAccess", func(t *testing.T) {
		factory := NewStandardBuildFactory()
		
		// Run multiple operations concurrently
		done := make(chan bool, 10)
		
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()
				
				// Perform various operations
				types := factory.SupportedTypes()
				assert.NotEmpty(t, types)
				
				build := createTestBuild(container.GoLang)
				_, err := factory.CreateLinter(build)
				assert.NoError(t, err)
				
				_, err = factory.CreateProd(build)
				assert.NoError(t, err)
			}()
		}
		
		// Wait for all goroutines to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("ConcurrentRegistryAccess", func(t *testing.T) {
		registry := NewBuilderRegistry()
		
		// Register some initial builders
		for i := 0; i < 5; i++ {
			registration := &BuilderRegistration{
				BuildType: container.BuildType("concurrent-" + string(rune(i))),
				Name:      "concurrent-builder",
				Constructor: func(build container.Build) (LanguageBuilder, error) {
					return &mockLanguageBuilder{}, nil
				},
			}
			err := registry.Register(registration)
			require.NoError(t, err)
		}
		
		// Run concurrent operations
		done := make(chan bool, 20)
		
		// Concurrent reads
		for i := 0; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()
				
				buildType := container.BuildType("concurrent-" + string(rune(idx%5)))
				_, exists := registry.Get(buildType)
				assert.True(t, exists)
				
				types := registry.List()
				assert.NotEmpty(t, types)
			}(i)
		}
		
		// Concurrent writes
		for i := 5; i < 10; i++ {
			go func(idx int) {
				defer func() { done <- true }()
				
				registration := &BuilderRegistration{
					BuildType: container.BuildType("concurrent-write-" + string(rune(idx))),
					Name:      "write-builder",
					Constructor: func(build container.Build) (LanguageBuilder, error) {
						return &mockLanguageBuilder{}, nil
					},
				}
				err := registry.Register(registration)
				assert.NoError(t, err)
			}(i)
		}
		
		// Wait for all operations to complete
		for i := 0; i < 20; i++ {
			<-done
		}
		
		// Verify final state
		types := registry.List()
		assert.GreaterOrEqual(t, len(types), 10, "Should have at least 10 registered types")
	})
}