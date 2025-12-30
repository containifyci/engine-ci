package claude

import (
	"github.com/containifyci/engine-ci/pkg/ai/claude/alpine"
	"github.com/containifyci/engine-ci/pkg/build"
)

// New creates a new Claude AI build step
func New() build.BuildStepv3 {
	return alpine.New()
}
