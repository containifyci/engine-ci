package build

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/containifyci/engine-ci/pkg/container"
)

// BuildCategory represents different phases of the build pipeline
type BuildCategory string

const (
	Auth      BuildCategory = "auth"      // Authentication & credentials
	PreBuild  BuildCategory = "prebuild"  // Setup, protobuf, dependencies
	Build     BuildCategory = "build"     // Language-specific compilation
	PostBuild BuildCategory = "postbuild" // Production artifacts, packaging
	Quality   BuildCategory = "quality"   // Linting, testing, security scanning
	Apply     BuildCategory = "apply"     // Infrastructure changes
	Publish   BuildCategory = "publish"   // Publishing, releases, notifications
)

type BuildContext struct {
	build    BuildStep
	category BuildCategory
	async    bool
}

type MatchesFunc func(build container.Build) bool

type BuildStep interface {
	Run() error
	Name() string
	Images() []string
	IsAsync() bool
	Matches(build container.Build) bool
}

type RunFunc func() error

type BuildSteps struct {
	Steps []*BuildContext
	init  bool
}

func ToBuildContexts(steps ...BuildStep) []*BuildContext {
	contexts := make([]*BuildContext, len(steps))
	for i, step := range steps {
		contexts[i] = &BuildContext{
			build:    step,
			async:    step.IsAsync(),
			category: Build, // Default category - should be specified explicitly
		}
	}
	return contexts
}

func NewBuildSteps(steps ...BuildStep) *BuildSteps {
	return &BuildSteps{
		init:  len(steps) > 0,
		Steps: ToBuildContexts(steps...),
	}
}

func (bs *BuildSteps) IsNotInit() bool { return !bs.init }
func (bs *BuildSteps) Init()           { bs.init = true }
func (bs *BuildSteps) Add(step BuildStep) {
	bs.Steps = append(bs.Steps, &BuildContext{build: step, category: Build, async: false}) // Default to Build category
}
func (bs *BuildSteps) AddAsync(step BuildStep) {
	bs.Steps = append(bs.Steps, &BuildContext{build: step, category: Build, async: true}) // Default to Build category
}

// Hook-based insertion methods
func (bs *BuildSteps) AddBefore(stepName string, step BuildStep) error {
	return bs.insertRelativeToStep(stepName, step, false, true)
}

func (bs *BuildSteps) AddAfter(stepName string, step BuildStep) error {
	return bs.insertRelativeToStep(stepName, step, false, false)
}

func (bs *BuildSteps) AddAsyncBefore(stepName string, step BuildStep) error {
	return bs.insertRelativeToStep(stepName, step, true, true)
}

func (bs *BuildSteps) AddAsyncAfter(stepName string, step BuildStep) error {
	return bs.insertRelativeToStep(stepName, step, true, false)
}

// Replace existing step by name
func (bs *BuildSteps) Replace(stepName string, step BuildStep) error {
	for i, bctx := range bs.Steps {
		if bctx.build.Name() == stepName {
			// Preserve the existing category and use the step's async setting
			bs.Steps[i] = &BuildContext{build: step, async: step.IsAsync(), category: bctx.category}
			return nil
		}
	}
	return fmt.Errorf("step '%s' not found", stepName)
}

// Helper method for relative insertion
func (bs *BuildSteps) insertRelativeToStep(stepName string, step BuildStep, async bool, before bool) error {
	for i, bctx := range bs.Steps {
		if bctx.build.Name() == stepName {
			// Use the same category as the reference step
			newStep := &BuildContext{build: step, async: async, category: bctx.category}
			insertPos := i
			if !before {
				insertPos = i + 1
			}

			// Insert at position
			bs.Steps = append(bs.Steps[:insertPos], append([]*BuildContext{newStep}, bs.Steps[insertPos:]...)...)
			return nil
		}
	}
	return fmt.Errorf("step '%s' not found", stepName)
}

// Category-based addition methods
func (bs *BuildSteps) AddToCategory(category BuildCategory, step BuildStep) error {
	return bs.insertAtCategoryEnd(category, step, false)
}

func (bs *BuildSteps) AddAsyncToCategory(category BuildCategory, step BuildStep) error {
	return bs.insertAtCategoryEnd(category, step, true)
}

// Helper method to find category boundaries and insert at the end of a category
func (bs *BuildSteps) insertAtCategoryEnd(category BuildCategory, step BuildStep, async bool) error {
	// Define category order for proper insertion
	categoryOrder := []BuildCategory{Auth, PreBuild, Build, PostBuild, Quality, Apply, Publish}

	// Find the target category index
	targetIndex := -1
	for i, cat := range categoryOrder {
		if cat == category {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return fmt.Errorf("unknown category: %s", category)
	}

	// Find insertion position (end of target category or before next category)
	insertPos := len(bs.Steps) // Default to end if no later categories found

	// Look for steps from categories that come after our target category
	for i := targetIndex + 1; i < len(categoryOrder); i++ {
		nextCategory := categoryOrder[i]
		if pos := bs.findFirstStepInCategory(nextCategory); pos != -1 {
			insertPos = pos
			break
		}
	}

	// Insert the new step
	newStep := &BuildContext{build: step, async: async, category: category}
	bs.Steps = append(bs.Steps[:insertPos], append([]*BuildContext{newStep}, bs.Steps[insertPos:]...)...)

	return nil
}

// Helper to find the first step in a given category
func (bs *BuildSteps) findFirstStepInCategory(category BuildCategory) int {
	for i, bctx := range bs.Steps {
		if bctx.category == category {
			return i
		}
	}
	return -1 // Not found
}

// GetStepsInCategory returns all steps in a specific category
func (bs *BuildSteps) GetStepsInCategory(category BuildCategory) []*BuildContext {
	var steps []*BuildContext
	for _, bctx := range bs.Steps {
		if bctx.category == category {
			steps = append(steps, bctx)
		}
	}
	return steps
}

// GetCategoryOrder returns the categories in the order they appear in the build steps
func (bs *BuildSteps) GetCategoryOrder() []BuildCategory {
	var seen = make(map[BuildCategory]bool)
	var order []BuildCategory

	for _, bctx := range bs.Steps {
		if !seen[bctx.category] {
			seen[bctx.category] = true
			order = append(order, bctx.category)
		}
	}
	return order
}

func (bs *BuildSteps) String() string {
	//TODO the for loop fails rarely looks like a race condition.
	/*
		panic: runtime error: index out of range [28] with length 28

		goroutine 38 [running]:
		github.com/containifyci/engine-ci/pkg/build.(*BuildSteps).String(0x106105760?)
		        /Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/build/build.go:216 +0x1b4
		github.com/containifyci/engine-ci/pkg/build.(*BuildSteps).PrintSteps(0x10519d8e0?)
		        /Users/frank.ittermann@goflink.com/private/github/engine-ci/pkg/build/build.go:222 +0x1c
	*/
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

func (bs *BuildSteps) Run(arg *container.Build, step ...string) error {
	return bs.runAllMatchingBuilds(arg, step)
}

func (bs *BuildSteps) runAllMatchingBuilds(arg *container.Build, step []string) error {
	var wg sync.WaitGroup

	for i, buildCtx := range bs.Steps {
		if !buildCtx.build.Matches(*arg) {
			// slog.Debug("Build step does not match config", "step", buildCtx.build.Name(), "index", i)
			continue
		}

		if step != nil && buildCtx.build.Name() != step[0] {
			continue
		}

		// slog.Debug("Build step matches config", "step", buildCtx.build.Name(), "index", i, "async", buildCtx.build.IsAsync())

		if buildCtx.build.IsAsync() {
			// Start async step immediately, don't wait
			wg.Add(1)
			go func(build BuildStep) {
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
