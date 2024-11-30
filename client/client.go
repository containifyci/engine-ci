package main

import (
	"github.com/containifyci/engine-ci/client/pkg/build"
	"github.com/containifyci/engine-ci/protos2"
)

var opt protos2.BuildArgs
var opts protos2.BuildArgsGroup

func init() {
	// os.Chdir("..")
	opt = protos2.BuildArgs{}
	opt.Verbose = true
	opt.File = "containifyci.go"
	opt.Application = "containifyci-cli"
	opt.BuildType = protos2.BuildType_GoLang
	opt.Environment = protos2.EnvType_local

	opts = protos2.BuildArgsGroup{
		Args: []*protos2.BuildArgs{&opt},
	}
}

func main() {
	build.Build(&opt)
	build.BuildAsync(&opt)
	build.BuildGroups(&opts)
}
