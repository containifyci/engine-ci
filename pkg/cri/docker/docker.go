package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	dockertypes "github.com/docker/docker/api/types"
)

type DockerManager struct {
	client *client.Client
}

func ToMount(v *types.Volume) mount.Mount {
	return mount.Mount{
		Type:   mount.Type(v.Type),
		Source: v.Source,
		Target: v.Target,
	}
}

func ToMounts(volumes []types.Volume) []mount.Mount {
	var mounts []mount.Mount
	for _, v := range volumes {
		mounts = append(mounts, ToMount(&v))
	}
	return mounts
}

func NewDockerManager() (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerManager{client: cli}, nil
}

func (d *DockerManager) Name() string {
	return "docker"
}

func (d *DockerManager) CreateContainer(ctx context.Context, opts *types.ContainerConfig, authBase64 string) (string, error) {
	config := &container.Config{
		Cmd:        opts.Cmd,
		Env:        opts.Env,
		Entrypoint: opts.Entrypoint,
		Image:      opts.Image,
		User:       opts.User,
		Tty:        true,
		WorkingDir: opts.WorkingDir,
	}

	portSet := nat.PortSet{
		// "9000/tcp": struct{}{},
	}
	config.ExposedPorts = portSet

	portMap := nat.PortMap{
		// "9000/tcp": []nat.PortBinding{portBinding},
	}

	for _, p := range opts.ExposedPorts {
		portBinding := nat.PortBinding{
			HostIP:   p.Host.IP,
			HostPort: p.Host.Port,
		}
		// binding, set := p.ToPortBinding()
		portMap[nat.Port(p.Container.IP)] = []nat.PortBinding{portBinding}
		portSet[nat.Port(p.Container.Port)] = struct{}{}
	}
	config.ExposedPorts = portSet

	hostConfig := &container.HostConfig{
		Mounts:       ToMounts(opts.Volumes),
		PortBindings: portMap,
	}

	netConfig := &network.NetworkingConfig{}

	// netConfig := &network.NetworkingConfig{
	// 	EndpointsConfig: map[string]*network.EndpointSettings{
	// 		"network": {
	// 			NetworkID: "my-network",
	// 		},
	// 	},
	// }

	var platform *v1.Platform
	if opts.Platform != nil {
		platform = opts.Platform.Container.ToOrg()

		//This ensure that the requested platform is pulled before creating the container.
		//Otherwise the container creation fail with does not match the specified platform error.
		r, err := d.PullImage(ctx, opts.Image, authBase64, opts.Platform.Container.String())
		if err != nil {
			return "", err
		}
		defer r.Close()
		_, err = io.Copy(os.Stdout, r)
		if err != nil {
			return "", err
		}
	}

	containerResp, err := d.client.ContainerCreate(ctx, config, hostConfig, netConfig, platform, "")
	if err != nil {
		return "", err
	}

	containerID := containerResp.ID

	return containerID, nil
}

func (d *DockerManager) StartContainer(ctx context.Context, id string) error {
	if err := d.client.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}

func (d *DockerManager) StopContainer(ctx context.Context, id string, signal string) error {
	return d.client.ContainerStop(ctx, id, container.StopOptions{Signal: signal})
}

func (d *DockerManager) CommitContainer(ctx context.Context, containerID string, opts types.CommitOptions) (string, error) {
	commitResp, err := d.client.ContainerCommit(ctx, containerID, container.CommitOptions{
		Reference: opts.Reference,
		Comment:   opts.Comment,
		Changes:   opts.Changes,
	})
	if err != nil {
		return "", err
	}
	return commitResp.ID, nil
}

func (d *DockerManager) RemoveContainer(ctx context.Context, containerID string) error {
	return d.client.ContainerRemove(ctx, containerID, container.RemoveOptions{})
}

func (d *DockerManager) ContainerList(ctx context.Context, all bool) ([]*types.Container, error) {
	containers, err := d.client.ContainerList(ctx, container.ListOptions{
		All: all,
	})
	if err != nil {
		return nil, err
	}

	var containerList []*types.Container
	for _, container := range containers {
		containerList = append(containerList, &types.Container{
			ID:      container.ID,
			Names:   container.Names,
			Image:   container.Image,
			ImageID: container.ImageID,
		})
	}

	return containerList, nil
}

func (d *DockerManager) CopyContentToContainer(ctx context.Context, id, content, dest string) error {
	tarBuf := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuf)

	header := &tar.Header{
		Name: dest,
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

	return d.client.CopyToContainer(ctx, id, "/", tarBuf, container.CopyToContainerOptions{})
}

func (d *DockerManager) CopyToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	buf, err := tarFile(srcPath, dstPath)
	if err != nil {
		slog.Error("Failed to create tar archive", "error", err)
		return err
	}
	err = d.client.CopyToContainer(ctx, id, "/", buf, container.CopyToContainerOptions{})
	if err != nil {
		slog.Error("Failed to copy to container", "error", err)
		return err
	}
	return nil
}

func (d *DockerManager) CopyDirectorToContainer(ctx context.Context, id, srcPath, dstPath string) error {
	buf, err := tarDir(srcPath)
	if err != nil {
		slog.Error("Failed to create tar archive", "error", err)
		return err
	}
	err = d.client.CopyToContainer(ctx, id, dstPath, buf, container.CopyToContainerOptions{})
	if err != nil {
		slog.Error("Failed to copy to container", "error", err)
		return err
	}
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

func (d *DockerManager) ExecContainer(ctx context.Context, id string, cmd []string, attachStdOut bool) (io.Reader, error) {
	execResp, err := d.client.ContainerExecCreate(ctx, id, container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: attachStdOut,
	})
	if err != nil {
		return nil, err
	}

	resp, err := d.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{
		// Tty: attachStdOut,
	})
	if err != nil {
		return nil, err
	}

	return resp.Reader, nil
}

func (d *DockerManager) InspectContainer(ctx context.Context, id string) (*types.ContainerConfig, error) {
	container, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	imageInfo, _, err := d.client.ImageInspectWithRaw(ctx, container.Image)
	if err != nil {
		slog.Error("Failed to inspect image", "error", err, "imageId", container.Image)
		return nil, fmt.Errorf("error inspecting image: %w", err)
	}

	return &types.ContainerConfig{
		User:         container.Config.User,
		ExposedPorts: nil,
		Tty:          container.Config.Tty,
		Env:          container.Config.Env,
		Cmd:          container.Config.Cmd,
		Image:        container.Config.Image,
		Volumes:      nil,
		WorkingDir:   container.Config.WorkingDir,
		Entrypoint:   container.Config.Entrypoint,
		Platform:     types.NewPlatform(imageInfo.Os, imageInfo.Architecture, imageInfo.Variant),
		Name:         container.Name,
	}, nil
}

func (d *DockerManager) WaitContainer(ctx context.Context, id string, waitCondition string) (*int64, error) {
	statusCh, errCh := d.client.ContainerWait(ctx, id, container.WaitCondition(waitCondition))
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	case status := <-statusCh:
		return &status.StatusCode, nil
	}
	return nil, fmt.Errorf("failed to wait for container")
}

func (d *DockerManager) ListImage(ctx context.Context, imageName string) ([]string, error) {
	images, err := d.client.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{
			Key:   "reference",
			Value: imageName,
		}),
	})
	if err != nil {
		return nil, err
	}

	var imageList []string
	for _, img := range images {
		imageList = append(imageList, img.ID)
	}

	return imageList, nil
}

func (d *DockerManager) PullImage(ctx context.Context, imageName string, authBase64 string, platform string) (io.ReadCloser, error) {
	resp, err := d.client.ImagePull(ctx, imageName, image.PullOptions{
		Platform:     platform,
		RegistryAuth: authBase64,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (d *DockerManager) TagImage(ctx context.Context, source, target string) error {
	return d.client.ImageTag(ctx, source, target)
}

func (d *DockerManager) PushImage(ctx context.Context, target string, authBase64 string) (io.ReadCloser, error) {
	resp, err := d.client.ImagePush(ctx, target, image.PushOptions{
		RegistryAuth: authBase64,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (d *DockerManager) RemoveImage(ctx context.Context, target string) error {
	_, err := d.client.ImageRemove(ctx, target, image.RemoveOptions{})
	return err
}

// CopyFileFromContainer reads a single file from a container and returns its content as a string.
func (d *DockerManager) CopyFileFromContainer(ctx context.Context, id string, srcPath string) (string, error) {
	// Create a reader for the tar archive
	reader, _, err := d.client.CopyFromContainer(ctx, id, srcPath)

	// reader, _, err := c.clientOld.CopyFromContainer(c.ctx, c.Resp.ID, srcPath)

	if err != nil && types.SameError(err, fmt.Errorf("Error response from daemon: Could not find the file ")) {
		slog.Info("File not exists", "error", err, "file", srcPath)
		return "", io.EOF
	}

	if err != nil {
		slog.Error("Failed to copy from container", "error", err)
		os.Exit(1)
	}
	defer reader.Close()

	// Extract the tar archive
	tarReader := tar.NewReader(reader)
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
	var buf bytes.Buffer
	_, err = io.Copy(&buf, tarReader)
	if err != nil {
		slog.Error("Failed to read file content", "error", err)
		os.Exit(1)
	}

	return buf.String(), nil
}

// createTarArchive creates a tar archive containing the Dockerfile.
func createTarArchive(dockerfileContent []byte) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Add the Dockerfile to the tar archive
	header := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfileContent)),
	}

	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}

	if _, err := tw.Write(dockerfileContent); err != nil {
		return nil, err
	}

	return buf, nil
}

func (d *DockerManager) ensureBuilderExists(ctx context.Context, builderName string) error {
	cmd := exec.CommandContext(ctx, "docker", "buildx", "ls")
	cmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error listing buildx builders: %v\n%s", err, errBuf.String())
	}

	builders := outBuf.String()
	if strings.Contains(builders, builderName) {
		return nil // Builder already exists, no need to create it
	}

	cmd = exec.CommandContext(ctx, "docker", "buildx", "create", "--driver", "docker-container", "--name", builderName, "--use")
	cmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error creating buildx builder: %v\n%s", err, errBuf.String())
	}

	return nil
}

// BuildMultiArchImage builds a multi-architecture image using docker cli because the golang client doesn't support it yet
func (d *DockerManager) BuildMultiArchImage(ctx context.Context, dockerfile []byte, dockerCtx *bytes.Buffer, imageName string, platforms []string, authBase64 string) (io.ReadCloser, []string, error) {
// func (d *DockerManager) BuildMultiArchImage(ctx context.Context, dockerfile []byte, imageName string, platforms []string, authBase64 string) (io.ReadCloser, []string, error) {
	err := d.ensureBuilderExists(ctx, "containifyci-builder")
	if err != nil {
		slog.Error("Error ensuring builder exists", "error", err)
		os.Exit(1)
	}

	dir, err := os.MkdirTemp("", "docker-build")
	if err != nil {
		slog.Error("Error creating temp directory", "error", err)
		os.Exit(1)
	}
	// defer os.RemoveAll(dir) // Clean up

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

	// Construct the platform string
	platformStr := platforms[0]
	if len(platforms) > 0 {
		for _, p := range platforms[1:] {
			platformStr += "," + p
		}
	}

	command := []string{"docker", "buildx", "build", "--progress", "plain", "--push", "--provenance", "false", "--platform", platformStr, "-t", imageName, "-f", file.Name(), dir}
	fmt.Printf("Running command: %v\n", command)
	// Create the Docker buildx command
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)

	// Set the environment variable to enable Docker CLI experimental features
	// cmd.Env = append(os.Environ(), "DOCKER_CLI_EXPERIMENTAL=enabled")

	// Capture the output
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	// Run the build command
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Output: %s\n", outBuf.String())
		fmt.Printf("Error Output: %s\n", errBuf.String())
		return nil, nil, err
	}

	for _, p := range platforms {
		reader, err := d.PullImage(ctx, imageName, authBase64, p)
		if err != nil {
			slog.Error("Failed to pull image", "error", err)
			os.Exit(1)
		}
		defer reader.Close()
		// Read the build output
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			slog.Error("Failed to pull image", "error", err)
			os.Exit(1)
		}
	}

	return utils.NewReadCloser(&outBuf), []string{}, nil
}

func (d *DockerManager) BuildImage(ctx context.Context, dockerfile []byte, imageName string, platform string) (io.ReadCloser, error) {
	tarReader, err := createTarArchive(dockerfile)
	if err != nil {
		return nil, err
	}

	platformSpec := types.ParsePlatform(platform)

	resp, err := d.client.ImageBuild(ctx, tarReader, dockertypes.ImageBuildOptions{
		Tags:       []string{imageName},
		Platform:   platform,
		Dockerfile: "Dockerfile",
		//TODO add docker context that contains the folders and files that are referenced in the Dockerfile
		// Context: ,
		BuildArgs: map[string]*string{
			"TARGETPLATFORM": &platform,
			"TARGETOS":       &platformSpec.OS,
			"TARGETARCH":     &platformSpec.Architecture,
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (d *DockerManager) ContainerLogs(ctx context.Context, id string, ShowStdout bool, ShowStderr bool, Follow bool) (io.ReadCloser, error) {
	return d.client.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: ShowStdout,
		ShowStderr: ShowStderr,
		Follow:     Follow,
	})
}

func (d *DockerManager) InspectImage(ctx context.Context, image string) (*types.ImageInfo, error) {
	imageInfo, _, err := d.client.ImageInspectWithRaw(ctx, image)
	if err != nil {
		slog.Error("Failed to inspect image", "error", err, "imageId", image)
		return nil, fmt.Errorf("error inspecting image: %w", err)
	}
	return &types.ImageInfo{
		ID: imageInfo.ID,
		Platform: &types.PlatformSpec{
			OS:           imageInfo.Os,
			Architecture: imageInfo.Architecture,
		},
	}, nil
}
