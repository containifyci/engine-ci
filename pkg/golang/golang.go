package golang

import (
	"log/slog"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/golang/alpine"
	"github.com/containifyci/engine-ci/pkg/golang/debian"
	"github.com/containifyci/engine-ci/pkg/golang/debiancgo"
)

// Legacy compatibility layer - these functions maintain the existing API
// The new unified builder functions are in factory.go and will replace these
// when migration is complete.

func NewLegacyAlpine(build container.Build) *alpine.GoContainer {
	slog.Debug("Using legacy alpine golang builder")
	return alpine.New(build)
}

func NewLegacyDebian(build container.Build) *debian.GoContainer {
	slog.Debug("Using legacy debian golang builder")
	return debian.New(build)
}

func NewLegacyProdDebian(build container.Build) build.Build {
	slog.Debug("Using legacy debian prod golang builder")
	return debian.NewProd(build)
}

func NewLegacyCGO(build container.Build) *debiancgo.GoContainer {
	slog.Debug("Using legacy debiancgo golang builder")
	return debiancgo.New(build)
}

func NewLegacyProd(build container.Build) build.Build {
	slog.Debug("Using legacy alpine prod golang builder")
	return alpine.NewProd(build)
}

func NewLegacyLinter(build container.Build) build.Build {
	slog.Debug("Using legacy alpine linter golang builder")
	return alpine.NewLinter(build)
}

// LegacyCacheFolder provides backward compatibility for cache folder access
func LegacyCacheFolder() string {
	return alpine.CacheFolder()
}

// LegacyLintImage provides backward compatibility for lint image access
func LegacyLintImage() string {
	return alpine.LintImage()
}
