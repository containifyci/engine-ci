package goreleaser

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/svc"
)

const (
	IMAGE = "goreleaser/goreleaser"
)

type GoReleaserContainer struct {
	*container.Container
}

func New() *GoReleaserContainer {
	return &GoReleaserContainer{
		Container: container.New(container.BuildEnv),
	}
}

func (c *GoReleaserContainer) IsAsync() bool {
	return false
}

func (c *GoReleaserContainer) Name() string {
	return "gorelease"
}

func (c *GoReleaserContainer) Images() []string {
	return []string{IMAGE}
}

func CacheFolder() string {
	// Command to get the GOMODCACHE location
	cmd := exec.Command("go", "env", "GOMODCACHE")

	// Run the command and capture its output
	output, err := cmd.Output()
	if err != nil {
		slog.Error("Failed to execute command: %s", "error", err)
		os.Exit(1)
	}

	// Print the GOMODCACHE location
	gomodcache := strings.Trim(string(output), "\n")
	fmt.Printf("GOMODCACHE location: %s\n", gomodcache)
	return gomodcache
}

func (c *GoReleaserContainer) ApplyEnvs(envs []string) []string {
	tag := os.Getenv("GORELEASER_CURRENT_TAG")
	if tag != "" {
		envs = append(envs, "GORELEASER_CURRENT_TAG="+tag)
	}
	return envs
}

func (c *GoReleaserContainer) Release(env container.EnvType) error {
	if v, ok := container.GetBuild().Custom["goreleaser"]; ok {
		if v[0] == "false" {
			slog.Info("Skip goreleaser")
			return nil
		}
	}
	token := container.GetEnv("CONTAINIFYCI_GITHUB_TOKEN")
	if token == "" {
		slog.Warn("Skip goreleaser missing CONTAINIFYCI_GITHUB_TOKEN")
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = IMAGE
	//FIX: this should fix the permission issue with the mounted cache folder
	// opts.User = "root"

	opts.Env = append(opts.Env, []string{
		"GOMODCACHE=/go/pkg/",
		"GOCACHE=/go/pkg/build-cache",
		"GOLANGCI_LINT_CACHE=/go/pkg/lint-cache",
		"GITHUB_TOKEN=" + token,
	}...)

	opts.Env = c.ApplyEnvs(opts.Env)

	opts.WorkingDir = "/usr/src"

	dir, _ := filepath.Abs(".")
	cache := CacheFolder()
	if cache == "" {
		cache, _ = filepath.Abs(".tmp/go")
	}

	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/usr/src",
		},
		{
			Type:   "bind",
			Source: cache,
			Target: "/go/pkg",
		},
	}

	opts.Cmd = []string{"release", "--skip=validate", "--verbose", "--clean"}
	err := c.Container.Create(opts)
	if err != nil {
		return err
	}

	err = c.Container.Start()
	if err != nil {
		return err
	}

	return c.Container.Wait()
}

func (c *GoReleaserContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *GoReleaserContainer) Run() error {
	slog.Info("Run gorelease")
	if svc.GitInfo().Tag == "" {
		slog.Info("Skipping goreleaser for non-main branch")
		return nil
	}
	env := container.GetBuild().Env

	err := c.Pull()
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Release(env)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}
	return nil
}
