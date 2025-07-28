package podman

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/registry"

	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/api/handlers"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/bindings/manifests"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/docker/docker/api/types/container"
	spec "github.com/opencontainers/runtime-spec/specs-go"

	buildahDefine "github.com/containers/buildah/define"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
)

// PodmanManager is a struct that implements the ContainerManager interface
type PodmanManager struct {
	conn context.Context
}

func ToMount(v *types.Volume) spec.Mount {
	return spec.Mount{
		Type:        v.Type,
		Source:      v.Source,
		Destination: v.Target,
		Options:     v.Options,
	}
	// return mount.Mount{
	// 	Type:   mount.Type(v.Type),
	// 	Source: v.Source,
	// 	Target: v.Target,
	// }
}

func ToMounts(volumes []types.Volume) []spec.Mount {
	var mounts []spec.Mount
	for _, v := range volumes {
		mounts = append(mounts, ToMount(&v))
	}
	return mounts
}

// NewPodmanManager returns a new PodmanManager
func NewPodmanManager() (*PodmanManager, error) {
	// Check if podman is available
	if _, err := exec.LookPath("podman"); err != nil {
		return nil, fmt.Errorf("podman not found in PATH: %w", err)
	}

	output, err := exec.Command("podman", "version", "-f", "{{.Version}}").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get podman version: %w", err)
	}

	var podmanSocket string

	if strings.HasPrefix(strings.TrimSpace(string(output)), "3.") ||
		strings.HasPrefix(strings.TrimSpace(string(output)), "4.") {
		cmd, err := exec.Command("podman", "info", "-f", "{{ .Host.RemoteSocket.Path }}").Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get podman socket info: %w", err)
		}
		podmanSocket = strings.TrimSpace(string(cmd))
	} else {
		cmd, err := exec.Command("podman", "machine", "inspect", "--format", "{{ .ConnectionInfo.PodmanSocket.Path }}").Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get podman machine socket: %w", err)
		}
		podmanSocket = strings.TrimSpace(string(cmd))
	}

	conn, err := bindings.NewConnection(context.Background(), "unix://"+podmanSocket)
	if err != nil {
		return nil, err
	}
	return &PodmanManager{conn: conn}, nil
}

func (d *PodmanManager) Name() string {
	return "podman"
}

// CreateContainer creates a container
func (p *PodmanManager) CreateContainer(ctx context.Context, opts *types.ContainerConfig, authBase64 string) (string, error) {
	s := specgen.NewSpecGenerator(opts.Image, false)

	wahr := true
	s.Command = opts.Cmd
	s.Entrypoint = opts.Entrypoint
	s.Image = opts.Image
	s.Privileged = &wahr
	limits := &spec.LinuxResources{}
	if opts.Memory != 0 {
		limits.Memory = &spec.LinuxMemory{
			Limit: &opts.Memory,
		}
	}
	if opts.CPU != 0 {
		limits.CPU = &spec.LinuxCPU{
			Shares: &opts.CPU,
		}
	}
	s.ResourceLimits = limits
	// s.ApparmorProfile = "unconfined"
	// s.SeccompPolicy = "unconfined"
	if opts.Env != nil {
		envs := map[string]string{}
		for _, e := range opts.Env {
			s := strings.SplitN(e, "=", 2)
			if len(s) != 2 {
				return "", fmt.Errorf("invalid env format: %s", e)
			}
			k, v := s[0], s[1]
			envs[k] = v
		}
		s.Env = envs
	}
	s.Name = opts.Name
	s.User = opts.User
	s.Terminal = &wahr
	if opts.WorkingDir != "" {
		s.WorkDir = opts.WorkingDir
	}
	if opts.Volumes == nil && opts.WorkingDir != "" {
		s.Volumes = []*specgen.NamedVolume{
			{
				Name: "wd",
				Dest: opts.WorkingDir,
			},
		}
	}
	if opts.Platform != nil {
		s.ImageOS = opts.Platform.Container.OS
		s.ImageArch = opts.Platform.Container.Architecture

		//This ensure that the requested platform is pulled before creating the container.
		//Otherwise the container creation fail with does not match the specified platform error.
		_, err := p.PullImage(ctx, opts.Image, authBase64, opts.Platform.Container.String())
		if err != nil {
			return "", err
		}
	}
	s.Mounts = ToMounts(opts.Volumes)
	s.ContainerSecurityConfig = specgen.ContainerSecurityConfig{
		Privileged: &wahr,
		// CapAdd:     []string{"ALL"},
	}
	// TODO add port mapping
	for _, p := range opts.ExposedPorts {
		hport, err := strconv.Atoi(p.Host.Port)
		if err != nil {
			slog.Error("Failed to convert host port to int", "error", err)
			return "", err
		}
		cport, err := strconv.Atoi(p.Container.Port)
		if err != nil {
			slog.Error("Failed to convert container port to int", "error", err)
			return "", err
		}

		s.PortMappings = append(s.PortMappings, nettypes.PortMapping{
			Range:         1,
			Protocol:      "tcp",
			HostPort:      uint16(hport),
			ContainerPort: uint16(cport),
		})
	}

	createResponse, err := containers.CreateWithSpec(p.conn, s, &containers.CreateOptions{})
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		return "", err
	}
	return createResponse.ID, nil
}

// StartContainer starts a container
func (p *PodmanManager) StartContainer(ctx context.Context, id string) error {
	err := containers.Start(p.conn, id, nil)
	if err != nil {
		return err
	}

	return nil
}

// StopContainer stops a container
func (p *PodmanManager) StopContainer(ctx context.Context, id string, signal string) error {
	return containers.Stop(p.conn, id, &containers.StopOptions{})
}

// CommitContainer commits a container
func (p *PodmanManager) CommitContainer(ctx context.Context, containerID string, opts types.CommitOptions) (string, error) {
	res, err := containers.Commit(p.conn, containerID, &containers.CommitOptions{
		// Comment: &opts.Comment,
		Changes: opts.Changes,
		Tag:     &opts.Reference,
	})
	if err != nil {
		return "", err
	}
	return res.ID, nil
}

// RemoveContainer removes a container
func (p *PodmanManager) RemoveContainer(ctx context.Context, containerID string) error {
	_, err := containers.Remove(p.conn, containerID, &containers.RemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}

// ContainerList lists containers
func (p *PodmanManager) ContainerList(ctx context.Context, all bool) ([]*types.Container, error) {
	cons, err := containers.List(p.conn, &containers.ListOptions{
		All: &all,
	})
	if err != nil {
		return nil, err
	}
	var containers []*types.Container
	for _, c := range cons {
		containers = append(containers, &types.Container{
			ID:      c.ID,
			Image:   c.Image,
			Names:   c.Names,
			ImageID: c.ImageID,
		})
	}
	return containers, nil
}

// ContainerLogs gets container logs
func (p *PodmanManager) ContainerLogs(ctx context.Context, id string, ShowStdout bool, ShowStderr bool, Follow bool) (io.ReadCloser, error) {
	dataCh := make(chan string, 1)

	go func() {
		err := containers.Logs(p.conn, id, &containers.LogOptions{
			Follow: &Follow,
			Stdout: &ShowStdout,
			Stderr: &ShowStderr,
		}, dataCh, dataCh)
		if err != nil {
			slog.Error("Failed to get container logs", "error", err)
			dataCh <- fmt.Errorf("containifyci: failed to get container logs: %v", err).Error()
			os.Exit(1)
		}
	}()
	return utils.NewChannelReadCloser(dataCh), nil
}

// CopyContentToContainer copies content to a container
func (p *PodmanManager) CopyContentToContainer(ctx context.Context, id, content, dest string) error {
	// Create a tar archive from the string content
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	header := &tar.Header{
		Name: filepath.Base(dest),
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tw.Write([]byte(content)); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	// Copy the tar archive into the container
	r := bytes.NewReader(buf.Bytes())
	fnc, err := containers.CopyFromArchive(p.conn, id, filepath.Dir(dest), r)
	if err != nil {
		return fmt.Errorf("failed to copy to container: %v", err)
	}
	err = fnc()
	if err != nil {
		return fmt.Errorf("failed to copy to container: %v", err)
	}

	fmt.Printf("Copied string content to %s in container %s\n", dest, id)
	return nil
}

// CopyDirectorToContainer copies a directory to a container
func (p *PodmanManager) CopyDirectorToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	buf, err := tarDir(srcPath)
	if err != nil {
		return err
	}
	fnc, err := containers.CopyFromArchive(p.conn, id, dstPath, buf)
	if err != nil {
		return fmt.Errorf("failed to copy to container: %v", err)
	}
	err = fnc()
	if err != nil {
		return fmt.Errorf("failed to copy to container: %v", err)
	}

	fmt.Printf("Copied string content to %s in container %s\n", dstPath, id)
	return nil
}

func tarDir(srcPath string) (*bytes.Buffer, error) {
	// Create a buffer to write our archive to
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	// Walk the directory and write each file to the tar writer
	err := filepath.Walk(srcPath, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create a tar header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		if srcPath == "." ||
			srcPath == "./" {
			srcPath = ""
		}
		// Ensure the header has the correct name
		header.Name = filepath.ToSlash(file[len(srcPath):])

		// Write the header to the tar writer
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If the file is not a directory, write the file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()

			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		slog.Error("Error walking the directory", "error", err)
		return nil, err
	}

	// Close the tar writer
	if err := tw.Close(); err != nil {
		slog.Error("Error closing the tar writer", "error", err)
		return nil, err
	}
	return buf, nil
}

// CopyToContainer copies to a container
func (p *PodmanManager) CopyToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	buf, err := tarFile(srcPath, dstPath)
	if err != nil {
		slog.Error("Failed to create tar archive", "error", err)
		return err
	}
	fnc, err := containers.CopyFromArchive(p.conn, id, "/", buf)
	if err != nil {
		return fmt.Errorf("failed to copy to container: %v", err)
	}
	err = fnc()
	if err != nil {
		return fmt.Errorf("failed to copy to container: %v", err)
	}

	fmt.Printf("Copied string content to %s in container %s\n", dstPath, id)
	return nil
}

// CopyFileFromContainer copies a file from a container
func (p *PodmanManager) CopyFileFromContainer(ctx context.Context, id string, srcPath string) (string, error) {
	// Create a reader for the tar archive
	// reader, _, err := d.client.CopyFromContainer(ctx, id, srcPath)

	var buf bytes.Buffer

	fnc, err := containers.CopyToArchive(p.conn, id, srcPath, &buf)
	// reader, _, err := c.clientOld.CopyFromContainer(c.ctx, c.Resp.ID, srcPath)

	if err != nil && types.SameError(err, fmt.Errorf("no such file or directory")) {
		slog.Info("File not exists", "error", err, "file", srcPath)
		return "", io.EOF
	}

	if err != nil {
		slog.Error("Failed to copy from container", "error", err)
		os.Exit(1)
	}
	err = fnc()
	if err != nil {
		slog.Error("Failed to copy from container", "error", err)
		os.Exit(1)
	}

	// Extract the tar archive
	tarReader := tar.NewReader(&buf)
	header, err := tarReader.Next()
	if err == io.EOF {
		slog.Error("File not found in container", "file", srcPath)
		return "", err
	}
	if err != nil {
		slog.Error("Failed to read tar archive", "error", err)
		os.Exit(1)
	}

	// Check if the header corresponds to a file
	if header.Typeflag != tar.TypeReg {
		slog.Error("Expected file but found type", "type", header.Typeflag)
		os.Exit(1)
	}

	// Read file content into a buffer
	var out bytes.Buffer
	_, err = io.Copy(&out, tarReader)
	if err != nil {
		slog.Error("Failed to read file content", "error", err)
		os.Exit(1)
	}

	return out.String(), nil
}

// tarFile creates a tar archive containing the specified file
func tarFile(srcPath, destPath string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	file, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return nil, err
	}
	header.Name = destPath

	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}

	if _, err := io.Copy(tw, file); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

// ExecContainer executes a container
func (p *PodmanManager) ExecContainer(ctx context.Context, id string, cmd []string, attachStdOut bool) (io.Reader, error) {
	id, err := containers.ExecCreate(p.conn, id, &handlers.ExecCreateConfig{
		ExecConfig: container.ExecOptions{
			Cmd:          cmd,
			AttachStdout: attachStdOut,
		},
	})
	if err != nil {
		return nil, err
	}
	// err = containers.ExecStart(p.conn, id, &containers.ExecStartOptions{})
	// if err != nil {
	// 	return nil, err
	// }

	var buf bytes.Buffer
	writer := io.Writer(&buf)

	err = containers.ExecStartAndAttach(p.conn, id, &containers.ExecStartAndAttachOptions{
		OutputStream: &writer,
		AttachOutput: &attachStdOut,
	})
	if err != nil {
		return nil, err
	}
	return &buf, nil
}

// InspectContainer inspects a container
func (p *PodmanManager) InspectContainer(ctx context.Context, id string) (*types.ContainerConfig, error) {
	meta, err := containers.Inspect(p.conn, id, &containers.InspectOptions{})
	if err != nil {
		return nil, err
	}

	volumes := []types.Volume{}
	for _, v := range meta.Mounts {
		volumes = append(volumes, types.Volume{
			Type:   v.Type,
			Source: v.Source,
			Target: v.Destination,
		})
	}

	ports := []types.Binding{}
	for _, p := range meta.NetworkSettings.Ports {
		for _, p := range p {
			ports = append(ports, types.Binding{
				Host: types.PortBinding{
					IP:   p.HostIP,
					Port: p.HostPort,
				},
				// TODO how to get container port mapping infos
				// Container: types.PortBinding{
				// 	IP:   p.ContainerIP,
				// 	Port: p.ContainerPort,
				// },
			})
		}
	}

	img, err := images.GetImage(p.conn, meta.Image, &images.GetOptions{})
	if err != nil {
		return nil, err
	}
	platform := types.GetPlatformSpec()
	platform.Container = &types.PlatformSpec{
		OS:           img.Os,
		Architecture: img.Architecture,
	}

	return &types.ContainerConfig{
		Cmd:          meta.Config.Cmd,
		Entrypoint:   meta.Config.Entrypoint,
		Env:          meta.Config.Env,
		ExposedPorts: ports,
		Image:        meta.Image,
		Name:         meta.Name,
		Platform:     platform,
		Tty:          meta.Config.Tty,
		User:         meta.Config.User,
		WorkingDir:   meta.Config.WorkingDir,
		Volumes:      volumes,
	}, nil
}

// WaitContainer waits for a container
func (p *PodmanManager) WaitContainer(ctx context.Context, id string, waitCondition string) (*int64, error) {
	res, err := containers.Wait(p.conn, id, &containers.WaitOptions{
		// TODO convert waitCondition to podman wait condition
		Condition: []define.ContainerStatus{
			define.ContainerStateStopped,
			define.ContainerStateExited,
		},
	})
	if err != nil {
		return nil, err
	}
	res64 := int64(res)
	return &res64, nil
}

// BuildImage builds an image
func (p *PodmanManager) BuildImage(ctx context.Context, dockerfile []byte, imageName string, platform string) (io.ReadCloser, error) {
	// Create a temporary directory for the Dockerfile
	dir, err := os.MkdirTemp("", "podman-build")
	if err != nil {
		slog.Error("Error creating temp directory", "error", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir) // Clean up

	// Write the Dockerfile
	dockerfilePath := dir + "/Dockerfile"
	file, err := os.Create(dockerfilePath)
	if err != nil {
		slog.Error("Error creating Dockerfile", "error", err)
		os.Exit(1)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.Write(dockerfile)
	if err != nil {
		slog.Error("Error writing Dockerfile", "error", err)
		os.Exit(1)
	}
	writer.Flush()

	var buf bytes.Buffer

	opts := buildahDefine.BuildOptions{
		Output: imageName,
		// TODO set platform
		// Platforms: ,
		Log: func(format string, args ...interface{}) {
			buf.WriteString(fmt.Sprintf(format, args...))
		},
	}

	platformSpec := types.ParsePlatform(platform)
	if platformSpec != nil {
		opts.Architecture = platformSpec.Architecture
		opts.OS = platformSpec.OS
	}

	_, err = images.Build(p.conn, []string{file.Name()}, images.BuildOptions{
		BuildOptions: opts,
	})
	if err != nil {
		slog.Error("Error building image", "error", err)
		os.Exit(1)
	}
	return utils.NewReadCloser(&buf), nil
}

// BuildImage builds an image
func (p *PodmanManager) BuildMultiArchImage(ctx context.Context, dockerfile []byte, dockerCtx *bytes.Buffer, imageName string, platforms []string, _ string) (io.ReadCloser, []string, error) {
	imageIDs := []struct {
		Platform *types.PlatformSpec
		ID       *string
		Image    string
	}{}
	// Create a temporary directory for the Dockerfile
	dir, err := os.MkdirTemp("", "podman-build")
	if err != nil {
		slog.Error("Error creating temp directory", "error", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir) // Clean up

	if dockerCtx != nil {
		// Extract the tar archive
		err := utils.ExtractTar(dockerCtx, dir)
		if err != nil {
			slog.Error("Error extracting tar archive", "error", err)
			os.Exit(1)
		}
	}

	// Write the Dockerfile
	dockerfilePath := dir + "/Dockerfile"
	file, err := os.Create(dockerfilePath)
	if err != nil {
		slog.Error("Error creating Dockerfile", "error", err)
		os.Exit(1)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, err = writer.Write(dockerfile)
	if err != nil {
		slog.Error("Error writing Dockerfile", "error", err)
		os.Exit(1)
	}
	writer.Flush()

	var buf bytes.Buffer

	info, err := utils.ParseDockerImage(imageName)
	if err != nil {
		slog.Error("Failed to parse image", "error", err)
		os.Exit(1)
	}

	opts := buildahDefine.BuildOptions{
		Log: func(format string, args ...interface{}) {
			buf.WriteString(fmt.Sprintf(format, args...))
		},
		PullPolicy:       buildahDefine.PullAlways,
		Out:              os.Stdout,
		ContextDirectory: dir,
	}

	if len(platforms) > 0 {
		// Build image for each platform separately because only the last image is properly tagged with the image name
		//and only one image id is returned by the images.Build function
		for _, plt := range platforms {
			platformSpec := types.ParsePlatform(plt)
			opts.Platforms = []struct {
				OS      string
				Arch    string
				Variant string
			}{
				{
					OS:   platformSpec.OS,
					Arch: platformSpec.Architecture,
				},
			}
			tmpImageName := fmt.Sprintf("%s/%s-%s:%s", info.Registry, info.Image, platformSpec.Architecture, info.Tag)
			opts.Output = tmpImageName
			res, err := images.Build(p.conn, []string{file.Name()}, images.BuildOptions{
				BuildOptions: opts,
			})
			if err != nil {
				slog.Error("Error building image", "error", err)
				os.Exit(1)
			}
			imageIDs = append(imageIDs, struct {
				Platform *types.PlatformSpec
				ID       *string
				Image    string
			}{
				ID:       &res.ID,
				Image:    tmpImageName,
				Platform: platformSpec,
			})
		}
	} else {
		opts.Output = imageName
		res, err := images.Build(p.conn, []string{file.Name()}, images.BuildOptions{
			BuildOptions: opts,
		})
		if err != nil {
			slog.Error("Error building image", "error", err)
			os.Exit(1)
		}
		imageIDs = append(imageIDs, struct {
			Platform *types.PlatformSpec
			ID       *string
			Image    string
		}{
			ID:    &res.ID,
			Image: imageName,
		})
	}

	var imageIDsStr []string
	for _, id := range imageIDs {
		imageIDsStr = append(imageIDsStr, *id.ID)
	}
	wahr := true
	mfst, err := manifests.Create(p.conn, imageName, nil, &manifests.CreateOptions{
		Amend: &wahr,
	})
	if err != nil {
		slog.Error("Error creating manifest", "error", err)
		os.Exit(1)
	}
	fmt.Println("Manifest ID: ", mfst)
	for _, img := range imageIDs {
		opts := &manifests.AddOptions{
			Images: []string{*img.ID},
		}
		id, err := manifests.Add(p.conn, mfst, opts)
		if err != nil {
			slog.Error("Error adding manifest artifact", "error", err)
			os.Exit(1)
		}
		fmt.Println("Manifest Artifact ID: ", id)
	}

	var progressWriter io.Writer = &buf

	_, err = manifests.Push(p.conn, mfst, imageName, &images.PushOptions{
		All:            &wahr,
		ProgressWriter: &progressWriter,
	})
	if err != nil {
		slog.Error("Error pushing manifest", "error", err)
		os.Exit(1)
	}

	// TODO add proper buffer and reader handling
	return utils.NewReadCloser(&buf), imageIDsStr, nil
}

// ListImage lists images
func (p *PodmanManager) ListImage(ctx context.Context, image string) ([]string, error) {
	imgs, err := images.List(p.conn, &images.ListOptions{
		Filters: map[string][]string{
			"reference": {image},
		},
	})
	if err != nil {
		return nil, err
	}
	var images []string
	for _, i := range imgs {
		images = append(images, i.Names...)
	}
	return images, nil
}

// CustomReadCloser wraps a bytes.Reader to implement io.ReadCloser.
type CustomReadCloser struct {
	*bytes.Reader
}

// Close is a no-op method to satisfy the io.ReadCloser interface.
func (CustomReadCloser) Close() error {
	return nil
}

// PullImage pulls an image
func (p *PodmanManager) PullImage(ctx context.Context, image string, authBase64 string, platform string) (io.ReadCloser, error) {
	authCfg, err := DecodeRegistryAuth(authBase64)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	unwahr := false
	var progressWriter io.Writer = &buf

	platformSpec := types.ParsePlatform(platform)

	opts := images.PullOptions{
		Username:       &authCfg.Username,
		Password:       &authCfg.Password,
		ProgressWriter: &progressWriter,
		Quiet:          &unwahr,
	}
	if platformSpec != nil {
		opts.Arch = &platformSpec.Architecture
		opts.OS = &platformSpec.OS
	}

	// TODO progress writer is not working
	_, err = images.Pull(p.conn, image, &opts)
	if err != nil {
		return nil, err
	}
	return CustomReadCloser{bytes.NewReader(buf.Bytes())}, nil
}

// TagImage tags an image
func (p *PodmanManager) TagImage(ctx context.Context, source, target string) error {
	info, err := utils.ParseDockerImage(target)
	if err != nil {
		slog.Error("Failed to parse image", "error", err)
		return err
	}
	// source = "localhost/" + info.Image + ":" + info.Tag
	target = info.Tag
	repo := "localhost/" + info.Image
	if info.Registry != "" {
		repo = info.Registry + "/" + info.Image
	}
	return images.Tag(p.conn, source, target, repo, &images.TagOptions{})
}

func DecodeRegistryAuth(authBase64 string) (*registry.AuthConfig, error) {
	base64Decoded, err := base64.StdEncoding.DecodeString(authBase64)
	if err != nil {
		slog.Error("Failed to decode base64", "error", err)
		return nil, err
	}

	var authCfg registry.AuthConfig

	if len(base64Decoded) <= 0 {
		return &authCfg, nil
	}

	err = json.Unmarshal(base64Decoded, &authCfg)
	if err != nil {
		// Mask sensitive auth data in error logs for security
		slog.Error("Failed to unmarshal auth config", "error", err, "auth_length", len(base64Decoded))
		return nil, err
	}

	return &authCfg, nil
}

// PushImage pushes an image
func (p *PodmanManager) PushImage(ctx context.Context, target string, authBase64 string) (io.ReadCloser, error) {

	authCfg, err := DecodeRegistryAuth(authBase64)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	var progressWriter io.Writer = buf
	err = images.Push(p.conn, target, target, &images.PushOptions{
		Username:       &authCfg.Username,
		Password:       &authCfg.Password,
		ProgressWriter: &progressWriter,
	})
	if err != nil {
		return nil, err
	}
	return CustomReadCloser{bytes.NewReader(buf.Bytes())}, nil
}

// RemoveImage removes an image
func (p *PodmanManager) RemoveImage(ctx context.Context, target string) error {
	_, errs := images.Remove(p.conn, []string{target}, &images.RemoveOptions{})
	if errs != nil {
		return errors.Join(errs...)
	}
	return nil
}

func (p *PodmanManager) InspectImage(ctx context.Context, image string) (*types.ImageInfo, error) {
	info, err := images.GetImage(p.conn, image, &images.GetOptions{})
	if err != nil {
		return nil, err
	}
	return &types.ImageInfo{
		ID: info.ID,
		Platform: &types.PlatformSpec{
			OS:           info.Os,
			Architecture: info.Architecture,
		},
	}, nil
}
