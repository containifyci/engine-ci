package build

import (
	"fmt"
	"os"

	"github.com/containifyci/engine-ci/protos2"
)

// NewAIBuild creates a new AI build step configuration
func NewAIBuild(name string) *BuildArgs {
	commitSha := os.Getenv("COMMIT_SHA")
	if commitSha == "" {
		commitSha = "local"
	}
	return &BuildArgs{
		Application: name,
		Environment: getEnv(),
		BuildType:   protos2.BuildType_AI,
		ImageTag:    commitSha,
	}
}

// NewClaudeBuild creates a Claude-specific AI build with a prompt
func NewClaudeBuild(prompt string) *BuildArgs {
	args := NewAIBuild("claude-agent")
	args.Properties = map[string]*ListValue{
		"ai_prompt": NewList(prompt),
	}
	return args
}

// NewAgentBuild creates an AI build configured for iterative agent mode
func NewAgentBuild(prompt string, maxIterations int) *BuildArgs {
	args := NewClaudeBuild(prompt)
	args.Properties["agent_mode"] = NewList("true")
	if maxIterations > 0 {
		args.Properties["max_iterations"] = NewList(fmt.Sprintf("%d", maxIterations))
	}
	return args
}
