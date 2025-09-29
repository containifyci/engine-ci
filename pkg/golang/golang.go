package golang

import (
	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/golang/alpine"
	"github.com/containifyci/engine-ci/pkg/golang/debian"
	"github.com/containifyci/engine-ci/pkg/golang/debiancgo"
)

func New(build container.Build) *alpine.GoContainer {
	return alpine.New(build)
}

func NewDebian(build container.Build) *debian.GoContainer {
	return debian.New(build)
}

func NewProdDebian(build container.Build) build.BuildStep {
	return debian.NewProd(build)
}

func NewCGO(build container.Build) *debiancgo.GoContainer {
	return debiancgo.New(build)
}

func NewProd(build container.Build) build.BuildStep {
	return alpine.NewProd(build)
}

func NewLinter(build container.Build) build.BuildStep {
	return alpine.NewLinter(build)
}
