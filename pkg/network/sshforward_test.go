package network

import (
	"os"
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/stretchr/testify/assert"
)

func TestSSHForward(t *testing.T) {
	tests := []struct {
		name    string
		want    *Forward
		os      string
		cri     utils.RuntimeType
		wantErr bool
	}{
		{
			name: "linux with SSH_AUTH_SOCK set",
			os:   "linux",
			cri:  utils.Docker,
			want: &Forward{
				Source: "linux_ssh_auth_socket",
				Target: "linux_ssh_auth_socket",
				Env:    "SSH_AUTH_SOCK=linux_ssh_auth_socket",
				Volume: &types.Volume{
					Type:   "bind",
					Source: "linux_ssh_auth_socket",
					Target: "linux_ssh_auth_socket",
				},
			},
		},
		{
			name: "darwin with Docker",
			os:   "darwin",
			cri:  utils.Docker,
			want: &Forward{
				Source: "/run/host-services/ssh-auth.sock",
				Target: "/run/host-services/ssh-auth.sock",
				Env:    "SSH_AUTH_SOCK=/run/host-services/ssh-auth.sock",
				Volume: &types.Volume{
					Type:   "bind",
					Source: "/run/host-services/ssh-auth.sock",
					Target: "/run/host-services/ssh-auth.sock",
				},
			},
		},
		{
			name:    "darwin with Podman returns nil (skip warning)",
			os:      "darwin",
			cri:     utils.Podman,
			want:    nil,
			wantErr: false, // SSHForward returns (nil, nil) for darwin+Podman
		},
		{
			name:    "unknown OS returns error",
			os:      "unknown",
			cri:     utils.Unknown,
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origSSHAuthSock := os.Getenv("SSH_AUTH_SOCK")
			t.Setenv("SSH_AUTH_SOCK", "linux_ssh_auth_socket")
			defer os.Setenv("SSH_AUTH_SOCK", origSSHAuthSock)

			build := &container.Build{
				Runtime: tt.cri,
				Platform: types.Platform{
					Host: &types.PlatformSpec{
						OS: tt.os,
					},
				},
			}
			got, err := SSHForward(*build)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestForwardApply_BindMount(t *testing.T) {
	// Default (SocatPort == 0) uses bind mount + env
	f := &Forward{
		Source: "/tmp/ssh-xyz/agent.sock",
		Target: "/tmp/ssh-xyz/agent.sock",
		Env:    "SSH_AUTH_SOCK=/tmp/ssh-xyz/agent.sock",
		Volume: &types.Volume{
			Type:   "bind",
			Source: "/tmp/ssh-xyz/agent.sock",
			Target: "/tmp/ssh-xyz/agent.sock",
		},
	}

	opts := types.ContainerConfig{}
	opts = f.Apply(&opts)

	assert.Contains(t, opts.Env, "SSH_AUTH_SOCK=/tmp/ssh-xyz/agent.sock")
	assert.Len(t, opts.Volumes, 1)
	assert.Empty(t, opts.Entrypoint, "bind mount mode should not set an entrypoint")
}

func TestForwardApply_SocatMode(t *testing.T) {
	// SocatPort > 0 uses TCP forwarding: no bind mount, sets entrypoint wrapper
	f := &Forward{
		SocatPort: 12345,
		Source:    "tcp:host.docker.internal:12345",
		Target:    SOCAT_CONTAINER_SOCK,
		Env:       "SSH_AUTH_SOCK=" + SOCAT_CONTAINER_SOCK,
	}

	opts := types.ContainerConfig{
		Cmd: []string{"go", "build", "./..."},
	}
	opts = f.Apply(&opts)

	assert.Contains(t, opts.Env, "SSH_AUTH_SOCK="+SOCAT_CONTAINER_SOCK)
	assert.Empty(t, opts.Volumes, "socat mode should not add bind mount volumes")
	// Entrypoint: ["/bin/sh", "-c", "<socat script>", "--"]
	assert.Len(t, opts.Entrypoint, 4, "entrypoint should be shell -c <script> --")
	assert.Contains(t, opts.Entrypoint[2], "socat UNIX-LISTEN:"+SOCAT_CONTAINER_SOCK+",fork")
	assert.Contains(t, opts.Entrypoint[2], "TCP:host.docker.internal:12345")
}

func TestForwardApply_NilReceiver(t *testing.T) {
	// A nil *Forward should be safe
	var f *Forward
	opts := types.ContainerConfig{}
	opts = f.Apply(&opts)
	assert.Empty(t, opts.Env)
	assert.Empty(t, opts.Volumes)
	assert.Empty(t, opts.Entrypoint)
}

func TestSSHForwardWithSocat_NoSocat_OrSuccess(t *testing.T) {
	// If socat is installed on the host, the test will actually try to
	// start it, which would leave a zombie.  We only test the error path
	// and skip if socat happens to be present.
	build := &container.Build{
		Runtime: utils.Docker,
		Platform: types.Platform{
			Host: &types.PlatformSpec{
				OS: "linux",
			},
		},
	}

	_, err := SSHForwardWithSocat(*build, 12345)
	if err != nil {
		assert.ErrorContains(t, err, "socat is not installed")
	}
	// If err == nil, socat is installed — the test passes silently.
	// In a real CI run, we can't easily test the full lifecycle here.
}

func TestSSHForwardWithSocat_UnsupportedPlatform(t *testing.T) {
	build := &container.Build{
		Runtime: utils.Docker,
		Platform: types.Platform{
			Host: &types.PlatformSpec{
				OS: "unknown",
			},
		},
	}

	_, err := SSHForwardWithSocat(*build, 12345)
	assert.Error(t, err)
}

func TestForwardCleanup_Nil(t *testing.T) {
	// Cleanup on nil or without process should not panic
	var f *Forward
	f.Cleanup() // must not panic

	f = &Forward{SocatPort: 12345}
	f.Cleanup() // must not panic (no hostSocatCmd)
}

func TestResolveSSHAuthSocket(t *testing.T) {
	tests := []struct {
		name    string
		os      string
		runtime utils.RuntimeType
		want    string
		wantErr bool
	}{
		{
			name:    "linux with SSH_AUTH_SOCK set",
			os:      "linux",
			runtime: utils.Docker,
			want:    "linux_ssh_auth_socket",
			wantErr: false,
		},
		{
			name:    "darwin with Docker",
			os:      "darwin",
			runtime: utils.Docker,
			want:    DARWIN_SSH_AUTH_SOCK,
			wantErr: false,
		},
		{
			name:    "darwin with Podman",
			os:      "darwin",
			runtime: utils.Podman,
			want:    "",
			wantErr: false,
		},
		{
			name:    "unknown OS",
			os:      "unknown",
			runtime: utils.Unknown,
			want:    "",
			wantErr: true,
		},
	}

	origSSHAuthSock := os.Getenv("SSH_AUTH_SOCK")
	t.Setenv("SSH_AUTH_SOCK", "linux_ssh_auth_socket")
	defer os.Setenv("SSH_AUTH_SOCK", origSSHAuthSock)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			build := container.Build{
				Runtime: tt.runtime,
				Platform: types.Platform{
					Host: &types.PlatformSpec{
						OS: tt.os,
					},
				},
			}
			got, err := resolveSSHAuthSocket(build)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
