package goreleaser

//go:generate go run ../../tools/dockerfile-metadata/ -input Dockerfile.goreleaser-zig -output docker_metadata_gen.go -package goreleaser

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	criutils "github.com/containifyci/engine-ci/pkg/cri/utils"
	utils "github.com/containifyci/engine-ci/pkg/utils"
	"github.com/containifyci/engine-ci/pkg/zig"

	"github.com/containifyci/engine-ci/pkg/svc"
)

//go:embed .goreleaser.yaml
var defaultGoreleaserConfig []byte

//go:embed .goreleaser-zig.yaml
var defaultZigGoreleaserConfig []byte

const (
	IMAGE             = "goreleaser/goreleaser:v2.15.2"
	defaultConfigPath = "/tmp/.goreleaser-default.yaml"
	zigCacheLocation  = "/root/.cache/zig"
)

type GoReleaserContainer struct {
	*container.Container
}

// Matches implements the Build interface - GoReleaser only runs for golang builds with goreleaser=true
func Matches(build container.Build) bool {
	if build.BuildType != container.GoLang && build.BuildType != container.Zig {
		return false
	}
	// Check if goreleaser is enabled
	if goreleaser, ok := build.Custom["goreleaser"]; ok && len(goreleaser) > 0 {
		return goreleaser[0] == "true"
	}
	return false
}

func New() build.BuildStep {
	return build.Stepper{
		// BuildType_: container.GoLang + container.Zig, // This step can run for both Go and Zig builds
		RunFn: func(build container.Build) (string, error) {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  goreleaserImages,
		Name_:     "gorelease",
		Alias_:    "release",
		Async_:    false,
	}
}

func goreleaserImages(b container.Build) []string {
	if b.BuildType == container.Zig {
		return []string{ZigGoreleaserImage(b), IMAGE}
	}
	return []string{IMAGE}
}

func new(build container.Build) *GoReleaserContainer {
	return &GoReleaserContainer{
		Container: container.New(build),
	}
}

// newWithManager creates a GoReleaserContainer with a custom container manager (for testing)
func newWithManager(build container.Build, manager cri.ContainerManager) *GoReleaserContainer {
	c := container.NewWithManager(manager)
	b := build.Defaults()
	c.Build = b
	c.Env = b.Env
	return &GoReleaserContainer{Container: c}
}

// cacheFolderFn allows injection of cache folder lookup for testing
var cacheFolderFn = defaultCacheFolder

func defaultCacheFolder() (string, error) {
	cmd := exec.Command("go", "env", "GOMODCACHE")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get GOMODCACHE: %w", err)
	}
	return strings.Trim(string(output), "\n"), nil
}

func CacheFolder() string {
	gomodcache, err := cacheFolderFn()
	if err != nil {
		slog.Error("Failed to execute command", "error", err)
		os.Exit(1)
	}
	slog.Debug("GOMODCACHE location", "path", gomodcache)
	return gomodcache
}

func (c *GoReleaserContainer) isZig() bool {
	return c.GetBuild().BuildType == container.Zig
}

// ZigGoreleaserImage returns the image URI for the goreleaser+zig intermediate image
func ZigGoreleaserImage(build container.Build) string {
	image := fmt.Sprintf("goreleaser-zig-%s", ImageVersion)
	return criutils.ImageURI(build.ContainifyRegistry, image, DockerfileChecksum)
}

// BuildZigImage builds the intermediate goreleaser+zig Docker image
func (c *GoReleaserContainer) BuildZigImage() error {
	b := c.GetBuild()
	image := ZigGoreleaserImage(*b)
	platforms := types.GetPlatforms(b.Platform)
	slog.Info("Building goreleaser-zig intermediate image", "image", image, "platforms", platforms)
	return c.BuildIntermidiateContainer(image, []byte(DockerfileContent), platforms...)
}

// zigCacheFolderFn allows injection of zig cache folder lookup for testing
var zigCacheFolderFn = zig.CacheFolder

func (c *GoReleaserContainer) ApplyEnvs(envs []string) []string {
	tag := os.Getenv("GORELEASER_CURRENT_TAG")
	if tag != "" {
		envs = append(envs, "GORELEASER_CURRENT_TAG="+tag)
	}
	return envs
}

// hasProjectConfig checks if the project has its own goreleaser config file
func hasProjectConfig(dir string) bool {
	configNames := []string{".goreleaser.yaml", ".goreleaser.yml", ".goreleaser.json"}
	for _, name := range configNames {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

// writeDefaultConfigContent writes the given config content to a temp file
func writeDefaultConfigContent(content []byte) (string, error) {
	if err := os.WriteFile(defaultConfigPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write default config: %w", err)
	}
	return defaultConfigPath, nil
}

// ErrMissingToken is returned when CONTAINIFYCI_GITHUB_TOKEN is not set
var ErrMissingToken = fmt.Errorf("missing CONTAINIFYCI_GITHUB_TOKEN")

func (c *GoReleaserContainer) Release(env container.EnvType) error {
	token := container.GetEnv("CONTAINIFYCI_GITHUB_TOKEN")
	if token == "" {
		slog.Warn("Skip goreleaser missing CONTAINIFYCI_GITHUB_TOKEN")
		return ErrMissingToken
	}

	opts := types.ContainerConfig{}

	envKeys := c.GetBuild().Custom.Strings("goreleaser_envs")
	envs := utils.GetAllEnvs(envKeys, c.Env.String())

	for k, v := range envs {
		opts.Env = append(opts.Env, k+"="+v)
	}

	opts.Env = append(opts.Env, "GITHUB_TOKEN="+token)
	opts.Env = c.ApplyEnvs(opts.Env)

	dir, _ := filepath.Abs(".")
	opts.WorkingDir = "/usr/src"

	if c.isZig() {
		opts.Image = ZigGoreleaserImage(*c.GetBuild())
		opts.Env = append(opts.Env, fmt.Sprintf("ZIG_GLOBAL_CACHE_DIR=%s", zigCacheLocation))
		opts.Volumes = []types.Volume{
			{Type: "bind", Source: dir, Target: "/usr/src"},
			{Type: "bind", Source: zigCacheFolderFn(), Target: zigCacheLocation},
		}
	} else {
		opts.Image = IMAGE
		opts.Env = append(opts.Env, []string{
			"GOMODCACHE=/go/pkg/",
			"GOCACHE=/go/pkg/build-cache",
			"GOLANGCI_LINT_CACHE=/go/pkg/lint-cache",
		}...)
		cache := CacheFolder()
		if cache == "" {
			cache, _ = filepath.Abs(".tmp/go")
		}
		opts.Volumes = []types.Volume{
			{Type: "bind", Source: dir, Target: "/usr/src"},
			{Type: "bind", Source: cache, Target: "/go/pkg"},
		}
	}

	opts.Cmd = []string{"release", "--skip=validate", "--verbose", "--clean"}

	// Use embedded default config if project doesn't have one
	if !hasProjectConfig(dir) {
		slog.Info("No goreleaser config found, using embedded default")
		configContent := defaultGoreleaserConfig
		if c.isZig() {
			configContent = defaultZigGoreleaserConfig
		}
		hostConfigPath, err := writeDefaultConfigContent(configContent)
		if err != nil {
			return fmt.Errorf("failed to write default goreleaser config: %w", err)
		}
		defer os.Remove(hostConfigPath)

		opts.Volumes = append(opts.Volumes, types.Volume{
			Type:   "bind",
			Source: hostConfigPath,
			Target: defaultConfigPath,
		})
		opts.Cmd = append(opts.Cmd, "--config="+defaultConfigPath)
	}

	err := c.Create(opts)
	if err != nil {
		return err
	}

	err = c.Start()
	if err != nil {
		return err
	}

	return c.Wait()
}

func (c *GoReleaserContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *GoReleaserContainer) Run() (string, error) {
	slog.Info("Run gorelease")
	if svc.GitInfo().IsNotTag() {
		slog.Info("Skipping goreleaser for non tag branch")
		return "", nil
	}
	if !c.GetBuild().Custom.Bool("goreleaser", false) {
		slog.Info("Skip goreleaser because its not explicit enabled", "build", c.GetBuild())
		return "", nil
	}

	if err := c.Pull(); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	if c.isZig() {
		if err := c.BuildZigImage(); err != nil {
			return "", fmt.Errorf("failed to build goreleaser-zig image: %w", err)
		}
	}

	if err := c.Release(c.GetBuild().Env); err != nil {
		return c.ID, fmt.Errorf("failed to release: %w", err)
	}
	return c.ID, nil
}
