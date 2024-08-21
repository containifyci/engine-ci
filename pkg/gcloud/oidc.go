package gcloud

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/filesystem"

	u "github.com/containifyci/engine-ci/pkg/utils"
)

//go:embed Dockerfile*
var f embed.FS

//go:embed auth_provider.go.tmpl
var authProvider string

const (
	CI_IMAGE = "golang:1.22.5-alpine"
)

type GCloudContainer struct {
	applicationCredentials string
	*container.Container
}

func New() *GCloudContainer {
	return &GCloudContainer{
		Container: container.New(container.BuildEnv),
	}
}

func (c *GCloudContainer) IsAsync() bool    { return false }
func (c *GCloudContainer) Name() string     { return "gcloud_oidc" }
func (c *GCloudContainer) Pull() error      { return c.Container.Pull(CI_IMAGE) }
func (c *GCloudContainer) Images() []string { return []string{CI_IMAGE} }

func (c *GCloudContainer) BuildImage() error {
	image := Image()

	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}

	platforms := types.GetPlatforms(container.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	return c.Container.BuildIntermidiateContainer(image, dockerFile, platforms...)
}

func (c *GCloudContainer) Auth() error {
	opts := types.ContainerConfig{}
	opts.Image = Image()

	dir, _ := filepath.Abs(".")
	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/src",
		},
	}

	opts.Env = []string{}

	googleADC := u.GetEnvWithDefault("GOOGLE_APPLICATION_CREDENTIALS", func() string {
		//TODO support multiple os (its only for macos)
		homeDir := filesystem.HomeDir()
		return filepath.Join(homeDir, ".config/gcloud/application_default_credentials.json")
	})

	if filesystem.FileExists(googleADC) {
		cnt, err := os.ReadFile(googleADC)
		if err != nil {
			return err
		}
		c.applicationCredentials = string(cnt)
		opts.Env = append(opts.Env,
			fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", "/tmp/.gcloud/adc.json"),
			"ACCOUNT_EMAIL_OR_UNIQUEID=frank.ittermann@goflink.com",
		)
	} else {
		opts.Env = append(opts.Env,
			fmt.Sprintf("WORKLOAD_IDENTITY_PROVIDER=%s", os.Getenv("WORKLOAD_IDENTITY_PROVIDER")),
			fmt.Sprintf("ACTIONS_ID_TOKEN_REQUEST_URL=%s", os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")),
			fmt.Sprintf("ACTIONS_ID_TOKEN_REQUEST_TOKEN=%s", os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")),
			fmt.Sprintf("ACCOUNT_EMAIL_OR_UNIQUEID=%s", os.Getenv("ACCOUNT_EMAIL_OR_UNIQUEID")),
		)
	}

	// opts.Cmd = []string{"sleep", "300"}

	err := c.Container.Create(opts)
	if err != nil {
		return err
	}

	err = c.Container.CopyContentTo(c.applicationCredentials, "/tmp/.gcloud/adc.json")
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

func (c *GCloudContainer) Code() string {
	return strings.ReplaceAll(authProvider, "package gcloud", "package main")
}

func Image() string {
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile.go", "error", err)
		os.Exit(1)
	}
	tag := container.ComputeChecksum(dockerFile)
	return utils.ImageURI(container.GetBuild().ContainifyRegistry, "gcloud", tag)
}

func (c *GCloudContainer) Run() error {
	// if os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL") == "" ||
	if container.GetBuild().CustomString("gcloud_oidc") == "" {
		slog.Info("No ACTIONS_ID_TOKEN_REQUEST_URL found and Custom property gcloud_oidc not set, skipping gcloud_oidc container", "ACTIONS_ID_TOKEN_REQUEST_URL", os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"), "gcloud_oidc", container.GetBuild().CustomString("gcloud_oidc"))
		return nil
	}

	err := c.BuildImage()
	if err != nil {
		slog.Error("Failed to build go image: %s", "error", err)
		return err
	}

	err = c.Auth()
	slog.Info("Container created", "containerId", c.Container.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		return err
	}
	return nil
}
