package network

import (
	"fmt"
	"log/slog"
	"net/url"
	"runtime"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
)

var RuntimeOS = runtime.GOOS

type Address struct {
	Host         string
	InternalHost string
}

func (a *Address) NewAddress(arg *container.Build) {
	parsedURL, err := url.Parse(a.InternalHost)
	slog.Info("Parsed URL", "parsedURL", parsedURL, "url", a.InternalHost)
	if err != nil {
		slog.Error("Error parsing URL", "error", err, "url", a.InternalHost)
		return
	}
	scheme := parsedURL.Scheme
	_ = parsedURL.Hostname()
	port := parsedURL.Port()
	var internalHost string

	switch RuntimeOS {
	case "windows":
		internalHost = hostname(scheme, "host.docker.internal", port)
	case "darwin":
		if arg.Runtime == utils.Docker {
			internalHost = hostname(scheme, "host.docker.internal", port)
		} else if arg.Runtime == utils.Podman {
			internalHost = hostname(scheme, "host.containers.internal", port)
		} else {
			internalHost = hostname(scheme, "host.docker.internal", port)
		}
	case "linux":
		internalHost = hostname(scheme, "localhost", port)
	default:
		internalHost = hostname(scheme, "host.docker.internal", port)
	}
	a.InternalHost = internalHost
}

func hostname(scheme, host, port string) string {
	if scheme == "" && port == "" {
		return host
	}
	if scheme == "" {
		return fmt.Sprintf("%s:%s", host, port)
	}
	if port == "" {
		return fmt.Sprintf("%s://%s", scheme, host)
	}

	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

func (a *Address) ForContainer(env container.EnvType) string {
	a.NewAddress(container.GetBuild())

	switch env {
	case container.LocalEnv:
		return a.InternalHost
	case container.BuildEnv:
		return a.Host
	default:
		return a.Host
	}
}

func (a *Address) ForContainerDefault(arg ...*container.Build) string {
	if len(arg) > 0 {
		a.NewAddress(arg[0])
	} else {
		a.NewAddress(container.GetBuild())
	}

	return a.InternalHost
}
