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

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/protos2"
	"github.com/spf13/cobra"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

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
		slog.Info("Process is not the leader. Waiting...\n", "id",id)
	}
}

func Engine(cmd *cobra.Command, _ []string) error {
	leader := LeaderElection{}
	fnc, addr := Start()
	defer fnc()
	arg := GetBuild()
	wg := sync.WaitGroup{}
	for _, a := range arg {
		wg.Add(1)
		go func() {
			time.Sleep(1 * time.Second)
			defer wg.Done()
			a.Leader = &leader
			c := NewCommand(*a)
			c.Run(addr, RootArgs.Target, a)
		}()
	}
	slog.Info("Waiting for all builds to complete")
	wg.Wait()
	slog.Info("Finish waiting for all builds to complete")

	return nil
}

func GetBuild() []*container.Build {
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
		Plugins:         protos2.PluginMap,
		Stderr:          os.Stderr,
		Cmd:             exec.Command("go", "run", "-C", path, file),
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

	// We should have a Counter store now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	containifyci := raw.(protos2.ContainifyCI)

	resp := containifyci.GetBuild()
	opts := resp.Args

	args := []*container.Build{}
	for _, opt := range opts {

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
		args = append(args, &arg)
		fmt.Printf("Builds: %v\n", arg)
	}
	return args
}
