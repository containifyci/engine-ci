package protobuf

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

//go:embed Dockerfile
var f embed.FS

type ProtogufContainer struct {
	Command        string
	WithHttp       bool
	WithTag        bool
	SourcePackages []string
	SourceFiles    []string
	*container.Container
}

func New() *ProtogufContainer {
	command := "protoc"
	if v, ok := container.GetBuild().Custom["protobuf_cmd"]; ok {
		command = v[0]
	}
	withHttp := false
	if v, ok := container.GetBuild().Custom["withHttp"]; ok {
		withHttp = v[0] == "true"
	}
	withTag := false
	if v, ok := container.GetBuild().Custom["withTag"]; ok {
		withTag = v[0] == "true"
	}
	return &ProtogufContainer{
		Command:        command,
		WithHttp:       withHttp,
		WithTag:        withTag,
		Container:      container.New(container.BuildEnv),
		SourcePackages: container.GetBuild().SourcePackages,
		SourceFiles:    container.GetBuild().SourceFiles,
	}
}

func (c *ProtogufContainer) Images() []string {
	return []string{Image()}
}

func Image() string {
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}
	tag := computeChecksum(dockerFile)
	return fmt.Sprintf("%s/%s/%s:%s", container.GetBuild().Registry, "containifyci", "protobuf", tag)
}

func (c *ProtogufContainer) Name() string {
	return "protobuf"
}

func (c *ProtogufContainer) Pull() error {
	image := Image()
	err := c.Container.Pull(image)
	if err != nil {
		slog.Info("Failed to pull image", "error", err, "image", image)
	}
	return nil
}

func (c *ProtogufContainer) CopyBuildScript() error {
	// Create a temporary script in-memory
	script := Script(NewBuildScript(c.Command, c.SourcePackages, c.SourceFiles, c.WithHttp, c.WithTag))
	err := c.Container.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func (c *ProtogufContainer) Generate() error {
	image := Image()

	opts := types.ContainerConfig{}
	opts.Image = image
	opts.Env = []string{}
	// opts.Cmd = []string{"protoc", "-I=/src/pkg/storage/", "--go-grpc_out=/src/pkg/storage", "--plugin=grpc", "--go_out=/src/pkg/storage", "/src/pkg/storage/token_service.proto"}
	opts.Cmd = []string{"sh", "/tmp/script.sh"}
	opts.Platform = types.AutoPlatform
	opts.WorkingDir = "/src"

	dir, _ := filepath.Abs(".")

	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/src",
		},
	}

	slog.Info("Protobuf container created")

	err := c.Container.Create(opts)
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		os.Exit(1)
	}

	err = c.CopyBuildScript()
	if err != nil {
		slog.Error("Failed to copy build script", "error", err)
		os.Exit(1)
	}

	err = c.Container.Start()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Container.Wait()
	if err != nil {
		slog.Error("Failed to wait for container", "error", err)
		os.Exit(1)
	}

	slog.Info("Protobuf generated")

	return err
}

func (c *ProtogufContainer) Build() error {
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

// TODO: provide a shorter checksum
func computeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (c *ProtogufContainer) Run() error {
	if len(c.SourcePackages) == 0 && len(c.SourceFiles) == 0 {
		slog.Info("Skip protobuf generate. No source packages or files provided")
		return nil
	}
	err := c.Pull()
	if err != nil {
		slog.Error("Failed to pull image", "error", err)
		return err
	}
	err = c.Build()
	if err != nil {
		slog.Error("Failed to build protobuf image", "error", err)
		os.Exit(1)
	}
	err = c.Generate()
	if err != nil {
		slog.Error("Failed to generate protobuf", "error", err)
		return err
	}
	return nil
}
