package autodiscovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containifyci/engine-ci/protos2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLanguageFilter(t *testing.T) {
	tests := []struct {
		expected map[ProjectType]bool
		name     string
		filter   LanguageFilter
	}{
		{
			name:   "all languages",
			filter: AllLanguages(),
			expected: map[ProjectType]bool{
				ProjectTypeGo:     true,
				ProjectTypePython: true,
				ProjectTypeJava:   true,
			},
		},
		{
			name:   "only go",
			filter: OnlyGo(),
			expected: map[ProjectType]bool{
				ProjectTypeGo:     true,
				ProjectTypePython: false,
				ProjectTypeJava:   false,
			},
		},
		{
			name: "custom filter",
			filter: LanguageFilter{
				Go:     false,
				Python: true,
				Java:   true,
			},
			expected: map[ProjectType]bool{
				ProjectTypeGo:     false,
				ProjectTypePython: true,
				ProjectTypeJava:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected[ProjectTypeGo], tt.filter.Go)
			assert.Equal(t, tt.expected[ProjectTypePython], tt.filter.Python)
			assert.Equal(t, tt.expected[ProjectTypeJava], tt.filter.Java)
		})
	}
}

func TestProjectCollection(t *testing.T) {
	goProject := Project{AppName: "go-app", IsService: true}
	pythonProject := Project{AppName: "python-app", IsService: false}
	javaProject := Project{AppName: "java-app", IsService: true}

	collection := &ProjectCollection{
		GoProjects:     []Project{goProject},
		PythonProjects: []Project{pythonProject},
		JavaProjects:   []Project{javaProject},
	}

	t.Run("AllProjects", func(t *testing.T) {
		allProjects := collection.AllProjects()
		assert.Len(t, allProjects, 3)

		// Check that all projects are included
		names := make(map[string]bool)
		for _, project := range allProjects {
			names[project.AppName] = true
		}

		assert.True(t, names["go-app"])
		assert.True(t, names["python-app"])
		assert.True(t, names["java-app"])
	})

	t.Run("CountByType", func(t *testing.T) {
		counts := collection.CountByType()
		assert.Equal(t, 1, counts[ProjectTypeGo])
		assert.Equal(t, 1, counts[ProjectTypePython])
		assert.Equal(t, 1, counts[ProjectTypeJava])
	})

	t.Run("IsEmpty", func(t *testing.T) {
		assert.False(t, collection.IsEmpty())

		emptyCollection := &ProjectCollection{}
		assert.True(t, emptyCollection.IsEmpty())
	})
}

func TestGenerateBuildGroupsFromCollection(t *testing.T) {
	goProject := Project{
		AppName:     "go-service",
		ModulePath:  "./go-service",
		IsService:   true,
		SourceFiles: []string{"main.go"},
		BuildType:   protos2.BuildType_GoLang,
	}

	pythonProject := Project{
		ModulePath:  "./python-lib",
		AppName:     "python-lib",
		IsService:   false,
		SourceFiles: []string{"lib.py"},
		BuildType:   protos2.BuildType_Python,
	}

	collection := &ProjectCollection{
		GoProjects:     []Project{goProject},
		PythonProjects: []Project{pythonProject},
		JavaProjects:   []Project{},
	}

	buildGroups := GenerateBuildGroupsFromCollection(collection)

	// Should have 2 build groups (one per project)
	assert.Len(t, buildGroups, 2)

	// Each build group should have 1 build
	for i, group := range buildGroups {
		assert.Len(t, group.Builds, 1, "Build group %d should have 1 build", i)
	}

	// Verify build types are correct
	buildTypes := make(map[string]bool)
	for _, group := range buildGroups {
		build := group.Builds[0]
		switch build.BuildType {
		case "GoLang":
			buildTypes["go"] = true
			assert.Equal(t, "go-service", build.App)
		case "Python":
			buildTypes["python"] = true
			assert.Equal(t, "python-lib", build.App)
		}
	}

	assert.True(t, buildTypes["go"], "Should have Go build")
	assert.True(t, buildTypes["python"], "Should have Python build")
}

func TestConvertGoProjectsToCollection(t *testing.T) {
	goProjects := []Project{
		{AppName: "app1", IsService: true},
		{AppName: "app2", IsService: false},
	}

	collection := ConvertGoProjectsToCollection(goProjects)

	assert.Len(t, collection.GoProjects, 2)
	assert.Len(t, collection.PythonProjects, 0)
	assert.Len(t, collection.JavaProjects, 0)

	assert.Equal(t, "app1", collection.GoProjects[0].AppName)
	assert.Equal(t, "app2", collection.GoProjects[1].AppName)
}

// Integration test for multi-language discovery
func TestDiscoverAllProjectsIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Go project
	goDir := filepath.Join(tmpDir, "go-service")
	err := os.MkdirAll(goDir, 0755)
	require.NoError(t, err)

	goModContent := "module github.com/example/go-service\n\ngo 1.21\n"
	err = os.WriteFile(filepath.Join(goDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	mainGoContent := "package main\n\nfunc main() {\n\tprintln(\"Go service\")\n}\n"
	err = os.WriteFile(filepath.Join(goDir, "main.go"), []byte(mainGoContent), 0644)
	require.NoError(t, err)

	// Create Python project
	pythonDir := filepath.Join(tmpDir, "python-app")
	err = os.MkdirAll(pythonDir, 0755)
	require.NoError(t, err)

	reqContent := "flask==2.0.1\n"
	err = os.WriteFile(filepath.Join(pythonDir, "requirements.txt"), []byte(reqContent), 0644)
	require.NoError(t, err)

	appPyContent := "from flask import Flask\napp = Flask(__name__)\n"
	err = os.WriteFile(filepath.Join(pythonDir, "app.py"), []byte(appPyContent), 0644)
	require.NoError(t, err)

	// Create Java project
	javaDir := filepath.Join(tmpDir, "java-lib")
	err = os.MkdirAll(filepath.Join(javaDir, "src/main/java"), 0755)
	require.NoError(t, err)

	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <groupId>com.example</groupId>
    <artifactId>java-lib</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
</project>`
	err = os.WriteFile(filepath.Join(javaDir, "pom.xml"), []byte(pomContent), 0644)
	require.NoError(t, err)

	utilsContent := "public class Utils { public static String helper() { return \"helper\"; } }"
	err = os.WriteFile(filepath.Join(javaDir, "src/main/java/Utils.java"), []byte(utilsContent), 0644)
	require.NoError(t, err)

	t.Run("discover all languages", func(t *testing.T) {
		options := DiscoveryOptions{
			RootDir:   tmpDir,
			Languages: AllLanguages(),
			Verbose:   false,
		}

		collection, err := DiscoverAllProjects(options)
		require.NoError(t, err)

		counts := collection.CountByType()
		assert.Equal(t, 1, counts[ProjectTypeGo])
		assert.Equal(t, 1, counts[ProjectTypePython])
		assert.Equal(t, 1, counts[ProjectTypeJava])

		allProjects := collection.AllProjects()
		assert.Len(t, allProjects, 3)
	})

	t.Run("discover only go", func(t *testing.T) {
		options := DiscoveryOptions{
			RootDir:   tmpDir,
			Languages: OnlyGo(),
			Verbose:   false,
		}

		collection, err := DiscoverAllProjects(options)
		require.NoError(t, err)

		counts := collection.CountByType()
		assert.Equal(t, 1, counts[ProjectTypeGo])
		assert.Equal(t, 0, counts[ProjectTypePython])
		assert.Equal(t, 0, counts[ProjectTypeJava])
	})

	t.Run("discover python and java only", func(t *testing.T) {
		filter := LanguageFilter{
			Go:     false,
			Python: true,
			Java:   true,
		}

		options := DiscoveryOptions{
			RootDir:   tmpDir,
			Languages: filter,
			Verbose:   false,
		}

		collection, err := DiscoverAllProjects(options)
		require.NoError(t, err)

		counts := collection.CountByType()
		assert.Equal(t, 0, counts[ProjectTypeGo])
		assert.Equal(t, 1, counts[ProjectTypePython])
		assert.Equal(t, 1, counts[ProjectTypeJava])
	})
}

func TestDiscoverProjectsConvenienceFunctions(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple Go project for testing
	goModContent := "module github.com/example/test\n\ngo 1.21\n"
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	mainGoContent := "package main\n\nfunc main() {}\n"
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGoContent), 0644)
	require.NoError(t, err)

	t.Run("DiscoverProjects", func(t *testing.T) {
		collection, err := DiscoverProjects(tmpDir)
		require.NoError(t, err)
		assert.False(t, collection.IsEmpty())
	})

	t.Run("DiscoverProjectsWithFilter", func(t *testing.T) {
		collection, err := DiscoverProjectsWithFilter(tmpDir, OnlyGo())
		require.NoError(t, err)

		counts := collection.CountByType()
		assert.Equal(t, 1, counts[ProjectTypeGo])
		assert.Equal(t, 0, counts[ProjectTypePython])
		assert.Equal(t, 0, counts[ProjectTypeJava])
	})

	t.Run("DiscoverAndGenerateBuildGroupsMultiLang", func(t *testing.T) {
		buildGroups, err := DiscoverAndGenerateBuildGroupsMultiLang(tmpDir)
		require.NoError(t, err)
		assert.Len(t, buildGroups, 1)
		assert.Len(t, buildGroups[0].Builds, 1)
	})

	t.Run("DiscoverAndGenerateBuildGroupsWithFilter", func(t *testing.T) {
		buildGroups, err := DiscoverAndGenerateBuildGroupsWithFilter(tmpDir, OnlyGo())
		require.NoError(t, err)
		assert.Len(t, buildGroups, 1)
	})
}

func TestDiscoveryErrorHandling(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		collection, err := DiscoverProjects(tmpDir)
		require.NoError(t, err)
		assert.True(t, collection.IsEmpty())

		_, err = DiscoverAndGenerateBuildGroupsMultiLang(tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no projects found")
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		nonexistentDir := "/nonexistent/directory"

		collection, err := DiscoverProjects(nonexistentDir)
		// Should not error but find no projects
		require.NoError(t, err)
		assert.True(t, collection.IsEmpty())
	})
}

func TestProjectInterface(t *testing.T) {
	// Test that all project types properly implement the Project interface
	var projects []Project

	goProject := Project{
		AppName:     "go-app",
		ModulePath:  "./go-app",
		IsService:   true,
		SourceFiles: []string{"main.go"},
	}

	pythonProject := Project{
		ModulePath:  "./python-app",
		AppName:     "python-app",
		IsService:   false,
		SourceFiles: []string{"app.py"},
	}

	javaProject := Project{
		ModulePath:  "./java-app",
		AppName:     "java-app",
		IsService:   true,
		SourceFiles: []string{"Main.java"},
	}

	// All should be assignable to Project interface
	projects = append(projects, goProject)
	projects = append(projects, pythonProject)
	projects = append(projects, javaProject)

	assert.Len(t, projects, 3)

	// Test common interface methods
	for _, project := range projects {
		assert.NotEmpty(t, project.AppName)
		assert.NotEmpty(t, project.ModulePath)
		assert.NotEmpty(t, project.BuilderFunction())
		// GetSourceFiles and IsServiceProject can be empty/false

		// Test ToBuild method
		build := project.ToBuild()
		assert.Equal(t, project.AppName, build.App)
		assert.Equal(t, project.BuilderFunction(), build.BuilderFunction)
	}
}
