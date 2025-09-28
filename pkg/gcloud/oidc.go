package gcloud

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/filesystem"

	u "github.com/containifyci/engine-ci/pkg/utils"
)

//go:embed Dockerfile*
var f embed.FS

//go:embed src/*
var d embed.FS

const (
	CI_IMAGE = "golang:1.22.5-alpine"
)

type GCloudContainer struct {
	*container.Container
	applicationCredentials string
}

func New(build container.Build) *GCloudContainer {
	return &GCloudContainer{
		Container: container.New(build),
	}
}

func (c *GCloudContainer) IsAsync() bool    { return false }
func (c *GCloudContainer) Name() string     { return "gcloud_oidc" }
func (c *GCloudContainer) Pull() error      { return c.Container.Pull(CI_IMAGE) }
func (c *GCloudContainer) Images() []string { return []string{CI_IMAGE} }

// Matches implements the Build interface - GCloud runs for all builds
func (c *GCloudContainer) Matches(build container.Build) bool {
	return true // GCloud setup runs for all builds
}

// calculateDirChecksum computes a combined SHA-256 checksum for all files in the specified directory within the embed.FS.
func calculateDirChecksum(_fs embed.FS) ([]byte, error) {
	// Initialize a SHA-256 hasher.
	hasher := sha256.New()

	// Walk through the directory in the embedded filesystem.
	err := fs.WalkDir(_fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		// Only process regular files (skip directories)
		if !d.IsDir() {
			// Open the embedded file.
			file, err := _fs.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			// Hash the content of the file.
			if _, err := io.Copy(hasher, file); err != nil {
				return fmt.Errorf("failed to hash content of file %s: %w", path, err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Return the combined checksum.
	return hasher.Sum(nil), nil
}

func (c *GCloudContainer) BuildImage() error {
	image := Image(c.GetBuild())

	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	c.Source = d

	return c.BuildIntermidiateContainer(image, dockerFile, platforms...)
}

func (c *GCloudContainer) Auth() error {
	opts := types.ContainerConfig{}
	opts.Image = Image(c.GetBuild())

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
			fmt.Sprintf("ACCOUNT_EMAIL_OR_UNIQUEID=%s", u.GetEnv("ACCOUNT_EMAIL_OR_UNIQUEID", "build")),
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

	err := c.Create(opts)
	if err != nil {
		return err
	}

	err = c.CopyContentTo(c.applicationCredentials, "/tmp/.gcloud/adc.json")
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

func Image(build *container.Build) string {
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile.go", "error", err)
		os.Exit(1)
	}

	fsCheckSum, err := calculateDirChecksum(d)
	if err != nil {
		slog.Error("Failed to calculate embed.FS checksum", "error", err)
		os.Exit(1)
	}

	dckCheckSum := sha256.Sum256(dockerFile)
	tag := container.SumChecksum(fsCheckSum, dckCheckSum[:])
	// tag := container.ComputeChecksum(dockerFile)
	return utils.ImageURI(build.ContainifyRegistry, "gcloud", tag)
}

func (c *GCloudContainer) Run() error {
	if os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL") == "" ||
		c.GetBuild().CustomString("gcloud_oidc") == "" {
		slog.Info("No ACTIONS_ID_TOKEN_REQUEST_URL found and Custom property gcloud_oidc not set, skipping gcloud_oidc container", "ACTIONS_ID_TOKEN_REQUEST_URL", os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"), "gcloud_oidc", c.GetBuild().CustomString("gcloud_oidc"))
		return nil
	}

	err := c.BuildImage()
	if err != nil {
		slog.Error("Failed to build go image: %s", "error", err)
		return err
	}

	err = c.Auth()
	slog.Info("Container created", "containerId", c.ID)
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		return err
	}
	return nil
}
