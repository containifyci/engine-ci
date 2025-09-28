package autodiscovery

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/filesystem"
)

// GoProject represents a discovered Go project/module
type GoProject struct {
	MainFile    string
	ModulePath  string
	ModuleName  string
	AppName     string
	SourceFiles []string
	ProtoFiles  []string
	IsService   bool
}

func (p *GoProject) BuilderFunction() string {
	if p.IsService {
		return "NewGoServiceBuild"
	}
	return "NewGoLibraryBuild"
}

// DiscoverGoProjects scans the given root directory recursively for Go projects
func DiscoverGoProjects(rootDir string) ([]GoProject, error) {
	var projects []GoProject

	// Use filesystem package to find all go.mod files
	fileCache := filesystem.NewFileCache("go_mod_cache.yaml")
	goModFiles, err := fileCache.FindFilesBySuffix(rootDir, "go.mod")
	if err != nil {
		return nil, fmt.Errorf("failed to find go.mod files: %w", err)
	}

	for _, goModPath := range goModFiles {
		project, err := analyzeGoProject(goModPath)
		moduleFolder, _ := strings.CutPrefix(project.ModulePath, rootDir)
		moduleFolder = filepath.Base(moduleFolder)
		fmt.Printf("Discovered Go project: %s (path: %s) (folder: %s)\n", project.AppName, project.ModulePath, moduleFolder)
		if moduleFolder == "containifyci" || moduleFolder == "dagger" ||
			moduleFolder == ".containifyci" || moduleFolder == ".dagger" {
			fmt.Println("Skipping .containifyci directory")
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to analyze project at %s: %w", goModPath, err)
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// analyzeGoProject analyzes a single Go project based on its go.mod file
func analyzeGoProject(goModPath string) (GoProject, error) {
	project := GoProject{
		ModulePath: filepath.Dir(goModPath),
	}

	// Parse go.mod to get module name
	moduleName, err := parseGoMod(goModPath)
	if err != nil {
		return project, fmt.Errorf("failed to parse go.mod: %w", err)
	}
	project.ModuleName = moduleName

	// Derive app name from module name or directory
	project.AppName = deriveAppName(moduleName, project.ModulePath)

	// Find all .go files in the project
	fileCache := filesystem.NewFileCache("go_files_cache.yaml")
	goFiles, err := fileCache.FindFilesBySuffix(project.ModulePath, ".go")
	if err != nil {
		return project, fmt.Errorf("failed to find .go files: %w", err)
	}
	project.SourceFiles = goFiles

	// Find all .proto files for compatibility with existing logic
	protoFiles, err := fileCache.FindFilesBySuffix(project.ModulePath, ".proto")
	if err != nil {
		// Proto files are optional, so we can ignore this error
		protoFiles = []string{}
	}
	project.ProtoFiles = protoFiles

	// Determine if this is a service (has main package) or library
	isService, goFile, err := hasMainPackage(project.ModulePath, goFiles)
	if err != nil {
		return project, fmt.Errorf("failed to check for main package: %w", err)
	}
	project.IsService = isService
	project.MainFile = goFile

	return project, nil
}

// parseGoMod extracts the module name from a go.mod file
func parseGoMod(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading go.mod: %w", err)
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}

// deriveAppName creates an application name from the module name or directory
func deriveAppName(moduleName, modulePath string) string {
	// Try to extract from module name first (e.g., "github.com/user/app" -> "app")
	if moduleName != "" {
		parts := strings.Split(moduleName, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Fallback to directory name
	return filepath.Base(modulePath)
}

// hasMainPackage checks if any of the Go files contains a main package
func hasMainPackage(modulePath string, goFiles []string) (bool, string, error) {
	for _, goFile := range goFiles {
		fmt.Printf("Examining Go file: %s != %s\n", filepath.Dir(goFile), modulePath)
		if filepath.Dir(goFile) != modulePath {
			continue
		}
		fmt.Printf("Checking Go file for main package: %s\n", goFile)
		isMain, err := isMainPackageFile(goFile)
		if err != nil {
			return false, "", fmt.Errorf("failed to check file %s: %w", goFile, err)
		}
		if isMain {
			return true, goFile, nil
		}
	}
	return false, "", nil
}

// isMainPackageFile checks if a single Go file declares package main
func isMainPackageFile(goFile string) (bool, error) {
	file, err := os.Open(goFile)
	if err != nil {
		return false, fmt.Errorf("failed to open %s: %w", goFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || line == "" {
			continue
		}
		// Check for package declaration
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[1] == "main" {
				return true, nil
			}
			// Package declaration found but not main, can stop here
			return false, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("error reading %s: %w", goFile, err)
	}

	return false, nil
}

// GoProjectToBuild converts a discovered Go project to a container.Build configuration
func GoProjectToBuild(project GoProject) container.Build {

	build := container.NewGoServiceBuild(project.AppName)
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

	if len(project.ProtoFiles) > 0 {
		build.SourceFiles = project.ProtoFiles
		build.SourcePackages = extractPackagesFromFiles(project.ProtoFiles)
	}

	return build
}

// extractPackagesFromFiles extracts unique directory paths from a list of files
func extractPackagesFromFiles(files []string) []string {
	packageSet := make(map[string]struct{})
	var packages []string

	for _, file := range files {
		pkg := filepath.Dir(file)
		if _, exists := packageSet[pkg]; !exists {
			packageSet[pkg] = struct{}{}
			packages = append(packages, pkg)
		}
	}

	return packages
}

// GenerateBuildGroups creates container.BuildGroups from discovered Go projects
func GenerateBuildGroups(projects []GoProject) container.BuildGroups {
	var groups container.BuildGroups

	//TODO support concurrent builds based on project dependencies
	for _, project := range projects {
		fmt.Printf("Generating build for project: %s (Service: %t)\n", project.AppName, project.IsService)
		build := GoProjectToBuild(project)
		build.Defaults()

		// Create a build group with this single build
		group := &container.BuildGroup{
			Builds: []*container.Build{&build},
		}

		groups = append(groups, group)
	}

	return groups
}

// DiscoverAndGenerateBuildGroups is a convenience function that combines discovery and build generation
func DiscoverAndGenerateBuildGroups(rootDir string) (container.BuildGroups, error) {
	projects, err := DiscoverGoProjects(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover Go projects: %w", err)
	}

	if len(projects) == 0 {
		return nil, fmt.Errorf("no Go projects found in %s", rootDir)
	}

	return GenerateBuildGroups(projects), nil
}
