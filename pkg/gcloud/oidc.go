package gcloud

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

const (
	CI_IMAGE = "google/cloud-sdk:slim"
)

type GCloudContainer struct {
	*container.Container
}

func New() *GCloudContainer {
	return &GCloudContainer{
		Container: container.New(container.BuildEnv),
	}
}

func (c *GCloudContainer) IsAsync() bool    { return false }
func (c *GCloudContainer) Name() 		string  { return "gcloud_oidc" }
func (c *GCloudContainer) Pull()    error   { return c.Container.Pull(CI_IMAGE) }
func (c *GCloudContainer) Images() []string { return []string{CI_IMAGE} }

func (c *GCloudContainer) Auth() error {
	opts := types.ContainerConfig{}
	opts.Image = CI_IMAGE

	opts.WorkingDir = "/src"

	dir, _ := filepath.Abs(".")
	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/src",
		},
	}

	opts.Script = `
#!/bin/sh
env | sort
gcloud auth list
gcloud auth application-default print-access-token > /src/.gcloud_token`

	opts.Env = []string{"GOOGLE_APPLICATION_CREDENTIALS=/src/creds.json"}
	opts.Cmd = []string{"sh", "/tmp/script.sh"}

	aud := os.Getenv("WORKLOAD_IDENTITY_PROVIDER")
	url := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL")
	token := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")

	err := c.Container.Create(opts)
	slog.Info("Container created", "containerId", c.Container.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Container.CopyContentTo(c.CredScript(aud, url, token), "/src/creds.json")
	if err != nil {
		slog.Error("Failed to copy cred.json", "error", err)
		os.Exit(1)
	}

	//TODO: maybe define a general entrypoint for all containers
	//only the containers can then define a script that is called by the entrypoint
	err = c.Container.CopyContentTo(opts.Script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Container.Start()
	if err != nil {
		slog.Error("Failed to start container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Container.Wait()
	if err != nil {
		slog.Error("Failed to wait for container: %s", "error", err)
		os.Exit(1)
	}

	return err
}

func (c *GCloudContainer) CredScript(audiens, url, token string) string {
	script := fmt.Sprintf(`
{
	"type": "external_account",
	"audience": "%s",
	"subject_token_type": "urn:ietf:params:oauth:token-type:jwt",
	"token_url": "https://sts.googleapis.com/v1/token",
	"credential_source": {
		"url": "%s&audience=%s",
		"headers": {
			"Authorization": "Bearer %s"
		},
		"format": {
			"type": "json",
			"subject_token_field_name": "value"
		}
	}
}
`, audiens, url, audiens, token)
	return script
}

func (c *GCloudContainer) Run() error {
	if os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL") == "" ||
		container.GetBuild().CustomString("gcloud_oidc") == "" {
		slog.Info("No ACTIONS_ID_TOKEN_REQUEST_URL found and Custom property gcloud_oidc not set, skipping gcloud_oidc container", "ACTIONS_ID_TOKEN_REQUEST_URL", os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"), "gcloud_oidc", container.GetBuild().CustomString("gcloud_oidc"))
		return nil
	}

	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull base images: %s", "error", err)
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
