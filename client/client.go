package main

import (
	"github.com/containifyci/engine-ci/client/pkg/build"
	"github.com/containifyci/engine-ci/protos2"
)

var opts protos2.BuildArgs

func init() {
	opts = protos2.BuildArgs{}
	opts.Verbose = true
	opts.File = "containifyci.go"
	opts.Application = "containifyci-cli"
	opts.BuildType = protos2.BuildType_GoLang
	opts.Environment = protos2.EnvType_local
}

func main() {
	build.Serve(&opts)
}
