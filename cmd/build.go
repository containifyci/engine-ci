package cmd

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/copier"
	"github.com/containifyci/engine-ci/pkg/dummy"
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
	container.InitRuntime(arg)
	return arg
}

func (c *Command) Pre() (*container.Build, *build.BuildSteps) {
	a, bs := Pre(c.buildArgs, c.buildSteps)
	return a, bs
}

func Pre(arg *container.Build, bs *build.BuildSteps) (*container.Build, *build.BuildSteps) {
	slog.Info("Pre build", "args", arg)
	a := Init(arg)

	if bs == nil {
		bs = build.NewBuildSteps()
	}
	// buildSteps := build.NewBuildSteps()
	// slog.Info("Build steps", "buildSteps", buildSteps)
	// slog.Info("Build steps", "buildSteps Check", buildSteps == nil)
	// if buildSteps == nil {
	// 	buildSteps = build.NewBuildSteps()
	// }
	// bs := build.NewBuildSteps()

	if bs.IsNotInit() {
		slog.Info("Registering all build steps by category", "build", *a)

		// Helper function to add step and log error
		addStep := func(category build.BuildCategory, step build.BuildStepv2) {
			var err error
			if step.IsAsync() {
				err = bs.AddAsyncToCategory(category, step)
			} else {
				err = bs.AddToCategory(category, step)
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

		// Build: Language-specific compilation
		addStep(build.Build, golang.New())       // Alpine variant
		addStep(build.Build, golang.NewDebian()) // Debian variant
		addStep(build.Build, golang.NewCGO())    // CGO variant
		addStep(build.Build, maven.New())        // Maven
		addStep(build.Build, python.New())       // Python

		// PostBuild: Production artifacts, packaging
		addStep(build.PostBuild, golang.NewProd())       // Alpine prod
		addStep(build.PostBuild, golang.NewProdDebian()) // Debian prod
		addStep(build.PostBuild, maven.NewProd())        // Maven prod
		addStep(build.PostBuild, python.NewProd())       // Python prod

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

		bs.Init()
	}

	bs.PrintSteps()
	return a, bs
}

type Command struct {
	targets    map[string]func() error
	buildArgs  *container.Build
	buildSteps *build.BuildSteps
}

func NewCommand(_buildArgs container.Build, _buildSteps *build.BuildSteps) *Command {
	_buildArgs.Defaults()
	return &Command{
		targets:    map[string]func() error{},
		buildArgs:  &_buildArgs,
		buildSteps: _buildSteps,
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

func (c *Command) Run(addr network.Address, target string, arg *container.Build) error {
	if arg.Custom == nil {
		arg.Custom = make(map[string][]string)
	}
	arg.Custom["CONTAINIFYCI_HOST"] = []string{fmt.Sprintf("%s:%d", addr.ForContainerDefault(arg), addr.Port)}
	arg.Secret = map[string]string{"CONTAINIFYCI_AUTH": addr.Secret}
	_, bs := Pre(arg, c.buildSteps)
	switch arg.BuildType {
	case container.GoLang:
		c.AddTarget("lint", func() error {
			return bs.Run(arg, "golangci-lint")
		})
		c.AddTarget("build", func() error {
			return bs.Run(arg, "golang")
		})
		c.AddTarget("push", func() error {
			return bs.Run(arg, "golang-prod")
		})
		c.AddTarget("protobuf", func() error {
			return bs.Run(arg, "protobuf")
		})
		c.AddTarget("release", func() error {
			return bs.Run(arg, "gorelease")
		})
		c.AddTarget("pulumi", func() error {
			return bs.Run(arg, "pulumi")
		})
	case container.Maven:
		c.AddTarget("build", func() error {
			return bs.Run(arg, "maven")
		})
		c.AddTarget("push", func() error {
			return bs.Run(arg, "maven-prod")
		})
	case container.Python:
		c.AddTarget("build", func() error {
			return bs.Run(arg, "python")
		})
		c.AddTarget("push", func() error {
			return bs.Run(arg, "python-prod")
		})
		c.AddTarget("release", func() error {
			return bs.Run(arg, "dummy")
		})
	}
	c.AddTarget("all", func() error {
		return bs.Run(arg)
	})
	c.AddTarget("sonar", func() error {
		return bs.Run(arg, "sonarcloud")
	})
	c.AddTarget("trivy", func() error {
		return bs.Run(arg, "trivy")
	})
	c.AddTarget("gcloud_oidc", func() error {
		return bs.Run(arg, "gcloud_oidc")
	})
	c.AddTarget("github", func() error {
		return bs.Run(arg, "github")
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
		slog.Info("Available targets", "targets", strings.Join(keys, " "))
		return nil
	})

	if fnc, ok := c.targets[target]; ok {
		if err := fnc(); err != nil {
			slog.Error("Failed to run command", "error", err)
			return err
		}
		return nil
	}
	keys := []string{}
	for k := range c.targets {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return fmt.Errorf("unknown target: %s (available: %s)", target, strings.Join(keys, " "))
}

// func (c *Command) Main(arg container.Build) {
// 	if len(os.Args) < 2 {
// 		fmt.Print("Usage: containifyci <command>\n")
// 		fmt.Print("Available commands: all, lint, build, push, sonar, protobuf\n")
// 		os.Exit(1)
// 	}
// 	c.Run(os.Args[1], buildArgs)
// }
