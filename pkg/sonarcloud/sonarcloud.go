package sonarcloud

import (
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/filesystem"
	"github.com/containifyci/engine-ci/pkg/network"
	"github.com/containifyci/engine-ci/pkg/svc"
)

const (
	// IMAGE = "sonarsource/sonar-scanner-cli"
	// IMAGE = "sonar:latest"
	IMAGE = "containifyci/sonar"
)

type SonarcloudContainer struct {
	*container.Container
}

func New() *SonarcloudContainer {
	return &SonarcloudContainer{
		Container: container.New(container.BuildEnv),
	}
}

func (c *SonarcloudContainer) IsAsync() bool {
	return true
}

func (c *SonarcloudContainer) Name() string {
	return "sonarcloud"
}

func (c *SonarcloudContainer) Images() []string {
	return []string{IMAGE}
}

func CacheFolder() string {
	dir := os.Getenv("CONTAINIFYCI_CACHE")
	if dir == "" {
		usr, _ := user.Current()
		dir = usr.HomeDir
	}
	folder, err := filepath.Abs(filepath.Join(dir, "/.sonar/cache"))
	if err != nil {
		slog.Error("Failed to get cache folder: %s", "error", err)
		os.Exit(1)
	}

	err = os.MkdirAll(folder, os.ModePerm)
	if err != nil {
		slog.Error("Failed to create cache folder: %s", "error", err)
		os.Exit(1)
	}
	slog.Info("Cache folder", "folder", folder)

	return folder
}

func (c *SonarcloudContainer) CopyScript() error {
	// Create a temporary script in-memory
	script := `#!/bin/sh
set -xe
sonar-scanner -X -Dsonar.projectBaseDir=/usr/src
`
	err := c.Container.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func (c *SonarcloudContainer) Analyze(env container.EnvType, token *string, address *network.Address) error {
	if token == nil || *token == "" {
		_token := container.GetEnv("SONAR_TOKEN")
		if _token == "" {
			slog.Warn("SONAR_TOKEN is not set skip sonar analysis")
			return nil
		}
		token = &_token
	}

	options := []string{
		fmt.Sprintf("-Dsonar.host.url=%s", address.ForContainer(env)),
		"-Dsonar.scm.disabled=true",
	}

	// TODO get branch name from git
	if env == container.LocalEnv {
		options = append(options,
			// Needs Developer Edition
			"-Dsonar.branch.name=",
			"-Dsonar.branch.target=",
		)
	} else if env == container.BuildEnv {
		if svc.GitInfo().PrNum == "" {
			options = append(options,
				fmt.Sprintf("-Dsonar.branch.name=%s", svc.GitInfo().Branch),
			)
		} else {
			options = append(options,
				fmt.Sprintf("-Dsonar.pullrequest.branch=%s", svc.GitInfo().Branch),
				fmt.Sprintf("-Dsonar.pullrequest.key=%s", svc.GitInfo().PrNum),
			)
		}
	}

	options = append(options, fmt.Sprintf("-Dsonar.verbose=%t", c.Container.Verbose))

	if !filesystem.FileExists("sonar-project.properties") {
		options = append(options,
			fmt.Sprintf("-Dsonar.projectKey=%s_%s", container.GetBuild().Organization, container.GetBuild().App),
			fmt.Sprintf("-Dsonar.projectName=%s", container.GetBuild().App),
			//TODO: get sonar organization from env
			"-Dsonar.organization=xxx",
		)
		switch container.GetBuild().BuildType {
		case container.GoLang:
			options = append(options,
				"-Dsonar.go.coverage.reportPaths=coverage.txt",
				"-Dsonar.exclusions=**/*.pb.go",
				"-Dsonar.sources=.",
				"-Dsonar.tests=.",
				"-Dsonar.test.inclusions=**/*_test.go",
				"-Dsonar.coverage.exclusions=**/*.pb.go,**/*_test.go",
				"-Dsonar.language=java",
				"-Dsonar.java.binaries=target/classes",
			)
		case container.Maven:
			options = append(options,
				"-Dsonar.language=kotlin",
				"-Dsonar.tests=src/test/java",
				"-Dsonar.exclusions=target/**",
				"-Dsonar.java.binaries=target/classes",
				"-Dsonar.coverage.jacoco.xmlReportPaths=target/jacoco-report/jacoco.xml",
			)
		}
	}

	opts := types.ContainerConfig{}
	opts.Image = IMAGE
	//FIX: this should fix the permission issue with the mounted cache folder
	opts.User = "root"

	cache := CacheFolder()

	dir, _ := filepath.Abs(".")
	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/usr/src",
		},
		//TODO: Fix accessing problem from Github Action Unable to create temp dir/opt/sonar-scanner/.sonar/cache/_tmp
		{
			Type:   "bind",
			Source: cache,
			Target: "/opt/sonar-scanner/.sonar/cache",
		},
	}

	opts.Cmd = []string{"sh", "/tmp/script.sh"}
	// opts.Cmd = []string{"sonar-scanner", "-Dsonar.projectBaseDir=/usr/src"}
	opts.Env = []string{fmt.Sprintf("SONAR_SCANNER_OPTS=%s", strings.Join(options, " ")), fmt.Sprintf("SONAR_TOKEN=%s", *token)}
	err := c.Container.Create(opts)
	if err != nil {
		return err
	}

	err = c.CopyScript()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Container.Start()
	if err != nil {
		return err
	}

	return c.Container.Wait()
}

func (c *SonarcloudContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *SonarcloudContainer) Run() error {
	slog.Info("Run sonarcloud")
	env := container.GetBuild().Env
	sonarqube := NewSonarQube()

	if env == container.LocalEnv {
		err := sonarqube.Start()
		if err != nil {
			slog.Error("Failed to start sonarqube container: %s", "error", err)
			os.Exit(1)
		}
	}

	err := c.Pull()
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}

	err = c.Analyze(env, sonarqube.Token(), sonarqube.Address())
	if err != nil {
		slog.Error("Failed to create container: %s", "error", err)
		os.Exit(1)
	}
	return nil
}
