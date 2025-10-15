package autodiscovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/protos2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetermineServiceType(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		files          map[string]string
		name           string
		expectedMain   string
		expectedResult bool
	}{
		{
			name: "service with __main__.py",
			files: map[string]string{
				"__main__.py": "if __name__ == '__main__':\n    print('Hello')",
			},
			expectedResult: true,
			expectedMain:   filepath.Join(tmpDir, "__main__.py"),
		},
		{
			name: "service with main.py",
			files: map[string]string{
				"main.py": "def main():\n    print('Hello')\n\nif __name__ == '__main__':\n    main()",
			},
			expectedResult: true,
			expectedMain:   filepath.Join(tmpDir, "main.py"),
		},
		{
			name: "service with app.py",
			files: map[string]string{
				"app.py": "from flask import Flask\napp = Flask(__name__)",
			},
			expectedResult: true,
			expectedMain:   filepath.Join(tmpDir, "app.py"),
		},
		{
			name: "library without main",
			files: map[string]string{
				"utils.py":  "def helper():\n    return 'helper'",
				"models.py": "class User:\n    pass",
			},
			expectedResult: false,
			expectedMain:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up from previous test
			for file := range tt.files {
				os.Remove(filepath.Join(tmpDir, file))
			}

			// Create test files
			var pyFiles []string
			for filename, content := range tt.files {
				fullPath := filepath.Join(tmpDir, filename)
				err := os.WriteFile(fullPath, []byte(content), 0644)
				require.NoError(t, err)
				pyFiles = append(pyFiles, fullPath)
			}

			isService, mainModule := determineServiceType(tmpDir, pyFiles)
			assert.Equal(t, tt.expectedResult, isService)
			assert.Equal(t, tt.expectedMain, mainModule)
		})
	}
}

func TestPythonProjectToBuild(t *testing.T) {
	tests := []struct {
		expected func(container.Build)
		name     string
		project  Project
	}{
		{
			name: "service project",
			project: Project{
				ModulePath:  "./testapp",
				AppName:     "testapp",
				IsService:   true,
				MainFile:    "main.py",
				SourceFiles: []string{"main.py", "utils.py"},
				BuildType:   protos2.BuildType_Python,
			},
			expected: func(build container.Build) {
				assert.Equal(t, "testapp", build.App)
				assert.Equal(t, container.Python, build.BuildType)
				assert.Equal(t, "./testapp", build.Folder)
				assert.Equal(t, "main.py", build.File)
				assert.Equal(t, "NewPythonServiceBuild", build.BuilderFunction)
			},
		},
		{
			name: "library project",
			project: Project{
				ModulePath:  "./testlib",
				AppName:     "testlib",
				IsService:   false,
				SourceFiles: []string{"lib.py"},
				BuildType:   protos2.BuildType_Python,
			},
			expected: func(build container.Build) {
				assert.Equal(t, "testlib", build.App)
				assert.Equal(t, container.Python, build.BuildType)
				assert.Equal(t, "./testlib", build.Folder)
				assert.Equal(t, "", build.File)
				assert.Equal(t, "", build.Image) // Library has no image
				assert.Equal(t, "NewPythonLibraryBuild", build.BuilderFunction)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PythonProjectToBuild(tt.project)
			tt.expected(result)
		})
	}
}

// Integration test for Python project discovery
func TestDiscoverPythonProjectsIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Flask service project
	serviceDir := filepath.Join(tmpDir, "flask-service")
	err := os.MkdirAll(serviceDir, 0755)
	require.NoError(t, err)

	// Write requirements.txt
	reqContent := "flask==2.0.1\nrequests==2.25.1\n"
	err = os.WriteFile(filepath.Join(serviceDir, "requirements.txt"), []byte(reqContent), 0644)
	require.NoError(t, err)

	// Write app.py
	appContent := `from flask import Flask

app = Flask(__name__)

@app.route('/')
def hello():
    return 'Hello, World!'

if __name__ == '__main__':
    app.run()
`
	err = os.WriteFile(filepath.Join(serviceDir, "app.py"), []byte(appContent), 0644)
	require.NoError(t, err)

	// Create pyproject.toml project
	pyprojectDir := filepath.Join(tmpDir, "modern-app")
	err = os.MkdirAll(pyprojectDir, 0755)
	require.NoError(t, err)

	pyprojectContent := `[project]
name = "modern-app"
version = "2.0.0"
description = "A modern Python application"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
	err = os.WriteFile(filepath.Join(pyprojectDir, "pyproject.toml"), []byte(pyprojectContent), 0644)
	require.NoError(t, err)

	mainContent := `def main():
    print("Modern app main")

if __name__ == "__main__":
    main()
`
	err = os.WriteFile(filepath.Join(pyprojectDir, "main.py"), []byte(mainContent), 0644)
	require.NoError(t, err)

	// Test discovery
	projects, err := DiscoverPythonProjects(tmpDir)
	require.NoError(t, err)

	// Should find 3 projects
	assert.Len(t, projects, 2)

	// Verify projects by name
	projectsByName := make(map[string]Project)
	for _, project := range projects {
		projectsByName[project.AppName] = project
	}

	// Check Flask service
	flaskProject, exists := projectsByName["flask-service"]
	require.True(t, exists, "Flask service project not found")
	assert.True(t, flaskProject.IsService)
	assert.Contains(t, flaskProject.MainFile, "app.py")

	// Check modern app
	modernProject, exists := projectsByName["modern-app"]
	require.True(t, exists, "Modern app project not found")
	assert.True(t, modernProject.IsService)
	assert.Contains(t, modernProject.MainFile, "main.py")
}

func TestDiscoverPythonProjectsIntegrationSingle(t *testing.T) {
	tmpDir := t.TempDir()

	// Create pyproject.toml project
	pyprojectDir := filepath.Join(tmpDir, "modern-app")
	err := os.MkdirAll(pyprojectDir, 0755)
	require.NoError(t, err)

	pyprojectContent := `[project]
name = "modern-app"
version = "2.0.0"
description = "A modern Python application"

[build-system]
requires = ["poetry-core"]
build-backend = "poetry.core.masonry.api"
`
	err = os.WriteFile(filepath.Join(pyprojectDir, "pyproject.toml"), []byte(pyprojectContent), 0644)
	require.NoError(t, err)

	mainContent := `def main():
    print("Modern app main")

if __name__ == "__main__":
    main()
`
	err = os.WriteFile(filepath.Join(pyprojectDir, "main.py"), []byte(mainContent), 0644)
	require.NoError(t, err)

	err = os.Chdir(pyprojectDir)
	require.NoError(t, err)

	// Test discovery
	projects, err := DiscoverPythonProjects(".")
	require.NoError(t, err)

	// Should find 3 projects
	assert.Len(t, projects, 1)

	// Verify projects by name
	projectsByName := make(map[string]Project)
	for _, project := range projects {
		projectsByName[project.AppName] = project
	}

	// Check modern app
	modernProject, exists := projectsByName["modern-app"]
	require.True(t, exists, "Modern app project not found")
	assert.True(t, modernProject.IsService)
	assert.Contains(t, modernProject.MainFile, "main.py")
}
