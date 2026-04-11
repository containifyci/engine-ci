package packer

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
)

const (
	IMAGE = "hashicorp/packer:full"
)

//go:embed Dockerfile
var f embed.FS

type packerContainer struct {
	*container.Container
	Folder string
}

// Matches implements the Build interface - packer only runs for golang builds
func Matches(build container.Build) bool {
	v, ok := build.Custom["packer"]
	if !ok || v[0] != "true" {
		return false
	}
	return build.BuildType == container.Generic

}

func New() build.BuildStep {
	return build.Stepper{
		RunFn: func(build container.Build) (string, error) {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  build.StepperImages(IMAGE),
		Name_:     "packer",
		Alias_:    "packer",
		Async_:    false,
	}
}

func new(build container.Build) *packerContainer {
	return &packerContainer{
		Container: container.New(build),
		Folder:    build.Folder,
	}
}

// TODO: provide a shorter checksum
func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (c *packerContainer) packerImage() string {
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}
	tag := ComputeChecksum(dockerFile)
	return utils.ImageURI(c.GetBuild().ContainifyRegistry, "packer", tag)
	// return fmt.Sprintf("%s/%s/%s:%s", container.GetBuild().Registry, "containifyci", "maven-3-eclipse-temurin-17-alpine", tag)
}

func (c *packerContainer) BuildpackerImage() error {
	image := c.packerImage()
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

func (c *packerContainer) CopyScript() error {
	// Create a temporary script in-memory
	script := `#!/bin/sh
set -xe
packer init .
packer validate .
packer build .
`
	err := c.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func (c *packerContainer) ApplyEnvs(envs []string) []string {
	tag := os.Getenv("packer_CONFIG_PASSPHRASE")
	if tag != "" {
		envs = append(envs, "packer_CONFIG_PASSPHRASE="+tag)
	}
	return envs
}

func (c *packerContainer) Release(env container.EnvType) error {
	if v, ok := c.GetBuild().Custom["packer"]; ok {
		if v[0] == "false" {
			slog.Info("Skip packer")
			return nil
		}
	}

	token := container.GetEnv("HCLOUD_TOKEN")
	if token == "" {
		slog.Warn("HCLOUD_TOKEN is not set skip packer")
		return nil
	}

	opts := types.ContainerConfig{}
	opts.Image = c.packerImage()
	opts.User = "root"

	opts.Env = c.ApplyEnvs(opts.Env)
	//TODO move the secret handling to kv instead of opts.Env this will hide the secret from docker envs
	opts.Secrets = map[string]string{
		"HCLOUD_TOKEN": token,
	}

	opts.WorkingDir = "/usr/src/" + c.Folder

	dir, _ := filepath.Abs(".")

	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/usr/src",
		},
	}

	opts.Entrypoint = []string{"sh", "/tmp/script.sh"}

	err := c.Create(opts)
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

func (c *packerContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *packerContainer) Run() (string, error) {
	if v, ok := c.GetBuild().Custom["packer"]; ok {
		if v[0] != "true" {
			slog.Debug("Skip packer")
			return "", nil
		}
	} else {
		slog.Debug("Skip packer")
		return "", nil
	}
	slog.Info("Run packer")
	env := c.GetBuild().Env

	err := c.Pull()
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.BuildpackerImage()
	if err != nil {
		slog.Error("Failed to build go image: %s", "error", err)
		return c.ID, err
	}

	err = c.Release(env)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}
	return c.ID, nil
}
