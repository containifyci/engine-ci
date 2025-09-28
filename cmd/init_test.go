package cmd

import (
	"os"
	"testing"

	"github.com/containifyci/engine-ci/pkg/autodiscovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateContainifyCIFileWithProjects(t *testing.T) {
	tests := []struct {
		name     string
		projects []autodiscovery.Project
		wantFile bool
		wantErr  bool
	}{
		{
			name:     "empty projects falls back to static template",
			projects: []autodiscovery.Project{},
			wantFile: true,
			wantErr:  false,
		},
		{
			name: "single service project",
			projects: []autodiscovery.Project{
				{
					ModulePath: "./myapp",
					ModuleName: "github.com/user/myapp",
					AppName:    "myapp",
					MainFile:   "./myapp/main.go",
					IsService:  true,
				},
			},
			wantFile: true,
			wantErr:  false,
		},
		{
			name: "multiple projects",
			projects: []autodiscovery.Project{
				{
					ModulePath: "./service1",
					ModuleName: "github.com/user/service1",
					AppName:    "service1",
					MainFile:   "./service1/main.go",
					IsService:  true,
				},
				{
					ModulePath: "./lib1",
					ModuleName: "github.com/user/lib1",
					AppName:    "lib1",
					IsService:  false,
				},
			},
			wantFile: true,
			wantErr:  false,
		},
		{
			name: "projects with empty app names are filtered",
			projects: []autodiscovery.Project{
				{
					ModulePath: "./invalid",
					ModuleName: "github.com/user/invalid",
					AppName:    "", // Empty app name - should be filtered
					IsService:  true,
				},
				{
					ModulePath: "./valid",
					ModuleName: "github.com/user/valid",
					AppName:    "valid",
					IsService:  true,
				},
			},
			wantFile: true,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()

			// Change to temp directory
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				err := os.Chdir(originalDir)
				require.NoError(t, err)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Create .containifyci directory
			err = os.MkdirAll(".containifyci", 0755)
			require.NoError(t, err)

			// Test creating file with projects
			err = createContainifyCIFileWithProjects(tt.projects)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.wantFile {
				// Verify file was created
				filePath := ".containifyci/containifyci.go"
				assert.FileExists(t, filePath)

				// Verify content is valid Go code
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				contentStr := string(content)

				assert.Contains(t, contentStr, "package main")
				assert.Contains(t, contentStr, "func main() {")
				assert.Contains(t, contentStr, "os.Chdir(\"../\")")
				// For empty projects, it falls back to static template which uses build.Build()
				if tt.name == "empty projects falls back to static template" {
					assert.Contains(t, contentStr, "build.Build(opts)")
				} else {
					assert.Contains(t, contentStr, "build.BuildGroups")
				}
				assert.Contains(t, contentStr, "}")
			}
		})
	}
}

func TestRunInitWithAutoFlag(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a simple Go project
	err = os.WriteFile("go.mod", []byte("module github.com/test/myapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile("main.go", []byte("package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n"), 0644)
	require.NoError(t, err)

	// Create and execute init command with auto flag
	cmd := initCmd
	err = cmd.Flags().Set("auto", "true")
	require.NoError(t, err)

	err = RunInit(cmd, []string{})
	require.NoError(t, err)

	// Verify .containifyci directory was created
	assert.DirExists(t, ".containifyci")

	// Verify containifyci.go was created
	assert.FileExists(t, ".containifyci/containifyci.go")

	// Verify content contains expected elements
	content, err := os.ReadFile(".containifyci/containifyci.go")
	require.NoError(t, err)
	contentStr := string(content)

	assert.Contains(t, contentStr, "package main")
	assert.Contains(t, contentStr, "myapp := build.NewGoServiceBuild(\"myapp\")")
	assert.Contains(t, contentStr, "build.BuildGroups(")
}

func TestRunInitWithoutAutoFlag(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create and execute init command without auto flag (default behavior)
	cmd := initCmd

	err = RunInit(cmd, []string{})
	require.NoError(t, err)

	// Verify .containifyci directory was created
	assert.DirExists(t, ".containifyci")

	// Verify containifyci.go was created
	assert.FileExists(t, ".containifyci/containifyci.go")

	// Verify content is from static template
	content, err := os.ReadFile(".containifyci/containifyci.go")
	require.NoError(t, err)
	contentStr := string(content)

	assert.Contains(t, contentStr, "package main")
	assert.Contains(t, contentStr, "build.NewGoServiceBuild(\"containifyci-example\")")
	assert.Contains(t, contentStr, "build.Build(opts)")
}
