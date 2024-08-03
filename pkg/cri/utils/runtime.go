package utils

const (
	Docker  RuntimeType = "docker"
	Podman  RuntimeType = "podman"
	Test    RuntimeType = "test"
	Unknown RuntimeType = "unknown"
)

type RuntimeType string
