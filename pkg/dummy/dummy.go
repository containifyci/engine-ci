package dummy

import (
	"log/slog"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
)

func Matches(build container.Build) bool {
	return true
}

func New() build.BuildStepv3 {
	return build.Stepper{
		BuildType_: container.Generic,
		RunFn: func(build container.Build) error {
			slog.Debug("Dummy build step executed", "build", build)
			return nil
		},
		MatchedFn: Matches,
		Name_:     "dummy",
		Alias_:    "dummy",
		Async_:    false,
	}
}
