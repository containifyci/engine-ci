package golang

import (
	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/golang/alpine"
	"github.com/containifyci/engine-ci/pkg/golang/debian"
	"github.com/containifyci/engine-ci/pkg/golang/debiancgo"
)

func New() build.BuildStep {
	return alpine.New()
}

func NewDebian() build.BuildStep {
	return debian.New()
}

func NewProdDebian() build.BuildStep {
	return debian.NewProd()
}

func NewCGO() build.BuildStep {
	return debiancgo.New()
}

func NewProd() build.BuildStep {
	return alpine.NewProd()
}

func NewLinter() build.BuildStep {
	return alpine.NewLinter()
}
