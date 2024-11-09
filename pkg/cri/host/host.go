package host

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

type HostManager struct {
	containers map[string]*HostContainer

	mu sync.RWMutex
}

type HostContainer struct {
	opts     *types.ContainerConfig
	stdout   io.ReadCloser
	exitCode *int64
	wg       sync.WaitGroup
}

func generateRandomString(length int) (string, error) {
	byteLength := (length*6 + 7) / 8 // Each base64 character represents 6 bits
	randomBytes := make([]byte, byteLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	randomString := base64.URLEncoding.EncodeToString(randomBytes)
	return randomString[:length], nil
}

func (d *HostManager) Name() string {
	return "host"
}

func (d *HostManager) CreateContainer(ctx context.Context, opts *types.ContainerConfig, authBase64 string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	id, err := generateRandomString(12)
	if err != nil {
		return "", err
	}

	d.containers[id] = &HostContainer{
		opts: opts,
	}

	return id, nil
}

func (d *HostManager) StartContainer(ctx context.Context, id string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	c := d.containers[id]

	commands := c.opts.Cmd
	if len(commands) == 0 {
		commands = c.opts.Entrypoint
	}

	// if commands[0] == "sleep" {
	// 	commands = []string{"echo", "sleeping"}
	// }

	if len(commands) > 1 && commands[1] == "/tmp/script.sh" {
		commands[1] = fmt.Sprintf("/tmp/%s/script.sh", id)
	}

	cmd := exec.Command(commands[0], commands[1:]...)
	stdout := NewWriterToReadCloser()
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	c.stdout = stdout

	// Start the command asynchronously
	slog.Info("Running command", "command", commands)
	if err := cmd.Start(); err != nil {
		slog.Error("Failed to start command", "command", commands, "error", err)
		return err
	}

	c.wg.Add(1)

	// Run command in a goroutin and handle exit code
	go func() {
		defer c.wg.Done()
		err := cmd.Wait()
		slog.Debug("Command finished", "command", commands, "error", err)

		// Set exit code based on command outcome
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				code := int64(exitError.ExitCode())
				c.exitCode = &code
				slog.Error("Error running command", "error", err, "command", commands)
			} else {
				slog.Error("Error running command", "error", err, "command", commands)
			}
		} else {
			code := int64(0)
			c.exitCode = &code
		}
	}()

	return nil
}

func (d *HostManager) WaitContainer(ctx context.Context, id string, waitCondition string) (*int64, error) {
	d.mu.RLock()
	c := d.containers[id]
	d.mu.RUnlock()

	c.wg.Wait()

	return c.exitCode, nil
}

func (d *HostManager) ContainerLogs(ctx context.Context, id string, ShowStdout bool, ShowStderr bool, Follow bool) (io.ReadCloser, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	c := d.containers[id]

	return io.NopCloser(c.stdout), nil
}

func (d *HostManager) CopyContentToContainer(ctx context.Context, id, content, dest string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	c := d.containers[id]
	if dest == "/tmp/script.sh" {
		err := os.MkdirAll("/tmp/"+id, 0755)
		if err != nil {
			slog.Error("Failed to create directory", "error", err)
			os.Exit(1)
		}
		dest = fmt.Sprintf("/tmp/%s/script.sh", id)
		content := strings.ReplaceAll(content, c.opts.WorkingDir+"/", "")
		return os.WriteFile(dest, []byte(content), 0755)
	}
	os.Exit(1)
	return nil
}

func (d *HostManager) ExecContainer(ctx context.Context, id string, cmd []string, attachStdOut bool) (io.Reader, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (d *HostManager) InspectContainer(ctx context.Context, id string) (*types.ContainerConfig, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	opts := d.containers[id].opts
	return opts, nil
}

func (d *HostManager) BuildImage(ctx context.Context, dockerfile []byte, imageName string, platform string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (d *HostManager) PullImage(ctx context.Context, image string, authBase64 string, platform string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (d *HostManager) PushImage(ctx context.Context, target string, authBase64 string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func NewHostManager() *HostManager {
	return &HostManager{
		containers: make(map[string]*HostContainer),
	}
}

//Dummy implementations

func (d *HostManager) StopContainer(ctx context.Context, id string, signal string) error {
	return nil
}

func (d *HostManager) CommitContainer(ctx context.Context, containerID string, opts types.CommitOptions) (string, error) {
	return "", nil
}

func (d *HostManager) RemoveContainer(ctx context.Context, containerID string) error {
	return nil
}

func (d *HostManager) ContainerList(ctx context.Context, all bool) ([]*types.Container, error) {
	return nil, nil
}

func (d *HostManager) BuildMultiArchImage(ctx context.Context, dockerfile []byte, dockerCtx *bytes.Buffer, imageName string, platforms []string, authBase64 string) (io.ReadCloser, []string, error) {
	return nil, nil, nil
}

func (d *HostManager) ListImage(ctx context.Context, image string) ([]string, error) {
	return nil, nil
}

func (d *HostManager) TagImage(ctx context.Context, source, target string) error {
	return nil
}

func (d *HostManager) RemoveImage(ctx context.Context, target string) error {
	return nil
}

func (d *HostManager) InspectImage(ctx context.Context, image string) (*types.ImageInfo, error) {
	return nil, nil
}

func (d *HostManager) CopyDirectorToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	return nil
}

func (d *HostManager) CopyToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	return nil
}

func (d *HostManager) CopyFileFromContainer(ctx context.Context, id string, srcPath string) (string, error) {
	return "", nil
}
