package cmd

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/kv"
	"github.com/containifyci/engine-ci/pkg/logger"
	"github.com/containifyci/engine-ci/pkg/network"
	"github.com/containifyci/engine-ci/pkg/svc"

	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Run: RunBuild,
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
	},
}

func init() {
	var buildArgs = &container.Build{}

	rootCmd.AddCommand(buildCmd)

	buildCmd.PersistentFlags().StringVarP(&buildArgs.App, "app", "a", "", "Application binary to build")
	_ = buildCmd.MarkPersistentFlagRequired("app")

	buildCmd.PersistentFlags().Var(&buildArgs.BuildType, "type", "Build type to run for example go, maven, generic")

	buildCmd.PersistentFlags().VarP(&buildArgs.Env, "env", "e", "Set the environment the build runs in")

	buildCmd.PersistentFlags().StringVarP(&buildArgs.Image, "image", "i", "", "Image to build")
	buildCmd.PersistentFlags().StringVar(&buildArgs.ImageTag, "tag", "", "Image tag to build")
	buildCmd.PersistentFlags().StringVarP(&buildArgs.Registry, "repo", "r", "containifyci", "the image repository to use")

	// golang
	buildCmd.PersistentFlags().StringVarP(&buildArgs.File, "file", "f", "/src/main.go", "Main file to build")
	buildCmd.PersistentFlags().StringVar(&buildArgs.Folder, "folder", "./", "Folder to build")

	// protobuf
	buildCmd.PersistentFlags().StringSliceVar(&buildArgs.SourcePackages, "protobuf-packages", nil, "package locations containing the proto files")
	buildCmd.PersistentFlags().StringSliceVar(&buildArgs.SourceFiles, "protobuf-files", nil, "protot file locations")

	// sonarcloud
	buildCmd.PersistentFlags().StringVarP(&buildArgs.Organization, "org", "o", "containifyci", "Organization to use for sonarcloud")

	buildCmd.PersistentFlags().BoolVarP(&buildArgs.Verbose, "verbose", "v", false, "Enable verbose logging")
}

func Init(arg *container.Build) *container.Build {
	bld := container.NewBuild(arg)

	logOpts := slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   false,
		ReplaceAttr: nil,
	}
	if bld.Verbose || RootArgs.Verbose {
		logOpts.Level = slog.LevelDebug
		logOpts.AddSource = true
	}

	logger := slog.New(logger.New(RootArgs.Progress, logOpts))
	slog.SetDefault(logger)

	git, err := svc.SetGitInfo()
	if err != nil {
		slog.Warn("Failed to get git info", "error", err)
		// os.Exit(1)
		git = svc.SetUnknowGitInfo()
	}
	slog.Info("Starting build", "build", bld, "git", git)
	return arg
}

func (c *Command) Pre() *container.Build {
	a := Pre(c.buildArgs)
	return a
}

func Pre(arg *container.Build) *container.Build {
	slog.Info("Pre build", "args", arg)
	a := Init(arg)
	return a
}

type Command struct {
	targets    map[string]func() ([]string, container.BuildLoop, error)
	buildArgs  *container.Build
	buildSteps *build.BuildSteps
}

func NewCommand(_buildArgs container.Build, _buildSteps *build.BuildSteps) *Command {
	_buildArgs.Defaults()
	return &Command{
		targets:    map[string]func() ([]string, container.BuildLoop, error){},
		buildArgs:  &_buildArgs,
		buildSteps: _buildSteps,
	}
}

func (c *Command) AddTarget(name string, fnc func() ([]string, container.BuildLoop, error)) {
	if _, ok := c.targets[name]; ok {
		slog.Info("Skip Overwriting target", "target", name)
		return
	}
	c.targets[name] = func() ([]string, container.BuildLoop, error) {
		slog.Info("Running custom target", "target", name)
		return fnc()
	}
}

func Start() (func(), network.Address, error) {
	srv, fnc, err := kv.StartHttpServer(kv.NewKeyValueStore())
	if err != nil {
		return nil, network.Address{}, fmt.Errorf("failed to start http server: %w", err)
	}
	go fnc()
	slog.Info("Started http server", "address", srv.Listener.Addr().String())
	addr := network.Address{Host: "localhost", Port: srv.Port, Secret: srv.Secret}
	return func() {
		slog.Info("Stopping http server")
		_ = srv.Listener.Close()
	}, addr, nil
}

func (c *Command) Run(addr network.Address, target string, arg *container.Build) ([]string, container.BuildLoop, error) {
	if arg.Custom == nil {
		arg.Custom = make(map[string][]string)
	}
	arg.Custom["CONTAINIFYCI_EXTERNAL_HOST"] = []string{fmt.Sprintf("%s:%d", addr.Host, addr.Port)}
	arg.Custom["CONTAINIFYCI_HOST"] = []string{fmt.Sprintf("%s:%d", addr.ForContainerDefault(arg), addr.Port)}
	arg.Secret = map[string]string{"CONTAINIFYCI_AUTH": addr.Secret}
	_ = Pre(arg)
	for _, b := range c.buildSteps.Steps {
		if b.Build().BuildType() == nil || *b.Build().BuildType() == arg.BuildType {
			slog.Info("Register Step", "step", b.Build().Name(), "buildtype", b.Build().BuildType(), "argtype", arg.BuildType)
			c.AddTarget(b.Build().Alias(), func() ([]string, container.BuildLoop, error) {
				return c.buildSteps.Run(arg, b.Build().Name())
			})
		}
	}
	c.AddTarget("all", func() ([]string, container.BuildLoop, error) {
		return c.buildSteps.Run(arg)
	})
	c.AddTarget("github_actions", func() ([]string, container.BuildLoop, error) {
		return []string{}, container.BuildContinue, RunGithubAction()
	})
	c.AddTarget("docker_load", func() ([]string, container.BuildLoop, error) {
		return []string{}, container.BuildContinue, LoadCache()
	})
	c.AddTarget("docker_save", func() ([]string, container.BuildLoop, error) {
		return []string{}, container.BuildContinue, SaveCache()
	})
	c.AddTarget("list", func() ([]string, container.BuildLoop, error) {
		keys := []string{}
		for k := range c.targets {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		slog.Info("Available targets", "targets", strings.Join(keys, " "))
		return []string{}, container.BuildContinue, nil
	})

	if fnc, ok := c.targets[target]; ok {
		ids, loop, err := fnc()
		if err != nil {
			slog.Error("Failed to run command", "error", err)
			return ids, container.BuildContinue, err
		}
		return ids, loop, nil
	}
	keys := []string{}
	for k := range c.targets {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return []string{}, container.BuildStop, fmt.Errorf("unknown target: %s (available: %s)", target, strings.Join(keys, " "))
}
