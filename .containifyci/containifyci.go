//go:generate bash -c "if [ ! -f go.mod ]; then echo 'Initializing go.mod...'; go mod init .containifyci; else echo 'go.mod already exists. Skipping initialization.'; fi"
//go:generate go get github.com/containifyci/engine-ci/protos2
//go:generate go get github.com/containifyci/engine-ci/client
//go:generate go mod tidy

package main

import (
	"os"

	"github.com/containifyci/engine-ci/client/pkg/build"

	"github.com/containifyci/engine-ci/protos2"
)

func registryAuth() map[string]*protos2.ContainerRegistry {
	return map[string]*protos2.ContainerRegistry{
		"docker.io": {
			Username: "env:DOCKER_USER",
			Password: "env:DOCKER_TOKEN",
		},
		"ghcr.io": {
			Username: "USERNAME",
			Password: "env:GHCR_TOKEN",
		},
	}
}

func main() {
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}

	pr2 := build.NewGoServiceBuild("engine-ci-protos2")
	pr2.Folder = "protos2"
	pr2.Image = ""

	client := build.NewGoServiceBuild("engine-ci-client")
	client.File = "client.go"
	client.Folder = "client"
	client.Image = ""

	opts1 := build.NewGoServiceBuild("engine-ci")
	opts1.File = "main.go"
	opts1.Properties = map[string]*build.ListValue{
		"goreleaser": build.NewList("true"),
		"tags":       build.NewList("containers_image_openpgp", "exclude_graphdriver_btrfs"),
	}
	opts1.Image = ""
	opts1.Registries = registryAuth()

	custom := build.NewGoServiceBuild("engine-ci-custom")
	custom.File = "main.go"
	custom.Properties = map[string]*build.ListValue{
		"tags": build.NewList("containers_image_openpgp", "exclude_graphdriver_btrfs"),
	}
	custom.Image = ""
	custom.ContainerFiles = map[string]*protos2.ContainerFile{
		"build": DockerFile(),
	}
	custom.Registries = registryAuth()

	opts2 := build.NewGoServiceBuild("engine-ci-debian")
	opts2.File = "main.go"
	opts2.Properties = map[string]*build.ListValue{
		"tags": build.NewList("containers_image_openpgp", "exclude_graphdriver_btrfs"),
		"from": build.NewList("debian"),
	}
	opts2.Image = ""
	opts2.Registries = registryAuth()

	opts3 := build.NewGoServiceBuild("engine-ci-debiancgo")
	opts3.File = "main.go"
	opts3.Properties = map[string]*build.ListValue{
		"tags": build.NewList("containers_image_openpgp", "exclude_graphdriver_btrfs"),
		"from": build.NewList("debiancgo"),
	}
	opts3.Image = ""
	opts3.Registries = registryAuth()

	build.BuildGroups(
		&protos2.BuildArgsGroup{
			Args: []*protos2.BuildArgs{pr2, client},
		},
		&protos2.BuildArgsGroup{
			Args: []*protos2.BuildArgs{opts1, custom, opts2, opts3},
		},
	)
}

func DockerFile() *protos2.ContainerFile {
	return &protos2.ContainerFile{
		Name: "golang-1.27.0-alpine-custom",
		Content: `FROM golang:1.27.0-alpine

	RUN apk --no-cache add git openssh-client && \
	  rm -rf /var/cache/apk/*

	RUN go install github.com/wadey/gocovmerge@latest && \
	  go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest && \
	  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.1 && \
	  go clean -cache && \
	  go clean -modcache
	WORKDIR /app`,
	}
}
