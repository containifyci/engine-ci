package github

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/kv"
	"github.com/containifyci/engine-ci/pkg/svc"
	"github.com/containifyci/engine-ci/pkg/trivy"
)

const (
	kvKeyGithubToken   = "github_token"
	kvKeyCommitMessage = "commit_message"
)

//go:embed Dockerfile*
var f embed.FS

const (
	IMAGE = "maniator/gh"
)

type GithubContainer struct {
	git *svc.Git
	*container.Container
}

// Matches implements the Build interface - GitHub runs for all builds
func Matches(build container.Build) bool {
	return true // GitHub integration runs for all builds
}

func New() build.BuildStepv3 {
	return build.Stepper{
		RunFn: func(build container.Build) error {
			container := new(build)
			return container.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  Images,
		Name_:     "github",
		Alias_:    "github",
		Async_:    true,
	}
}

func new(build container.Build) *GithubContainer {
	return &GithubContainer{
		Container: container.New(build),
		git:       svc.GitInfo(),
	}
}

func Images(build container.Build) []string {
	return []string{Image(build)}
}

func (c *GithubContainer) CopyScript() error {
	// Create a temporary script in-memory
	script := fmt.Sprintf(`#!/bin/sh
set -xe
gh pr comment %s --repo %s --edit-last --body-file /src/trivy.md || gh pr comment %s --repo %s --body-file /src/trivy.md
`, c.git.PrNum, c.git.FullRepo(), c.git.PrNum, c.git.FullRepo())
	err := c.CopyContentTo(script, "/tmp/script.sh")
	if err != nil {
		slog.Error("Failed to copy script to container: %s", "error", err)
		os.Exit(1)
	}
	return err
}

func Image(build container.Build) string {
	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile.go", "error", err)
		os.Exit(1)
	}
	tag := container.ComputeChecksum(dockerFile)
	return utils.ImageURI(build.ContainifyRegistry, "gh", tag)

	// return fmt.Sprintf("%s/%s/%s:%s", container.GetBuild().Registry, "containifyci", "gh", tag)
}

func (c *GithubContainer) BuildImage() error {
	image := Image(*c.GetBuild())

	dockerFile, err := f.ReadFile("Dockerfile")
	if err != nil {
		slog.Error("Failed to read Dockerfile", "error", err)
		os.Exit(1)
	}

	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building intermediate image", "image", image, "platforms", platforms)

	return c.BuildIntermidiateContainer(image, dockerFile, platforms...)
}

func (c *GithubContainer) Comment() error {
	opts := types.ContainerConfig{}
	opts.Image = Image(*c.GetBuild())
	//FIX: this should fix the permission issue with the mounted cache folder
	// opts.User = "root"

	// cache := CacheFolder()

	dir, _ := filepath.Abs(".")
	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/src",
		},
	}

	// Open the JSON file
	file, err := os.ReadFile("trivy.json")
	if err != nil {
		slog.Error("Failed to open JSON file", "error", err)
		os.Exit(1)
	}

	comment, err := trivy.Parse(string(file))
	if err != nil {
		slog.Error("Failed to parse trivy JSON", "error", err)
		return err
	}

	err = os.WriteFile("trivy.md", []byte(comment), 0644)
	if err != nil {
		slog.Error("Failed to write JSON file", "error", err)
		os.Exit(1)
	}

	opts.Cmd = []string{"sh", "/tmp/script.sh"}
	// opts.Cmd = []string{"pr", "comment", "4", "--repo", "containifyci/engine-ci-example", "--edit-last", "--body-file", "/src/trivy.json"}
	opts.Env = []string{"GITHUB_TOKEN=" + container.GetEnv("CONTAINIFYCI_GITHUB_TOKEN")}
	err = c.Create(opts)
	if err != nil {
		return err
	}

	err = c.CopyScript()
	if err != nil {
		slog.Error("Failed to start container", "error", err)
		os.Exit(1)
	}

	err = c.Start()
	if err != nil {
		return err
	}

	return c.Wait()
}

func (c *GithubContainer) Pull() error {
	return c.Container.Pull(IMAGE)
}

func (c *GithubContainer) Run() error {
	shouldComment := c.git.IsPR() && ifTrivyFileExists()
	shouldCommit := c.git.IsPR() && c.shouldCommit()

	if !shouldComment && !shouldCommit {
		slog.Info("Skip github step - no PR comment needed and no commit required")
		return nil
	}

	// Build image once (shared between operations)
	if err := c.BuildImage(); err != nil {
		slog.Error("Failed to build github image", "error", err)
		return err
	}

	// Operation 1: PR Comment (existing behavior)
	if shouldComment {
		if err := c.Comment(); err != nil {
			slog.Error("Failed to comment on PR", "error", err)
			// Continue to commit even if comment fails
		}
	}

	// Operation 2: Commit (new behavior)
	if shouldCommit {
		if err := c.Commit(); err != nil {
			return fmt.Errorf("failed to commit changes: %w", err)
		}
	}

	return nil
}

// shouldCommit returns true if auto_commit is enabled and there are uncommitted changes
func (c *GithubContainer) shouldCommit() bool {
	autoCommit := c.GetBuild().Custom.Bool("auto_commit", false)
	if !autoCommit {
		return false
	}

	if !hasUncommittedChanges() {
		slog.Info("Skip commit - no uncommitted changes")
		return false
	}

	return true
}

// Commit creates a git commit with changes and pushes to the remote
func (c *GithubContainer) Commit() error {
	host := c.GetBuild().Custom.String("CONTAINIFYCI_EXTERNAL_HOST")
	auth := c.GetBuild().Secret["CONTAINIFYCI_AUTH"]

	// Get GitHub token - prefer PAT for workflow triggers
	token := container.GetEnvs("CONTAINIFYCI_PAT_TOKEN", "CONTAINIFYCI_GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("no GitHub token (either CONTAINIFYCI_PAT_TOKEN or CONTAINIFYCI_GITHUB_TOKEN) available for commit")
	}

	// Store GitHub token in KV (fetched inside container for security)
	if err := kv.SetValue(host, auth, kvKeyGithubToken, token); err != nil {
		return fmt.Errorf("failed to store github token: %w", err)
	}

	// Get commit message from KV (set by upstream step like Claude)
	commitMsg, err := kv.GetValue(host, auth, kvKeyCommitMessage)
	if err != nil {
		slog.Warn("Failed to get commit message from KV", "error", err)
	}

	if strings.TrimSpace(commitMsg) == "" {
		files := getChangedFiles()
		commitMsg = generateFallbackMessage(files)
		slog.Info("Using fallback commit message", "message", commitMsg)
	}

	opts := types.ContainerConfig{}
	opts.Image = Image(*c.GetBuild())

	// Only pass host/auth for KV access - NO tokens in Env for security
	opts.Env = []string{
		"CONTAINIFYCI_HOST=http://" + host,
		"CONTAINIFYCI_AUTH=" + auth,
	}

	dir, _ := filepath.Abs(".")
	opts.Volumes = []types.Volume{
		{
			Type:   "bind",
			Source: dir,
			Target: "/src",
		},
	}

	opts.Cmd = []string{"sh", "/tmp/commit.sh"}

	if err := c.Create(opts); err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Copy commit script to container
	if err := c.CopyCommitScript(commitMsg); err != nil {
		return fmt.Errorf("failed to copy commit script: %w", err)
	}

	if err := c.Start(); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	return c.Wait()
}

// CopyCommitScript copies the commit script to the container
func (c *GithubContainer) CopyCommitScript(commitMsg string) error {
	// Escape single quotes in commit message
	escapedMsg := strings.ReplaceAll(commitMsg, "'", "'\"'\"'")

	script := fmt.Sprintf(`#!/bin/sh
set -xe
cd /src

if [ -z "$(git status --porcelain)" ]; then
    echo "No changes to commit"
    exit 0
fi

# Fetch token from KV store (secure - not in container config)
export GITHUB_TOKEN="$(curl -fsS -H "Authorization: Bearer ${CONTAINIFYCI_AUTH}" "${CONTAINIFYCI_HOST}/mem/%s")"

git config --global user.email "bot@containifyci.io"
git config --global user.name "containifyci"
git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

git add -A
git commit -m '%s

Co-Authored-By: containifyci <bot@containifyci.io>'

git push origin HEAD
`, kvKeyGithubToken, escapedMsg)

	return c.CopyContentTo(script, "/tmp/commit.sh")
}

func ifTrivyFileExists() bool {
	_, err := os.Stat("trivy.json")
	if err == nil {
		return true
	} else if os.IsNotExist(err) {
		return false
	}

	slog.Error("Failed to read trivy.json file", "error", err)
	os.Exit(1)
	return false
}
