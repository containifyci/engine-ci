package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/gcloud"
	"github.com/containifyci/engine-ci/pkg/github"
	"github.com/containifyci/engine-ci/pkg/golang"
	"github.com/containifyci/engine-ci/pkg/goreleaser"
	"github.com/containifyci/engine-ci/pkg/kv"
	"github.com/containifyci/engine-ci/pkg/logger"
	"github.com/containifyci/engine-ci/pkg/maven"
	"github.com/containifyci/engine-ci/pkg/network"
	"github.com/containifyci/engine-ci/pkg/protobuf"
	"github.com/containifyci/engine-ci/pkg/pulumi"
	"github.com/containifyci/engine-ci/pkg/python"
	"github.com/containifyci/engine-ci/pkg/sonarcloud"
	"github.com/containifyci/engine-ci/pkg/svc"
	"github.com/containifyci/engine-ci/pkg/trivy"

	"github.com/spf13/cobra"
)

var buildSteps = build.NewBuildSteps()

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
	// var arg *container.Build
	// if len(args) <= 0 {
	// 	arg = buildArgs
	// } else {
	// arg = args[0]
	// }
	bld := container.NewBuild(arg)

	logOpts := slog.HandlerOptions{
		Level:       slog.LevelDebug,
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
	container.InitRuntime(arg)
	return arg
}

func (c *Command) Pre() (*container.Build, *build.BuildSteps) {
	a, bs := Pre(c.buildArgs)
	return a, bs
}

func Pre(arg *container.Build) (*container.Build, *build.BuildSteps) {
	slog.Info("Pre build", "args", arg)
	a := Init(arg)
	// bs := build.NewBuildSteps()
	bs := buildSteps

	var from string
	if v, ok := a.Custom["from"]; ok {
		slog.Info("Using custom build", "from", v[0])
		from = v[0]
	}

	if bs.IsNotInit() {
		slog.Info("Init build steps", "build", *a)
		bs.Add(gcloud.New(*a))
		switch a.BuildType {
		case container.GoLang:
			bs.Add(protobuf.New(*a))
			//TODO: register different build images automatically or at least in the build implementation itself
			switch from {
			case "debian":
				bs.Add(golang.NewDebian(*a))
				bs.Add(golang.NewProdDebian(*a))
			case "debiancgo":
				bs.Add(golang.NewCGO(*a))
				bs.Add(golang.NewProdDebian(*a))
			default:
				bs.Add(golang.New(*a))
				bs.Add(golang.NewProd(*a))
			}
			bs.AddAsync(golang.NewLinter(*a))
			bs.Add(goreleaser.New(*a))
			bs.Add(pulumi.New(*a))
		case container.Maven:
			bs.Add(maven.New(*a))
			bs.Add(maven.NewProd(*a))
		case container.Python:
			bs.Add(python.New(*a))
			bs.Add(python.NewProd(*a))
		}
		bs.AddAsync(sonarcloud.New(*a))
		bs.Add(trivy.New(*a))
		bs.AddAsync(github.New(*a))
		bs.Init()
	}

	bs.PrintSteps()
	return a, bs
}

// func RunBuild(_ *cobra.Command, _ []string) {
// 	fmt.Println("build called2")
// 	_, bs := Pre(buildArgs)
// 	err := bs.Run()
// 	if err != nil {
// 		slog.Error("Failed to build", "error", err)
// 		os.Exit(1)
// 	}
// }

func (c *Command) RunBuild() {
	fmt.Println("build called")
	_, bs := c.Pre()
	err := bs.Run()
	if err != nil {
		slog.Error("Failed to build", "error", err)
		os.Exit(1)
	}
}

type Command struct {
	targets   map[string]func() error
	buildArgs *container.Build
}

func NewCommand(_buildArgs container.Build) *Command {
	// buildArgs = &_buildArgs
	_buildArgs.Defaults()
	return &Command{
		targets:   map[string]func() error{},
		buildArgs: &_buildArgs,
	}
}

func (c *Command) AddTarget(name string, fnc func() error) {
	if _, ok := c.targets[name]; ok {
		slog.Info("Skip Overwriting target", "target", name)
		return
	}
	c.targets[name] = func() error {
		slog.Info("Running custom target", "target", name)
		return fnc()
	}
}

func Start() (func(), network.Address) {
	srv, fnc, err := kv.StartHttpServer(kv.NewKeyValueStore())
	if err != nil {
		slog.Error("Failed to start http server", "error", err)
		os.Exit(1)
	}
	go fnc()
	slog.Info("Started http server", "address", srv.Listener.Addr().String())
	addr := network.Address{Host: "localhost", Port: srv.Port}
	// if arg.Custom == nil {
	// 	arg.Custom = make(map[string][]string)
	// }
	// arg.Custom["CONTAINIFYCI_HOST"] = []string{fmt.Sprintf("%s:%d", addr.ForContainerDefault(arg), srv.Port)}
	return func() {
		slog.Info("Stopping http server")
		srv.Listener.Close()

		// buildSteps = build.NewBuildSteps()
	}, addr
}

func (c *Command) Run(addr network.Address, target string, arg *container.Build) {
	if arg.Custom == nil {
		arg.Custom = make(map[string][]string)
	}
	arg.Custom["CONTAINIFYCI_HOST"] = []string{fmt.Sprintf("%s:%d", addr.ForContainerDefault(arg), addr.Port)}
	_, bs := Pre(arg)
	switch arg.BuildType {
	case container.GoLang:
		c.AddTarget("lint", func() error {
			return bs.Run("golangci-lint")
		})
		c.AddTarget("build", func() error {
			return bs.Run("golang")
		})
		c.AddTarget("push", func() error {
			return bs.Run("golang-prod")
		})
		c.AddTarget("protobuf", func() error {
			return bs.Run("protobuf")
		})
		c.AddTarget("release", func() error {
			return bs.Run("gorelease")
		})
		c.AddTarget("pulumi", func() error {
			return bs.Run("pulumi")
		})
	case container.Maven:
		c.AddTarget("build", func() error {
			return bs.Run("maven")
		})
		c.AddTarget("push", func() error {
			return bs.Run("maven-prod")
		})
	case container.Python:
		c.AddTarget("build", func() error {
			return bs.Run("python")
		})
		c.AddTarget("push", func() error {
			return bs.Run("python-prod")
		})
	}
	c.AddTarget("all", func() error {
		// return All(arg)
		c.RunBuild()
		return nil
	})
	c.AddTarget("sonar", func() error {
		return bs.Run("sonarcloud")
	})
	c.AddTarget("trivy", func() error {
		return bs.Run("trivy")
	})
	c.AddTarget("gcloud_oidc", func() error {
		return bs.Run("gcloud_oidc")
	})
	c.AddTarget("github", func() error {
		return bs.Run("github")
	})
	c.AddTarget("github_actions", func() error {
		return RunGithubAction()
	})
	c.AddTarget("docker_load", func() error {
		return LoadCache()
	})
	c.AddTarget("docker_save", func() error {
		return SaveCache()
	})
	c.AddTarget("list", func() error {
		keys := []string{}
		for k := range c.targets {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		fmt.Printf("Available targets: %s\n", strings.Join(keys, " "))
		return nil
	})

	if fnc, ok := c.targets[target]; ok {
		err := fnc()
		if err != nil {
			hclog.Default().Debug("Failed to run command", "error", err)

			slog.Error("Failed to run command", "error", err)
			os.Exit(1)
		}
	} else {
		keys := []string{}
		for k := range c.targets {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		hclog.Default().Debug("Unknown target", "target", target, "available", keys)

		fmt.Printf("Unknown target: %s, available targets: %s\n", target, strings.Join(keys, " "))
		os.Exit(1)
	}
}

// func (c *Command) Main(arg container.Build) {
// 	if len(os.Args) < 2 {
// 		fmt.Print("Usage: containifyci <command>\n")
// 		fmt.Print("Available commands: all, lint, build, push, sonar, protobuf\n")
// 		os.Exit(1)
// 	}
// 	c.Run(os.Args[1], buildArgs)
// }

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
