package container

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"

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

// TODO: add target container platform
type Build struct {
	App      string `json:"app"`
	Env      EnvType
	File     string
	Folder   string
	Image    string `json:"image"`
	ImageTag string `json:"image_tag"`
	Custom   map[string][]string

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

	defaults bool
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

	if b.File != "/src/main.go" {
		b.File = "/src/" + b.File
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
