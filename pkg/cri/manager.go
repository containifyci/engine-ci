package cri

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"sync"

	"github.com/containifyci/engine-ci/pkg/cri/critest"
	"github.com/containifyci/engine-ci/pkg/cri/docker"
	"github.com/containifyci/engine-ci/pkg/cri/host"
	"github.com/containifyci/engine-ci/pkg/cri/podman"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
)

type ContainerManager interface {
	CreateContainer(ctx context.Context, opts *types.ContainerConfig, authBase64 string) (string, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, signal string) error
	CommitContainer(ctx context.Context, containerID string, opts types.CommitOptions) (string, error)
	RemoveContainer(ctx context.Context, containerID string) error
	ContainerList(ctx context.Context, all bool) ([]*types.Container, error)
	ContainerLogs(ctx context.Context, id string, ShowStdout bool, ShowStderr bool, Follow bool) (io.ReadCloser, error)
	CopyContentToContainer(ctx context.Context, id, content, dest string) error
	CopyDirectorToContainer(ctx context.Context, id, srcPath, dstPath string) error
	CopyToContainer(ctx context.Context, id, srcPath, dstPath string) error
	CopyFileFromContainer(ctx context.Context, id string, srcPath string) (string, error)
	ExecContainer(ctx context.Context, id string, cmd []string, attachStdOut bool) (io.Reader, error)
	InspectContainer(ctx context.Context, id string) (*types.ContainerConfig, error)
	WaitContainer(ctx context.Context, id string, waitCondition string) (*int64, error)

	BuildImage(ctx context.Context, dockerfile []byte, imageName string, platform string) (io.ReadCloser, error)
	BuildMultiArchImage(ctx context.Context, dockerfile []byte, dockerCtx *bytes.Buffer, imageName string, platforms []string, authBase64 string) (io.ReadCloser, []string, error)

	ListImage(ctx context.Context, image string) ([]string, error)
	PullImage(ctx context.Context, image string, authBase64 string, platform string) (io.ReadCloser, error)
	TagImage(ctx context.Context, source, target string) error
	PushImage(ctx context.Context, target string, authBase64 string) (io.ReadCloser, error)
	RemoveImage(ctx context.Context, target string) error
	InspectImage(ctx context.Context, image string) (*types.ImageInfo, error)

	Name() string
}

var (
	lazyValue ContainerManager
	once      sync.Once
)

func InitContainerRuntime() (ContainerManager, error) {
	var err error
	once.Do(func() {
		lazyValue, err = getRuntime()
	})
	return lazyValue, err
}

func getRuntime() (ContainerManager, error) {
	switch DetectContainerRuntime() {
	case utils.Docker:
		slog.Info("Using Docker")
		return docker.NewDockerManager()
	case utils.Podman:
		slog.Info("Using Podman")
		return podman.NewPodmanManager()
	case utils.Test:
		slog.Info("Using Test")
		return critest.NewMockContainerManager()
	case utils.Host:
		slog.Info("Using Host")
		return host.NewHostManager(), nil
	default:
		return nil, fmt.Errorf("unknown container runtime stop")
	}
}

func DetectContainerRuntime() utils.RuntimeType {
	runtime := os.Getenv("CONTAINER_RUNTIME")
	if runtime != "" {
		switch runtime {
		case "docker":
			slog.Info("Detect Docker")
			return utils.Docker
		case "podman":
			slog.Info("Detect Podman")
			return utils.Podman
		case "test":
			slog.Info("Detect Test")
			return utils.Test
		case "host":
			slog.Info("Detect Host")
			return utils.Host
		default:
			slog.Error("unknown container runtime", "runtime", runtime)
			os.Exit(1)
		}
	}

	// Use parallel detection for faster startup
	return detectRuntimeParallel()
}

// detectRuntimeParallel checks for container runtimes in parallel
func detectRuntimeParallel() utils.RuntimeType {
	type runtimeResult struct {
		runtime utils.RuntimeType
		found   bool
	}

	results := make(chan runtimeResult, 2)

	// Check Docker
	go func() {
		_, err := exec.LookPath("docker")
		results <- runtimeResult{runtime: utils.Docker, found: err == nil}
	}()

	// Check Podman
	go func() {
		_, err := exec.LookPath("podman")
		results <- runtimeResult{runtime: utils.Podman, found: err == nil}
	}()

	// Collect results with priority (Docker first)
	var dockerFound, podmanFound bool
	for i := 0; i < 2; i++ {
		result := <-results
		switch result.runtime {
		case utils.Docker:
			dockerFound = result.found
		case utils.Podman:
			podmanFound = result.found
		}
	}

	// Return in priority order
	if dockerFound {
		return utils.Docker
	}
	if podmanFound {
		return utils.Podman
	}

	slog.Error("unknown container runtime")
	return utils.RuntimeType("unknown")
}
