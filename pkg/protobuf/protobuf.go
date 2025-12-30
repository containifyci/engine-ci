package protobuf

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

//go:embed Dockerfile
var f embed.FS

type ProtogufContainer struct {
	*container.Container
	Command        string
	SourcePackages []string
	SourceFiles    []string
	WithHttp       bool
	WithTag        bool
}

// Matches implements the Build interface - Protobuf only runs for golang builds
func Matches(build container.Build) bool {
	return build.BuildType == container.GoLang &&
		len(build.SourceFiles) > 0
}

func New() build.BuildStepv3 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := newC(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  Images,
		Name_:     "protobuf",
		Alias_:    "protobuf",
		Async_:    false,
	}
}

func newC(build container.Build) *ProtogufContainer {
	command := "protoc"
	if v, ok := build.Custom["protobuf_cmd"]; ok {
		command = v[0]
	}
	withHttp := false
	if v, ok := build.Custom["withHttp"]; ok {
		withHttp = v[0] == "true"
	}
	withTag := false
	if v, ok := build.Custom["withTag"]; ok {
		withTag = v[0] == "true"
	}
	return &ProtogufContainer{
		Command:        command,
		WithHttp:       withHttp,
		WithTag:        withTag,
		Container:      container.New(build),
		SourcePackages: build.SourcePackages,
		SourceFiles:    build.SourceFiles,
	}
}

func Images(build container.Build) []string {
	return []string{Image(build)}
}

func Image(build container.Build) string {
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}
	tag := computeChecksum(dockerFile)
	return utils.ImageURI(build.ContainifyRegistry, "protobuf", tag)
	// return fmt.Sprintf("%s/%s/%s:%s", container.GetBuild().Registry, "containifyci", "protobuf", tag)
}
func (c *ProtogufContainer) Pull() error {
	image := Image(*c.GetBuild())
	err := c.Container.Pull(image)
	if err != nil {
		slog.Info("Failed to pull image", "error", err, "image", image)
	}
	return nil
}

func (c *ProtogufContainer) CopyBuildScript() error {
	// Create a temporary script in-memory
	script := Script(NewBuildScript(c.Command, c.SourcePackages, c.SourceFiles, c.WithHttp, c.WithTag))
	err := c.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func (c *ProtogufContainer) Generate() error {
	image := Image(*c.GetBuild())

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

	err := c.Create(opts)
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		os.Exit(1)
	}

	err = c.CopyBuildScript()
	if err != nil {
		slog.Error("Failed to copy build script", "error", err)
		os.Exit(1)
	}

	err = c.Start()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Wait()
	if err != nil {
		slog.Error("Failed to wait for container", "error", err)
		os.Exit(1)
	}

	slog.Info("Protobuf generated")

	return err
}

func (c *ProtogufContainer) Build() error {
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
