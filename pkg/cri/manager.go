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
		fmt.Println("Using Docker")
		return docker.NewDockerManager()
	case utils.Podman:
		fmt.Println("Using Podman")
		return podman.NewPodmanManager()
	case utils.Test:
		fmt.Println("Using Test")
		return critest.NewMockContainerManager()
	default:
		slog.Error("unknown container runtime stop")
		os.Exit(1)
	}
	return nil, fmt.Errorf("unknown container runtime stop")
}

func DetectContainerRuntime() utils.RuntimeType {
	runtime := os.Getenv("CONTAINER_RUNTIME")
	if runtime != "" {
		switch runtime {
		case "docker":
			fmt.Println("Detect Docker")
			return utils.Docker
		case "podman":
			fmt.Println("Detect Podman")
			return utils.Podman
		case "test":
			fmt.Println("Detect Test")
			return utils.Test
		default:
			slog.Error("unknown container runtime", "runtime", runtime)
			os.Exit(1)
		}
	}
	if _, err := exec.LookPath("docker"); err == nil {
		return utils.Docker
	}
	if _, err := exec.LookPath("podman"); err == nil {
		return utils.Podman
	}
	slog.Error("unknown container runtime")
	os.Exit(1)
	return utils.RuntimeType("unknown")
}
