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

type Build interface {
	Run() error
	Name() string
	Images() []string
	IsAsync() bool
	// Set(container.Build)
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
	var wg sync.WaitGroup
	for _, bctx := range bs.Steps {
		if len(step) > 0 && bctx.build.Name() != step[0] {
			continue
		}
		if bctx.async {
			slog.Info("Running async build step")
			wg.Add(1)
			go func(bctx *BuildContext) {
				defer wg.Done()
				err := bctx.build.Run()
				if err != nil {
					slog.Error("Failed to run build step: %s", "error", err)
					os.Exit(1)
				}
			}(bctx)
			continue
		}
		err := bctx.build.Run()
		if err != nil {
			slog.Error("Failed to run build step: %s", "error", err)
			return err
		}
	}
	wg.Wait()
	return nil
}

func (bs *BuildSteps) Images(step ...string) []string {
	images := []string{}
	for _, bctx := range bs.Steps {
		if len(step) > 0 && bctx.build.Name() != step[0] {
			continue
		}
		images = append(images, bctx.build.Images()...)
	}
	return images
}
