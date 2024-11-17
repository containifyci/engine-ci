package golang

import (
	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/golang/alpine"
	"github.com/containifyci/engine-ci/pkg/golang/debian"
	"github.com/containifyci/engine-ci/pkg/golang/debiancgo"
)

func New() *alpine.GoContainer {
	return alpine.New()
}

func NewDebian() *debian.GoContainer {
	return debian.New()
}

func NewProdDebian() build.Build {
	return debian.NewProd()
}

func NewCGO() *debiancgo.GoContainer {
	return debiancgo.New()
}

func NewProd() build.Build {
	return alpine.NewProd()
}

func NewLinter() build.Build {
	return alpine.NewLinter()
}
