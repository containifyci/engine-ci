package network

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

const (
	DARWIN_SSH_AUTH_SOCK = "/run/host-services/ssh-auth.sock"
)

type Forward struct {
	Source string
	Target string
	Env    string
	Volume *types.Volume
}

// TODO implement ssh socket forward with
// socat TCP-LISTEN:12345,reuseaddr,fork UNIX-CONNECT:$SSH_AUTH_SOCK on host
// socat TCP-LISTEN:12345,reuseaddr,fork UNIX-CONNECT:$SSH_AUTH_SOCK in container
func (f *Forward) Apply(opts *types.ContainerConfig) types.ContainerConfig {
	if f == nil {
		return *opts
	}
	if f.Env != "" {
		opts.Env = append(opts.Env, f.Env)
	}
	if f.Volume != nil {
		opts.Volumes = append(opts.Volumes, *f.Volume)
	}
	return *opts
}

func SSHForward() (*Forward, error) {
	switch container.GetBuild().Platform.Host.OS {
	case "linux":
		sshAuthSocket := os.Getenv("SSH_AUTH_SOCK")
		if sshAuthSocket == "" {
			slog.Error("SSH_AUTH_SOCK is not set")
			os.Exit(1)
		}
		return &Forward{
			Source: sshAuthSocket,
			Target: sshAuthSocket,
			Env:    "SSH_AUTH_SOCK=" + sshAuthSocket,
			Volume: &types.Volume{
				Type:   "bind",
				Source: sshAuthSocket,
				Target: sshAuthSocket,
			},
		}, nil
	case "darwin":
		if container.GetBuild().Runtime == "podman" {
			slog.Warn("SSH forwarding is not supported on macOS with Podman")
			return &Forward{}, nil
		}
		// _, err := os.Stat(DARWIN_SSH_AUTH_SOCK)
		// if err != nil {
		// 	slog.Error("SSH_AUTH_SOCK is not available on the host")
		// 	os.Exit(1)
		// }
		return &Forward{
			Source: DARWIN_SSH_AUTH_SOCK,
			Target: DARWIN_SSH_AUTH_SOCK,
			Env:    "SSH_AUTH_SOCK=" + DARWIN_SSH_AUTH_SOCK,
			Volume: &types.Volume{
				Type:   "bind",
				Source: DARWIN_SSH_AUTH_SOCK,
				Target: DARWIN_SSH_AUTH_SOCK,
			},
		}, nil
	}
	return nil, fmt.Errorf("unsupported platform: %s", container.GetBuild().Platform.Host)
}
