package trivy

import (
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/network"
)

const (
	IMAGE = "public.ecr.aws/aquasecurity/trivy:canary"
)

type TrivyContainer struct {
	*container.Container
}

func New() *TrivyContainer {
	return &TrivyContainer{
		Container: container.New(container.BuildEnv),
	}
}

func (c *TrivyContainer) IsAsync() bool {
	return false
}

func (c *TrivyContainer) Name() string {
	return "trivy"
}

func (c *TrivyContainer) Images() []string {
	return []string{IMAGE}
}

func CacheFolder() string {
	dir := os.Getenv("CONTAINIFYCI_CACHE")
	if dir == "" {
		usr, _ := user.Current()
		dir = usr.HomeDir
	}
	folder, err := filepath.Abs(filepath.Join(dir, "/.trivy/cache"))
	if err != nil {
		slog.Error("Failed to get cache folder: %s", "error", err)
		os.Exit(1)
	}

	err = os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		slog.Error("Failed to create cache folder: %s", "error", err)
		os.Exit(1)
	}
	slog.Info("Cache folder", "folder", folder)

	return folder
}

func (c *TrivyContainer) CopyScript() error {
	// TODO add the --podman-host /var/run/podman.sock  only when runtime is podman
	image := container.GetBuild().ImageURI()
	if container.GetBuild().Runtime == utils.Podman {
		info, err := c.Container.InspectImage(image)
		if err != nil {
			slog.Error("Failed to inspect image", "error", err)
			os.Exit(1)
		}
		image = info.ID
	}

	script := fmt.Sprintf(`#!/bin/sh
set -xe
trivy image --podman-host /var/run/podman.sock --severity CRITICAL,HIGH --ignore-unfixed -d --scanners vuln --format json --output /usr/src/trivy.json %s
#chmod 0755 -R /root/.cache/trivy
`, image)
	err := c.Container.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func (c *TrivyContainer) Scan() error {
	// options := []string{}

	opts := types.ContainerConfig{}
	opts.Image = IMAGE
	//FIX: this should fix the permission issue with the mounted cache folder
	// opts.User = "root"

	cache := CacheFolder()

	dir, _ := filepath.Abs(".")
	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/usr/src",
		},
		{
			Type:   "bind",
			Source: cache,
			Target: "/root/.cache/trivy",
		},
	}
	opts.Env = []string{
		"TRIVY_CACHE_DIR=/root/.cache/trivy",
		"TRIVY_INSECURE=true",
		"TRIVY_NON_SSL=true",
		"TRIVY_DB_REPOSITORY=ghcr.io/aquasecurity/trivy-db,public.ecr.aws/aquasecurity/trivy-db",
		"TRIVY_JAVA_DB_REPOSITORY=ghcr.io/aquasecurity/trivy-java-db,public.ecr.aws/aquasecurity/trivy-java-db",
	}

	ssh, err := network.SSHForward()
	if err != nil {
		slog.Error("Failed to create ssh forward", "error", err)
		os.Exit(1)
	}

	opts = ssh.Apply(&opts)
	opts = utils.ApplySocket(container.GetBuild().Runtime, &opts)

	opts.Entrypoint = []string{"sh", "/tmp/script.sh"}

	// opts.Cmd = []string{"/tmp/script.sh"}
	// opts.Entrypoint = []string{"sh", "-c", "ls", "-lha", "/tmp"}
	// opts.Cmd = []string{"3600s"}
	// opts.Cmd = []string{"image", "--severity", "CRITICAL,HIGH", "--ignore-unfixed", "--scanners", "vuln", "--format", "json", "--output", "/usr/src/trivy.json", container.GetBuild().ImageURI()}

	// opts.Cmd = []string{"sonar-scanner", "-Dsonar.projectBaseDir=/usr/src"}
	err = c.Container.Create(opts)
	if err != nil {
		return err
	}

	err = c.CopyScript()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Container.Start()
	if err != nil {
		return err
	}

	return c.Container.Wait()
}

func (c *TrivyContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *TrivyContainer) Run() error {
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Scan()
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}
	return nil
}
