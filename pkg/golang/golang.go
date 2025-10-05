package golang

import (
	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/golang/alpine"
	"github.com/containifyci/engine-ci/pkg/golang/debian"
	"github.com/containifyci/engine-ci/pkg/golang/debiancgo"
)

func New() build.BuildStepv2 {
	return alpine.New()
}

func NewDebian() build.BuildStepv2 {
	return debian.New()
}

func NewProdDebian() build.BuildStepv2 {
	return debian.NewProd()
}

func NewCGO() build.BuildStepv2 {
	return debiancgo.New()
}

func NewProd() build.BuildStepv2 {
	return alpine.NewProd()
}

func NewLinter() build.BuildStepv2 {
	return alpine.NewLinter()
}
