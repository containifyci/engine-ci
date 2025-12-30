package copier

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

const (
	COPIER_IMAGE = "webedia/copier:9.1-2025-09-30"
	COPIER_FILE  = "copier.yml"
)

type CopierContainer struct {
	*container.Container
	Build        *container.Build
	SourceFolder string
	TargetFolder string
	TemplatePath string
	TemplateData []string
}

func New() build.BuildStepv3 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			copierContainer := new(build)
			return copierContainer.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  CopierImages,
		Name_:     "copier",
		Alias_:    "copier",
		Async_:    false,
	}
}

func new(build container.Build) *CopierContainer {
	return &CopierContainer{
		Container:    container.New(build),
		Build:        &build,
		SourceFolder: build.Folder,
		TargetFolder: extractOutputPath(build),
		TemplateData: extractTemplateData(build),
		TemplatePath: extractTemplatePath(build),
	}
}

func CopierImages(build container.Build) []string {
	return []string{COPIER_IMAGE}
}

func Matches(build container.Build) bool {
	folder := build.Folder
	if folder == "" {
		folder = "./"
	}

	tmpPath := extractTemplatePath(build)
	if tmpPath == "" {
		tmpPath = folder
	}

	copierFile := filepath.Join(tmpPath, COPIER_FILE)
	if _, err := os.Stat(copierFile); os.IsNotExist(err) {
		slog.Debug("No copier.yml file found", "path", copierFile)
		return false
	}

	slog.Debug("Found copier.yml file", "path", copierFile)
	return true
}

func extractTemplateData(build container.Build) []string {
	tmpData := []string{}

	// Extract common template parameters from build.Custom
	if data, ok := build.Custom["data"]; ok && len(data) > 0 {
		tmpData = append(tmpData, data...)
	}
	return tmpData
}

func extractTemplatePath(build container.Build) string {
	if templatePath, ok := build.Custom["template_path"]; ok && len(templatePath) > 0 {
		return templatePath[0]
	}
	return ""
}

func extractOutputPath(build container.Build) string {
	if outputPath, ok := build.Custom["output_path"]; ok && len(outputPath) > 0 {
		return outputPath[0]
	}
	return ""
}

func (c *CopierContainer) buildCopierCommand() []string {
	args := []string{"copier", "copy"}

	// Add data parameters
	for _, value := range c.TemplateData {
		args = append(args, "--data", value)
	}

	// Add defaults if template path is provided
	if c.TemplatePath != "" {
		args = append(args, "--defaults", c.TemplatePath)
	}
	// Add overwrite flag
	args = append(args, "--overwrite")

	// Add target directory
	args = append(args, c.TargetFolder)

	return args
}

func (c *CopierContainer) Run() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull copier image", "error", err)
		return err
	}

	return c.Execute()
}

func (c *CopierContainer) Pull() error {
	return c.Container.Pull(COPIER_IMAGE)
}

func (c *CopierContainer) Execute() error {
	opts := types.ContainerConfig{}
	opts.Image = COPIER_IMAGE
	opts.Cmd = c.buildCopierCommand()
	opts.WorkingDir = "/src"

	// Get absolute path for the folder
	dir, err := filepath.Abs(".")
	if err != nil {
		slog.Error("Failed to get absolute path", "error", err)
		return err
	}

	// Mount the source folder as workspace
	opts.Volumes = []types.Volume{{
		Type:   "bind",
		Source: dir,
		Target: "/src",
	}}

	if c.Build.Verbose {
		slog.Info("Executing copier command", "cmd", strings.Join(opts.Cmd, " "), "volumes", opts.Volumes)
	}

	err = c.Create(opts)
	if err != nil {
		slog.Error("Failed to create copier container", "error", err)
		return err
	}

	err = c.Start()
	if err != nil {
		slog.Error("Failed to start copier container", "error", err)
		return err
	}

	err = c.Wait()
	if err != nil {
		slog.Error("Copier execution failed", "error", err)
		return err
	}

	slog.Info("Copier execution completed successfully")
	return nil
}
