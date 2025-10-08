package python

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Tool string
type Commands [][]string

func (c Commands) String() string {
	var cmds []string
	for _, cmd := range c {
		cmds = append(cmds, strings.Join(cmd, " "))
	}
	return strings.Join(cmds, "\n")
}

const (
	ToolUV     Tool = "uv"
	ToolPoetry Tool = "poetry"
	ToolPip    Tool = "pip"
)

type Builder struct {
	Folder string
	Tool   Tool

	files Files
}

type Files struct {
	pyproject    string
	requirements string
	poetryLock   string
	uvLock       string
}

func NewBuilder(folder string) *Builder {
	builder := &Builder{
		Folder: folder,
	}

	return builder
}

func (b *Builder) Analyze() (Tool, error) {
	pyproject := filepath.Join(b.Folder, "pyproject.toml")
	requirements := filepath.Join(b.Folder, "requirements.txt")
	poetryLock := filepath.Join(b.Folder, "poetry.lock")
	uvLock := filepath.Join(b.Folder, "uv.lock")

	pyprojectContent := readFile(pyproject)

	files := Files{
		pyproject:    pyproject,
		requirements: requirements,
		poetryLock:   poetryLock,
		uvLock:       uvLock,
	}
	b.files = files

	// Heuristics (prefer explicit signals):
	// 1) Poetry if pyproject has [tool.poetry] or poetry.lock
	// 2) uv if pyproject suggests uv (tool.uv, dependency-groups) or uv.lock present
	// 3) pip if requirements.txt or a generic pyproject without poetry/uv hints
	// 4) fallback to what's installed: uv > poetry > pip
	var chosen Tool
	switch {
	case containsSection(pyprojectContent, "tool.uv") || fileExists(uvLock) || strings.Contains(pyprojectContent, "[dependency-groups]"):
		chosen = ToolUV
	case containsSection(pyprojectContent, "tool.poetry") || fileExists(poetryLock):
		chosen = ToolPoetry
	case fileExists(requirements) || (fileExists(pyproject) && !containsSection(pyprojectContent, "tool.poetry")):
		// leaning pip for plain pyproject or requirements.txt
		chosen = ToolPip
	default:
		chosen = ToolUV
	}
	b.Tool = chosen
	return chosen, nil
}

func (b *Builder) Build() (Commands, error) {

	fmt.Printf("🔎 Detected build tool: %s\n", b.Tool)

	// Build command plan
	var plan Commands
	switch b.Tool {
	case ToolUV:
		// Install deps
		plan = append(plan, []string{"uv", "sync"})
		// Build distribution if pyproject exists (most modern projects)
		if fileExists(b.files.pyproject) {
			plan = append(plan, []string{"uv", "build"})
		}
	case ToolPoetry:
		plan = append(plan, []string{"poetry", "install", "--no-interaction"})
		plan = append(plan, []string{"poetry", "build"})
	case ToolPip:
		python := "python3"
		// Install deps
		if fileExists(b.files.requirements) {
			plan = append(plan, []string{python, "-m", "pip", "install", "-r", "requirements.txt"})
		} else if fileExists(b.files.pyproject) {
			// PEP 517 build front-end; install build tool if missing
			plan = append(plan,
				[]string{python, "-m", "pip", "install", "--upgrade", "pip", "build"},
				[]string{python, "-m", "build", "--wheel", "--sdist"},
			)
		} else {
			return nil, errors.New("no recognizable Python project files found (need pyproject.toml or requirements.txt)")
		}
	default:
		return nil, fmt.Errorf("unknown tool %q", b.Tool)
	}

	return plan, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func readFile(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

func containsSection(toml string, dotted string) bool {
	// crude but effective: match either [tool.poetry] or [tool.poetry.something]
	needle := "[" + dotted + "]"
	needlePrefix := "[" + dotted + "."
	return strings.Contains(toml, needle) || strings.Contains(toml, needlePrefix)
}
