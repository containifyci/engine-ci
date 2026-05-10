package network

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
)

const (
	DARWIN_SSH_AUTH_SOCK = "/run/host-services/ssh-auth.sock"
	// SOCAT_CONTAINER_SOCK is the UNIX socket path inside the container
	// that socat listens on and forwards to the host via TCP.
	SOCAT_CONTAINER_SOCK = "/tmp/ssh-agent.sock"
	// DefaultSocatPort is the default TCP port used for socat SSH forwarding.
	DefaultSocatPort = 12345
)

// Forward holds the configuration for forwarding the SSH agent socket
// into a container. By default it uses a bind mount, but when SocatPort
// is set it uses TCP-based forwarding via socat instead.
type Forward struct {
	Volume       *types.Volume
	hostSocatCmd *exec.Cmd
	Source       string
	Target       string
	Env          string
	SocatPort    int
}

// Apply applies the SSH forward configuration to the container options.
//
// When SocatPort is set (socat-based forwarding), it sets an entrypoint
// that runs socat inside the container to forward from a UNIX socket
// to host.docker.internal:PORT, and sets SSH_AUTH_SOCK to that socket.
// Socat must be installed in the container image.
//
// When SocatPort is 0 (default), it uses a bind mount of the host's
// SSH agent socket into the container.
func (f *Forward) Apply(opts *types.ContainerConfig) types.ContainerConfig {
	if f == nil {
		return *opts
	}
	if f.Env != "" {
		opts.Env = append(opts.Env, f.Env)
	}
	if f.Volume != nil && f.SocatPort <= 0 {
		opts.Volumes = append(opts.Volumes, *f.Volume)
	}

	if f.SocatPort > 0 {
		// Use an entrypoint wrapper that starts socat in the container
		// to forward TCP:host.docker.internal:PORT -> UNIX:SOCAT_CONTAINER_SOCK
		// in the background, then execs the original Cmd.
		entrypoint := fmt.Sprintf(
			`socat UNIX-LISTEN:%s,fork TCP:host.docker.internal:%d & exec "$@"`,
			SOCAT_CONTAINER_SOCK, f.SocatPort,
		)
		opts.Entrypoint = []string{"/bin/sh", "-c", entrypoint, "--"}
	}

	return *opts
}

// Cleanup kills the host-side socat process if one was started.
// It should be called with 'defer' after a successful SSHForwardWithSocat call.
func (f *Forward) Cleanup() {
	if f == nil || f.hostSocatCmd == nil || f.hostSocatCmd.Process == nil {
		return
	}
	slog.Debug("Cleaning up host-side socat process", "pid", f.hostSocatCmd.Process.Pid)
	if err := f.hostSocatCmd.Process.Kill(); err != nil {
		slog.Warn("Failed to kill host socat process", "error", err)
	}
	// Wait to reap the process; ignore errors after kill.
	_ = f.hostSocatCmd.Wait()
	f.hostSocatCmd = nil
}

// SSHForward creates a bind-mount based SSH forward (default approach).
// This is the recommended approach for Linux and works on macOS when
// /run/host-services/ssh-auth.sock is bind-mountable.
func SSHForward(build container.Build) (*Forward, error) {
	switch build.Platform.Host.OS {
	case "linux":
		sshAuthSocket := os.Getenv("SSH_AUTH_SOCK")
		if sshAuthSocket == "" {
			slog.Warn("SSH_AUTH_SOCK is not set")
			return nil, nil
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
		if build.Runtime == "podman" {
			slog.Warn("SSH forwarding is not supported on macOS with Podman")
			return nil, nil
		}
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
	return nil, fmt.Errorf("unsupported platform: %s", build.Platform.Host)
}

// SSHForwardWithSocat creates a TCP-based SSH forward via socat.
//
// It starts a host-side socat process that listens on the given port
// and forwards connections to the host's SSH agent UNIX socket. The
// returned Forward's Apply method configures the container to connect
// to that TCP port via host.docker.internal.
//
// The caller MUST call f.Cleanup() (typically via defer) when the
// forward is no longer needed, to kill the host-side socat process.
// Socat must be installed on the host and in the container image.
//
// If port is 0, DefaultSocatPort is used.
func SSHForwardWithSocat(build container.Build, port int) (*Forward, error) {
	if port <= 0 {
		port = DefaultSocatPort
	}

	sshAuthSocket, err := resolveSSHAuthSocket(build)
	if err != nil {
		return nil, err
	}
	if sshAuthSocket == "" {
		// No SSH socket available; not an error, just skip forwarding.
		return nil, nil
	}

	// Check that socat is available on the host.
	if _, err := exec.LookPath("socat"); err != nil {
		return nil, fmt.Errorf("socat is not installed on the host: %w", err)
	}

	// Start host-side socat: TCP-LISTEN:PORT -> UNIX-CONNECT:$SSH_AUTH_SOCK
	socatArgs := []string{
		fmt.Sprintf("TCP-LISTEN:%d,reuseaddr,fork", port),
		fmt.Sprintf("UNIX-CONNECT:%s", sshAuthSocket),
	}
	cmd := exec.Command("socat", socatArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start host-side socat: %w", err)
	}

	slog.Debug("Started host-side socat for SSH forwarding",
		"port", port,
		"socket", sshAuthSocket,
		"pid", cmd.Process.Pid,
	)

	portStr := strconv.Itoa(port)

	return &Forward{
		SocatPort:    port,
		Source:       fmt.Sprintf("tcp:host.docker.internal:%s", portStr),
		Target:       SOCAT_CONTAINER_SOCK,
		Env:          "SSH_AUTH_SOCK=" + SOCAT_CONTAINER_SOCK,
		hostSocatCmd: cmd,
	}, nil
}

// resolveSSHAuthSocket resolves the host's SSH agent socket path
// depending on the platform.
func resolveSSHAuthSocket(build container.Build) (string, error) {
	switch build.Platform.Host.OS {
	case "linux":
		sshAuthSocket := os.Getenv("SSH_AUTH_SOCK")
		if sshAuthSocket == "" {
			slog.Warn("SSH_AUTH_SOCK is not set")
			return "", nil
		}
		return sshAuthSocket, nil
	case "darwin":
		if build.Runtime == "podman" {
			slog.Warn("SSH forwarding is not supported on macOS with Podman")
			return "", nil
		}
		return DARWIN_SSH_AUTH_SOCK, nil
	default:
		return "", errors.New("unsupported platform")
	}
}
