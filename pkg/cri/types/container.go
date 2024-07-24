package types

import "time"

type Container struct {
	ID      string
	Names   []string
	Image   string
	ImageID string
}

type PortBinding struct {
	IP   string
	Port string
}

type Binding struct {
	Host      PortBinding
	Container PortBinding
}

type ReadinessProbe struct {
	Endpoint string
	Timeout  time.Duration
	Validate func([]byte) bool
}

type ContainerConfig struct {
	Cmd          []string
	Entrypoint   []string
	Env          []string // List of environment variable to set in the container
	ExposedPorts []Binding
	Image        string // Name of the image as it was passed by the operator (e.g. could be symbolic)
	Name         string
	Platform     *Platform
	Readiness    *ReadinessProbe `json:"-"`
	Tty          bool            // Attach standard streams to a tty, including stdin if it is not closed.
	User         string          // User that will run the command(s) inside the container, also support user:group
	Volumes      []Volume        // List of volumes (mounts) used for the container
	WorkingDir   string          // Current directory (PWD) in the command will be launched
	Memory       int64
	CPU          uint64
	Script       string
}

type ImageInfo struct {
	ID       string
	Platform *PlatformSpec
}
