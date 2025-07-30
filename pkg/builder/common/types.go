package common

import (
	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
)

// LanguageBuild provides a common implementation of the build.Build interface
// that can be used by all language builders. This eliminates the duplicate
// XxxBuild structs (GoBuild, MavenBuild, PythonBuild, etc.) across packages.
type LanguageBuild struct {
	runFunc build.RunFunc
	name    string
	images  []string
	async   bool
}

// NewLanguageBuild creates a new LanguageBuild instance with the specified parameters.
func NewLanguageBuild(runFunc build.RunFunc, name string, images []string, async bool) *LanguageBuild {
	return &LanguageBuild{
		runFunc: runFunc,
		name:    name,
		images:  images,
		async:   async,
	}
}

// Run executes the build function.
func (l *LanguageBuild) Run() error {
	return l.runFunc()
}

// Name returns the build name.
func (l *LanguageBuild) Name() string {
	return l.name
}

// Images returns the list of images used by this build.
func (l *LanguageBuild) Images() []string {
	return l.images
}

// IsAsync returns whether this build can run asynchronously.
func (l *LanguageBuild) IsAsync() bool {
	return l.async
}

// ContainerConfig encapsulates common container configuration patterns
// used across all language builders.
type ContainerConfig struct {
	Image      string
	WorkingDir string
	Platform   string
	Script     string
	User       string
	Env        []string
	Cmd        []string
	Memory     int64
	CPU        uint64
}

// BuildOptions contains common options used during the build process
// across all language builders.
type BuildOptions struct {
	CacheDir    string
	CacheMount  string
	SourceDir   string
	SourceMount string
	OutputDir   string
	Tags        []string
	Flags       []string
	Targets     []string
	Verbose     bool
	NoCoverage  bool
}

// ImageConfiguration defines how Docker images are configured and built
// for different language builders.
type ImageConfiguration struct {
	// Base images
	BaseImage       string
	IntermediateImg string
	ProductionImg   string
	LintImage       string

	// Image building
	DockerfilePath string
	BuildContext   string
	BuildArgs      map[string]string

	// Registry configuration
	Registry     string
	Organization string
	Repository   string
}

// CacheConfiguration defines cache settings for different language builders.
type CacheConfiguration struct {
	CacheEnvVars    map[string]string
	HostCacheDir    string
	HostBuildCache  string
	ContainerCache  string
	ContainerBuild  string
	ValidateCache   bool
	CreateIfMissing bool
}

// NetworkConfiguration defines network settings for container builds.
type NetworkConfiguration struct {
	SSHAgent          string
	NetworkMode       string
	TestcontainerHost string
	DockerHost        string
	ExposedPorts      []string
	EnableSSH         bool
}

// SecurityConfiguration defines security settings for container builds.
type SecurityConfiguration struct {
	RunAsUser  string
	RunAsGroup string
	UserName   string
	GroupName  string
	AddCaps    []string
	DropCaps   []string
	Privileged bool
	ReadOnly   bool
	NoNewPrivs bool
}

// LanguageDefaults provides default configuration values for specific languages.
// This helps maintain consistency and reduces configuration overhead.
type LanguageDefaults struct {
	// Language identification
	Language  string
	BuildType container.BuildType

	// Default images
	BaseImage string
	LintImage string

	// Default versions
	LanguageVersion string
	ToolVersions    map[string]string

	// Default paths
	SourceMount string
	CacheMount  string
	OutputDir   string

	// Default environment variables
	DefaultEnv map[string]string

	// Required files for validation
	RequiredFiles []string
}

// GetGoDefaults returns default configuration for Go language builds.
func GetGoDefaults() LanguageDefaults {
	return LanguageDefaults{
		Language:        "golang",
		BuildType:       container.GoLang,
		BaseImage:       "golang:1.24.2-alpine",
		LintImage:       "golangci/golangci-lint:v2.1.2",
		LanguageVersion: "1.24.2",
		SourceMount:     "/src",
		CacheMount:      "/go/pkg",
		OutputDir:       "/out",
		DefaultEnv: map[string]string{
			"GOMODCACHE": "/go/pkg/",
			"GOCACHE":    "/go/pkg/build-cache",
		},
		RequiredFiles: []string{"go.mod"},
	}
}

// GetMavenDefaults returns default configuration for Maven language builds.
func GetMavenDefaults() LanguageDefaults {
	return LanguageDefaults{
		Language:        "maven",
		BuildType:       container.Maven,
		BaseImage:       "maven:3-eclipse-temurin-17-alpine",
		LanguageVersion: "17",
		SourceMount:     "/src",
		CacheMount:      "/root/.m2",
		OutputDir:       "target/quarkus-app",
		DefaultEnv: map[string]string{
			"MAVEN_OPTS": "-Xms512m -Xmx512m -XX:MaxDirectMemorySize=512m",
		},
		RequiredFiles: []string{"pom.xml"},
	}
}

// GetPythonDefaults returns default configuration for Python language builds.
func GetPythonDefaults() LanguageDefaults {
	return LanguageDefaults{
		Language:        "python",
		BuildType:       container.Python,
		BaseImage:       "python:3.11-slim-bookworm",
		LanguageVersion: "3.11",
		SourceMount:     "/src",
		CacheMount:      "/root/.cache/pip",
		OutputDir:       "/app",
		DefaultEnv: map[string]string{
			"_PIP_USE_IMPORTLIB_METADATA": "0",
			"UV_CACHE_DIR":                "/root/.cache/pip",
		},
		RequiredFiles: []string{"requirements.txt", "pyproject.toml"},
	}
}

// LanguageDefaultsRegistry provides a centralized way to access language defaults.
var LanguageDefaultsRegistry = map[container.BuildType]LanguageDefaults{
	container.GoLang: GetGoDefaults(),
	container.Maven:  GetMavenDefaults(),
	container.Python: GetPythonDefaults(),
}

// GetLanguageDefaults returns the default configuration for a specific build type.
func GetLanguageDefaults(buildType container.BuildType) (LanguageDefaults, bool) {
	defaults, exists := LanguageDefaultsRegistry[buildType]
	return defaults, exists
}
