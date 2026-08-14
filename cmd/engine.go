package cmd

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/containifyci/engine-ci/pkg/autodiscovery"
	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/copier"
	"github.com/containifyci/engine-ci/pkg/cri"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/dummy"
	"github.com/containifyci/engine-ci/pkg/gcloud"
	"github.com/containifyci/engine-ci/pkg/github"
	"github.com/containifyci/engine-ci/pkg/golang"
	"github.com/containifyci/engine-ci/pkg/goreleaser"
	"github.com/containifyci/engine-ci/pkg/maven"
	"github.com/containifyci/engine-ci/pkg/network"
	"github.com/containifyci/engine-ci/pkg/packer"
	"github.com/containifyci/engine-ci/pkg/protobuf"
	"github.com/containifyci/engine-ci/pkg/pulumi"
	"github.com/containifyci/engine-ci/pkg/python"
	"github.com/containifyci/engine-ci/pkg/sonarcloud"
	"github.com/containifyci/engine-ci/pkg/trivy"
	"github.com/containifyci/engine-ci/pkg/utils"
	"github.com/containifyci/engine-ci/pkg/zig"
	"github.com/containifyci/engine-ci/protos2"
	"github.com/spf13/cobra"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// TODO (tight coupling buildsteps and build paramater) we have to uncouple the BuildSteps initialization from the build parameters.
// Otherwise we have to initialize the BuildSteps for every build new which is not optimal.
var buildSteps *build.BuildSteps

// buildCmd represents the build command
var engineCmd = &cobra.Command{
	Use:   "engine",
	Short: "Command to generate Github Action file for gflip",
	Long: `Command to generate Github Action file for gflip.
`,
	RunE: Engine,
}

func init() {
	rootCmd.AddCommand(engineCmd)
	buildSteps = build.NewBuildSteps()
}

type LeaderElection struct {
	mu sync.Mutex
}

func (l *LeaderElection) Leader(id string, fnc func() error) {
	// Try to acquire the lock
	if l.mu.TryLock() {
		// Process becomes leader
		defer l.mu.Unlock()
		slog.Info("Process is the leader\n", "id", id)
		err := fnc()
		if err != nil {
			slog.Error("Error runing func as leader", "error", err)
		}
	} else {
		l.mu.Lock()
		defer l.mu.Unlock()
		err := fnc()
		if err != nil {
			slog.Error("Error runing func as non leader", "error", err)
		}
		slog.Info("Process is not the leader. Waiting...\n", "id", id)
	}
}

func Engine(cmd *cobra.Command, _ []string) error {
	leader := LeaderElection{}
	fnc, addr, err := Start()
	if err != nil {
		return err
	}
	defer fnc()

	InitBuildSteps()

	groups := GetBuild(false)
	idStore := utils.IDStore{}
	for _, group := range groups {
		executeBuildGroup(group, &leader, &idStore, addr)
	}
	slog.Info("Finish waiting for all builds to complete")
	return nil
}

// executeBuild executes a single build with proper context and error handling.
// It handles leader assignment, command execution, and result tracking.
// This function is designed to be called from a goroutine.
func executeBuild(b *container.Build, leader *LeaderElection, idStore *utils.IDStore, addr network.Address) {
	time.Sleep(1 * time.Second)
	b.Leader = leader
	slog.Info("Starting build", "build", b, "steps", buildSteps.String())

	c := NewCommand(*b, buildSteps)
	result := c.Run(addr, RootArgs.Target, b)
	slog.Info("Build completed", "app", b.App, "ids", result.IDs, "loop", result.Loop)

	if result.Error != nil {
		slog.Error("Executing command", "error", result.Error, "command", c)
		os.Exit(1)
	}

	if result.Loop == container.BuildStop {
		slog.Info("Build requested to stop further builds", "app", b.App)
		os.Exit(0)
	}

	idStore.Add(result.IDs...)
}

// executeBuildGroup executes all builds in a group in parallel using goroutines.
// It spawns a goroutine for each build and waits for all to complete before returning.
func executeBuildGroup(group *container.BuildGroup, leader *LeaderElection, idStore *utils.IDStore, addr network.Address) {
	wg := sync.WaitGroup{}

	for _, b := range group.Builds {
		wg.Add(1)
		go func(b *container.Build) {
			defer wg.Done()
			executeBuild(b, leader, idStore, addr)
		}(b)
	}

	slog.Info("Waiting for all builds to complete")
	wg.Wait()
}

func GetBuild(auto bool) container.BuildGroups {
	if auto {
		// Use auto-discovery to detect Go projects
		projects, err := autodiscovery.DiscoverProjects(".")
		if err != nil {
			slog.Error("Auto-discovery failed", "error", err)
			os.Exit(1)
		}
		slog.Info("Auto-discovered projects", "count", len(projects.AllProjects()))
		return autodiscovery.GenerateBuildGroups(projects.AllProjects())
	}

	logger := hclog.New(&hclog.LoggerOptions{
		Level:           hclog.Error,
		Output:          os.Stderr,
		IncludeLocation: true,
	})

	hclog.SetDefault(logger)
	hclog.Default().SetLevel(hclog.Error)

	//TODO - make it configurable to support different languages
	file := os.Getenv("CONTAINIFYCI_FILE")

	path := filepath.Dir(file)
	if file == "" {
		file = "containifyci.go"
	} else {
		file = filepath.Base(file)
	}

	// We don't want to see the plugin logs.
	log.SetOutput(os.Stderr)

	fmt.Printf("go run -C %s %s\n", path, file)

	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: protos2.Handshake,
		// Plugins:          protos2.PluginMap,
		VersionedPlugins: protos2.PluginMap,
		Stderr:           os.Stderr,
		Cmd:              exec.Command("go", "run", "-C", path, file),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
			plugin.ProtocolGRPC,
		},
		Logger: logger,
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		logger.Error("Error:", "error", err.Error())
		os.Exit(1)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("containifyci")
	if err != nil {
		logger.Error("Error:", "error", err.Error())
		os.Exit(1)
	}

	opts := CallPlugin(logger, raw)

	groups := container.BuildGroups{}

	for _, group := range opts {
		g := &container.BuildGroup{}

		for _, opt := range group.Args {
			arg := container.Build{
				App:            opt.Application,
				BuildType:      container.BuildType(opt.BuildType.String()),
				ContainerFiles: opt.ContainerFiles,
				Env:            container.EnvType(opt.Environment.String()),
				Image:          opt.Image,
				ImageTag:       opt.ImageTag,
				Registry:       opt.Registry,
				Registries:     opt.Registries,
				Repository:     opt.Repository,
				Runtime:        cri.DetectContainerRuntime(),
				File:           opt.File,
				Folder:         opt.Folder,
				SourcePackages: opt.SourcePackages,
				SourceFiles:    opt.SourceFiles,
				Organization:   opt.Organization,
				Verbose:        opt.Verbose,
			}

			if opt.Properties != nil {
				arg.Custom = make(map[string][]string)
				for k, v := range opt.Properties {
					arg.Custom[k] = make([]string, len(v.Values))
					for i, l := range v.Values {
						arg.Custom[k][i] = l.GetStringValue()
					}
				}
			}

			if opt.Platform != "" {
				platform := types.Platform{
					Host: types.ParsePlatform(opt.Platform),
				}

				platform.Container = types.GetContainerPlatform(platform.Host)
				arg.Platform = platform
			}

			if len(opt.Secrets) > 0 {
				m := make(map[string]*container.BuildSecret, len(opt.Secrets))
				for _, s := range opt.Secrets {
					m[s.Key] = container.NewBuildSecret(s)
				}
				arg.Secrets = m
			}
			arg.Defaults()
			// args = append(args, &arg)
			g.Builds = append(g.Builds, &arg)
			// slog.Info("Build Arg", "arg", arg)
		}
		groups = append(groups, g)
	}
	return groups
}

func CallPlugin(logger hclog.Logger, plugin interface{}) []*protos2.BuildArgsGroup {
	// We should have a Counter store now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	{
		containifyci, ok := plugin.(protos2.ContainifyCIv2)
		if ok {
			resp, err := containifyci.GetBuilds()
			if err != nil {
				logger.Error("Failed to get builds from plugin", "error", err)
				os.Exit(1)
			}
			return resp.Args
		}
	}
	{
		containifyci, ok := plugin.(protos2.ContainifyCIv1)
		if !ok {
			logger.Error("Can't use v1 plugin version.")
			os.Exit(1)
		}
		resp, err := containifyci.GetBuild()
		if err != nil {
			logger.Error("Failed to get build from plugin", "error", err)
			os.Exit(1)
		}

		groups := []*protos2.BuildArgsGroup{}
		for _, a := range resp.Args {
			groups = append(groups, &protos2.BuildArgsGroup{
				Args: []*protos2.BuildArgs{a},
			})
		}
		return groups
	}
}

// GetDefaultBuildSteps returns the current BuildSteps instance with all default build steps.
// This allows engines to extend the default pipeline instead of replacing it completely.
func GetDefaultBuildSteps(arg *container.Build) *build.BuildSteps {
	if buildSteps.IsNotInit() {
		InitBuildSteps()
	}
	return buildSteps
}



func InitBuildSteps() {
	if buildSteps.IsNotInit() {
		slog.Info("Registering all build steps by category")

		// Helper function to add step and log error
		addStep := func(category build.BuildCategory, step build.BuildStep) {
			var err error
			if step.IsAsync() {
				err = buildSteps.AddAsyncToCategory(category, step)
			} else {
				err = buildSteps.AddToCategory(category, step)
			}
			if err != nil {
				slog.Error("Failed to add build step", "step", step.Name(), "category", category, "error", err)
			}
		}

		// Auth: Authentication & credentials
		addStep(build.Auth, gcloud.New())

		// PreBuild: Setup, protobuf, dependencies
		addStep(build.PreBuild, copier.New())
		addStep(build.PreBuild, protobuf.New())
		addStep(build.PreBuild, packer.New())

		// Build: Language-specific compilation
		addStep(build.Build, golang.New())       // Alpine variant
		addStep(build.Build, golang.NewDebian()) // Debian variant
		addStep(build.Build, golang.NewCGO())    // CGO variant
		addStep(build.Build, maven.New())        // Maven
		addStep(build.Build, python.New())       // Python
		addStep(build.Build, zig.New())          // Zig

		// PostBuild: Production artifacts, packaging
		addStep(build.PostBuild, golang.NewProd())       // Alpine prod
		addStep(build.PostBuild, golang.NewProdDebian()) // Debian prod
		addStep(build.PostBuild, maven.NewProd())        // Maven prod
		addStep(build.PostBuild, python.NewProd())       // Python prod
		addStep(build.PostBuild, zig.NewProd())          // Zig prod

		// Quality: Linting, testing, security scanning
		addStep(build.Quality, golang.NewLinter()) // Golang linter (async)
		addStep(build.Quality, sonarcloud.New())   // SonarCloud (async)
		addStep(build.Quality, trivy.New())        // Trivy

		// Apply: Infrastructure changes
		addStep(build.Apply, pulumi.New()) // Pulumi


		// Publish: Publishing, releases, notifications
		addStep(build.Publish, goreleaser.New()) // Goreleaser
		addStep(build.Publish, github.New())     // GitHub (async)

		addStep(build.Publish, dummy.New()) // Goreleaser
		buildSteps.Init()
	}
}
