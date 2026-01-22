package network

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/utils"

	"github.com/stretchr/testify/assert"
)

func TestAddress_ForContainer(t *testing.T) {
	tests := []struct {
		name string
		os   string
		cri  utils.RuntimeType
		env  container.EnvType
		want string
		addr Address
	}{
		{
			name: "Local URL",
			os:   "darwin",
			cri:  utils.Docker,
			addr: Address{
				Host:         "https://sonarcloud.io:443",
				InternalHost: "http://localhost:9000",
			},
			env:  container.LocalEnv,
			want: "http://host.docker.internal:9000",
		},
		{
			name: "Local URL",
			os:   "darwin",
			cri:  utils.Podman,
			addr: Address{
				Host:         "https://sonarcloud.io:443",
				InternalHost: "http://localhost:9000",
			},
			env:  container.LocalEnv,
			want: "http://host.containers.internal:9000",
		},
		{
			name: "Build URL",
			os:   "darwin",
			cri:  utils.Podman,
			addr: Address{
				Host:         "https://sonarcloud.io:443",
				InternalHost: "http://localhost:9000",
			},
			env:  container.BuildEnv,
			want: "https://sonarcloud.io:443",
		},
		{
			name: "Local URL",
			os:   "linux",
			cri:  utils.Docker,
			addr: Address{
				Host:         "https://sonarcloud.io:443",
				InternalHost: "http://localhost:9000",
			},
			env:  container.LocalEnv,
			want: "http://host.docker.internal:9000",
		},
		{
			name: "Build URL",
			os:   "linux",
			cri:  utils.Podman,
			addr: Address{
				Host:         "https://sonarcloud.io:443",
				InternalHost: "http://localhost:9000",
			},
			env:  container.BuildEnv,
			want: "https://sonarcloud.io:443",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name+" os "+tt.os+" cri "+string(tt.cri), func(t *testing.T) {
			RuntimeOS = tt.os
			host := tt.addr.ForContainer(container.Build{Env: tt.env, Runtime: tt.cri})
			assert.Equal(t, tt.want, host)
		})
	}
}

func TestHostname(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		host   string
		port   string
		want   string
	}{
		{
			name:   "Scheme and Host",
			scheme: "http",
			host:   "localhost",
			port:   "",
			want:   "http://localhost",
		},
		{
			name:   "Scheme and Port",
			scheme: "http",
			host:   "",
			port:   "9000",
			want:   "http://:9000",
		},
		{
			name:   "Host and Port",
			scheme: "",
			host:   "localhost",
			port:   "9000",
			want:   "localhost:9000",
		},
		{
			name:   "Scheme, Host and Port",
			scheme: "http",
			host:   "localhost",
			port:   "9000",
			want:   "http://localhost:9000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := hostname(tt.scheme, tt.host, tt.port)
			assert.Equal(t, tt.want, host)
		})
	}
}
