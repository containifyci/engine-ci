package container

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/registry"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	u "github.com/containifyci/engine-ci/pkg/utils"
)

type t struct {
	client func() cri.ContainerManager
	ctx    context.Context
}

type EnvType string

const (
	LocalEnv EnvType = "local"
	BuildEnv EnvType = "build"
	ProdEnv  EnvType = "production"
)

// String is used both by fmt.Print and by Cobra in help text
func (e *EnvType) String() string {
	return string(*e)
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (e *EnvType) Set(v string) error {
	switch v {
	case "local", "build", "production":
		*e = EnvType(v)
		return nil
	default:
		return errors.New(`must be one of "local", "build", or "production"`)
	}
}

// Type is only used in help text
func (e *EnvType) Type() string {
	return "EnvType"
}

type Container struct {
	// private fields
	ID      string
	Name    string
	Image   string
	Env     EnvType
	Verbose bool

	Opts types.ContainerConfig
	t
}

type PushOption struct {
	Remove bool
}

func Getenv(key string) string {
	return u.Getenv(key, string(BuildEnv))
}

func GetEnv(key string) string {
	return u.GetEnv(key, string(BuildEnv))
}

func New(env EnvType) *Container {
	_client := func() cri.ContainerManager {
		client, err := cri.InitContainerRuntime()
		if err != nil {
			slog.Error("Failed to detect container runtime", "error", err)
			os.Exit(1)
		}
		return client
	}
	// _client()
	if _build != nil {
		return &Container{t: t{client: _client, ctx: context.TODO()}, Env: env, Verbose: _build.Verbose}
	}
	return &Container{t: t{client: _client, ctx: context.TODO()}, Env: env}
}

func (c *Container) Create(opts types.ContainerConfig) error {
	c.Opts = opts

	if opts.Name != "" {
		// List all containers
		containers, err := c.client().ContainerList(c.ctx, true)
		if err != nil {
			slog.Error("Failed to list containers: %s", "error", err)
			os.Exit(1)
		}

		// Find the container by name
		var foundContainer *types.Container
		for _, container := range containers {
			for _, name := range container.Names {
				if name == "/"+opts.Name { // Container names have a leading '/'
					foundContainer = container
					break
				}
			}
			if foundContainer != nil {
				c.ID = foundContainer.ID
				return nil
			}
		}
	}

	if opts.Platform == types.AutoPlatform {
		opts.Platform = types.GetPlatformSpec()
	}

	slog.Info("Creating container", "opts", opts, "platform", opts.Platform)

	authConfig := registryAuthBase64(opts.Image)
	id, err := c.client().CreateContainer(c.ctx, &opts, authConfig)

	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}
	c.ID = id

	info, err := c.client().InspectContainer(c.ctx, c.ID)
	if err != nil {
		slog.Error("Failed to inspect container", "error", err)
		os.Exit(1)
	}
	c.Name = info.Name
	c.Image = info.Image
	return err
}

func (c *Container) Start() error {
	err := c.client().StartContainer(context.TODO(), c.ID)
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	// TODO make this optional or provide a way to opt out
	shortImage := strings.ReplaceAll(c.Opts.Image, GetBuild().Registry+"/", "")
	img, tag := ParseImageTag(shortImage)

	short := fmt.Sprintf("%s:%s", img, safeShort(tag, 8))
	go func() {
		streamContainerLogs(c.ctx, c.client(), c.ID, short)
	}()
	return err
}

func safeShort(str string, end int) string {
	if end > len(str) {
		end = len(str)
	}
	return str[:end]
}

func ParseImageTag(imageTag string) (string, string) {
	parts := strings.Split(imageTag, ":")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func streamContainerLogs(ctx context.Context, cli cri.ContainerManager, containerID, image string) {
	out, err := cli.ContainerLogs(ctx, containerID, true, true, true)
	if err != nil {
		slog.Error("Error getting logs for container", "containerId", containerID, "error", err)
		return
	}
	defer out.Close()

	prefix := fmt.Sprintf("[%s (%s)] ", containerID[:6], image)

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		logLine := scanner.Text()
		fmt.Printf("%s%s\n", prefix, logLine)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Error reading logs for container", "containerId", containerID, "error", err)
	}
}

func (c *Container) Stop() error {
	return c.client().StopContainer(context.TODO(), c.ID, "SIGTERM")
}

func (c *Container) CopyContentTo(content, dest string) error {
	return c.client().CopyContentToContainer(c.ctx, c.ID, content, dest)
}

func (c *Container) Commit(imageTag string, comment string, changes ...string) (string, error) {
	// Commit the container to create a new image
	id, err := c.client().CommitContainer(c.ctx, c.ID, types.CommitOptions{
		Reference: imageTag,
		Comment:   comment,
		Changes:   changes,
	})

	if err != nil {
		slog.Error("Failed to commit container", "error", err, "imageTag", imageTag)
		os.Exit(1)
	}
	return id, err
}

// TODO: ignore hidden folder and files maybe support .dockerignore file or more .dockerinclude file to
// include folder and files that are ignored by default
func (c *Container) CopyDirectoryTo(srcPath, dstPath string) error {
	err := c.client().CopyDirectorToContainer(c.ctx, c.ID, srcPath, dstPath)

	if err != nil {
		slog.Error("Failed to copy to container", "error", err)
		return err
	}
	return nil
}

func (c *Container) CopyFileTo(srcPath, destPath string) error {
	return c.client().CopyToContainer(c.ctx, c.ID, srcPath, destPath)
}

func (c *Container) Exec(cmd ...string) error {
	reader, err := c.client().ExecContainer(c.ctx, c.ID, cmd, true)
	if err != nil {
		slog.Error("Failed to exec command", "error", err)
		return err
	}
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		slog.Error("Failed to copy output", "error", err)
	}
	return err
}

func (c *Container) Inspect() (*types.ContainerConfig, error) {
	return c.client().InspectContainer(c.ctx, c.ID)
}

func (c *Container) InspectImage(image string) (*types.ImageInfo, error) {
	return c.client().InspectImage(c.ctx, image)
}

func (c *Container) Wait() error {
	statusCode, err := c.client().WaitContainer(c.ctx, c.ID, string(container.WaitConditionNotRunning))
	if err != nil {
		c.ctx.Done()
		log.Fatal(err)
	}
	if statusCode == nil {
		log.Fatal(fmt.Errorf("Failed to wait for container status code is nil"))
	}
	if *statusCode != 0 {
		// Inspect the container to retrieve metadata
		inspection, err := c.client().InspectContainer(c.ctx, c.ID)
		if err != nil {
			c.ctx.Done()
			slog.Error("Failed to inspect container", "error", err)
		}
		return fmt.Errorf("Container %s exited with status %d", inspection.Image, *statusCode)
	}
	return nil
}

func (c *Container) Ready() error {
	if c.Opts.Readiness != nil {
		err := waitForApplication(c.ctx, c.Opts.Readiness)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Container) PullByPlatform(platform string, imageTags ...string) error {
	return ensureImagesExists(c.ctx, c.client(), imageTags, platform)
}

func (c *Container) Pull(imageTags ...string) error {
	return ensureImagesExists(c.ctx, c.client(), imageTags, GetBuild().Platform.Container.String())
}

func (c *Container) PullDefault(imageTags ...string) error {
	return c.PullByPlatform("", imageTags...)
}

// ensureImageExists checks if a Docker image exists locally and pulls it if it doesn't.
func ensureImagesExists(ctx context.Context, cli cri.ContainerManager, imageNames []string, platform string) error {
	for _, imageName := range imageNames {
		images, err := cli.ListImage(ctx, imageName)
		if err != nil {
			return err
		}

		if len(images) == 0 {
			slog.Info("Image not found locally. Pulling from registry...", "image", imageName)
			out, err := cli.PullImage(ctx, imageName, registryAuthBase64(imageName), platform)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = io.Copy(os.Stdout, out)
			if err != nil {
				return err
			}
		} else {
			info, err := cli.InspectImage(ctx, imageName)
			if err != nil {
				slog.Error("Failed to inspect image", "error", err)
				return err
			}
			if info.Platform.String() != platform {
				slog.Warn("Image found locally but with different platform", "image", imageName, "platform", info.Platform.String())
				slog.Warn("Pulling from registry...", "image", imageName)
				out, err := cli.PullImage(ctx, imageName, registryAuthBase64(imageName), platform)
				if err != nil {
					slog.Error("Failed to pull image", "error", err)
					return err
				}
				defer out.Close()
				_, err = io.Copy(os.Stdout, out)
				if err != nil {
					slog.Error("Failed to copy stdout", "error", err)
					return err
				}
				return nil
			}
			slog.Info("Image found locally.\n", "image", imageName, "platform", info.Platform.String())
		}
	}
	return nil
}

// TODO: make registry authentification configurable and support multiple registries
func registryAuthBase64(imageName string) string {

	imgInfo, err := utils.ParseDockerImage(imageName)
	slog.Debug("Auth for image", "info", imgInfo)
	if err != nil {
		slog.Error("Failed to parse image", "error", err, "image", imageName)
		return ""
	}

	if reg, ok := GetBuild().Registries[imgInfo.Server]; ok {
		authConfig := registry.AuthConfig{
			Username:      u.GetValue(reg.Username, GetBuild().Env.String()), // Username for GCR
			Password:      u.GetValue(reg.Password, GetBuild().Env.String()),
			ServerAddress: imgInfo.Server, // Server address for GCR
		}
		return encodeAuthToBase64(authConfig)
	}

	slog.Warn("No registry auth found for image", "image", imageName, "server", imgInfo.Server)
	return ""
}

func (c *Container) Tag(source, target string) error {
	err := c.client().TagImage(c.ctx, source, target)
	if err != nil {
		slog.Error("Failed to tag image", "error", err)
		return err
	}
	return nil
}

// TODO: find a better way to provide optional parameters like PushOption
func (c *Container) Push(source, target string, opts ...PushOption) error {
	if opts == nil {
		opts = []PushOption{{Remove: true}}
		if GetBuild().Runtime == utils.Podman {
			opts = []PushOption{{Remove: false}}
		}
	}

	err := c.Tag(source, target)
	if err != nil {
		slog.Error("Failed to tag image", "error", err)
		return err
	}

	authConfig := registryAuthBase64(target)

	reader, err := c.client().PushImage(c.ctx, target, authConfig)
	if err != nil {
		slog.Error("Failed to push image", "error", err)
		return err
	}
	defer reader.Close()
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		slog.Error("Failed to copy output", "error", err)
	}
	if opts[0].Remove {
		return c.client().RemoveImage(c.ctx, target)
	}
	return nil
}

// encodeAuthToBase64 encodes the authentication configuration to base64.
func encodeAuthToBase64(auth registry.AuthConfig) string {
	authJSON, _ := json.Marshal(auth)
	if GetBuild().Verbose {
		slog.Debug("Auth config", "auth", string(authJSON))
	}
	return base64.URLEncoding.EncodeToString(authJSON)
}

func waitForApplication(ctx context.Context, readiness *types.ReadinessProbe) error {
	ctx, cancel := context.WithTimeout(ctx, readiness.Timeout)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(5 * time.Second)

			// Perform a health check or readiness check against the container's application
			resp, err := http.Get(readiness.Endpoint)
			if err != nil {
				break
				// slog.Error("Waiting for application to start", "error", err)
				// return err
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if readiness.Validate != nil {
					buf := new(bytes.Buffer)
					_, err := buf.ReadFrom(resp.Body)
					if err != nil {
						slog.Error("Failed to read response body", "error", err)
						break
					}
					body := buf.String()
					if readiness.Validate([]byte(body)) {
						return nil
					}
				} else {
					return nil
				}
			}

			fmt.Printf("Waiting for application to start: HTTP status code %d\n", resp.StatusCode)
		}
	}
}

func GetBuild() *Build {
	return _build
}

func (c *Container) BuildImageByPlatforms(dockerfile []byte, imageName string, platforms []string) ([]string, error) {
	authConfig := registryAuthBase64(imageName)
	reader, imageIds, err := c.client().BuildMultiArchImage(c.ctx, dockerfile, imageName, platforms, authConfig)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// Read the build output
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return nil, err
	}
	return imageIds, err
}

func (c *Container) BuildImageByPlatform(dockerfile []byte, imageName string, platform string) error {
	// return nil
	reader, err := c.client().BuildImage(c.ctx, dockerfile, imageName, platform)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Read the build output
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}
	return err
}

func (c *Container) BuildImage(dockerfile []byte, imageName string) error {
	return c.BuildImageByPlatform(dockerfile, imageName, GetBuild().Platform.Container.String())
}

// imageExists checks if the image with the specified tag exists.
func (c *Container) ImageExists(imageName string, platforms ...string) (bool, error) {
	images, err := c.client().ListImage(c.ctx, imageName)
	if err != nil {
		return false, err
	}

	foundByPlatform := map[string]bool{}
	for _, platform := range platforms {
		foundByPlatform[platform] = false
	}
	for _, image := range images {
		if image == imageName {
			if len(platforms) == 0 {
				return true, nil
			}
			info, err := c.client().InspectImage(c.ctx, imageName)
			if err != nil {
				return false, err
			}
			for _, platform := range platforms {
				if info.Platform.String() == platform {
					foundByPlatform[platform] = true
					break
				}
			}
		}
	}

	for _, found := range foundByPlatform {
		if !found {
			return false, nil
		}
	}

	return true, nil
}

// CopyFileFromContainer reads a single file from a container and returns its content as a string.
func (c *Container) CopyFileFromContainer(srcPath string) (string, error) {
	return c.client().CopyFileFromContainer(c.ctx, c.ID, srcPath)
}

// TODO: provide a shorter checksum
func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (c *Container) BuildingContainer(opts types.ContainerConfig) error {
	opts.Cmd = []string{"sh", "/tmp/script.sh"}

	err := c.Create(opts)
	if err != nil {
		slog.Error("Failed to create container", "error", err)
		os.Exit(1)
	}

	//TODO: maybe define a general entrypoint for all containers
	//only the containers can then define a script that is called by the entrypoint
	err = c.CopyContentTo(opts.Script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}

	//TODO: load the secrets in the build scripts from above
	//The secret could be loaded as part of the entrypoint.
	if secrets, ok := GetBuild().Custom["secrets"]; ok {
		var buf bytes.Buffer
		buf.WriteString("#!/bin/sh\nset +xe\n")
		for _, secret := range secrets {
			v := u.GetEnv(secret, "build")
			buf.WriteString(fmt.Sprintf("export %s=%s\n", secret, v))
		}
		err = c.CopyContentTo(buf.String(), "/tmp/secrets.sh")
		if err != nil {
			slog.Error("Failed to copy secrets to container: %s", "error", err)
			os.Exit(1)
		}
	}

	err = c.Start()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Wait()
	if err != nil {
		slog.Error("Failed to wait for container", "error", err, "name", c.Name, "image", c.Image)
		os.Exit(1)
	}
	return err
}

func (c *Container) BuildIntermidiateContainer(image string, dockerFile []byte, platforms ...string) error {
	if len(platforms) == 0 {
		platforms = []string{GetBuild().Platform.Container.String()}
	}

	exists, err := c.ImageExists(image, platforms...)
	if err != nil {
		slog.Error("Failed to check if image exists", "error", err)
		os.Exit(1)
	}
	if exists {
		slog.Info("Image already exists", "image", image)
		return nil
	}

	err = c.PullByPlatform(platforms[0], image)
	if err != nil {
		slog.Warn("Failed to pull intermediate image. Has to build now then", "error", err, "image", image, "platform", platforms[0])
	}

	if err == nil {
		slog.Info("Image successfully pulled", "image", image, "platforms", platforms)
		return nil
	}

	if len(platforms) == 1 {
		slog.Info("Start building intermediate container image", "image", image)
		err = c.BuildImage(dockerFile, image)
		if err != nil {
			slog.Error("Failed to build image", "error", err)
			os.Exit(1)
		}

		err = c.Push(
			image,
			//TODO: define where the build container should be stored
			image,
			PushOption{Remove: false},
		)
		if err != nil {
			slog.Error("Failed to push image", "error", err)
			os.Exit(1)
		}
	} else {
		//TODO: how to pull multi platform images
		platform := GetBuild().Platform.Container.String()
		err = c.PullByPlatform(platform, image)
		if err != nil {
			slog.Warn("Failed to pull intermediate image. Has to build now then", "error", err, "image", image)
		}

		if err == nil {
			slog.Info("Image successfully pulled", "image", image)
			return nil
		}

		// err = c.Container.BuildImage(dockerFile, image)
		// TODO get list of platforms
		// Multi-platform builds are already pushed otherwise there are not usable by podman or docker
		_, err = c.BuildImageByPlatforms(dockerFile, image, platforms)
		if err != nil {
			slog.Error("Failed to build image", "error", err)
			os.Exit(1)
		}
	}

	return err
}

func (c *Container) Apply(opts *types.ContainerConfig) {
	if envs, ok := GetBuild().Custom["envs"]; ok {
		for _, env := range envs {
			v := u.GetEnv(env, "build")
			opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", env, v))
		}
	}
}
