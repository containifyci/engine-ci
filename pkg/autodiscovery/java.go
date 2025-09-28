package autodiscovery

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/filesystem"
	"github.com/containifyci/engine-ci/protos2"
)

// // JavaProject represents a discovered Java/Maven project
// type JavaProject struct {
// 	ProjectRoot   string
// 	AppName       string
// 	ConfigFile    string
// 	ConfigType    string
// 	PackagingType string
// 	GroupId       string
// 	ArtifactId    string
// 	Version       string
// 	MainClass     string
// 	SourceFiles   []string
// 	IsService     bool
// }

// // Implement Project interface methods
// func (p JavaProject) GetAppName() string {
// 	return p.AppName
// }

// func (p JavaProject) GetModulePath() string {
// 	return p.ProjectRoot
// }

// func (p JavaProject) IsServiceProject() bool {
// 	return p.IsService
// }

// func (p JavaProject) GetProjectType() ProjectType {
// 	return ProjectTypeJava
// }

// func (p JavaProject) GetSourceFiles() []string {
// 	return p.SourceFiles
// }

// func (p JavaProject) BuilderFunction() string {
// 	if p.IsService {
// 		return "NewMavenServiceBuild"
// 	}
// 	return "NewMavenLibraryBuild"
// }

// func (p JavaProject) ToBuild() container.Build {
// 	return JavaProjectToBuild(p)
// }

// DiscoverJavaProjects scans the given root directory recursively for Java projects
func DiscoverJavaProjects(rootDir string) ([]Project, error) {
	var projects []Project

	// Use filesystem package to find Java configuration files
	fileCache := filesystem.NewFileCache("java_cache.yaml")

	// Look for Maven and Gradle project indicators
	configFiles := []string{}

	// Find pom.xml files (Maven)
	pomFiles, err := fileCache.FindFilesBySuffix(rootDir, "pom.xml")
	if err == nil {
		configFiles = append(configFiles, pomFiles...)
	}

	// Find build.gradle files (Gradle)
	gradleFiles, err := fileCache.FindFilesBySuffix(rootDir, "build.gradle")
	if err == nil {
		configFiles = append(configFiles, gradleFiles...)
	}

	// Find build.gradle.kts files (Gradle Kotlin DSL)
	gradleKtsFiles, err := fileCache.FindFilesBySuffix(rootDir, "build.gradle.kts")
	if err == nil {
		configFiles = append(configFiles, gradleKtsFiles...)
	}

	for _, configFile := range configFiles {
		projectDir := filepath.Dir(configFile)

		// Skip if this is a subdirectory of another discovered project
		if isJavaSubproject(projectDir, projects) {
			continue
		}

		project, err := analyzeJavaProject(projectDir, configFile)
		if err != nil {
			fmt.Printf("Warning: Failed to analyze Java project at %s: %v\n", projectDir, err)
			continue
		}

		fmt.Printf("Discovered Java project: %s (path: %s) (type: %s)\n",
			project.AppName, project.ModulePath, project.BuildType)
		projects = append(projects, project)
	}

	return projects, nil
}

// analyzeJavaProject analyzes a single Java project based on its configuration file
func analyzeJavaProject(projectDir, configFile string) (Project, error) {
	project := Project{
		ModulePath: projectDir,
		BuildType:  protos2.BuildType_Maven,
	}

	// Parse the configuration file
	err := parseJavaConfig(&project, configFile)
	if err != nil {
		return project, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Derive app name if not found in config
	if project.AppName == "" {
		project.AppName = filepath.Base(projectDir)
	}

	// Find all Java source files
	fileCache := filesystem.NewFileCache("java_files_cache.yaml")
	javaFiles, err := fileCache.FindFilesBySuffix(projectDir, ".java")
	if err != nil {
		return project, fmt.Errorf("failed to find .java files: %w", err)
	}
	project.SourceFiles = javaFiles

	// Determine if this is a service or library based on packaging type
	project.IsService = isJavaService(javaFiles, projectDir)

	// Find main class for services
	if project.IsService {
		mainClass := findMainClass(javaFiles)
		project.MainFile = mainClass
	}

	return project, nil
}

func isJavaService(javaFiles []string, projectDir string) bool {
	// Check for main method in any Java file
	for _, javaFile := range javaFiles {
		if hasMainMethod(javaFile) {
			return true
		}
	}

	return false
}

// parseJavaConfig extracts project metadata from the configuration file
func parseJavaConfig(project *Project, configFile string) error {
	if strings.HasSuffix(configFile, "pom.xml") {
		return parsePomXml(project, configFile)
	} else if strings.HasSuffix(configFile, "build.gradle") || strings.HasSuffix(configFile, "build.gradle.kts") {
		return parseBuildGradle(project, configFile)
	}
	return fmt.Errorf("unsupported Java config file: %s", configFile)
}

// parsePomXml extracts project metadata from pom.xml (basic XML parsing)
func parsePomXml(project *Project, pomFile string) error {
	file, err := os.Open(pomFile)
	if err != nil {
		return fmt.Errorf("failed to open pom.xml: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var inProject bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Simple XML parsing - look for opening tags
		if strings.HasPrefix(line, "<project") {
			inProject = true
			continue
		}

		if !inProject {
			continue
		}

		// // Extract tag content (basic approach)
		// if strings.HasPrefix(line, "<groupId>") && strings.HasSuffix(line, "</groupId>") {
		// 	content := extractXmlContent(line, "groupId")
		// 	if content != "" && project.GroupId == "" {
		// 		project.GroupId = content
		// 	}
		// }

		if strings.HasPrefix(line, "<artifactId>") && strings.HasSuffix(line, "</artifactId>") {
			content := extractXmlContent(line, "artifactId")
			if content != "" {
				project.AppName = content
			}
		}

		// if strings.HasPrefix(line, "<version>") && strings.HasSuffix(line, "</version>") {
		// 	content := extractXmlContent(line, "version")
		// 	if content != "" && project.Version == "" {
		// 		project.Version = content
		// 	}
		// }

		// if strings.HasPrefix(line, "<packaging>") && strings.HasSuffix(line, "</packaging>") {
		// 	content := extractXmlContent(line, "packaging")
		// 	if content != "" {
		// 		project.PackagingType = content
		// 	}
		// }
	}

	// // Default packaging type for Maven is jar
	// if project.PackagingType == "" {
	// 	project.PackagingType = "jar"
	// }

	return scanner.Err()
}

// parseBuildGradle extracts project metadata from build.gradle (basic parsing)
func parseBuildGradle(project *Project, gradleFile string) error {
	file, err := os.Open(gradleFile)
	if err != nil {
		return fmt.Errorf("failed to open build.gradle: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		// line := strings.TrimSpace(scanner.Text())

		// // Look for version assignment
		// if strings.HasPrefix(line, "version") && strings.Contains(line, "=") {
		// 	if idx := strings.Index(line, "="); idx != -1 {
		// 		version := strings.TrimSpace(line[idx+1:])
		// 		version = strings.Trim(version, `"'`)
		// 		if version != "" {
		// 			project.Version = version
		// 		}
		// 	}
		// }

		// // Look for war plugin (highest priority) - either "id 'war'" or "war" with "plugin"
		// if strings.Contains(line, "war") && (strings.Contains(line, "plugin") || strings.Contains(line, "id")) {
		// 	project.PackagingType = "war"
		// }

		// // Look for application plugin (indicates it's a service)
		// if strings.Contains(line, "application") && (strings.Contains(line, "plugin") || strings.Contains(line, "apply") || strings.Contains(line, "id")) {
		// 	if project.PackagingType == "" {
		// 		project.PackagingType = "jar"
		// 	}
		// }
	}

	// // Default packaging type for Gradle is jar
	// if project.PackagingType == "" {
	// 	project.PackagingType = "jar"
	// }

	return scanner.Err()
}

// extractXmlContent extracts content from simple XML tags
func extractXmlContent(line, tagName string) string {
	openTag := "<" + tagName + ">"
	closeTag := "</" + tagName + ">"

	if strings.HasPrefix(line, openTag) && strings.HasSuffix(line, closeTag) {
		content := line[len(openTag) : len(line)-len(closeTag)]
		return strings.TrimSpace(content)
	}

	return ""
}

// hasMainMethod checks if a Java file contains a main method
func hasMainMethod(javaFile string) bool {
	file, err := os.Open(javaFile)
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Look for main method signature
		if strings.Contains(line, "public static void main") {
			return true
		}
	}

	return false
}

// hasSpringBootIndicators checks for Spring Boot application indicators
func hasSpringBootIndicators(javaFiles []string, projectDir string) bool {
	// Look for @SpringBootApplication annotation
	for _, javaFile := range javaFiles {
		file, err := os.Open(javaFile)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.Contains(line, "@SpringBootApplication") {
				file.Close()
				return true
			}
		}
		file.Close()
	}

	return false
}

// findMainClass attempts to find the main class in Java source files
func findMainClass(javaFiles []string) string {
	for _, javaFile := range javaFiles {
		if hasMainMethod(javaFile) {
			// Extract class name from file path
			fileName := filepath.Base(javaFile)
			className := strings.TrimSuffix(fileName, ".java")
			return className
		}
	}
	return ""
}

// isJavaSubproject checks if a directory is a subdirectory of an already discovered project
func isJavaSubproject(dir string, projects []Project) bool {
	for _, project := range projects {
		relPath, err := filepath.Rel(project.ModulePath, dir)
		if err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
			return true
		}
	}
	return false
}

// JavaProjectToBuild converts a discovered Java project to a container.Build configuration
func JavaProjectToBuild(project Project) container.Build {
	build := container.NewMavenServiceBuild(project.AppName)
	build.BuilderFunction = project.BuilderFunction()

	if !project.IsService {
		build.Image = ""
	}

	if project.ModulePath != "" {
		build.Folder = project.ModulePath
	}

	// Add Java-specific source files and packages
	// if len(project.SourceFiles) > 0 {
	// 	build.SourceFiles = project.SourceFiles
	// 	build.SourcePackages = extractPackagesFromFiles(project.SourceFiles)
	// }

	return build
}
