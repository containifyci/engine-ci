package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

type ContainerSocket struct {
	RuntimeType RuntimeType
	Source      string
	Target      string
}

func Socket(runtimeType RuntimeType) (*ContainerSocket, error) {
	switch runtimeType {
	case Docker:
		return DockerSocket()
	case Podman:
		return PodmanSocket()
	default:
		return nil, fmt.Errorf("unknown runtime: %s", runtimeType)
	}
}

func DockerSocket() (*ContainerSocket, error) {
	dockerSocket := "/var/run/docker.sock"
	return &ContainerSocket{
		RuntimeType: Docker,
		Source:      dockerSocket,
		Target:      dockerSocket,
	}, nil
}

func PodmanSocket() (*ContainerSocket, error) {
	// Check if podman is available
	if _, err := exec.LookPath("podman"); err != nil {
		return nil, fmt.Errorf("podman not found in PATH: %w", err)
	}

	cmd, err := exec.Command("podman", "info", "-f", "{{ .Host.RemoteSocket.Path }}").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get podman socket info: %w", err)
	}
	if strings.HasPrefix(string(cmd), "unix://") {
		cmd = cmd[7:]
	}
	podmanSocket := strings.TrimSpace(string(cmd))
	return &ContainerSocket{
		RuntimeType: Podman,
		Source:      podmanSocket,
		Target:      "/var/run/podman.sock",
	}, nil
}

func ApplySocket(runtimeType RuntimeType, opts *types.ContainerConfig) types.ContainerConfig {
	socket, err := Socket(runtimeType)
	if err != nil {
		return *opts
	}
	opts.Volumes = append(opts.Volumes, types.Volume{
		Type:   "bind",
		Source: socket.Source,
		Target: socket.Target,
	})

	return *opts
}
