package build

import (
	"os"
	"testing"

	"github.com/containifyci/engine-ci/protos2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewList(t *testing.T) {
	t.Parallel()
	t.Run("creates list with single value", func(t *testing.T) {
		t.Parallel()
		list := NewList("value1")
		require.NotNil(t, list)
		assert.Len(t, list.Values, 1)
		assert.Equal(t, "value1", list.Values[0].GetStringValue())
	})

	t.Run("creates list with multiple values", func(t *testing.T) {
		t.Parallel()
		list := NewList("value1", "value2", "value3")
		require.NotNil(t, list)
		assert.Len(t, list.Values, 3)
		assert.Equal(t, "value1", list.Values[0].GetStringValue())
		assert.Equal(t, "value2", list.Values[1].GetStringValue())
		assert.Equal(t, "value3", list.Values[2].GetStringValue())
	})

	t.Run("creates empty list", func(t *testing.T) {
		t.Parallel()
		list := NewList()
		require.NotNil(t, list)
		assert.Len(t, list.Values, 0)
	})
}

//nolint:testparallel // use t.Setenv which is not compatible with t.Parallel
func TestGetEnv(t *testing.T) {
	t.Run("returns local for ENV=local", func(t *testing.T) {
		t.Setenv("ENV", "local")

		env := getEnv()
		assert.Equal(t, protos2.EnvType_local, env)
	})

	t.Run("returns build for other ENV values", func(t *testing.T) {
		t.Setenv("ENV", "production")

		env := getEnv()
		assert.Equal(t, protos2.EnvType_build, env)
	})

	t.Run("returns build for empty ENV", func(t *testing.T) {
		env := getEnv()
		assert.Equal(t, protos2.EnvType_build, env)
	})
}

//nolint:testparallel // use t.Setenv which is not compatible with t.Parallel
func TestNewGoServiceBuild(t *testing.T) {
	t.Run("creates go service build", func(t *testing.T) {
		build := NewGoServiceBuild("test-app")
		require.NotNil(t, build)
		assert.Equal(t, "test-app", build.Application)
		assert.Equal(t, protos2.BuildType_GoLang, build.BuildType)
		assert.Equal(t, "test-app", build.Image)
	})

	t.Run("uses COMMIT_SHA for image tag", func(t *testing.T) {
		t.Setenv("COMMIT_SHA", "abc123")

		build := NewGoServiceBuild("test-app")
		assert.Equal(t, "abc123", build.ImageTag)
	})

	t.Run("uses local for missing COMMIT_SHA", func(t *testing.T) {
		// Ensure COMMIT_SHA is not set
		err := os.Unsetenv("COMMIT_SHA")
		require.NoError(t, err)

		build := NewGoServiceBuild("test-app")
		assert.Equal(t, "local", build.ImageTag)
	})
}

func TestNewGoLibraryBuild(t *testing.T) {
	t.Parallel()
	t.Run("creates go library build", func(t *testing.T) {
		t.Parallel()
		build := NewGoLibraryBuild("test-lib")
		require.NotNil(t, build)
		assert.Equal(t, "test-lib", build.Application)
		assert.Equal(t, protos2.BuildType_GoLang, build.BuildType)
		assert.Empty(t, build.Image, "library should have empty image")
	})
}

func TestNewMavenServiceBuild(t *testing.T) {
	t.Parallel()
	build := NewMavenServiceBuild("test-maven")
	require.NotNil(t, build)
	assert.Equal(t, "test-maven", build.Application)
	assert.Equal(t, protos2.BuildType_Maven, build.BuildType)
	assert.Equal(t, "test-maven", build.Image)
	assert.Equal(t, "target/quarkus-app", build.Folder)
}

func TestNewMavenLibraryBuild(t *testing.T) {
	t.Parallel()
	build := NewMavenLibraryBuild("test-maven-lib")
	require.NotNil(t, build)
	assert.Equal(t, "test-maven-lib", build.Application)
	assert.Equal(t, protos2.BuildType_Maven, build.BuildType)
	assert.Empty(t, build.Image, "library should have empty image")
	assert.Equal(t, "target/quarkus-app", build.Folder)
}

func TestNewPythonServiceBuild(t *testing.T) {
	t.Parallel()
	build := NewPythonServiceBuild("test-python")
	require.NotNil(t, build)
	assert.Equal(t, "test-python", build.Application)
	assert.Equal(t, protos2.BuildType_Python, build.BuildType)
	assert.Equal(t, "test-python", build.Image)
}

func TestNewPythonLibraryBuild(t *testing.T) {
	t.Parallel()
	build := NewPythonLibraryBuild("test-python-lib")
	require.NotNil(t, build)
	assert.Equal(t, "test-python-lib", build.Application)
	assert.Equal(t, protos2.BuildType_Python, build.BuildType)
	assert.Empty(t, build.Image, "library should have empty image")
}
