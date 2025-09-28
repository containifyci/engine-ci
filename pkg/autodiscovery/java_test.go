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

func TestParsePomXml(t *testing.T) {
	tests := []struct {
		name    string
		content string
		appName string
	}{
		{
			name: "basic pom.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>myapp</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
</project>`,
			appName: "myapp",
		},
		{
			name: "war packaging",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <groupId>com.example</groupId>
    <artifactId>webapp</artifactId>
    <version>2.1.0</version>
    <packaging>war</packaging>
</project>`,
			appName: "webapp",
		},
		{
			name: "no packaging specified",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<project>
    <groupId>com.example</groupId>
    <artifactId>defaultapp</artifactId>
    <version>1.0.0</version>
</project>`,
			appName: "defaultapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "pom.xml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			require.NoError(t, err)
			tmpFile.Close()

			project := &Project{}
			err = parsePomXml(project, tmpFile.Name())
			require.NoError(t, err)

			assert.Equal(t, tt.appName, project.AppName)
		})
	}
}

func TestExtractXmlContent(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		tagName  string
		expected string
	}{
		{
			name:     "simple tag",
			line:     "<groupId>com.example</groupId>",
			tagName:  "groupId",
			expected: "com.example",
		},
		{
			name:     "tag with spaces",
			line:     "<version>  1.0.0  </version>",
			tagName:  "version",
			expected: "1.0.0",
		},
		{
			name:     "no matching tag",
			line:     "<groupId>com.example</groupId>",
			tagName:  "version",
			expected: "",
		},
		{
			name:     "incomplete tag",
			line:     "<groupId>com.example",
			tagName:  "groupId",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractXmlContent(tt.line, tt.tagName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasMainMethod(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name: "has main method",
			content: `package com.example;

public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
    }
}`,
			expected: true,
		},
		{
			name: "no main method",
			content: `package com.example;

public class Utils {
    public static String helper() {
        return "helper";
    }
}`,
			expected: false,
		},
		{
			name: "main method with different formatting",
			content: `package com.example;

public class App {
    public static void main(String... args) {
        // Main method with varargs
    }
}`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(tmpDir, "Test.java")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			require.NoError(t, err)
			defer os.Remove(tmpFile)

			result := hasMainMethod(tmpFile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasSpringBootIndicators(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Spring Boot application file
	springBootContent := `package com.example;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}`

	springBootFile := filepath.Join(tmpDir, "Application.java")
	err := os.WriteFile(springBootFile, []byte(springBootContent), 0644)
	require.NoError(t, err)

	// Create a regular Java file
	regularContent := `package com.example;

public class Utils {
    public static String helper() {
        return "helper";
    }
}`

	regularFile := filepath.Join(tmpDir, "Utils.java")
	err = os.WriteFile(regularFile, []byte(regularContent), 0644)
	require.NoError(t, err)

	javaFiles := []string{springBootFile, regularFile}

	// Test with Spring Boot indicator
	result := hasSpringBootIndicators(javaFiles, tmpDir)
	assert.True(t, result)

	// Test without Spring Boot indicator
	regularFiles := []string{regularFile}
	result = hasSpringBootIndicators(regularFiles, tmpDir)
	assert.False(t, result)
}

func TestJavaProjectToBuild(t *testing.T) {
	tests := []struct {
		expected func(container.Build)
		project  Project
		name     string
	}{
		{
			name: "maven service project",
			project: Project{
				ModulePath:  "./testapp",
				AppName:     "testapp",
				IsService:   true,
				SourceFiles: []string{"Main.java", "Utils.java"},
				BuildType:   protos2.BuildType_Maven,
			},
			expected: func(build container.Build) {
				assert.Equal(t, "testapp", build.App)
				assert.Equal(t, container.Maven, build.BuildType)
				assert.Equal(t, "./testapp", build.Folder)
				assert.Equal(t, "NewMavenServiceBuild", build.BuilderFunction)
			},
		},
		{
			name: "gradle war project",
			project: Project{
				ModulePath:  "./webapp",
				AppName:     "webapp",
				IsService:   true,
				SourceFiles: []string{"WebApp.java"},
				BuildType:   protos2.BuildType_Maven,
			},
			expected: func(build container.Build) {
				assert.Equal(t, "webapp", build.App)
				assert.Equal(t, container.Maven, build.BuildType)
				assert.Equal(t, "./webapp", build.Folder)
				assert.Equal(t, "NewMavenServiceBuild", build.BuilderFunction)
			},
		},
		{
			name: "library project",
			project: Project{
				ModulePath:  "./testlib",
				AppName:     "testlib",
				IsService:   false,
				SourceFiles: []string{"Lib.java"},
				BuildType:   protos2.BuildType_Maven,
			},
			expected: func(build container.Build) {
				assert.Equal(t, "testlib", build.App)
				assert.Equal(t, container.Maven, build.BuildType)
				assert.Equal(t, "", build.Image) // Library has no image
				assert.Equal(t, "NewMavenLibraryBuild", build.BuilderFunction)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JavaProjectToBuild(tt.project)
			tt.expected(result)
		})
	}
}

// Integration test for Java project discovery
func TestDiscoverJavaProjectsIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create Maven service project
	mavenDir := filepath.Join(tmpDir, "maven-service")
	err := os.MkdirAll(filepath.Join(mavenDir, "src/main/java/com/example"), 0755)
	require.NoError(t, err)

	// Write pom.xml
	pomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>maven-service</artifactId>
    <version>1.0.0</version>
    <packaging>jar</packaging>
</project>`
	err = os.WriteFile(filepath.Join(mavenDir, "pom.xml"), []byte(pomContent), 0644)
	require.NoError(t, err)

	// Write Main.java
	mainContent := `package com.example;

public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, Maven!");
    }
}`
	err = os.WriteFile(filepath.Join(mavenDir, "src/main/java/com/example/Main.java"), []byte(mainContent), 0644)
	require.NoError(t, err)

	// Create Gradle library project
	gradleDir := filepath.Join(tmpDir, "gradle-lib")
	err = os.MkdirAll(filepath.Join(gradleDir, "src/main/java/com/example"), 0755)
	require.NoError(t, err)

	// Write build.gradle
	gradleContent := `plugins {
    id 'java'
}

version = '2.0.0'
group = 'com.example'

repositories {
    mavenCentral()
}`
	err = os.WriteFile(filepath.Join(gradleDir, "build.gradle"), []byte(gradleContent), 0644)
	require.NoError(t, err)

	// Write Utils.java (no main method - library)
	utilsContent := `package com.example;

public class Utils {
    public static String helper() {
        return "helper";
    }
}`
	err = os.WriteFile(filepath.Join(gradleDir, "src/main/java/com/example/Utils.java"), []byte(utilsContent), 0644)
	require.NoError(t, err)

	// Create Spring Boot WAR project
	springDir := filepath.Join(tmpDir, "spring-webapp")
	err = os.MkdirAll(filepath.Join(springDir, "src/main/java/com/example"), 0755)
	require.NoError(t, err)

	// Write pom.xml for WAR
	warPomContent := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>spring-webapp</artifactId>
    <version>1.0.0</version>
    <packaging>war</packaging>
</project>`
	err = os.WriteFile(filepath.Join(springDir, "pom.xml"), []byte(warPomContent), 0644)
	require.NoError(t, err)

	// Write Spring Boot Application
	springContent := `package com.example;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class Application {
    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}`
	err = os.WriteFile(filepath.Join(springDir, "src/main/java/com/example/Application.java"), []byte(springContent), 0644)
	require.NoError(t, err)

	// Test discovery
	projects, err := DiscoverJavaProjects(tmpDir)
	require.NoError(t, err)

	// Should find 3 projects
	assert.Len(t, projects, 3)

	// Verify projects by name
	projectsByName := make(map[string]Project)
	for _, project := range projects {
		projectsByName[project.AppName] = project
	}

	// Check Maven service
	mavenProject, exists := projectsByName["maven-service"]
	require.True(t, exists, "Maven service project not found")
	assert.True(t, mavenProject.IsService)
	assert.Equal(t, "maven-service", mavenProject.AppName)

	// Check Gradle library
	gradleProject, exists := projectsByName["gradle-lib"]
	require.True(t, exists, "Gradle library project not found")
	assert.False(t, gradleProject.IsService)
	assert.Equal(t, "gradle-lib", gradleProject.AppName)

	assert.Contains(t, gradleProject.ModulePath, "gradle-lib")
	// assert.Equal(t, "gradle", gradleProject.ConfigType)
	// assert.Equal(t, "jar", gradleProject.PackagingType)

	// Check Spring Boot WAR
	springProject, exists := projectsByName["spring-webapp"]
	require.True(t, exists, "Spring webapp project not found")
	assert.True(t, springProject.IsService)
	assert.Contains(t, springProject.ModulePath, "spring-webapp")
	assert.Equal(t, "spring-webapp", springProject.AppName)
}
