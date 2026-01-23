package goreleaser

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	utils "github.com/containifyci/engine-ci/pkg/utils"

	"github.com/containifyci/engine-ci/pkg/svc"
)

//go:embed .goreleaser.yaml
var defaultGoreleaserConfig []byte

const (
	IMAGE = "goreleaser/goreleaser:nightly"
)

type GoReleaserContainer struct {
	*container.Container
}

// Matches implements the Build interface - GoReleaser only runs for golang builds with goreleaser=true
func Matches(build container.Build) bool {
	if build.BuildType != container.GoLang {
		return false
	}
	// Check if goreleaser is enabled
	if goreleaser, ok := build.Custom["goreleaser"]; ok && len(goreleaser) > 0 {
		return goreleaser[0] == "true"
	}
	return false
}

func New() build.BuildStepv3 {
	return build.Stepper{
		BuildType_: container.GoLang,
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  build.StepperImages(IMAGE),
		Name_:     "gorelease",
		Alias_:    "release",
		Async_:    false,
	}
}

func new(build container.Build) *GoReleaserContainer {
	return &GoReleaserContainer{
		Container: container.New(build),
	}
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
	slog.Debug("GOMODCACHE location", "path", gomodcache)
	return gomodcache
}

func (c *GoReleaserContainer) ApplyEnvs(envs []string) []string {
	tag := os.Getenv("GORELEASER_CURRENT_TAG")
	if tag != "" {
		envs = append(envs, "GORELEASER_CURRENT_TAG="+tag)
	}
	return envs
}

// hasProjectConfig checks if the project has its own goreleaser config file
func hasProjectConfig(dir string) bool {
	configNames := []string{".goreleaser.yaml", ".goreleaser.yml", ".goreleaser.json"}
	for _, name := range configNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// writeDefaultConfig writes the embedded default config to a temp file
func writeDefaultConfig() (string, error) {
	configPath := "/tmp/.goreleaser-default.yaml"
	if err := os.WriteFile(configPath, defaultGoreleaserConfig, 0644); err != nil {
		return "", fmt.Errorf("failed to write default config: %w", err)
	}
	return configPath, nil
}

func (c *GoReleaserContainer) Release(env container.EnvType) error {
	token := container.GetEnv("CONTAINIFYCI_GITHUB_TOKEN")
	if token == "" {
		slog.Warn("Skip goreleaser missing CONTAINIFYCI_GITHUB_TOKEN")
		os.Exit(1)
	}

	opts := types.ContainerConfig{}
	opts.Image = IMAGE
	//FIX: this should fix the permission issue with the mounted cache folder
	// opts.User = "root"

	envKeys := c.GetBuild().Custom.Strings("goreleaser_envs")
	envs := utils.GetAllEnvs(envKeys, c.Env.String())

	for k, v := range envs {
		opts.Env = append(opts.Env, []string{k + "=" + v}...)
	}

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

	// Use embedded default config if project doesn't have one
	if !hasProjectConfig(dir) {
		slog.Info("No goreleaser config found, using embedded default")
		hostConfigPath, err := writeDefaultConfig()
		if err != nil {
			return fmt.Errorf("failed to write default goreleaser config: %w", err)
		}
		defer os.Remove(hostConfigPath)

		opts.Volumes = append(opts.Volumes, types.Volume{
			Type:   "bind",
			Source: hostConfigPath,
			Target: "/tmp/.goreleaser-default.yaml",
		})
		opts.Cmd = append(opts.Cmd, "--config=/tmp/.goreleaser-default.yaml")
	}

	err := c.Create(opts)
	if err != nil {
		return err
	}

	err = c.Start()
	if err != nil {
		return err
	}

	return c.Wait()
}

func (c *GoReleaserContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *GoReleaserContainer) Run() error {
	slog.Info("Run gorelease")
	if svc.GitInfo().IsNotTag() {
		slog.Info("Skipping goreleaser for non tag branch")
		return nil
	}
	if !c.GetBuild().Custom.Bool("goreleaser", false) {
		slog.Info("Skip goreleaser because its not explicit enabled", "build", c.GetBuild())
		return nil
	}
	env := c.GetBuild().Env

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
