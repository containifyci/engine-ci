package alpine

//go:generate go run ../../../../tools/dockerfile-metadata/ -package alpine -output docker_metadata_gen.go -input Dockerfile_claude -variant ""

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containifyci/engine-ci/pkg/build"
	"github.com/containifyci/engine-ci/pkg/container"
	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/containifyci/engine-ci/pkg/cri/utils"
	"github.com/containifyci/engine-ci/pkg/kv"
	"github.com/containifyci/engine-ci/pkg/network"
	u "github.com/containifyci/engine-ci/pkg/utils"
	"github.com/containifyci/engine-ci/protos2"
)

//go:embed roles/*.md
var rolesFS embed.FS

// getRoleTemplate returns the role template for the given role name, or empty string if not found
func getRoleTemplate(role string) string {
	if role == "" {
		return ""
	}

	content, err := rolesFS.ReadFile("roles/" + role + ".md")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(content))
}

const (
	PROJ_MOUNT         = "/src"
	kvKeyCommitMessage = "commit_message"
)

// ClaudeContainer represents a Claude AI build step container
type ClaudeContainer struct {
	*container.Container
	Prompt    string // Main prompt for Claude
	Context   string // Additional context (e.g., build logs from previous iteration)
	Folder    string
	Role      string // Role identifier (e.g., "docker_expert", "planner")
	AgentMode bool   // If true, enables iterative loop behavior
	MaxIter   int    // Maximum iterations for agent mode
}

// New creates a new Claude AI build step
func New() build.BuildStepv3 {
	return build.Stepper{
		BuildType_: container.AI,
		RunFnV3: func(b container.Build) (string, error) {
			c := newContainer(b)
			slog.Info("Claude build", "custom", b.Custom)
			return c.Run()
		},
		MatchedFn: Matches,
		ImagesFn:  ClaudeImages,
		Name_:     "claude",
		Alias_:    "ai",
		Async_:    false,
	}
}

func newContainer(b container.Build) *ClaudeContainer {
	const defaultMaxIterations = 5

	maxIter := defaultMaxIterations
	if v := b.Custom.UInt("max_iterations"); v > 0 {
		maxIter = int(v)
	}

	return &ClaudeContainer{
		Container: container.New(b),
		Prompt:    b.Custom.String("ai_prompt"),
		Context:   b.Custom.String("ai_context"),
		Folder:    b.Folder,
		Role:      b.Custom.String("ai_role"),
		AgentMode: b.Custom.Bool("agent_mode", false),
		MaxIter:   maxIter,
	}
}

// Matches returns true if this step should run for the given build
func Matches(b container.Build) bool {
	if b.BuildType != container.AI {
		return false
	}

	claudeKey := u.GetValue(b.Custom.String("claude_api_key"), "build")
	if claudeKey == "" {
		slog.Info("Claude API key not provided, skipping Claude AI step")
		return false
	}

	return true
}

func dockerFile(b *container.Build) (*protos2.ContainerFile, error) {
	if b != nil {
		if v, ok := b.ContainerFiles["build"]; ok && v.Content != "" {
			return v, nil
		}
	}

	version, _, content := GetDockerfileMetadata("")
	return &protos2.ContainerFile{
		Name:    fmt.Sprintf("claude-%s", version),
		Content: content,
	}, nil
}

// ClaudeImage returns the image URI for the Claude container
func ClaudeImage(b container.Build) string {
	df, err := dockerFile(&b)
	if err != nil {
		slog.Error("Failed to get Dockerfile", "error", err)
		os.Exit(1)
	}
	tag := container.ComputeChecksum([]byte(df.Content))
	return utils.ImageURI(b.ContainifyRegistry, df.Name, tag)
}

// ClaudeImages returns all images needed for the Claude build step
func ClaudeImages(b container.Build) []string {
	return []string{"node:22-alpine", ClaudeImage(b)}
}

// BuildClaudeImage builds the intermediate Claude container image
func (c *ClaudeContainer) BuildClaudeImage() error {
	image := ClaudeImage(*c.GetBuild())
	df, err := dockerFile(c.GetBuild())
	if err != nil {
		return fmt.Errorf("failed to get Dockerfile: %w", err)
	}
	platforms := types.GetPlatforms(c.GetBuild().Platform)
	slog.Info("Building Claude intermediate image", "image", image, "platforms", platforms)
	return c.BuildIntermidiateContainer(image, []byte(df.Content), platforms...)
}

// Pull pulls the base images needed for the Claude build step
func (c *ClaudeContainer) Pull() error {
	return c.Container.Pull("node:22-alpine")
}

// Run executes the Claude AI build step
func (c *ClaudeContainer) Run() (string, error) {
	host := c.Build.Custom.String("CONTAINIFYCI_EXTERNAL_HOST")
	auth := c.Build.Secret["CONTAINIFYCI_AUTH"]
	claudeKey := u.GetValue(c.GetBuild().Custom.String("claude_api_key"), "build")

	err := kv.SetValue(host, auth, "claudecodeoauthtoken", claudeKey)
	if err != nil {
		return "", fmt.Errorf("failed to set claude_code_oauth_token: %w", err)
	}

	// 1) Pull base images
	if err := c.Pull(); err != nil {
		return "", fmt.Errorf("failed to pull base images: %w", err)
	}

	// 2) Build intermediate image
	if err := c.BuildClaudeImage(); err != nil {
		return "", fmt.Errorf("failed to build Claude image: %w", err)
	}

	// 3) Prepare prompt with context
	payload := strings.TrimSpace(c.Prompt)

	// Prepend role template if specified
	if roleTemplate := getRoleTemplate(c.Role); roleTemplate != "" {
		payload = roleTemplate + "\n\n" + payload
	}

	stopWord := c.Build.Custom.String("ai_done_word")
	if stopWord != "" {
		stopInstruction := fmt.Sprintf("Also if you get the build fixed, please print the following %s to indicate that you finished. Also add this as the last entry to the claude-actions.log file.", stopWord)
		payload = payload + "\n\n" + stopInstruction
	}

	// Add auto_commit instruction if enabled
	autoCommit := c.Build.Custom.Bool("auto_commit", false)
	if autoCommit {
		commitInstruction := `

When you complete your changes, write a commit message to a file called "commit-message.txt" in the project root.
The commit message should:
- Be a single line summary (max 72 chars) following conventional commit format (e.g., "feat(scope): description" or "fix(scope): description")
- Accurately describe what changes were made
- Not include any additional body text or footers`
		payload = payload + commitInstruction
	}

	context := ""
	if ctx := strings.TrimSpace(c.Context); ctx != "" {
		context = "\n\n Just edit the files no need to build the project that will be done outside of the session. \n\n---\nPrevious build context:\n" + ctx
	}

	if payload == "" {
		slog.Warn("No prompt provided for Claude AI step, skipping")
		return "", nil
	}

	// 4) Setup container config
	image := ClaudeImage(*c.GetBuild())

	ssh, err := network.SSHForward(*c.GetBuild())
	if err != nil {
		return "", fmt.Errorf("SSH forward failed: %w", err)
	}

	opts := types.ContainerConfig{}
	opts.Image = image
	opts.WorkingDir = PROJ_MOUNT

	// Pass through authentication
	opts.Env = append(opts.Env,
		"CI=true",
	)

	opts.Secrets = c.Secret

	// Mount project directory
	dir, _ := filepath.Abs(".")
	if c.Folder != "" {
		dir, _ = filepath.Abs(c.Folder)
	}
	opts.Volumes = []types.Volume{
		{Type: "bind", Source: dir, Target: PROJ_MOUNT},
	}
	opts = ssh.Apply(&opts)

	// Script to run Claude CLI with prompt from file
	opts.Script = `#!/bin/bash
set -euo pipefail
cat /tmp/prompt.txt
cat /tmp/prompt.txt /tmp/context.txt > /tmp/instructions.txt
export CLAUDE_CODE_OAUTH_TOKEN="$(curl -fsS -H "Authorization: Bearer ${CONTAINIFYCI_AUTH}" "${CONTAINIFYCI_HOST}/mem/claudecodeoauthtoken")"
claude --verbose --permission-mode acceptEdits --debug -p --output-format text "$(cat /tmp/instructions.txt)" 2>&1 | tee /src/claude-output.log
`
	// Set command to execute the script (like BuildingContainer does)
	opts.Cmd = []string{"sh", "/tmp/script.sh"}

	if err := c.Create(opts); err != nil {
		return c.ID, fmt.Errorf("failed to create container: %w", err)
	}

	// Copy script to container
	if err := c.CopyContentTo(opts.Script, "/tmp/script.sh"); err != nil {
		return c.ID, fmt.Errorf("failed to copy script to container: %w", err)
	}

	// Copy prompt to container
	if err := c.CopyContentTo(payload, "/tmp/prompt.txt"); err != nil {
		return c.ID, fmt.Errorf("failed to copy prompt to container: %w", err)
	}

	// Copy context to container
	if err := c.CopyContentTo(context, "/tmp/context.txt"); err != nil {
		return c.ID, fmt.Errorf("failed to copy context to container: %w", err)
	}

	if err := c.Start(); err != nil {
		return c.ID, fmt.Errorf("failed to start container: %w", err)
	}

	if err := c.Wait(); err != nil {
		time.Sleep(3 * time.Second) // Allow logs to flush
		return c.ID, fmt.Errorf("container execution failed: %w", err)
	}

	// Store commit message in KV if auto_commit is enabled
	if autoCommit {
		commitMsgPath := filepath.Join(dir, "commit-message.txt")
		if msg, err := os.ReadFile(commitMsgPath); err == nil {
			trimmedMsg := strings.TrimSpace(string(msg))
			if trimmedMsg != "" {
				if err := kv.SetValue(host, auth, kvKeyCommitMessage, trimmedMsg); err != nil {
					slog.Warn("Failed to store commit message in KV", "error", err)
				} else {
					slog.Info("Stored commit message in KV", "message", trimmedMsg)
				}
			}
		} else {
			slog.Info("No commit-message.txt file found", "path", commitMsgPath)
		}
	}

	return c.ID, nil
}
