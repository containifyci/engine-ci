package utils

const (
	Docker  RuntimeType = "docker"
	Podman  RuntimeType = "podman"
	Test    RuntimeType = "test"
	Host    RuntimeType = "host"
	Unknown RuntimeType = "unknown"
)

type RuntimeType string
