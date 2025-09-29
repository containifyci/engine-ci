package autodiscovery

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/filesystem"
	"github.com/containifyci/engine-ci/protos2"
)

// // PythonProject represents a discovered Python project
// type PythonProject struct {
// 	ProjectRoot string
// 	AppName     string
// 	ConfigFile  string
// 	ConfigType  string
// 	MainModule  string
// 	PackageName string
// 	SourceFiles []string
// 	IsService   bool
// }

// // Implement Project interface methods
// func (p PythonProject) GetAppName() string {
// 	return p.AppName
// }

// func (p PythonProject) GetModulePath() string {
// 	return p.ProjectRoot
// }

// func (p PythonProject) IsServiceProject() bool {
// 	return p.IsService
// }

// func (p PythonProject) GetProjectType() ProjectType {
// 	return ProjectTypePython
// }

// func (p PythonProject) GetSourceFiles() []string {
// 	return p.SourceFiles
// }

// func (p PythonProject) BuilderFunction() string {
// 	if p.IsService {
// 		return "NewPythonServiceBuild"
// 	}
// 	return "NewPythonLibraryBuild"
// }

func (p Project) ToBuild() container.Build {
	switch p.BuildType {
	case protos2.BuildType_GoLang:
		return GoProjectToBuild(p)
	case protos2.BuildType_Python:
		return PythonProjectToBuild(p)
	case protos2.BuildType_Maven:
		return JavaProjectToBuild(p)
	default:
		return container.Build{}
	}
}

// DiscoverPythonProjects scans the given root directory recursively for Python projects
func DiscoverPythonProjects(rootDir string) ([]Project, error) {
	var projects []Project

	// Use filesystem package to find Python configuration files
	fileCache := filesystem.NewFileCache("python_cache.yaml")

	// Look for different types of Python project indicators
	configFiles := []string{}

	// Find requirements.txt files
	reqFiles, err := fileCache.FindFilesBySuffix(rootDir, "requirements.txt")
	if err == nil {
		configFiles = append(configFiles, reqFiles...)
	}

	// Find pyproject.toml files
	pyprojectFiles, err := fileCache.FindFilesBySuffix(rootDir, "pyproject.toml")
	if err == nil {
		configFiles = append(configFiles, pyprojectFiles...)
	}

	// Group config files by directory to identify unique projects
	projectDirs := make(map[string][]string)
	for _, configFile := range configFiles {
		dir := filepath.Dir(configFile)
		projectDirs[dir] = append(projectDirs[dir], configFile)
	}

	for projectDir, configs := range projectDirs {
		project, err := analyzePythonProject(projectDir, configs)
		if err != nil {
			slog.Warn("Failed to analyze Python project", "path", projectDir, "error", err)
			continue
		}

		// Skip if this is a subdirectory of another discovered project
		if isSubproject(projectDir, projects) {
			continue
		}

		slog.Info("Discovered Python project", "name", project.AppName, "path", project.ModulePath)
		projects = append(projects, project)
	}

	return projects, nil
}

// analyzePythonProject analyzes a single Python project based on its configuration files
func analyzePythonProject(projectDir string, configFiles []string) (Project, error) {
	project := Project{
		ModulePath: projectDir,
		BuildType:  protos2.BuildType_Python,
	}

	// Derive app name if not found in config
	if project.AppName == "" {
		project.AppName = filepath.Base(projectDir)
	}

	// Find all Python source files
	fileCache := filesystem.NewFileCache("python_files_cache.yaml")
	pyFiles, err := fileCache.FindFilesBySuffix(projectDir, ".py")
	if err != nil {
		return project, fmt.Errorf("failed to find .py files: %w", err)
	}
	project.SourceFiles = pyFiles

	// Determine if this is a service or library
	isService, mainFile := determineServiceType(projectDir, pyFiles)
	project.IsService = isService
	project.MainFile = mainFile

	return project, nil
}

// parsePyprojectToml extracts project name from pyproject.toml (basic parsing)
func parsePyprojectToml(project *Project, pyprojectFile string) error {
	file, err := os.Open(pyprojectFile)
	if err != nil {
		return fmt.Errorf("failed to open pyproject.toml: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inProjectSection := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for [project] section
		if line == "[project]" {
			inProjectSection = true
			continue
		}

		// Check for other sections (exit project section)
		if strings.HasPrefix(line, "[") && line != "[project]" {
			inProjectSection = false
			continue
		}

		// Parse name in project section
		if inProjectSection && strings.HasPrefix(line, "name") {
			if idx := strings.Index(line, "="); idx != -1 {
				nameStr := strings.TrimSpace(line[idx+1:])
				nameStr = strings.Trim(nameStr, `"'`)
				if nameStr != "" {
					project.AppName = nameStr
					// project.PackageName = nameStr
					break
				}
			}
		}
	}

	return scanner.Err()
}

// determineServiceType checks if this is a deployable service or a library
func determineServiceType(projectDir string, pyFiles []string) (bool, string) {
	// Look for __main__.py in the project root or main package
	for _, pyFile := range pyFiles {
		if filepath.Base(pyFile) == "__main__.py" {
			// Check if it's in the project root or immediate subdirectory
			relPath, err := filepath.Rel(projectDir, pyFile)
			if err == nil {
				// If __main__.py is in root or one level deep, it's likely a service
				if filepath.Dir(relPath) == "." || strings.Count(relPath, string(filepath.Separator)) <= 1 {
					return true, pyFile
				}
			}
		}
	}

	// Look for common service patterns: app.py, main.py, server.py, run.py
	serviceFiles := []string{"app.py", "main.py", "server.py", "run.py", "wsgi.py", "asgi.py"}
	for _, pyFile := range pyFiles {
		filename := filepath.Base(pyFile)
		for _, serviceFile := range serviceFiles {
			if filename == serviceFile {
				// Check if it's in the project root
				if filepath.Dir(pyFile) == projectDir {
					return true, pyFile
				}
			}
		}
	}

	// Default to library if no service indicators found
	return false, ""
}

// isSubproject checks if a directory is a subdirectory of an already discovered project
func isSubproject(dir string, projects []Project) bool {
	for _, project := range projects {
		relPath, err := filepath.Rel(project.ModulePath, dir)
		if err == nil && !strings.HasPrefix(relPath, "..") && relPath != "." {
			return true
		}
	}
	return false
}

// PythonProjectToBuild converts a discovered Python project to a container.Build configuration
func PythonProjectToBuild(project Project) container.Build {
	build := container.NewPythonServiceBuild(project.AppName)
	build.BuilderFunction = project.BuilderFunction()

	if !project.IsService {
		build.Image = ""
	}

	if project.MainFile != "" {
		build.File = project.MainFile
	}

	if project.ModulePath != "" {
		build.Folder = project.ModulePath
	}

	// Add Python-specific source files and packages
	if len(project.SourceFiles) > 0 {
		build.SourceFiles = project.SourceFiles
		build.SourcePackages = extractPackagesFromFiles(project.SourceFiles)
	}

	return build
}
