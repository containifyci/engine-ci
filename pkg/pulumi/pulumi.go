package pulumi

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
)

const (
	IMAGE = "pulumi/pulumi-go"
)

//go:embed Dockerfile
var f embed.FS

type PulumiContainer struct {
	*container.Container
}

// Matches implements the Build interface - Pulumi only runs for golang builds
func Matches(build container.Build) bool {
	return build.BuildType == container.GoLang
}

func New() build.BuildStepv2 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  build.StepperImages(IMAGE),
		Name_:     "pulumi",
		Async_:    false,
	}
}

func new(build container.Build) *PulumiContainer {
	return &PulumiContainer{
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

// TODO: provide a shorter checksum
func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (c *PulumiContainer) PulumiImage() string {
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}
	tag := ComputeChecksum(dockerFile)
	return utils.ImageURI(c.GetBuild().ContainifyRegistry, "pulumi-go", tag)
	// return fmt.Sprintf("%s/%s/%s:%s", container.GetBuild().Registry, "containifyci", "maven-3-eclipse-temurin-17-alpine", tag)
}

func (c *PulumiContainer) BuildPulumiImage() error {
	image := c.PulumiImage()
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	err = c.BuildIntermidiateContainer(image, dockerFile, platforms...)
	if err != nil {
		slog.Error("Failed to build maven image", "error", err)
		os.Exit(1)
	}
	return nil
}

func (c *PulumiContainer) CopyScript() error {
	var stack, command string
	if v, ok := c.GetBuild().Custom["stack"]; ok {
		stack = v[0]
	}

	if v, ok := c.GetBuild().Custom["cmd"]; ok {
		command = v[0]
	}

	if len(stack) <= 0 {
		stack = c.GetBuild().App
	}

	if len(command) <= 0 {
		command = "preview"
	}

	// Create a temporary script in-memory
	script := `#!/bin/sh
set -xe
pulumi login --local
pulumi stack init %s || echo ignore if stack %s already exists
pulumi stack select -c %s
pulumi %s --non-interactive
`
	err := c.CopyContentTo(fmt.Sprintf(script, stack, stack, stack, command), "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func (c *PulumiContainer) ApplyEnvs(envs []string) []string {
	tag := os.Getenv("PULUMI_CONFIG_PASSPHRASE")
	if tag != "" {
		envs = append(envs, "PULUMI_CONFIG_PASSPHRASE="+tag)
	}
	return envs
}

func (c *PulumiContainer) Release(env container.EnvType) error {
	if v, ok := c.GetBuild().Custom["pulumi"]; ok {
		if v[0] == "false" {
			slog.Info("Skip pulumi")
			return nil
		}
	}

	opts := types.ContainerConfig{}
	opts.Image = c.PulumiImage()
	opts.User = "root"

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

	home, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("home directory couldn't be found. (Skip mounting pulumi stacks)", "error", err)
	} else {
		opts.Volumes = append(opts.Volumes, types.Volume{
			Type:   "bind",
			Source: home + "/.pulumi/stacks",
			Target: "/root/.pulumi/stacks",
		})
	}

	opts.Cmd = []string{"sh", "/tmp/script.sh"}

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

func (c *PulumiContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *PulumiContainer) Run() error {
	if v, ok := c.GetBuild().Custom["pulumi"]; ok {
		if v[0] != "true" {
			slog.Debug("Skip pulumi")
			return nil
		}
	} else {
		slog.Debug("Skip pulumi")
		return nil
	}
	slog.Info("Run pulumi")
	env := c.GetBuild().Env

	err := c.Pull()
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.BuildPulumiImage()
	if err != nil {
		slog.Error("Failed to build go image: %s", "error", err)
		return err
	}

	err = c.Release(env)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}
	return nil
}
