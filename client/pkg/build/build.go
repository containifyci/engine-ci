package build

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/client/pkg/filesystem"
	"github.com/containifyci/engine-ci/protos2"
	"google.golang.org/protobuf/types/known/structpb"
)

type BuildArgs = protos2.BuildArgs
type ListValue = structpb.ListValue

func NewList(v ...string) *structpb.ListValue {
	x := &ListValue{Values: make([]*structpb.Value, len(v))}
	for i, v := range v {
		x.Values[i] = structpb.NewStringValue(v)
	}
	return x
}

func getEnv() protos2.EnvType {
	env := os.Getenv("ENV")
	if env == "local" {
		return protos2.EnvType_local
	}
	return protos2.EnvType_build
}

func NewServiceBuild(appName string, buildType protos2.BuildType) *BuildArgs {
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
	return &BuildArgs{
		Application:    appName,
		Environment:    getEnv(),
		Image:          appName,
		ImageTag:       commitSha,
		BuildType:      buildType,
		SourcePackages: packages,
		SourceFiles:    files,
	}
}

func NewGoServiceBuild(appName string) *BuildArgs {
	return NewServiceBuild(appName, protos2.BuildType_GoLang)
}

func NewGoLibraryBuild(appName string) *BuildArgs {
	lib := NewGoServiceBuild(appName)
	lib.Image = ""
	return lib
}

func NewMavenServiceBuild(appName string) *BuildArgs {
	build := NewServiceBuild(appName, protos2.BuildType_Maven)
	build.Folder = "target/quarkus-app"
	return build
}

func NewMavenLibraryBuild(appName string) *BuildArgs {
	lib := NewMavenServiceBuild(appName)
	lib.Image = ""
	return lib
}

func NewPythonServiceBuild(appName string) *BuildArgs {
	return NewServiceBuild(appName, protos2.BuildType_Python)
}

func NewPythonLibraryBuild(appName string) *BuildArgs {
	lib := NewPythonServiceBuild(appName)
	lib.Image = ""
	return lib
}
