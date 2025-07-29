package builder

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/builder/common"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/stretchr/testify/assert"
)

func TestLanguageDefaults(t *testing.T) {
	tests := []struct {
		name      string
		buildType container.BuildType
		expected  string
	}{
		{
			name:      "Go defaults",
			buildType: container.GoLang,
			expected:  "golang",
		},
		{
			name:      "Maven defaults",
			buildType: container.Maven,
			expected:  "maven",
		},
		{
			name:      "Python defaults",
			buildType: container.Python,
			expected:  "python",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaults, exists := common.GetLanguageDefaults(tt.buildType)
			assert.True(t, exists, "Language defaults should exist for %s", tt.buildType)
			assert.Equal(t, tt.expected, defaults.Language)
			assert.NotEmpty(t, defaults.BaseImage, "Base image should not be empty")
			assert.NotEmpty(t, defaults.SourceMount, "Source mount should not be empty")
			assert.NotEmpty(t, defaults.CacheMount, "Cache mount should not be empty")
		})
	}
}

func TestStandardBuildFactory(t *testing.T) {
	factory := NewStandardBuildFactory()
	assert.NotNil(t, factory)

	// Test supported types
	supportedTypes := factory.SupportedTypes()
	assert.Contains(t, supportedTypes, container.GoLang)
	assert.Contains(t, supportedTypes, container.Maven)
	assert.Contains(t, supportedTypes, container.Python)

	// Test factory methods (they should return errors since no builders are registered yet)
	build := container.Build{BuildType: container.GoLang}
	
	_, err := factory.CreateBuilder(build)
	assert.Error(t, err, "Should return error when no builder is registered")
	
	_, err = factory.CreateLinter(build)
	assert.NoError(t, err, "Should be able to create linter build")
	
	_, err = factory.CreateProd(build)
	assert.NoError(t, err, "Should be able to create prod build")
}

func TestBuilderRegistry(t *testing.T) {
	registry := NewBuilderRegistry()
	assert.NotNil(t, registry)

	// Test registration
	registration := &BuilderRegistration{
		BuildType: container.GoLang,
		Name:      "test-golang",
		Constructor: func(build container.Build) (LanguageBuilder, error) {
			return nil, nil
		},
		Features: BuilderFeatures{
			SupportsLinting:    true,
			SupportsProduction: true,
		},
	}

	err := registry.Register(registration)
	assert.NoError(t, err)

	// Test retrieval
	retrieved, exists := registry.Get(container.GoLang)
	assert.True(t, exists)
	assert.Equal(t, "test-golang", retrieved.Name)

	// Test listing
	types := registry.List()
	assert.Contains(t, types, container.GoLang)
}

func TestLanguageBuild(t *testing.T) {
	executed := false
	runFunc := func() error {
		executed = true
		return nil
	}

	build := common.NewLanguageBuild(runFunc, "test-build", []string{"test:latest"}, false)
	assert.NotNil(t, build)
	assert.Equal(t, "test-build", build.Name())
	assert.Equal(t, []string{"test:latest"}, build.Images())
	assert.False(t, build.IsAsync())

	err := build.Run()
	assert.NoError(t, err)
	assert.True(t, executed, "Run function should have been executed")
}