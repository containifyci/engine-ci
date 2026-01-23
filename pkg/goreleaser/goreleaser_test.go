package goreleaser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultGoreleaserConfigEmbedded(t *testing.T) {
	// Verify the embedded config is not empty
	assert.NotEmpty(t, defaultGoreleaserConfig, "embedded default config should not be empty")

	// Verify it contains expected goreleaser config content
	configStr := string(defaultGoreleaserConfig)
	assert.Contains(t, configStr, "version:", "config should contain version field")
	assert.Contains(t, configStr, "builds:", "config should contain builds section")
}

func TestHasProjectConfig(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles []string
		want       bool
	}{
		{
			name:       "no config file",
			setupFiles: nil,
			want:       false,
		},
		{
			name:       "has .goreleaser.yaml",
			setupFiles: []string{".goreleaser.yaml"},
			want:       true,
		},
		{
			name:       "has .goreleaser.yml",
			setupFiles: []string{".goreleaser.yml"},
			want:       true,
		},
		{
			name:       "has .goreleaser.json",
			setupFiles: []string{".goreleaser.json"},
			want:       true,
		},
		{
			name:       "has unrelated yaml file",
			setupFiles: []string{"goreleaser.yaml"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()

			// Create test files
			for _, file := range tt.setupFiles {
				filePath := filepath.Join(tempDir, file)
				err := os.WriteFile(filePath, []byte("test"), 0644)
				require.NoError(t, err)
			}

			got := hasProjectConfig(tempDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	// Write the default config
	configPath, err := writeDefaultConfig()
	require.NoError(t, err)

	// Cleanup after test
	defer os.Remove(configPath)

	// Verify file was created
	assert.Equal(t, "/tmp/.goreleaser-default.yaml", configPath)

	// Verify file exists and has content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, defaultGoreleaserConfig, content)
}

func TestMatches(t *testing.T) {
	tests := []struct {
		name  string
		build container.Build
		want  bool
	}{
		{
			name: "golang build with goreleaser enabled",
			build: container.Build{
				BuildType: container.GoLang,
				Custom: map[string][]string{
					"goreleaser": {"true"},
				},
			},
			want: true,
		},
		{
			name: "golang build with goreleaser disabled",
			build: container.Build{
				BuildType: container.GoLang,
				Custom: map[string][]string{
					"goreleaser": {"false"},
				},
			},
			want: false,
		},
		{
			name: "golang build without goreleaser custom field",
			build: container.Build{
				BuildType: container.GoLang,
				Custom:    map[string][]string{},
			},
			want: false,
		},
		{
			name: "non-golang build with goreleaser enabled",
			build: container.Build{
				BuildType: container.Python,
				Custom: map[string][]string{
					"goreleaser": {"true"},
				},
			},
			want: false,
		},
		{
			name: "golang build with empty goreleaser value",
			build: container.Build{
				BuildType: container.GoLang,
				Custom: map[string][]string{
					"goreleaser": {},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Matches(tt.build)
			assert.Equal(t, tt.want, got)
		})
	}
}
