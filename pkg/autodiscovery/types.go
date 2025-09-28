package autodiscovery

// ProjectType represents the language/framework type of a discovered project
type ProjectType string

const (
	ProjectTypeGo     ProjectType = "go"
	ProjectTypePython ProjectType = "python"
	ProjectTypeJava   ProjectType = "java"
)

// // Project represents a discovered project that can be built
// type Project interface {
// 	// GetAppName returns the application name for the project
// 	GetAppName() string

// 	// GetModulePath returns the root path of the project
// 	GetModulePath() string

// 	// IsServiceProject returns true if this is a deployable service, false if it's a library
// 	IsServiceProject() bool

// 	// GetProjectType returns the language/framework type
// 	GetProjectType() ProjectType

// 	// BuilderFunction returns the name of the client build function to use
// 	BuilderFunction() string

// 	// ToBuild converts the project to a container.Build configuration
// 	ToBuild() container.Build

// 	// GetSourceFiles returns the list of source files for the project
// 	GetSourceFiles() []string
// }

// ProjectCollection holds discovered projects from all supported languages
type ProjectCollection struct {
	GoProjects     []Project
	PythonProjects []Project
	JavaProjects   []Project
}

// AllProjects returns all discovered projects as a slice of Project interfaces
func (pc *ProjectCollection) AllProjects() []Project {
	var all []Project

	for _, p := range pc.GoProjects {
		all = append(all, p)
	}

	for _, p := range pc.PythonProjects {
		all = append(all, p)
	}

	for _, p := range pc.JavaProjects {
		all = append(all, p)
	}

	return all
}

// CountByType returns the number of projects discovered for each type
func (pc *ProjectCollection) CountByType() map[ProjectType]int {
	return map[ProjectType]int{
		ProjectTypeGo:     len(pc.GoProjects),
		ProjectTypePython: len(pc.PythonProjects),
		ProjectTypeJava:   len(pc.JavaProjects),
	}
}

// IsEmpty returns true if no projects were discovered
func (pc *ProjectCollection) IsEmpty() bool {
	return len(pc.GoProjects) == 0 && len(pc.PythonProjects) == 0 && len(pc.JavaProjects) == 0
}
