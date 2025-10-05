package build

import (
	"fmt"
	"os"

	"github.com/containifyci/engine-ci/pkg/container"
)

type Stepper struct {
	RunFn     RunFuncv2
	MatchedFn func(build container.Build) bool
	ImagesFn  func(build container.Build) []string
	Name_     string
	Async_    bool
}

func (g Stepper) Run() error {
	fmt.Printf("deprected Run call without build %v\n", g)
	os.Exit(1)
	return nil
}
func (g Stepper) RunWithBuild(build container.Build) error { return g.RunFn(build) }
func (g Stepper) Name() string                             { return g.Name_ }

func (g Stepper) Images(build container.Build) []string {
	if g.ImagesFn != nil {
		return g.ImagesFn(build)
	}
	return []string{}
}
func (g Stepper) IsAsync() bool { return g.Async_ }

// Matches implements the Build interface provider matching logic
func (g Stepper) Matches(build container.Build) bool {
	if g.MatchedFn != nil {
		return g.MatchedFn(build)
	}
	return false
}

func StepperImages(images ...string) func(build container.Build) []string {
	return func(build container.Build) []string {
		return images
	}
}

var _ BuildStepv2 = (*Stepper)(nil)
