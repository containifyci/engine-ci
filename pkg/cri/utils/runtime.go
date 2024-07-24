package utils

const (
	Docker  RuntimeType = "docker"
	Podman  RuntimeType = "podman"
	Unknown RuntimeType = "unknown"
)

type RuntimeType string
