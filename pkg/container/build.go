package container

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
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
var _build *Build

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

func (c Custom) Bool(key string) bool {
	if v, ok := c[key]; ok {
		return v[0] == "true"
	}
	return false
}

func (c Custom) UInt(key string) uint {
	if v, ok := c[key]; ok {
		i, err := strconv.Atoi(v[0])
		if err != nil {
			slog.Error("Error converting string to int", "error", err)
			os.Exit(1)
		}
		return uint(i)
	}
	return 0
}

// TODO: add target container platform
type Build struct {
	App      string `json:"app"`
	Env      EnvType
	File     string
	Folder   string
	Image    string `json:"image"`
	ImageTag string `json:"image_tag"`
	Custom   Custom

	BuildType BuildType `json:"build_type"`
	// docker or podman
	Runtime utils.RuntimeType

	Organization       string
	Platform           types.Platform
	Registry           string
	ContainifyRegistry string
	Repository         string
	SourcePackages     []string
	SourceFiles        []string
	Verbose            bool
	Registries         map[string]*protos2.ContainerRegistry

	defaults bool
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

func (b *Build) ImageURI() string {
	return b.Image + ":" + b.ImageTag
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
	files, err := filesystem.NewFileCache("file_cache.yaml").
		FindFilesBySuffix(".", ".proto")

	packages := []string{}
	for _, file := range files {
		pkg := filepath.Dir(file)
		packages = append(packages, pkg)
	}
	if err != nil {
		slog.Error("Error finding proto files", "error", err)
		os.Exit(1)
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
	build.Folder = "target/quarkus-app"
	return build
}

func NewPythonServiceBuild(appName string) Build {
	return NewServiceBuild(appName, Python)
}

func NewBuild(build *Build) *Build {
	_build = build
	if _build.Runtime == "" {
		InitRuntime()
	}
	return _build
}

func InitRuntime() *Build {
	_build.Runtime = cri.DetectContainerRuntime()
	return _build
}

func (b *Build) Defaults() *Build {
	if b.defaults {
		return b
	}
	if b.Env == "" {
		b.Env = BuildEnv
	}

	if b.File != "/src/main.go" && b.File != "" {
		b.File = "/src/" + b.File
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
func (b *Build) AsFlags() []string {
	flags := []string{
		"--app", b.App,
		"--env", string(b.Env),
		"--image", b.Image,
		"--tag", b.ImageTag,
		"--repo", b.Registry,
		"--file", b.File,
		"--folder", b.Folder,
		"--type", string(b.BuildType),
	}
	if b.Verbose {
		flags = append(flags, "--verbose")
	}

	for _, pkg := range b.SourcePackages {
		flags = append(flags, "--protobuf-packages", pkg)
	}

	for _, file := range b.SourceFiles {
		flags = append(flags, "--protobuf-files", file)
	}

	return flags
}
