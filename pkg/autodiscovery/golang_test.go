package autodiscovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/stretchr/testify/assert"
)

func TestParseGoMod(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    string
		expectError bool
	}{
		{
			name:     "valid go.mod",
			content:  "module github.com/user/myapp\n\ngo 1.21\n",
			expected: "github.com/user/myapp",
		},
		{
			name:     "go.mod with extra spaces",
			content:  "module   github.com/user/myapp   \n\ngo 1.21\n",
			expected: "github.com/user/myapp",
		},
		{
			name:        "go.mod without module",
			content:     "go 1.21\n",
			expectError: true,
		},
		{
			name:     "go.mod with comments",
			content:  "// This is a comment\nmodule github.com/user/myapp\n// Another comment\ngo 1.21\n",
			expected: "github.com/user/myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "go.mod")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content to file
			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Test parseGoMod
			result, err := parseGoMod(tmpFile.Name())

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestDeriveAppName(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		modulePath string
		expected   string
	}{
		{
			name:       "from module name",
			moduleName: "github.com/user/myapp",
			modulePath: "/path/to/project",
			expected:   "myapp",
		},
		{
			name:       "from simple module name",
			moduleName: "myapp",
			modulePath: "/path/to/project",
			expected:   "myapp",
		},
		{
			name:       "from directory when no module name",
			moduleName: "",
			modulePath: "/path/to/myproject",
			expected:   "myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deriveAppName(tt.moduleName, tt.modulePath)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsMainPackageFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "main package",
			content:  "package main\n\nfunc main() {\n}\n",
			expected: true,
		},
		{
			name:     "non-main package",
			content:  "package utils\n\nfunc Helper() {\n}\n",
			expected: false,
		},
		{
			name:     "main package with comments",
			content:  "// This is a comment\n// Another comment\npackage main\n\nfunc main() {\n}\n",
			expected: true,
		},
		{
			name:     "package with extra spaces",
			content:  "package   main   \n\nfunc main() {\n}\n",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "*.go")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content to file
			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Test isMainPackageFile
			result, err := isMainPackageFile(tmpFile.Name())
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractPackagesFromFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected []string
	}{
		{
			name:     "multiple files same directory",
			files:    []string{"./pkg/utils/file1.go", "./pkg/utils/file2.go"},
			expected: []string{"pkg/utils"},
		},
		{
			name:     "files in different directories",
			files:    []string{"./pkg/utils/file1.go", "./pkg/models/file2.go", "./cmd/main.go"},
			expected: []string{"pkg/utils", "pkg/models", "cmd"},
		},
		{
			name:     "empty file list",
			files:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPackagesFromFiles(tt.files)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d packages, got %d", len(tt.expected), len(result))
				return
			}

			// Check contents (order may vary due to map iteration)
			expectedSet := make(map[string]struct{})
			for _, pkg := range tt.expected {
				expectedSet[pkg] = struct{}{}
			}

			for _, pkg := range result {
				if _, exists := expectedSet[pkg]; !exists {
					t.Errorf("Unexpected package: %s", pkg)
				}
			}
		})
	}
}

func TestGoProjectToBuild(t *testing.T) {
	tests := []struct {
		expected func(container.Build)
		name     string
		project  GoProject
	}{
		{
			name: "service project",
			project: GoProject{
				ModulePath: "./testapp",
				ModuleName: "github.com/user/testapp",
				AppName:    "testapp",
				IsService:  true,
				MainFile:   "main.go",
				ProtoFiles: []string{"./proto/service.proto"},
			},
			expected: func(build container.Build) {
				assert.Equal(t, "testapp", build.App)
				assert.Equal(t, container.GoLang, build.BuildType)
				assert.Equal(t, "./testapp", build.Folder)
				assert.Equal(t, "main.go", build.File)
				assert.Len(t, build.SourceFiles, 1)
			},
		},
		{
			name: "library project",
			project: GoProject{
				ModulePath: "./testlib",
				ModuleName: "github.com/user/testlib",
				AppName:    "testlib",
				IsService:  false,
				ProtoFiles: []string{},
			},
			expected: func(build container.Build) {
				assert.Equal(t, "testlib", build.App)
				assert.Equal(t, container.GoLang, build.BuildType)
				assert.Equal(t, "./testlib", build.Folder)
				assert.Equal(t, "", build.File)  // No main file for library
				assert.Equal(t, "", build.Image) // No image for library
				assert.Len(t, build.SourceFiles, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GoProjectToBuild(tt.project)
			tt.expected(result)
		})
	}
}

func TestGenerateBuildGroups(t *testing.T) {
	projects := []GoProject{
		{
			ModulePath:  "./testapp",
			ModuleName:  "github.com/user/testapp",
			AppName:     "testapp",
			IsService:   true,
			SourceFiles: []string{"./main.go"},
		},
		{
			ModulePath:  "./testlib",
			ModuleName:  "github.com/user/testlib",
			AppName:     "testlib",
			IsService:   false,
			SourceFiles: []string{"./lib.go"},
		},
	}

	result := GenerateBuildGroups(projects)

	// Should have one build group per project
	if len(result) != len(projects) {
		t.Errorf("Expected %d build groups, got %d", len(projects), len(result))
	}

	// Each build group should have one build
	for i, group := range result {
		if len(group.Builds) != 1 {
			t.Errorf("Build group %d should have 1 build, got %d", i, len(group.Builds))
		}
	}
}

// Integration test using temporary directory structure
func TestDiscoverGoProjectsIntegration(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create a service project
	serviceDir := filepath.Join(tmpDir, "service")
	err := os.MkdirAll(serviceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create service directory: %v", err)
	}

	// Write go.mod for service
	goModContent := "module github.com/user/service\n\ngo 1.21\n"
	err = os.WriteFile(filepath.Join(serviceDir, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Write main.go for service
	mainGoContent := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n"
	err = os.WriteFile(filepath.Join(serviceDir, "main.go"), []byte(mainGoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Create a library project
	libDir := filepath.Join(tmpDir, "lib/")
	err = os.MkdirAll(libDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create lib directory: %v", err)
	}

	libContainifyCiDir := filepath.Join(tmpDir, ".containifyci")
	err = os.MkdirAll(libContainifyCiDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .containifyci directory: %v", err)
	}

	// Write go.mod for library
	libGoModContent := "module github.com/user/lib\n\ngo 1.21\n"
	err = os.WriteFile(filepath.Join(libDir, "go.mod"), []byte(libGoModContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write lib go.mod: %v", err)
	}

	// Write go.mod for library

	libContainifyCIGoModContent := "module github.com/user/containifyci \n\ngo 1.21\n"
	err = os.WriteFile(filepath.Join(libContainifyCiDir, "go.mod"), []byte(libContainifyCIGoModContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write lib go.mod: %v", err)
	}

	// Write lib.go for library
	libGoContent := "package lib\n\nfunc Helper() string {\n\treturn \"helper\"\n}\n"
	err = os.WriteFile(filepath.Join(libDir, "lib.go"), []byte(libGoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write lib.go: %v", err)
	}

	// Test discovery
	projects, err := DiscoverGoProjects(tmpDir)
	if err != nil {
		t.Fatalf("Failed to discover projects: %v", err)
	}

	// Should find 2 projects
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	// Verify service project
	var serviceProject, libProject *GoProject
	for i := range projects {
		switch projects[i].AppName {
		case "service":
			serviceProject = &projects[i]
		case "lib":
			libProject = &projects[i]
		}
	}

	if serviceProject == nil {
		t.Error("Service project not found")
	} else {
		if !serviceProject.IsService {
			t.Error("Service project should be identified as service")
		}
		if serviceProject.ModuleName != "github.com/user/service" {
			t.Errorf("Expected module name 'github.com/user/service', got '%s'", serviceProject.ModuleName)
		}
	}

	if libProject == nil {
		t.Error("Library project not found")
	} else {
		if libProject.IsService {
			t.Error("Library project should not be identified as service")
		}
		if libProject.ModuleName != "github.com/user/lib" {
			t.Errorf("Expected module name 'github.com/user/lib', got '%s'", libProject.ModuleName)
		}
	}
}

func TestDiscoverAndGenerateBuildGroups(t *testing.T) {
	// Create temporary directory with a simple Go project
	tmpDir := t.TempDir()

	// Write go.mod
	goModContent := "module github.com/user/testapp\n\ngo 1.21\n"
	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Write main.go
	mainGoContent := "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}\n"
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write main.go: %v", err)
	}

	// Test the full workflow
	buildGroups, err := DiscoverAndGenerateBuildGroups(tmpDir)
	if err != nil {
		t.Fatalf("Failed to discover and generate build groups: %v", err)
	}

	// Should have 1 build group
	if len(buildGroups) != 1 {
		t.Errorf("Expected 1 build group, got %d", len(buildGroups))
	}

	// Build group should have 1 build
	if len(buildGroups[0].Builds) != 1 {
		t.Errorf("Expected 1 build in group, got %d", len(buildGroups[0].Builds))
	}

	// Verify build configuration
	build := buildGroups[0].Builds[0]
	if build.App != "testapp" {
		t.Errorf("Expected app name 'testapp', got '%s'", build.App)
	}
	if build.BuildType != container.GoLang {
		t.Errorf("Expected build type GoLang, got %s", build.BuildType)
	}
}
