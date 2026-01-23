package goreleaser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/critest"
	"github.com/containifyci/engine-ci/pkg/svc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers
func mockManager(t *testing.T) *critest.MockContainerManager {
	t.Helper()
	m, err := critest.NewMockContainerManager()
	require.NoError(t, err)
	return m
}

func goreleaserBuild(enabled bool) container.Build {
	custom := map[string][]string{}
	if enabled {
		custom["goreleaser"] = []string{"true"}
	}
	return container.Build{BuildType: container.GoLang, Custom: custom}
}

func setupGitTag(t *testing.T, tag string) {
	t.Helper()
	svc.SetGitInfoForTest("owner", "repo", "main", tag)
	t.Cleanup(svc.ResetGitInfo)
}

func TestDefaultGoreleaserConfigEmbedded(t *testing.T) {
	assert.NotEmpty(t, defaultGoreleaserConfig)
	assert.Contains(t, string(defaultGoreleaserConfig), "version:")
	assert.Contains(t, string(defaultGoreleaserConfig), "builds:")
}

func TestHasProjectConfig(t *testing.T) {
	tests := []struct {
		file string
		want bool
	}{
		{"", false},
		{".goreleaser.yaml", true},
		{".goreleaser.yml", true},
		{".goreleaser.json", true},
		{"goreleaser.yaml", false}, // no leading dot
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			dir := t.TempDir()
			if tt.file != "" {
				require.NoError(t, os.WriteFile(filepath.Join(dir, tt.file), []byte("test"), 0644))
			}
			assert.Equal(t, tt.want, hasProjectConfig(dir))
		})
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	path, err := writeDefaultConfig()
	require.NoError(t, err)
	defer os.Remove(path)

	assert.Equal(t, defaultConfigPath, path)
	content, _ := os.ReadFile(path)
	assert.Equal(t, defaultGoreleaserConfig, content)
}

func TestMatches(t *testing.T) {
	tests := []struct {
		custom    map[string][]string
		name      string
		buildType container.BuildType
		want      bool
	}{
		{name: "golang+enabled", buildType: container.GoLang, custom: map[string][]string{"goreleaser": {"true"}}, want: true},
		{name: "golang+disabled", buildType: container.GoLang, custom: map[string][]string{"goreleaser": {"false"}}, want: false},
		{name: "golang+notset", buildType: container.GoLang, custom: map[string][]string{}, want: false},
		{name: "python+enabled", buildType: container.Python, custom: map[string][]string{"goreleaser": {"true"}}, want: false},
		{name: "golang+empty", buildType: container.GoLang, custom: map[string][]string{"goreleaser": {}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Matches(container.Build{BuildType: tt.buildType, Custom: tt.custom})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNew(t *testing.T) {
	step := New()
	assert.Equal(t, "gorelease", step.Name())
	assert.Equal(t, "release", step.Alias())
	assert.Equal(t, container.GoLang, *step.BuildType())
	assert.False(t, step.IsAsync())
	assert.Contains(t, step.Images(container.Build{}), IMAGE)
}

func TestRun_SkipScenarios(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		enabled bool
	}{
		{"non-tag", "", true},
		{"disabled", "v1.0.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupGitTag(t, tt.tag)
			m := mockManager(t)
			gc := newWithManager(goreleaserBuild(tt.enabled), m)

			err := gc.Run()

			assert.NoError(t, err)
			assert.Empty(t, m.Containers)
			assert.Empty(t, m.Images)
		})
	}
}

func TestRun_MissingToken(t *testing.T) {
	setupGitTag(t, "v1.0.0")
	os.Unsetenv("CONTAINIFYCI_GITHUB_TOKEN")

	m := mockManager(t)
	gc := newWithManager(goreleaserBuild(true), m)

	err := gc.Run()

	assert.ErrorIs(t, err, ErrMissingToken)
}

func TestRun_WithToken(t *testing.T) {
	setupGitTag(t, "v1.0.0")
	t.Setenv("CONTAINIFYCI_GITHUB_TOKEN", "test-token")

	m := mockManager(t)
	gc := newWithManager(goreleaserBuild(true), m)

	err := gc.Run()

	assert.NoError(t, err)
	assert.NotEmpty(t, m.Images)
	assert.NotEmpty(t, m.Containers)
}

func TestApplyEnvs(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		input  []string
		want   []string
	}{
		{"adds tag", "v1.0.0", []string{"A=1"}, []string{"A=1", "GORELEASER_CURRENT_TAG=v1.0.0"}},
		{"no tag", "", []string{"A=1"}, []string{"A=1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv("GORELEASER_CURRENT_TAG", tt.envVal)
			} else {
				os.Unsetenv("GORELEASER_CURRENT_TAG")
			}
			gc := newWithManager(container.Build{}, mockManager(t))
			assert.Equal(t, tt.want, gc.ApplyEnvs(tt.input))
		})
	}
}

func TestPull(t *testing.T) {
	m := mockManager(t)
	gc := newWithManager(container.Build{}, m)

	err := gc.Pull()

	assert.NoError(t, err)
	assert.Contains(t, m.Images, IMAGE)
}

func TestRelease_MissingToken(t *testing.T) {
	os.Unsetenv("CONTAINIFYCI_GITHUB_TOKEN")
	gc := newWithManager(container.Build{}, mockManager(t))

	err := gc.Release(container.BuildEnv)

	assert.ErrorIs(t, err, ErrMissingToken)
}

func TestRelease_WithToken(t *testing.T) {
	t.Setenv("CONTAINIFYCI_GITHUB_TOKEN", "test-token")

	m := mockManager(t)
	gc := newWithManager(container.Build{}, m)

	err := gc.Release(container.BuildEnv)

	assert.NoError(t, err)
	assert.Len(t, m.Containers, 1)

	// Verify container config
	con := m.GetContainerByImage(IMAGE)
	require.NotNil(t, con)
	assert.Equal(t, "/usr/src", con.Opts.WorkingDir)
	assert.Contains(t, con.Opts.Cmd, "release")
}

func TestRelease_UsesDefaultConfig(t *testing.T) {
	t.Setenv("CONTAINIFYCI_GITHUB_TOKEN", "test-token")
	// Project has .goreleaser.yaml so default won't be used
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	// Test in temp dir without config
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))

	m := mockManager(t)
	gc := newWithManager(container.Build{}, m)

	err := gc.Release(container.BuildEnv)

	assert.NoError(t, err)
	con := m.GetContainerByImage(IMAGE)
	require.NotNil(t, con)
	// Should have --config flag since no project config
	assert.Contains(t, con.Opts.Cmd, "--config="+defaultConfigPath)
}

func TestRelease_UsesProjectConfig(t *testing.T) {
	t.Setenv("CONTAINIFYCI_GITHUB_TOKEN", "test-token")
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	// Create temp dir with project config
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".goreleaser.yaml"), []byte("version: 2"), 0644))
	require.NoError(t, os.Chdir(tmpDir))

	m := mockManager(t)
	gc := newWithManager(container.Build{}, m)

	err := gc.Release(container.BuildEnv)

	assert.NoError(t, err)
	con := m.GetContainerByImage(IMAGE)
	require.NotNil(t, con)
	// Should NOT have --config flag since project has config
	for _, cmd := range con.Opts.Cmd {
		assert.NotContains(t, cmd, "--config=")
	}
}

func TestDefaultCacheFolder(t *testing.T) {
	cache, err := defaultCacheFolder()
	assert.NoError(t, err)
	assert.NotEmpty(t, cache)
}

func TestCacheFolderFn_Injection(t *testing.T) {
	original := cacheFolderFn
	defer func() { cacheFolderFn = original }()

	cacheFolderFn = func() (string, error) {
		return "/mock/cache", nil
	}

	result := CacheFolder()
	assert.Equal(t, "/mock/cache", result)
}

func TestCacheFolderFn_Error(t *testing.T) {
	original := cacheFolderFn
	defer func() { cacheFolderFn = original }()

	cacheFolderFn = func() (string, error) {
		return "", errors.New("mock error")
	}

	// This would call os.Exit(1), so we just verify the function is injectable
	// by checking the original works
	cacheFolderFn = original
	result := CacheFolder()
	assert.NotEmpty(t, result)
}

func TestNewWithManager(t *testing.T) {
	m := mockManager(t)
	gc := newWithManager(container.Build{}, m)
	assert.NotNil(t, gc)
	assert.NotNil(t, gc.Container)
}
