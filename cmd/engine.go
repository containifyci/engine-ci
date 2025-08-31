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

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/protos2"
	"github.com/spf13/cobra"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

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
	fnc, addr := Start()
	defer fnc()
	arg := GetBuild()
	wg := sync.WaitGroup{}
	for _, a := range arg {
		for _, b := range a.Builds {
			wg.Add(1)
			go func() {
				time.Sleep(1 * time.Second)
				defer wg.Done()
				b.Leader = &leader
				_buildSteps := buildSteps
				slog.Info("Starting build", "build", b, "steps", _buildSteps.String())
				if _buildSteps == nil {
					_buildSteps = build.NewBuildSteps()
				}
				slog.Info("Starting build2", "build", b, "steps", _buildSteps.String())
				c := NewCommand(*b, _buildSteps)
				c.Run(addr, RootArgs.Target, b)
			}()
		}
		slog.Info("Waiting for all builds to complete")
		wg.Wait()
	}
	slog.Info("Finish waiting for all builds to complete")

	return nil
}

func GetBuild() container.BuildGroups {
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

	// args := []*container.Build{}
	for _, group := range opts {
		g := &container.BuildGroup{}

		for _, opt := range group.Args {
			arg := container.Build{
				App:            opt.Application,
				BuildType:      container.BuildType(opt.BuildType.String()),
				Env:            container.EnvType(opt.Environment.String()),
				Image:          opt.Image,
				ImageTag:       opt.ImageTag,
				Registry:       opt.Registry,
				Registries:     opt.Registries,
				Repository:     opt.Repository,
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
			arg.Defaults()
			// args = append(args, &arg)
			fmt.Printf("Builds: %v\n", arg)
			g.Builds = append(g.Builds, &arg)
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

// InitBuildSteps can be used to set the build steps for the build command
// This is useful for registering a new build step as part of a extension
// of the engine-ci with to support new build types for different languages
// or to customize the build steps for a specific project.
func InitBuildSteps(_buildSteps *build.BuildSteps) *build.BuildSteps {
	//TODO make it possible to register new build steps without the need of a gloabl variable
	buildSteps = _buildSteps
	// return buildSteps
	return _buildSteps
}
