package dummy

import (
	"log/slog"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
)

func Matches(build container.Build) bool {
	return true
}

func New() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			slog.Info("Dummy build step executed", "build", build)
			return nil
		},
		MatchedFn: Matches,
		Name_:     "dummy",
		Async_:    false,
	}
}
