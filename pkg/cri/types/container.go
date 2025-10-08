package types

import "time"

type Container struct {
	ID      string
	Image   string
	ImageID string
	Names   []string
}

type PortBinding struct {
	IP   string
	Port string
}

func (p PortBinding) String() string {
	if p.IP != "" {
		return p.IP + ":" + p.Port
	}
	return p.Port
}

type Binding struct {
	Host      PortBinding
	Container PortBinding
}

type ReadinessProbe struct {
	Validate func([]byte) bool
	Endpoint string
	Timeout  time.Duration
}

type ContainerConfig struct {
	ExposedPorts []Binding
	Volumes      []Volume // List of volumes (mounts) used for the container
	Platform     *Platform
	Readiness    *ReadinessProbe `json:"-"`
	Image        string          // Name of the image as it was passed by the operator (e.g., could be symbolic)
	Name         string
	Script       string
	User         string // User that will run the command(s) inside the container, also supports user:group
	WorkingDir   string // Current directory (PWD) in which the command will be launched
	Cmd          []string
	Entrypoint   []string
	Env          []string // List of environment variable to set in the container
	Memory       int64
	CPU          uint64
	Tty          bool // Attach standard streams to a tty, including stdin if it is not closed

	Secrets map[string]string
}

// type ContainerConfig struct {
// 	Platform     *Platform
// 	Readiness    *ReadinessProbe `json:"-"`
// 	Tty          bool            // Attach standard streams to a tty, including stdin if it is not closed.
// 	Memory       int64
// 	CPU          uint64
// 	User         string          // User that will run the command(s) inside the container, also support user:group
// 	WorkingDir   string          // Current directory (PWD) in the command will be launched
// 	Image        string // Name of the image as it was passed by the operator (e.g. could be symbolic)
// 	Name         string
// 	Script       string
// 	Cmd          []string
// 	Entrypoint   []string
// 	Env          []string // List of environment variable to set in the container
// 	Volumes      []Volume        // List of volumes (mounts) used for the container
// 	ExposedPorts []Binding
// }

type ImageInfo struct {
	Platform *PlatformSpec
	ID       string
}
