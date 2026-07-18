package network

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/stretchr/testify/assert"
)

func TestSSHForward(t *testing.T) {
	// Todo: add test for podman runtime
	tests := []struct {
		want *Forward
		os   string
		cri  utils.RuntimeType
	}{
		{
			os:  "linux",
			cri: utils.Docker,
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
			os:  "darwin",
			cri: utils.Docker,
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
			os:   "unknown",
			cri:  utils.Unknown,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Setenv("SSH_AUTH_SOCK", "linux_ssh_auth_socket")
		t.Run("test for "+tt.os, func(t *testing.T) {
			build := &container.Build{
				Runtime: tt.cri,
				Platform: types.Platform{
					Host: &types.PlatformSpec{
						OS: tt.os,
					},
				},
			}
			got, err := SSHForward(*build)
			if tt.want == nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
