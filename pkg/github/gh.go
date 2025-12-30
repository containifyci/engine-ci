package github

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/svc"
	"github.com/containifyci/engine-ci/pkg/trivy"
)

//go:embed Dockerfile*
var f embed.FS

const (
	IMAGE = "maniator/gh"
)

type GithubContainer struct {
	git *svc.Git
	*container.Container
}

// Matches implements the Build interface - GitHub runs for all builds
func Matches(build container.Build) bool {
	return true // GitHub integration runs for all builds
}

func New() build.BuildStepv3 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  Images,
		Name_:     "github",
		Alias_:    "github",
		Async_:    true,
	}
}

func new(build container.Build) *GithubContainer {
	return &GithubContainer{
		Container: container.New(build),
		git:       svc.GitInfo(),
	}
}

func Images(build container.Build) []string {
	return []string{Image(build)}
}

func (c *GithubContainer) CopyScript() error {
	// Create a temporary script in-memory
	script := fmt.Sprintf(`#!/bin/sh
set -xe
gh pr comment %s --repo %s --edit-last --body-file /src/trivy.md || gh pr comment %s --repo %s --body-file /src/trivy.md
`, c.git.PrNum, c.git.FullRepo(), c.git.PrNum, c.git.FullRepo())
	err := c.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func Image(build container.Build) string {
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile.go", "error", err)
		os.Exit(1)
	}
	tag := container.ComputeChecksum(dockerFile)
	return utils.ImageURI(build.ContainifyRegistry, "gh", tag)

	// return fmt.Sprintf("%s/%s/%s:%s", container.GetBuild().Registry, "containifyci", "gh", tag)
}

func (c *GithubContainer) BuildImage() error {
	image := Image(*c.GetBuild())

	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	return c.BuildIntermidiateContainer(image, dockerFile, platforms...)
}

func (c *GithubContainer) Comment() error {
	opts := types.ContainerConfig{}
	opts.Image = Image(*c.GetBuild())
	//FIX: this should fix the permission issue with the mounted cache folder
	// opts.User = "root"

	// cache := CacheFolder()

	dir, _ := filepath.Abs(".")
	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/src",
		},
	}

	// Open the JSON file
	file, err := os.ReadFile("trivy.json")
	if err != nil {
		slog.Error("Failed to open JSON file", "error", err)
		os.Exit(1)
	}

	comment, err := trivy.Parse(string(file))
	if err != nil {
		slog.Error("Failed to parse trivy JSON", "error", err)
		return err
	}

	err = os.WriteFile("trivy.md", []byte(comment), 0644)
	if err != nil {
		slog.Error("Failed to write JSON file", "error", err)
		os.Exit(1)
	}

	opts.Cmd = []string{"sh", "/tmp/script.sh"}
	// opts.Cmd = []string{"pr", "comment", "4", "--repo", "containifyci/engine-ci-example", "--edit-last", "--body-file", "/src/trivy.json"}
	opts.Env = []string{"GITHUB_TOKEN=" + container.GetEnv("CONTAINIFYCI_GITHUB_TOKEN")}
	err = c.Create(opts)
	if err != nil {
		return err
	}

	err = c.CopyScript()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Start()
	if err != nil {
		return err
	}

	return c.Wait()
}

func (c *GithubContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *GithubContainer) Run() error {
	if c.git.IsNotPR() {
		slog.Info("Skip github PR comment because PR number is not set")
		return nil
	}

	if !ifTrivyFileExists() {
		slog.Info("Skip github PR comment because trivy.json file does not exist")
		return nil
	}

	err := c.BuildImage()
	if err != nil {
		slog.Error("Failed to build go image: %s", "error", err)
		return err
	}

	err = c.Comment()
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}
	return nil
}

func ifTrivyFileExists() bool {
	_, err := os.Stat("trivy.json")
	if err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	}

	slog.Error("Failed to read trivy.json file", "error", err)
	os.Exit(1)
	return false
}
