package golang

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestBuild creates a minimal container.Build for testing
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
		Custom: make(container.Custom),
	}
}

func TestNewGoBuilder(t *testing.T) {
	build := createTestBuild()

	t.Run("Alpine variant", func(t *testing.T) {
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, VariantAlpine, builder.Variant)
		assert.Equal(t, "golang-alpine", builder.Name())
	})

	t.Run("Debian variant", func(t *testing.T) {
		builder, err := NewGoBuilder(build, VariantDebian)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, VariantDebian, builder.Variant)
		assert.Equal(t, "golang-debian", builder.Name())
	})

	t.Run("DebianCGO variant", func(t *testing.T) {
		builder, err := NewGoBuilder(build, VariantDebianCGO)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, VariantDebianCGO, builder.Variant)
		assert.Equal(t, "golang-debiancgo", builder.Name())
	})

	t.Run("Invalid variant", func(t *testing.T) {
		_, err := NewGoBuilder(build, GoVariant("invalid"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported Go variant")
	})
}

func TestGoBuilder_LintImage(t *testing.T) {
	build := createTestBuild()

	builder, err := NewGoBuilder(build, VariantAlpine)
	require.NoError(t, err)

	lintImage := builder.LintImage()
	assert.Equal(t, "golangci/golangci-lint:v2.1.2", lintImage)
}

func TestGoBuilder_Images(t *testing.T) {
	build := createTestBuild()

	t.Run("Alpine images", func(t *testing.T) {
		builder, err := NewGoBuilder(build, VariantAlpine)
		require.NoError(t, err)

		images := builder.Images()
		assert.Len(t, images, 3)
		assert.Contains(t, images, "golang:1.24.2-alpine")
		assert.Contains(t, images, "alpine:latest")
		// Third image is the intermediate image which varies based on checksum
	})

	t.Run("Debian images", func(t *testing.T) {
		builder, err := NewGoBuilder(build, VariantDebian)
		require.NoError(t, err)

		images := builder.Images()
		assert.Len(t, images, 3)
		assert.Contains(t, images, "golang:1.24.2")
		assert.Contains(t, images, "alpine:latest")
	})
}

func TestGoBuilder_BuildScript(t *testing.T) {
	build := createTestBuild()
	build.Custom = container.Custom{
		"tags": []string{"integration", "e2e"},
	}

	builder, err := NewGoBuilder(build, VariantAlpine)
	require.NoError(t, err)

	script := builder.BuildScript()
	assert.NotEmpty(t, script)
	// The script should contain the app name and build tags
	assert.Contains(t, script, "test-app")
}

func TestGoBuilderFactory(t *testing.T) {
	factory, err := NewGoBuilderFactory()
	require.NoError(t, err)
	assert.NotNil(t, factory)

	build := createTestBuild()

	t.Run("CreateBuilder", func(t *testing.T) {
		builder, err := factory.CreateBuilder(build)
		require.NoError(t, err)
		assert.NotNil(t, builder)
	})

	t.Run("CreateLinter", func(t *testing.T) {
		linter, err := factory.CreateLinter(build)
		require.NoError(t, err)
		assert.NotNil(t, linter)
		assert.Equal(t, "golangci-lint", linter.Name())
	})

	t.Run("CreateProd", func(t *testing.T) {
		prod, err := factory.CreateProd(build)
		require.NoError(t, err)
		assert.NotNil(t, prod)
		assert.Equal(t, "golang-prod", prod.Name())
	})

	t.Run("SupportedTypes", func(t *testing.T) {
		types := factory.SupportedTypes()
		assert.Len(t, types, 1)
		assert.Contains(t, types, container.GoLang)
	})
}

func TestBackwardCompatibilityFunctions(t *testing.T) {
	build := createTestBuild()

	t.Run("New", func(t *testing.T) {
		builder, err := New(build)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, VariantAlpine, builder.Variant)
	})

	t.Run("NewDebian", func(t *testing.T) {
		builder, err := NewDebian(build)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, VariantDebian, builder.Variant)
	})

	t.Run("NewCGO", func(t *testing.T) {
		builder, err := NewCGO(build)
		require.NoError(t, err)
		assert.NotNil(t, builder)
		assert.Equal(t, VariantDebianCGO, builder.Variant)
	})

	t.Run("NewLinter", func(t *testing.T) {
		linter := NewLinter(build)
		assert.NotNil(t, linter)
		assert.Equal(t, "golangci-lint", linter.Name())
	})

	t.Run("NewProd", func(t *testing.T) {
		prod := NewProd(build)
		assert.NotNil(t, prod)
		assert.Equal(t, "golang-prod", prod.Name())
	})

	t.Run("LintImage", func(t *testing.T) {
		image := LintImage()
		assert.Equal(t, "golangci/golangci-lint:v2.1.2", image)
	})

	t.Run("CacheFolder", func(t *testing.T) {
		cache := CacheFolder()
		assert.NotEmpty(t, cache)
	})
}