package build

import (
	"fmt"
	"os"

	"github.com/containifyci/engine-ci/pkg/container"
)

type Stepper struct {
	RunFn              RunFunc
	MatchedFn          func(build container.Build) bool
	ImagesFn           func(build container.Build) []string
	IntermediateImagesFn func(build container.Build) []IntermediateImage
	BuildType_         container.BuildType
	Name_              string
	Alias_             string
	Async_             bool
}

func (g Stepper) Run() error {
	fmt.Printf("deprected Run call without build %v\n", g)
	os.Exit(1)
	return nil
}
func (g Stepper) RunWithBuild(build container.Build) (string, error) { return g.RunFn(build) }

func (g Stepper) Name() string  { return g.Name_ }
func (g Stepper) Alias() string { return g.Alias_ }

func (g Stepper) BuildType() *container.BuildType {
	if g.BuildType_ != "" {
		return &g.BuildType_
	}
	return nil
}

func (g Stepper) Images(build container.Build) []string {
	if g.ImagesFn != nil {
		return g.ImagesFn(build)
	}
	return []string{}
}

// IntermediateImages returns all containifyci intermediate images this step can
// build, each paired with its Dockerfile path. Empty by default; steps that
// produce intermediate images set IntermediateImagesFn.
func (g Stepper) IntermediateImages(build container.Build) []IntermediateImage {
	if g.IntermediateImagesFn != nil {
		return g.IntermediateImagesFn(build)
	}
	return nil
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

// SingleIntermediateImage returns an IntermediateImagesFn that produces a
// single containifyci intermediate image. It is a convenience helper for the
// common case where a build step produces exactly one intermediate image from
// one Dockerfile — most steps use this. Steps with multiple variants (e.g.
// golang/alpine has default + chromium) define their own IntermediateImagesFn.
func SingleIntermediateImage(uriFn func(container.Build) string, dockerfile string) func(container.Build) []IntermediateImage {
	return func(b container.Build) []IntermediateImage {
		return []IntermediateImage{{URI: uriFn(b), Dockerfile: dockerfile}}
	}
}

var _ BuildStep = (*Stepper)(nil)
