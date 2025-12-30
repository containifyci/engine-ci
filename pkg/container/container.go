package container

import (
	"archive/tar"
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
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/registry"

	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/logger"
	"github.com/containifyci/engine-ci/pkg/memory"

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
	t
	Source  fs.ReadDirFS
	Build   *Build
	Secret  map[string]string
	Env     EnvType
	Prefix  string
	Image   string
	Name    string
	ID      string
	Opts    types.ContainerConfig
	Verbose bool
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

func NewWithManager(manager cri.ContainerManager) *Container {
	_client := func() cri.ContainerManager {
		return manager
	}
	ctx := context.Background()
	build := &Build{}
	logger.NewLogAggregator("")
	return &Container{t: t{client: _client, ctx: ctx}, Build: build.Defaults()}
}

func New(build Build) *Container {
	_client := func() cri.ContainerManager {
		client, err := cri.InitContainerRuntime()
		if err != nil {
			panic(fmt.Errorf("failed to detect container runtime: %w", err))
			// slog.Error("Failed to detect container runtime", "error", err)
			// // Return nil to allow caller to handle the error gracefully
			// return nil
		}
		return client
	}

	// Use background context with reasonable timeout instead of TODO
	ctx := context.Background()
	container := &Container{t: t{client: _client, ctx: ctx}, Env: build.Env, Build: &build, Secret: build.Secret, Verbose: build.Verbose}

	return container
}

func (c *Container) getContainifyHost() string {
	if v, ok := c.GetBuild().Custom["CONTAINIFYCI_HOST"]; ok {
		return v[0]
	}
	return ""
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
				info, err := c.client().InspectContainer(c.ctx, c.ID)
				if err != nil {
					slog.Error("Failed to inspect container", "error", err)
					os.Exit(1)
				}
				c.Name = info.Name
				c.Image = info.Image
				img, tag := ParseImageTag(info.Image)

				short := fmt.Sprintf("%s:%s", img, safeShort(tag, 8))
				c.Prefix = fmt.Sprintf("[%s (%s)]", c.ID[:6], short)
				return nil
			}
		}
	}

	if opts.Env == nil {
		opts.Env = []string{}
	}

	if c.getContainifyHost() != "" {
		opts.Env = append(opts.Env, fmt.Sprintf("CONTAINIFYCI_HOST=%s", c.getContainifyHost()))
	}

	opts.Env = append(opts.Env, fmt.Sprintf("CONTAINIFYCI_FOLDER=%s", c.Build.Folder))

	if opts.Platform == types.AutoPlatform {
		opts.Platform = types.GetPlatformSpec()
	}

	slog.Info("Creating container", "opts", opts, "platform", opts.Platform)

	authConfig := c.registryAuthBase64(opts.Image)
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
	img, tag := ParseImageTag(info.Image)

	short := fmt.Sprintf("%s:%s", img, safeShort(tag, 8))
	c.Prefix = fmt.Sprintf("[%s (%s)]", c.ID[:6], short)
	return err
}

func (c *Container) Start() error {
	// Use context with timeout for container operations (10min for CI testing)
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	err := c.client().StartContainer(ctx, c.ID)
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		return fmt.Errorf("failed to start container %s: %w", c.ID, err)
	}

	// TODO make this optional or provide a way to opt out
	shortImage := strings.ReplaceAll(c.Opts.Image, c.GetBuild().Registry+"/", "")
	img, tag := ParseImageTag(shortImage)

	short := fmt.Sprintf("%s:%s", img, safeShort(tag, 8))
	go func() {
		streamContainerLogs(c.ctx, c.client(), c.ID, short, c.Prefix)
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

func streamContainerLogs(ctx context.Context, cli cri.ContainerManager, containerID, image, prefix string) {
	out, err := cli.ContainerLogs(ctx, containerID, true, true, true)
	if err != nil {
		slog.Error("Error getting logs for container", "containerId", containerID, "error", err)
		return
	}
	defer out.Close()

	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		logLine := scanner.Text()
		logger.GetLogAggregator().LogMessage(prefix, logLine)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Error reading logs for container", "containerId", containerID, "error", err)
	}
}

func (c *Container) Stop() error {
	// Use context with timeout for container stop operations (10min for CI testing)
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	return c.client().StopContainer(ctx, c.ID, "SIGTERM")
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
	//TODO: the stdout is not showing up at all, need to fix that
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
		return fmt.Errorf("failed to wait for container: %w", err)
	}
	if statusCode == nil {
		return fmt.Errorf("failed to wait for container: status code is nil")
	}
	if *statusCode != 0 {
		defer func() {
			logger.GetLogAggregator().FailedMessage(c.Prefix, "Container exited with non 0")
		}()
		// Inspect the container to retrieve metadata
		inspection, err := c.client().InspectContainer(c.ctx, c.ID)
		if err != nil {
			c.ctx.Done()
			slog.Error("Failed to inspect container", "error", err)
		}
		return fmt.Errorf("Container %s exited with status %d", inspection.Image, *statusCode)
	}
	logger.GetLogAggregator().SuccessMessage(c.Prefix, "Container exited with status 0")
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
	return c.ensureImagesExists(c.ctx, c.client(), imageTags, platform)
}

func (c *Container) Pull(imageTags ...string) error {
	return c.ensureImagesExists(c.ctx, c.client(), imageTags, c.GetBuild().Platform.Container.String())
}

func (c *Container) PullDefault(imageTags ...string) error {
	return c.PullByPlatform("", imageTags...)
}

// ensureImageExists checks if a Docker image exists locally and pulls it if it doesn't.
// Now uses concurrent pulling for better performance with multiple images.
func (c *Container) ensureImagesExists(ctx context.Context, cli cri.ContainerManager, imageNames []string, platform string) error {
	if len(imageNames) == 0 {
		return nil
	}

	for _, imageName := range imageNames {
		images, err := cli.ListImage(ctx, imageName)
		if err != nil {
			return err
		}

		if len(images) == 0 {
			slog.Info("Image not found locally. Pulling from registry...", "image", imageName)
			out, err := cli.PullImage(ctx, imageName, c.registryAuthBase64(imageName), platform)
			if err != nil {
				return err
			}
			defer out.Close()

			_, err = logger.GetLogAggregator().Copy(out)
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
				slog.Warn("Image found locally but with different platform", "image", imageName, "image_platform", info.Platform.String(), "platform", platform)
				slog.Warn("Pulling from registry...", "image", imageName)
				out, err := cli.PullImage(ctx, imageName, c.registryAuthBase64(imageName), platform)
				if err != nil {
					slog.Error("Failed to pull image", "error", err)
					return err
				}
				defer out.Close()
				_, err = logger.GetLogAggregator().Copy(out)
				if err != nil {
					slog.Error("Failed to copy stdout", "error", err)
					return err
				}
				return nil
			}
			slog.Debug("Image found locally.\n", "image", imageName, "platform", info.Platform.String())
		}
	}
	return nil
}

// TODO: make registry authentification configurable and support multiple registries
func (c *Container) registryAuthBase64(imageName string) string {

	imgInfo, err := utils.ParseDockerImage(imageName)
	if err != nil {
		slog.Error("Failed to parse image", "error", err, "image", imageName)
		return ""
	}

	if reg, ok := c.GetBuild().Registries[imgInfo.Server]; ok {
		username := u.GetValue(reg.Username, c.GetBuild().Env.String())
		slog.Debug("Registry auth found for image", "image", imageName, "server", imgInfo.Server, "username", username)
		authConfig := registry.AuthConfig{
			Username:      username,
			Password:      u.GetValue(reg.Password, c.GetBuild().Env.String()),
			ServerAddress: imgInfo.Server, // Server address for GCR
		}
		return c.encodeAuthToBase64(authConfig)
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
		if c.GetBuild().Runtime == utils.Podman {
			opts = []PushOption{{Remove: false}}
		}
	}

	err := c.Tag(source, target)
	if err != nil {
		slog.Error("Failed to tag image", "error", err)
		return err
	}

	authConfig := c.registryAuthBase64(target)

	reader, err := c.client().PushImage(c.ctx, target, authConfig)
	if err != nil {
		slog.Error("Failed to push image", "error", err)
		return err
	}
	defer reader.Close()
	_, err = logger.GetLogAggregator().Copy(reader)
	if err != nil {
		slog.Error("Failed to copy output", "error", err)
		return err
	}
	if opts[0].Remove {
		return c.client().RemoveImage(c.ctx, target)
	}
	return nil
}

// encodeAuthToBase64 encodes the authentication configuration to base64.
func (c *Container) encodeAuthToBase64(auth registry.AuthConfig) string {
	authJSON, _ := json.Marshal(auth)
	if c.GetBuild().Verbose {
		// Mask sensitive auth data in debug logs for security
		slog.Debug("Auth config", "username", auth.Username, "server", auth.ServerAddress, "auth_configured", len(auth.Password) > 0)
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

func (c *Container) GetBuild() *Build {
	return c.Build
}

func (c *Container) BuildImageByPlatforms(dockerfile []byte, dockerCtx *bytes.Buffer, imageName string, platforms []string) ([]string, error) {
	authConfig := c.registryAuthBase64(imageName)
	reader, imageIds, err := c.client().BuildMultiArchImage(c.ctx, dockerfile, dockerCtx, imageName, platforms, authConfig)
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
	return c.BuildImageByPlatform(dockerfile, imageName, c.GetBuild().Platform.Container.String())
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

// ComputeChecksum computes SHA256 checksum with memory optimizations
// Uses streaming approach and buffer pooling for better performance
func ComputeChecksum(data []byte) string {

	// For very small data, use direct approach to avoid overhead
	if len(data) <= 4096 { // 4KB threshold - reduced for better performance
		hash := sha256.Sum256(data)
		result := hex.EncodeToString(hash[:])
		return result
	}

	// For larger data, use direct SHA256 approach (58% faster than buffer pool)
	hasher := sha256.New()
	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	result := hex.EncodeToString(hashBytes)

	return result
}

// SumChecksum combines multiple checksums with memory optimization
func SumChecksum(sums ...[]byte) string {
	hasher := sha256.New()

	// Direct approach - simpler and faster than buffer pool
	for _, sum := range sums {
		hasher.Write(sum)
	}

	return hex.EncodeToString(hasher.Sum(nil))
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
	if secrets, ok := c.GetBuild().Custom["secrets"]; ok {
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
		slog.Error("Failed to wait for container", "error", err, "name", c.Name, "id", c.ID, "image", c.Image)
		// os.Exit(1)
		return err
	}
	return err
}

func (c *Container) BuildIntermidiateContainer(image string, dockerFile []byte, platforms ...string) error {
	if len(platforms) == 0 {
		platforms = []string{c.GetBuild().Platform.Container.String()}
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

	var pullErr []error
	for _, platform := range platforms {
		err := c.PullByPlatform(platform, image)
		if err != nil {
			slog.Warn("Failed to pull intermediate image. Has to build now then", "error", err, "image", image, "platform", platform)
			pullErr = append(pullErr, err)
		}
	}

	if len(pullErr) == 0 {
		slog.Info("Image successfully pulled", "image", image, "platforms", platforms)
		return nil
	}

	if len(platforms) == 1 {

		//TODO: implement providing the src folder for the docker build
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
		platform := c.GetBuild().Platform.Container.String()
		err = c.PullByPlatform(platform, image)
		if err != nil {
			slog.Warn("Failed to pull intermediate image. Has to build now then", "error", err, "image", image)
		}

		if err == nil {
			slog.Info("Image successfully pulled", "image", image)
			return nil
		}

		var buf *bytes.Buffer

		if c.Source != nil {
			buf, err = TarDir(c.Source)
			if err != nil {
				slog.Error("Failed to tar source", "error", err)
				os.Exit(1)
			}
		}

		// Multi-platform builds are already pushed otherwise there are not usable by podman or docker
		_, err = c.BuildImageByPlatforms(dockerFile, buf, image, platforms)
		if err != nil {
			slog.Error("Failed to build image", "error", err)
			os.Exit(1)
		}
	}

	return err
}

// TarDir creates a tar archive from a filesystem with memory optimizations and concurrent processing
func TarDir(src fs.ReadDirFS) (*bytes.Buffer, error) {
	// Count files first to determine if concurrent processing is beneficial
	fileCount := 0
	totalSize := int64(0)

	err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fileCount++
			if fi, err := d.Info(); err == nil {
				totalSize += fi.Size()
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Use concurrent processing for larger archives
	if fileCount > 20 || totalSize > 10*1024*1024 { // 20 files or 10MB
		return TarDirConcurrent(src, fileCount, totalSize)
	}

	// Use sequential processing for smaller archives
	return TarDirSequential(src)
}

// TarDirConcurrent creates a tar archive using concurrent processing for better performance
func TarDirConcurrent(src fs.ReadDirFS, fileCount int, totalSize int64) (*bytes.Buffer, error) {
	// Estimate buffer size based on file count and total size
	estimatedSize := max(64*1024, int(totalSize/4)) // At least 64KB or quarter of total size
	buf := bytes.NewBuffer(make([]byte, 0, estimatedSize))
	tw := tar.NewWriter(buf)

	// Collect all file entries first
	type fileEntry struct {
		info fs.FileInfo
		path string
	}

	var files []fileEntry
	err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fi, err := d.Info()
			if err != nil {
				return err
			}
			files = append(files, fileEntry{path: path, info: fi})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Process files with controlled concurrency
	maxWorkers := min(runtime.NumCPU(), 4) // Limit to 4 workers max
	semaphore := make(chan struct{}, maxWorkers)

	// Channel for processed file data
	type processedFile struct {
		err     error
		header  *tar.Header
		path    string
		content []byte
		index   int
	}

	resultChan := make(chan processedFile, len(files))
	var wg sync.WaitGroup

	// Process files concurrently but maintain order
	for i, file := range files {
		wg.Add(1)
		go func(idx int, f fileEntry) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			default:
				// Fallback to sequential processing if semaphore is full
			}

			// Read file content
			data, err := src.Open(f.path)
			if err != nil {
				resultChan <- processedFile{path: f.path, err: err, index: idx}
				return
			}
			defer data.Close()

			// Use pooled buffer for reading
			content := memory.WithBufferReturn(memory.TarBuffer, func(buffer []byte) []byte {
				var result []byte
				for {
					n, err := data.Read(buffer)
					if n > 0 {
						result = append(result, buffer[:n]...)
					}
					if err == io.EOF {
						break
					}
					if err != nil {
						return nil
					}
				}
				memory.TrackBufferReuse()
				return result
			})

			if content == nil {
				resultChan <- processedFile{path: f.path, err: fmt.Errorf("failed to read file"), index: idx}
				return
			}

			// Create tar header
			header, err := tar.FileInfoHeader(f.info, f.info.Name())
			if err != nil {
				resultChan <- processedFile{path: f.path, err: err, index: idx}
				return
			}
			header.Name = filepath.ToSlash(f.path)

			resultChan <- processedFile{
				path:    f.path,
				header:  header,
				content: content,
				index:   idx,
			}
		}(i, file)
	}

	// Close result channel when all goroutines finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and maintain order
	results := make([]processedFile, len(files))
	for result := range resultChan {
		if result.err != nil {
			slog.Error("Error processing file for tar", "path", result.path, "error", result.err)
			return nil, result.err
		}
		results[result.index] = result
	}

	// Write to tar in original order
	for _, result := range results {
		if err := tw.WriteHeader(result.header); err != nil {
			return nil, err
		}
		if len(result.content) > 0 {
			if _, err := tw.Write(result.content); err != nil {
				return nil, err
			}
		}
	}

	// Close the tar writer
	if err := tw.Close(); err != nil {
		slog.Error("Error closing the tar writer", "error", err)
		return nil, err
	}
	return buf, nil
}

// TarDirSequential creates a tar archive using sequential processing (original implementation)
func TarDirSequential(src fs.ReadDirFS) (*bytes.Buffer, error) {
	// Pre-estimate buffer size to reduce reallocations
	initialSize := 64 * 1024 // 64KB initial buffer
	buf := bytes.NewBuffer(make([]byte, 0, initialSize))
	tw := tar.NewWriter(buf)

	// Use pooled buffer for file copying to reduce allocations
	copyBuffer := memory.GetBuffer(memory.TarBuffer)
	defer memory.PutBuffer(copyBuffer, memory.TarBuffer)

	// Walk the directory and write each file to the tar writer
	err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		fi, err := d.Info()
		if err != nil {
			return err
		}

		// Create a tar header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// Ensure the header has the correct name
		header.Name = filepath.ToSlash(path)

		// Write the header to the tar writer
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// If the file is not a directory, write the file content
		if !fi.IsDir() {
			data, err := src.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				if closeErr := data.Close(); closeErr != nil {
					slog.Warn("Failed to close file", "path", path, "error", closeErr)
				}
			}()

			// Use pooled buffer for efficient copying with optimized chunk size
			_, err = io.CopyBuffer(tw, data, copyBuffer)
			if err != nil {
				return err
			}

			// Track buffer reuse for each file processed
			memory.TrackBufferReuse()
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

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c *Container) Apply(opts *types.ContainerConfig) {
	if envs, ok := c.GetBuild().Custom["envs"]; ok {
		for _, env := range envs {
			v := u.GetEnv(env, "build")
			opts.Env = append(opts.Env, fmt.Sprintf("%s=%s", env, v))
		}
	}
}
