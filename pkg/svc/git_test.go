package svc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type GitTestCommand struct {
	branch    string
	tag       string
	exists    bool
	remoteURL string
	err       error
}

func (g *GitTestCommand) Exists() bool {
	return g.exists
}

func (g *GitTestCommand) RemoteURL() (*string, error) {
	if g.err != nil {
		return nil, g.err
	}
	return &g.remoteURL, nil
}

func (g *GitTestCommand) Branch() (*string, error) {
	if g.err != nil {
		return nil, g.err
	}
	return &g.branch, nil
}

func (g *GitTestCommand) Tag() (*string, error) {
	if g.err != nil {
		return nil, g.err
	}
	return &g.tag, nil
}

func (g *GitTestCommand) PrNumber() (*string, error) {
	gitCmd := &GitCommander{}
	return gitCmd.PrNumber()
}

func NewGitCommand(branch, remoteURL string, tag string, exists bool) GitCommand {
	return &GitTestCommand{branch: branch, tag: tag, exists: exists, remoteURL: remoteURL}
}

func TestSetGitInfo(t *testing.T) {
	tests := []struct {
		name    string
		command GitCommand
		expect  Git
		err     error
	}{
		{
			name:    "Git https url",
			command: NewGitCommand("master", "https://github.com/owner/repo.git", "", true),
			expect: Git{
				Owner:  "owner",
				Repo:   "repo",
				Branch: "master",
				Tag:    "",
			},
		},
		{
			name:    "Git ssh url",
			command: NewGitCommand("main", "git@github.com:owner/repo.git", "v1.0.2", true),
			expect: Git{
				Owner:  "owner",
				Repo:   "repo",
				Branch: "main",
				Tag:    "v1.0.2",
			},
		},
	}

	RunGitInfoTest(t, tests)
}

func TestSetGitInfoWithEnv(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY_OWNER", "owner")
	t.Setenv("GITHUB_REPOSITORY_NAME", "repo")
	t.Setenv("GITHUB_HEAD_REF", "feature/branch")
	t.Setenv("GITHUB_REF", "refs/tags/v1.0.4")
	tests := []struct {
		name    string
		command GitCommand
		expect  Git
		err     error
	}{
		{
			name:    "Without Git command only with env",
			command: &GitTestCommand{exists: false},
			expect: Git{
				Owner:  "owner",
				Repo:   "repo",
				Branch: "feature/branch",
				Tag:    "v1.0.4",
			},
		},
	}

	RunGitInfoTest(t, tests)
}

func TestSetGitInfoWithError(t *testing.T) {
	tests := []struct {
		name    string
		command GitCommand
		expect  Git
		err     error
	}{
		{
			name:    "Git https url",
			command: &GitTestCommand{exists: false},
			err:     fmt.Errorf("missing environment variables: [GITHUB_REPOSITORY_OWNER GITHUB_REPOSITORY_NAME GITHUB_HEAD_REF GITHUB_REF]"),
		},
	}

	RunGitInfoTest(t, tests)
}

func RunGitInfoTest(t *testing.T, tests []struct {
	name    string
	command GitCommand
	expect  Git
	err     error
}) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git = nil
			git, err := setGitInfo(tt.command)
			if tt.err != nil {
				assert.ErrorContains(t, err, tt.err.Error())
				assert.Nil(t, git)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, tt.expect.Owner, git.Owner)
			assert.Equal(t, tt.expect.Repo, git.Repo)
			assert.Equal(t, tt.expect.Branch, git.Branch)
			assert.Equal(t, tt.expect.Tag, git.Tag)

		})
	}
}

func TestPrNumber(t *testing.T) {
	t.Setenv("GITHUB_REF_NAME", "1234/merge")
	_git, err := setGitInfo(NewGitCommand("main", "", "", true))
	assert.NoError(t, err)

	assert.Equal(t, "1234", _git.PrNum)

	t.Setenv("GITHUB_REF_NAME", "feature_branch")
	git = nil
	_git, err = setGitInfo(NewGitCommand("main", "", "", true))
	assert.NoError(t, err)

	assert.Equal(t, "", _git.PrNum)
}
