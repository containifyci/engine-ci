//go:generate bash -c "if [ ! -f go.mod ]; then echo 'Initializing go.mod...'; go mod init .containifyci; else echo 'go.mod already exists. Skipping initialization.'; fi"
//go:generate go get github.com/containifyci/engine-ci/protos2
//go:generate go get github.com/containifyci/engine-ci/client
//go:generate go mod tidy

package main

import (
	"os"

	"github.com/containifyci/engine-ci/client/pkg/build"
)

func main() {
	os.Chdir("..")
	opts1 := build.NewGoServiceBuild("engine-ci")
	opts1.Verbose = false
	opts1.File = "main.go"
	opts1.Properties = map[string]*build.ListValue{
		"tags": build.NewList("containers_image_openpgp"),
	}

	opts2 := build.NewGoServiceBuild("engine-ci")
	opts2.Verbose = false
	opts2.File = "main.go"
	opts2.Properties = map[string]*build.ListValue{
		"tags": build.NewList("containers_image_openpgp"),
		"from": build.NewList("debian"),
		"goreleaser": build.NewList("false"),
	}
	build.Serve(opts1, opts2)
}
