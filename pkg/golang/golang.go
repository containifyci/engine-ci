package golang

import (
	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/golang/alpine"
	"github.com/containifyci/engine-ci/pkg/golang/debian"
	"github.com/containifyci/engine-ci/pkg/golang/debiancgo"
)

func New() build.BuildStepv3 {
	return alpine.New()
}

func NewDebian() build.BuildStepv3 {
	return debian.New()
}

func NewProdDebian() build.BuildStepv3 {
	return debian.NewProd()
}

func NewCGO() build.BuildStepv3 {
	return debiancgo.New()
}

func NewProd() build.BuildStepv3 {
	return alpine.NewProd()
}

func NewLinter() build.BuildStepv3 {
	return alpine.NewLinter()
}
