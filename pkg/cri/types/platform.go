package types

import (
	"fmt"
	"log/slog"
	_ "os"
	"runtime"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type PlatformSpec ocispec.Platform

func (p *PlatformSpec) String() string {
	if p.Variant != "" {
		return fmt.Sprintf("%s/%s/%s", p.OS, p.Architecture, p.Variant)
	}
	return fmt.Sprintf("%s/%s", p.OS, p.Architecture)
}

func (p *PlatformSpec) ToOrg() *ocispec.Platform {
	spec := ocispec.Platform(*p)
	return &spec
}

type Platform struct {
	Host      *PlatformSpec
	Container *PlatformSpec
}

var AutoPlatform = &Platform{}

func NewPlatform(os, arch, variant string) *Platform {
	return &Platform{
		Host: &PlatformSpec{
			OS:           os,
			Architecture: arch,
			Variant:      variant,
		},
		Container: &PlatformSpec{
			OS:           os,
			Architecture: arch,
			Variant:      variant,
		},
	}
}

func (p *Platform) Same() bool {
	return p.Host.OS == p.Container.OS && p.Host.Architecture == p.Container.Architecture
}

func GetContainerPlatform(host *PlatformSpec) *PlatformSpec {
	//TODO: convert darwin/arm64 to linux/arm64 and darwin/amd64 to linux/amd64
	switch host.OS + "/" + host.Architecture {
	case "darwin/arm64", "darwin/amd64": // MacOS M1/M2
		slog.Info("Convert MacOS M1/M2 platform to linux/amd64 for container")
		return &PlatformSpec{
			OS:           "linux",
			Architecture: "amd64",
		}
	default:
		slog.Info("Use host platform for container")
		return host
	}

	// switch host.OS {
	// case "darwin": // MacOS M1/M2
	// 	slog.Info("Convert MacOS M1/M2 platform to linux for container")
	// 	return &PlatformSpec{
	// 		OS:           "linux",
	// 		Architecture: host.Architecture,
	// 	}
	// default:
	// 	slog.Info("Use host platform for container")
	// 	return host
	// }
}

func GetPlatformSpec() *Platform {
	// Get the host OS and architecture
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH

	// var platform *Platform

	platform := &Platform{
		Host: &PlatformSpec{
			OS:           hostOS,
			Architecture: hostArch,
		},
	}

	platform.Container = GetContainerPlatform(platform.Host)
	return platform
}

func ParsePlatform(platform string) *PlatformSpec {
	if platform == "" {
		return nil
	}
	parts := strings.Split(platform, "/")
	if len(parts) != 2 {
		return nil
	}

	return &PlatformSpec{
		OS:           parts[0],
		Architecture: parts[1],
	}
}

func GetImagePlatform(host *PlatformSpec) *PlatformSpec {
	switch host.OS {
	case "darwin": // MacOS M1/M2
		slog.Info("Convert MacOS M1/M2 platform to linux for container")
		return &PlatformSpec{
			OS:           "linux",
			Architecture: host.Architecture,
		}
	default:
		slog.Info("Use host platform for container")
		return host
	}
}

func GetContainerPlatform2(host *PlatformSpec) *PlatformSpec {
	//TODO: convert darwin/arm64 to linux/arm64 and darwin/amd64 to linux/amd64
	switch host.OS + "/" + host.Architecture {
	case "darwin/arm64": // MacOS M1/M2
		slog.Info("Convert MacOS M1/M2 platform to linux/amd64 for container")
		return &PlatformSpec{
			OS:           "linux",
			Architecture: "amd64",
		}
	case "darwin/amd64": // MacOS M1/M2
		slog.Info("Convert MacOS M1/M2 platform to linux/amd64 for container")
		return &PlatformSpec{
			OS:           "linux",
			Architecture: "arm64",
		}
	default:
		slog.Info("Use host platform for container")
		return host
	}
}

func GetPlatforms(platform Platform) []string {
	//TODO: Get target platform for build for darwin arm64 it should be linux/arm64
	// and for darwin amd64 it should be linux/amd64
	cPlatform := GetImagePlatform(platform.Host)
	platform.Container = cPlatform
	platforms := []string{cPlatform.String()}
	if !platform.Same() {
		platforms = append(platforms, GetContainerPlatform2(platform.Host).String())
	}
	return platforms
}
