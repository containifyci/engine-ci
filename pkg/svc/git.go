package svc

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/containifyci/engine-ci/pkg/utils"
)

// TODO: Find a better way then a package global var
var git *Git

type Git struct {
	Owner  string
	Repo   string
	Branch string
	Tag    string
	PrNum  string
}

func (g *Git) IsUnknown() bool { return g == nil || g.Owner == "unknown" }
func (g *Git) IsPR() bool      { return g.PrNum != "" }
func (g *Git) IsNotPR() bool   { return !g.IsPR() }
func (g *Git) IsTag() bool     { return g.Tag != "" }
func (g *Git) IsNotTag() bool  { return !g.IsTag() }

func (g *Git) FullRepo() string {
	return fmt.Sprintf("%s/%s", g.Owner, g.Repo)
}

// CustomError is a basic custom error type.
type GitError struct {
	Message string
	Code    int
}

const (
	ErrGitCommandNotFound = 1
	ErrGitExec            = 2
)

type GitCommand interface {
	Exists() bool
	RemoteURL() (*string, error)
	Branch() (*string, error)
	Tag() (*string, error)
	PrNumber() (*string, error)
}

type GitCommander struct{}

func (g *GitCommander) Exists() bool {
	return gitExists()
}

func (g *GitCommander) RemoteURL() (*string, error) {
	// Get the remote URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return nil, &GitError{
			Code:    ErrGitExec,
			Message: err.Error(),
		}
	}
	remoteURL := strings.TrimSpace(string(output))
	return &remoteURL, nil
}

func (g *GitCommander) Branch() (*string, error) {
	branch := os.Getenv("GITHUB_HEAD_REF")

	if branch == "" {
		branch = os.Getenv("GITHUB_REF_NAME")
	}

	if branch == "" {
		// Get the current branch
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			return nil, &GitError{
				Code:    ErrGitExec,
				Message: err.Error(),
			}
		}
		branch = strings.TrimSpace(string(output))
	}
	return &branch, nil
}

func (g *GitCommander) Tag() (*string, error) {
	empty := ""
	//TODO: fetch the right build type
	tag := utils.GetEnvs([]string{"CONTAINIFYCI_GITHUB_REF", "GITHUB_REF"}, "build")

	slog.Debug("Found tag", "tag", tag)

	if tag != "" {
		if !strings.HasPrefix(tag, "refs/tags/") {
			slog.Debug("Invalid tag skip", "tag", tag)
			return &empty, nil
		}
		tag = strings.TrimPrefix(tag, "refs/tags/")
	}
	return &tag, nil
}

func (g *GitCommander) PrNumber() (*string, error) {
	empty := ""
	ref := os.Getenv("GITHUB_REF_NAME")

	if ref == "" {
		slog.Warn("GITHUB_REF_NAME is not set")
		return &empty, nil
	}
	if !strings.HasSuffix(ref, "/merge") {
		return &empty, nil
	}
	ref = strings.TrimSuffix(ref, "/merge")
	return &ref, nil
}

func (e *GitError) Error() string {
	return fmt.Sprintf("Code %d: %s", e.Code, e.Message)
}

// Check if git command exists
func gitExists() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func GitInfo() *Git {
	if git == nil {
		slog.Error("GitInfo not initialized")
	}
	return git
}

func SetUnknowGitInfo() *Git {
	git = &Git{
		Owner:  "unknown",
		Repo:   "unknown",
		Branch: "unknown",
		Tag:    "",
	}
	return git
}

// SetGitInfoForTest sets git info with custom values for testing purposes
func SetGitInfoForTest(owner, repo, branch, tag string) *Git {
	git = &Git{
		Owner:  owner,
		Repo:   repo,
		Branch: branch,
		Tag:    tag,
	}
	return git
}

// ResetGitInfo resets the git info to nil (for test cleanup)
func ResetGitInfo() {
	git = nil
}

func SetGitInfo() (*Git, error) {
	return setGitInfo(&GitCommander{})
}

func setGitInfo(cmd GitCommand) (*Git, error) {
	if git == nil {
		if !cmd.Exists() {
			slog.Debug("Retrieve git info from env. (git command not found)")
			_git, err := GitInfoFromEnv()
			if err != nil {
				slog.Error("Error getting Git info from environment", "error", err)
				return nil, err
			}
			git = _git
			return git, nil
		}

		remoteURL, err := cmd.RemoteURL()
		if err != nil {
			slog.Error("Error getting Git remote URL", "error", err)
			return nil, err
		}

		// Parse the remote URL to extract owner and repo
		var owner, repo string
		if strings.HasPrefix(*remoteURL, "http") {
			// HTTP/HTTPS URL
			parts := strings.Split(*remoteURL, "/")
			if len(parts) >= 2 {
				owner = parts[len(parts)-2]
				repo = strings.TrimSuffix(parts[len(parts)-1], ".git")
			}
		} else if strings.HasPrefix(*remoteURL, "git@") {
			// SSH URL
			parts := strings.Split(*remoteURL, ":")
			if len(parts) == 2 {
				pathParts := strings.Split(parts[1], "/")
				if len(pathParts) == 2 {
					owner = pathParts[0]
					repo = strings.TrimSuffix(pathParts[1], ".git")
				}
			}
		}

		branch, err := cmd.Branch()
		if err != nil {
			slog.Error("Error getting Git branch", "error", err)
			return nil, err
		}

		prNum, err := cmd.PrNumber()
		if err != nil {
			slog.Error("Error getting PR number", "error", err)
			return nil, err
		}

		tag, err := cmd.Tag()
		if err != nil {
			slog.Error("Error getting Git branch", "error", err)
			return nil, err
		}

		git = &Git{
			Owner:  owner,
			Repo:   repo,
			Branch: *branch,
			Tag:    *tag,
			PrNum:  *prNum,
		}
	}

	return git, nil
}

// checkEnvVars checks if the specified environment variables are set.
func checkEnvVars(vars []string) (map[string]string, error) {
	missingVars := make([]string, 0)
	existingVars := make(map[string]string)

	for _, v := range vars {
		value, exists := os.LookupEnv(v)
		if !exists {
			missingVars = append(missingVars, v)
		} else {
			existingVars[v] = value
		}
	}

	if len(missingVars) > 0 {
		return existingVars, fmt.Errorf("missing environment variables: %v", missingVars)
	}

	return existingVars, nil
}

func GitInfoFromEnv() (*Git, error) {
	envs, err := checkEnvVars([]string{"GITHUB_REPOSITORY_OWNER", "GITHUB_REPOSITORY_NAME", "GITHUB_HEAD_REF", "GITHUB_REF"})
	if err != nil {
		return nil, err
	}
	owner := envs["GITHUB_REPOSITORY_OWNER"]
	repo := envs["GITHUB_REPOSITORY_NAME"]
	branch := envs["GITHUB_HEAD_REF"]
	branch = strings.TrimPrefix(branch, "refs/heads/")
	tag := envs["GITHUB_REF"]
	tag = strings.TrimPrefix(tag, "refs/tags/")

	git = &Git{
		Owner:  owner,
		Repo:   repo,
		Branch: branch,
		Tag:    tag,
	}
	return git, nil
}
