package build

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/containifyci/engine-ci/pkg/container"
)

type BuildContext struct {
	build Build
	async bool
}

type MatchesFunc func(build container.Build) bool

type Build interface {
	Run() error
	Name() string
	Images() []string
	IsAsync() bool
	Matches(build container.Build) bool
}

type RunFunc func() error

type BuildSteps struct {
	Steps []*BuildContext
	build container.Build
	init  bool
}

func ToBuildContexts(steps ...Build) []*BuildContext {
	contexts := make([]*BuildContext, len(steps))
	for i, step := range steps {
		contexts[i] = &BuildContext{
			build: step,
			async: step.IsAsync(),
		}
	}
	return contexts
}

func NewBuildSteps(steps ...Build) *BuildSteps {
	return &BuildSteps{
		init:  len(steps) > 0,
		Steps: ToBuildContexts(steps...),
	}
}

func NewBuildStepsWithArg(arg container.Build, steps ...Build) *BuildSteps {
	return &BuildSteps{
		build: arg,
		init:  len(steps) > 0,
		Steps: ToBuildContexts(steps...),
	}
}

func (bs *BuildSteps) IsNotInit() bool { return !bs.init }
func (bs *BuildSteps) Init()           { bs.init = true }
func (bs *BuildSteps) Add(step Build) {
	bs.Steps = append(bs.Steps, &BuildContext{step, false})
}
func (bs *BuildSteps) AddAsync(step Build) {
	bs.Steps = append(bs.Steps, &BuildContext{build: step, async: true})
}

func (bs *BuildSteps) String() string {
	if bs == nil || len(bs.Steps) == 0 {
		return ""
	}
	names := make([]string, len(bs.Steps))
	for i, bctx := range bs.Steps {
		if bctx.async {
			names[i] = fmt.Sprintf("%s(A)", bctx.build.Name())
			continue
		}
		names[i] = bctx.build.Name()
	}
	return strings.Join(names, ", ")
}

func (bs *BuildSteps) PrintSteps() {
	slog.Info("Build step", "steps", bs.String())
}

func (bs *BuildSteps) Run(step ...string) error {
	return bs.runAllMatchingBuilds(step)
}

func (bs *BuildSteps) runAllMatchingBuilds(step []string) error {
	var wg sync.WaitGroup

	for i, buildCtx := range bs.Steps {
		if !buildCtx.build.Matches(bs.build) {
			slog.Debug("Build step does not match config", "step", buildCtx.build.Name(), "index", i)
			continue
		}

		if step != nil && buildCtx.build.Name() != step[0] {
			continue
		}

		slog.Debug("Build step matches config", "step", buildCtx.build.Name(), "index", i, "async", buildCtx.build.IsAsync())

		if buildCtx.build.IsAsync() {
			// Start async step immediately, don't wait
			wg.Add(1)
			go func(build Build) {
				defer wg.Done()
				slog.Debug("Starting async step", "step", build.Name())

				if err := build.Run(); err != nil {
					slog.Error("Failed to run build step: %s", "error", err)
					os.Exit(1)
				}

				slog.Debug("Completed async step", "step", build.Name())
			}(buildCtx.build)
			continue
		}
		// Execute sync step and wait for completion
		slog.Debug("Executing sync step", "step", buildCtx.build.Name(), "index", i)
		if err := buildCtx.build.Run(); err != nil {
			slog.Error("Failed to run build step: %s", "error", err)
			return err
		}
		slog.Debug("Completed sync step", "step", buildCtx.build.Name(), "index", i)

	}

	// Wait for all async steps to complete
	slog.Debug("Waiting for all async steps to complete")
	wg.Wait()

	slog.Info("All build steps completed successfully")
	return nil
}

func (bs *BuildSteps) Images(groups container.BuildGroups) []string {
	images := []string{}
	for _, group := range groups {
		for _, build := range group.Builds {
			for _, bctx := range bs.Steps {
				if !bctx.build.Matches(*build) {
					continue
				}
				images = append(images, bctx.build.Images()...)
			}
		}
	}
	//deduplicate images
	images = uniqueStrings(images)
	return images
}

// uniqueStrings returns a slice containing only unique strings from the input.
func uniqueStrings(input []string) []string {
	seen := make(map[string]struct{})
	result := []string{}
	for _, str := range input {
		if _, ok := seen[str]; !ok {
			seen[str] = struct{}{}
			result = append(result, str)
		}
	}
	return result
}
