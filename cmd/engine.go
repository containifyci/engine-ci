package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/protos2"
	"github.com/spf13/cobra"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

type engineArgs struct {
	Target string
}

var engineArg = engineArgs{}

// buildCmd represents the build command
var engineCmd = &cobra.Command{
	Use:   "engine",
	Short: "Command to generate Github Action file for gflip",
	Long: `Command to generate Github Action file for gflip.
`,
	RunE: Engine,
}

func init() {
	engineCmd.PersistentFlags().StringVarP(&engineArg.Target, "target", "t", "all", "The build target to run")

	rootCmd.AddCommand(engineCmd)
}

func Engine(cmd *cobra.Command, args []string) error {
	arg := GetBuild()
	for _, a := range arg {
		c := NewCommand(*a)
		c.Run(engineArg.Target, a)
	}
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
