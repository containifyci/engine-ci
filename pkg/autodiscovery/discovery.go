package autodiscovery

import (
	"fmt"
	"log/slog"

	"github.com/containifyci/engine-ci/pkg/container"
)

// LanguageFilter represents which languages to discover
type LanguageFilter struct {
	Go     bool
	Python bool
	Java   bool
}

// AllLanguages returns a filter that includes all supported languages
func AllLanguages() LanguageFilter {
	return LanguageFilter{
		Go:     true,
		Python: true,
		Java:   true,
	}
}

// OnlyGo returns a filter that includes only Go projects (for backward compatibility)
func OnlyGo() LanguageFilter {
	return LanguageFilter{
		Go:     true,
		Python: false,
		Java:   false,
	}
}

// DiscoveryOptions holds configuration for project discovery
type DiscoveryOptions struct {
	RootDir   string
	Languages LanguageFilter
	Verbose   bool
}

// DiscoverAllProjects scans for projects in all supported languages
func DiscoverAllProjects(options DiscoveryOptions) (*ProjectCollection, error) {
	collection := &ProjectCollection{}

	if options.Verbose {
		slog.Info("Starting multi-language project discovery", "rootDir", options.RootDir)
	}

	// Discover Go projects
	if options.Languages.Go {
		if options.Verbose {
			slog.Info("Discovering Go projects...")
		}
		goProjects, err := DiscoverGoProjects(options.RootDir)
		if err != nil {
			slog.Warn("Failed to discover Go projects", "error", err)
		} else {
			collection.GoProjects = goProjects
			if options.Verbose {
				slog.Info("Discovered Go projects", "count", len(goProjects))
			}
		}
	}

	// Discover Python projects
	if options.Languages.Python {
		if options.Verbose {
			slog.Info("Discovering Python projects...")
		}
		pythonProjects, err := DiscoverPythonProjects(options.RootDir)
		if err != nil {
			slog.Warn("Failed to discover Python projects", "error", err)
		} else {
			collection.PythonProjects = pythonProjects
			if options.Verbose {
				slog.Info("Discovered Python projects", "count", len(pythonProjects))
			}
		}
	}

	// Discover Java projects
	if options.Languages.Java {
		if options.Verbose {
			slog.Info("Discovering Java projects...")
		}
		javaProjects, err := DiscoverJavaProjects(options.RootDir)
		if err != nil {
			slog.Warn("Failed to discover Java projects", "error", err)
		} else {
			collection.JavaProjects = javaProjects
			if options.Verbose {
				slog.Info("Discovered Java projects", "count", len(javaProjects))
			}
		}
	}

	if options.Verbose {
		logDiscoverySummary(collection)
	}

	return collection, nil
}

// DiscoverProjects is a convenience function that discovers all projects with default options
func DiscoverProjects(rootDir string) (*ProjectCollection, error) {
	options := DiscoveryOptions{
		RootDir:   rootDir,
		Languages: AllLanguages(),
		Verbose:   false,
	}
	return DiscoverAllProjects(options)
}

// DiscoverProjectsWithFilter discovers projects for specific languages
func DiscoverProjectsWithFilter(rootDir string, filter LanguageFilter) (*ProjectCollection, error) {
	options := DiscoveryOptions{
		RootDir:   rootDir,
		Languages: filter,
		Verbose:   false,
	}
	return DiscoverAllProjects(options)
}

// logDiscoverySummary logs a summary of discovered projects
func logDiscoverySummary(collection *ProjectCollection) {
	counts := collection.CountByType()
	totalProjects := len(collection.AllProjects())

	slog.Info("Project discovery completed",
		"total", totalProjects,
		"go", counts[ProjectTypeGo],
		"python", counts[ProjectTypePython],
		"java", counts[ProjectTypeJava])

	// Log details about discovered projects
	for _, project := range collection.AllProjects() {
		slog.Info("Found project",
			"name", project.AppName,
			"type", project.BuildType,
			"path", project.ModulePath,
			"isService", project.IsService)
	}
}

// GenerateBuildGroupsFromCollection creates container.BuildGroups from a project collection
func GenerateBuildGroupsFromCollection(collection *ProjectCollection) container.BuildGroups {
	var groups container.BuildGroups

	// Convert all projects to builds
	allProjects := collection.AllProjects()

	// TODO: Implement dependency-aware build ordering
	// For now, create individual build groups for each project
	for _, project := range allProjects {
		slog.Info("Generating build for project",
			"name", project.AppName,
			"type", project.BuildType,
			"isService", project.IsService)

		build := project.ToBuild()
		build.Defaults()

		// Create a build group with this single build
		group := &container.BuildGroup{
			Builds: []*container.Build{&build},
		}

		groups = append(groups, group)
	}

	return groups
}

// DiscoverAndGenerateBuildGroupsMultiLang is the multi-language equivalent of the Go-only function
func DiscoverAndGenerateBuildGroupsMultiLang(rootDir string) (container.BuildGroups, error) {
	collection, err := DiscoverProjects(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover projects: %w", err)
	}

	if collection.IsEmpty() {
		return nil, fmt.Errorf("no projects found in %s", rootDir)
	}

	return GenerateBuildGroupsFromCollection(collection), nil
}

// DiscoverAndGenerateBuildGroupsWithFilter discovers projects with language filtering
func DiscoverAndGenerateBuildGroupsWithFilter(rootDir string, filter LanguageFilter) (container.BuildGroups, error) {
	collection, err := DiscoverProjectsWithFilter(rootDir, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to discover projects: %w", err)
	}

	if collection.IsEmpty() {
		return nil, fmt.Errorf("no projects found in %s", rootDir)
	}

	return GenerateBuildGroupsFromCollection(collection), nil
}

// Legacy Functions for Backward Compatibility

// DiscoverGoProjectsAndGenerateBuildGroups maintains backward compatibility
func DiscoverGoProjectsAndGenerateBuildGroups(rootDir string) (container.BuildGroups, error) {
	// Use the existing Go-only implementation for backward compatibility
	return DiscoverAndGenerateBuildGroups(rootDir)
}

// ConvertGoProjectsToCollection converts existing Go projects to the new collection format
func ConvertGoProjectsToCollection(goProjects []Project) *ProjectCollection {
	return &ProjectCollection{
		GoProjects:     goProjects,
		PythonProjects: []Project{},
		JavaProjects:   []Project{},
	}
}
