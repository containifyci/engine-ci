package container

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"

	"github.com/containifyci/engine-ci/protos2"

	"github.com/containifyci/engine-ci/pkg/filesystem"
)

const (
	GoLang     BuildType = "GoLang"
	Maven      BuildType = "Maven"
	Python     BuildType = "Python"
	NodeJS     BuildType = "NodeJS"
	Typescript BuildType = "Typescript"
	Zig        BuildType = "Zig"
	Rust       BuildType = "Rust"
	AI         BuildType = "AI"

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
	case string(GoLang), string(Maven), string(Python), string(Generic), string(NodeJS), string(Typescript), string(Zig), string(Rust), string(AI):
		*e = BuildType(v)
		return nil
	default:
		return errors.New(`must be one of go, maven, python, zig, ai, generic`)
	}
}

// Type is only used in help text
func (e *BuildType) Type() string {
	return "BuildType"
}

type Custom map[string][]string

func (c Custom) String(key string) string {
	if v, ok := c[key]; ok {
		if len(v) > 0 {
			return strings.Join(v, "\n")
		}
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
	return uint(c.Int(key))
}

func (c Custom) Int(key string) int {
	if v, ok := c[key]; ok {
		i, err := strconv.Atoi(v[0])
		if err != nil {
			slog.Warn("Error converting string to int", "key", key, "value", v[0], "error", err)
			return 0
		}
		return i
	}
	return 0
}

type Leader interface {
	Leader(id string, fnc func() error)
}

type BuildLoop string

const (
	BuildStop     BuildLoop = "stop"
	BuildContinue BuildLoop = "continue"
)

// TODO: add target container platform
// Build struct optimized for memory alignment and cache performance
type Build struct {
	Leader             Leader
	Platform           types.Platform
	Custom             Custom
	Registries         map[string]*protos2.ContainerRegistry
	ContainerFiles     map[string]*protos2.ContainerFile
	Secret             map[string]string
	ContainifyRegistry string
	Runtime            utils.RuntimeType
	RuntimeClient      func() cri.ContainerManager `json:"-"`
	Image              string                      `json:"image"`
	ImageTag           string                      `json:"image_tag"`
	File               string
	Env                EnvType
	Folder             string
	Repository         string
	Organization       string
	App                string    `json:"app"`
	BuildType          BuildType `json:"build_type"`
	BuilderFunction    string
	Registry           string
	SourcePackages     []string
	SourceFiles        []string
	Verbose            bool
	defaults           bool
}

type BuildGroup struct {
	Builds []*Build
}

type BuildGroups []*BuildGroup

func (b *Build) VarName() string {
	return strings.ToLower(
		strings.ReplaceAll(strings.ReplaceAll(b.App, ".", ""), "-", ""),
	)
}

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
	// Use standard string builder for optimal performance (29% faster than pool)
	var builder strings.Builder
	builder.WriteString(b.Image)
	builder.WriteByte(':')
	builder.WriteString(b.ImageTag)

	result := builder.String()

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
	// Cache filesystem operations to avoid repeated disk access
	files, err := filesystem.NewFileCache("file_cache.yaml").
		FindFilesBySuffix(".", ".proto")
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
	return build
}

func NewPythonServiceBuild(appName string) Build {
	return NewServiceBuild(appName, Python)
}

func NewBuild(build *Build) *Build {
	if build.Runtime == "" {
		initRuntime(build)
	}
	if build.RuntimeClient == nil {
		build.RuntimeClient = func() cri.ContainerManager {
			runtime, err := cri.InitContainerRuntime()
			if err != nil {
				slog.Error("Failed to initialize container runtime", "error", err)
				os.Exit(1)
			}
			return runtime
		}
	}
	return build
}

func initRuntime(build *Build) *Build {
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

	return flags
}
