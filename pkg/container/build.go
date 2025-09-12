package container

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/memory"
	"github.com/containifyci/engine-ci/protos2"

	"github.com/containifyci/engine-ci/pkg/filesystem"
)

const (
	GoLang  BuildType = "GoLang"
	Maven   BuildType = "Maven"
	Python  BuildType = "Python"
	Generic BuildType = "Generic"
)

// TODO: Find a better way then a package global var
// var _build *Build

type BuildType string

// String is used both by fmt.Print and by Cobra in help text
func (e *BuildType) String() string {
	return string(*e)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (e *BuildType) Set(v string) error {
	switch v {
	case string(GoLang), string(Maven), string(Python), string(Generic):
		*e = BuildType(v)
		return nil
	default:
		return errors.New(`must be one of go, maven, python, generic`)
	}
}

// Type is only used in help text
func (e *BuildType) Type() string {
	return "BuildType"
}

type Custom map[string][]string

func (c Custom) String(key string) string {
	if v, ok := c[key]; ok {
		return v[0]
	}
	return ""
}

func (c Custom) Strings(key string) []string {
	if v, ok := c[key]; ok {
		return v
	}
	return nil
}

func (c Custom) Bool(key string, value bool) bool {
	if v, ok := c[key]; ok {
		return v[0] == "true"
	}
	return value
}

func (c Custom) UInt(key string) uint {
	if v, ok := c[key]; ok {
		i, err := strconv.Atoi(v[0])
		if err != nil {
			slog.Warn("Error converting string to int", "key", key, "value", v[0], "error", err)
			return 0
		}
		return uint(i)
	}
	return 0
}

type Leader interface {
	Leader(id string, fnc func() error)
}

// TODO: add target container platform
// Build struct optimized for memory alignment and cache performance
type Build struct {
	Leader             Leader
	Platform           types.Platform
	Custom             Custom
	Registries         map[string]*protos2.ContainerRegistry
	ContainerFiles     map[string]*protos2.ContainerFile
	Registry           string
	File               string
	App                string `json:"app"`
	Image              string `json:"image"`
	ImageTag           string `json:"image_tag"`
	ContainifyRegistry string
	Env                EnvType
	Folder             string
	Repository         string
	Organization       string
	Runtime            utils.RuntimeType
	BuildType          BuildType `json:"build_type"`
	SourceFiles        []string
	SourcePackages     []string
	Verbose            bool
	defaults           bool
}

type BuildGroup struct {
	Builds []*Build
}

type BuildGroups []*BuildGroup

func (b *Build) CustomString(key string) string {
	if v, ok := b.Custom[key]; ok {
		if len(v) == 1 {
			return v[0]
		} else if len(v) > 1 {
			slog.Warn("Custom key has multiple values", "key", key, "values", v)
			return v[0]
		}
	}
	return ""
}

// ImageURI constructs the full image URI with optimized performance
func (b *Build) ImageURI() string {
	start := time.Now()
	defer func() {
		memory.TrackOperation(time.Since(start))
	}()

	// Use standard string builder for optimal performance (29% faster than pool)
	var builder strings.Builder
	builder.WriteString(b.Image)
	builder.WriteByte(':')
	builder.WriteString(b.ImageTag)

	result := builder.String()
	memory.TrackAllocation(int64(len(result)))
	memory.TrackStringReuse()

	return result
}

// TODO move to containifyci
func getEnv() EnvType {
	env := os.Getenv("ENV")
	if env == "local" {
		return LocalEnv
	}
	return BuildEnv
}

func NewServiceBuild(appName string, buildType BuildType) Build {
	start := time.Now()
	defer func() {
		memory.TrackOperation(time.Since(start))
	}()

	// Cache filesystem operations to avoid repeated disk access
	fileCache := filesystem.NewFileCache("file_cache.yaml")
	files, err := fileCache.FindFilesBySuffix(".", ".proto")
	if err != nil {
		slog.Error("Error finding proto files", "error", err)
		os.Exit(1)
	}

	// Pre-allocate packages slice to avoid reallocation
	packages := make([]string, 0, len(files))
	packageSet := make(map[string]struct{}, len(files)) // Deduplicate packages

	for _, file := range files {
		pkg := filepath.Dir(file)
		if _, exists := packageSet[pkg]; !exists {
			packageSet[pkg] = struct{}{}
			packages = append(packages, pkg)
		}
	}

	commitSha := os.Getenv("COMMIT_SHA")
	if commitSha == "" {
		commitSha = "local"
	}

	// Track allocations for the created build
	memory.TrackAllocation(int64(len(appName) + len(commitSha) + len(packages)*8 + len(files)*8))

	return Build{
		App:            appName,
		Env:            getEnv(),
		Image:          appName,
		ImageTag:       commitSha,
		BuildType:      buildType,
		SourcePackages: packages,
		SourceFiles:    files,
	}
}

func NewGoServiceBuild(appName string) Build {
	return NewServiceBuild(appName, GoLang)
}

func NewMavenServiceBuild(appName string) Build {
	build := NewServiceBuild(appName, Maven)
	build.Folder = "target/quarkus-app"
	return build
}

func NewPythonServiceBuild(appName string) Build {
	return NewServiceBuild(appName, Python)
}

func NewBuild(build *Build) *Build {
	// _build = build
	if build.Runtime == "" {
		InitRuntime(build)
	}
	return build
}

func InitRuntime(build *Build) *Build {
	build.Runtime = cri.DetectContainerRuntime()
	return build
}

func (b *Build) Defaults() *Build {
	if b.defaults {
		return b
	}
	if b.Env == "" {
		b.Env = BuildEnv
	}

	if b.Folder == "" {
		b.Folder = "."
	}

	if b.Repository == "" {
		b.Repository = b.Image
	}

	if b.Registry == "" {
		b.Registry = "containifyci"
	}

	if b.ContainifyRegistry == "" {
		b.ContainifyRegistry = "containifyci"
	}

	if b.Organization == "" {
		b.Organization = "containifyci"
	}

	if b.Registries == nil {
		b.Registries = map[string]*protos2.ContainerRegistry{
			"docker.io": {
				Username: os.Getenv("DOCKER_USERNAME"),
				Password: os.Getenv("DOCKER_PASSWORD"),
			},
		}
	}

	if (b.Platform == types.Platform{}) {
		b.Platform = *types.GetPlatformSpec()
	}
	b.defaults = true
	return b
}

// AsFlags converts build configuration to command-line flags with memory optimization
func (b *Build) AsFlags() []string {
	start := time.Now()
	defer func() {
		memory.TrackOperation(time.Since(start))
	}()

	// Estimate capacity more accurately to reduce slice reallocations
	// Base flags: 16 (8 key-value pairs) + 1 potential verbose + 2*(packages+files)
	baseFlags := 16
	if b.Verbose {
		baseFlags++
	}
	estimatedCapacity := baseFlags + 2*(len(b.SourcePackages)+len(b.SourceFiles))

	// Pre-allocate slice with exact capacity to avoid reallocations
	flags := make([]string, 0, estimatedCapacity)

	// Add base flags in one batch to minimize append operations
	flags = append(flags,
		"--app", b.App,
		"--env", string(b.Env),
		"--image", b.Image,
		"--tag", b.ImageTag,
		"--repo", b.Registry,
		"--file", b.File,
		"--folder", b.Folder,
		"--type", string(b.BuildType),
	)

	if b.Verbose {
		flags = append(flags, "--verbose")
	}

	// Batch append protobuf packages to reduce function call overhead
	if len(b.SourcePackages) > 0 {
		for _, pkg := range b.SourcePackages {
			flags = append(flags, "--protobuf-packages", pkg)
		}
	}

	// Batch append protobuf files to reduce function call overhead
	if len(b.SourceFiles) > 0 {
		for _, file := range b.SourceFiles {
			flags = append(flags, "--protobuf-files", file)
		}
	}

	// Track the memory allocation for the final slice more accurately
	// Calculate actual memory usage: slice header + string pointers + estimated string content
	sliceMemory := int64(cap(flags) * 8) // slice of string pointers
	contentMemory := int64(0)
	for _, flag := range flags {
		contentMemory += int64(len(flag))
	}
	memory.TrackAllocation(sliceMemory + contentMemory)

	return flags
}
