package buildscript

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"text/template"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

type CoverageMode string

type Image string

type BuildScript struct {
	AppName      string
	MainFile     string
	Folder       string
	Output       string
	CoverageMode CoverageMode
	FileName     string
	Tags         []string
	Platforms    []*types.PlatformSpec
	Artifacts    []string
	Verbose      bool
	NoCoverage   bool
}

func NewBuildScript(appName, mainfile string, folder string, tags []string, verbose bool, nocoverage bool, coverageMode CoverageMode, platforms ...*types.PlatformSpec) *BuildScript {
	filename := "{{.app}}-{{.os}}-{{.arch}}"
	output := "-o /src/" + filename
	if mainfile == "" {
		mainfile = "./..."
		output = ""
		filename = ""
	}
	if folder == "" {
		folder = "."
	}
	script := &BuildScript{
		AppName:      appName,
		CoverageMode: coverageMode,
		Folder:       folder,
		MainFile:     mainfile,
		NoCoverage:   nocoverage,
		Output:       output,
		Platforms:    platforms,
		Tags:         tags,
		Verbose:      verbose,
		FileName:     filename,
		Artifacts:    []string{},
	}
	return script
}

// TODO: the -race flag needs CDO enabled for now https://github.com/golang/go/issues/6508
func (bs *BuildScript) String() string {
	return script(bs)
}

func script(bs *BuildScript) string {
	goBuildCmd := goBuildCmds(bs)
	script := fmt.Sprintf(`#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
cd %s
%s
`, bs.Folder, goBuildCmd)

	return script
}

func trim(str string, args ...any) string {
	s := strings.TrimSpace(str)
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, args...)
}

func renderTestCommand(bs *BuildScript, m map[string]interface{}) string {
	var cmd string
	switch bs.CoverageMode {
	case "binary":
		cmd = "mkdir -p ${PWD}/.coverdata/unit\ngo test {{- .verbose }} -timeout 120s {{- .tags }} -cover ./... {{- .coverage}}"
	case "text":
		fallthrough
	default:
		cmd = "go test {{- .verbose }} -timeout 120s {{- .tags }} {{- .coverage}} ./..."
	}

	t := template.Must(template.New("").
		Parse(trim(cmd)))
	buf := new(bytes.Buffer)
	err := t.Execute(buf, m)
	if err != nil {
		slog.Error("Failed render go test cmd", "error", err)
		os.Exit(1)
	}
	return buf.String()
}

func (bs *BuildScript) renderCompileCommand(m map[string]interface{}) string {
	t := template.Must(template.New("").
		Parse(trim(`
env GOOS={{.os}} GOARCH={{ .arch }} go build {{- .tags }} {{- .verbose }} %s {{.mainfile}}
`, bs.Output)))
	buf := new(bytes.Buffer)
	err := t.Execute(buf, m)
	if err != nil {
		slog.Error("Failed render go build cmd", "error", err)
		os.Exit(1)
	}

	if bs.FileName == "" {
		return buf.String()
	}
	t2 := template.Must(template.New("").
		Parse(bs.FileName))
	buf2 := new(bytes.Buffer)
	err2 := t2.Execute(buf2, m)
	if err2 != nil {
		slog.Error("Failed render go build cmd", "error", err)
		os.Exit(1)
	}

	bs.Artifacts = append(bs.Artifacts, buf2.String())
	return buf.String()
}

func coverage(nocoverage bool, coveragemode CoverageMode) string {
	if nocoverage {
		return ""
	}
	switch coveragemode {
	case "binary":
		return " -args -test.gocoverdir=${PWD}/.coverdata/unit"
	case "text":
		fallthrough
	default:
		return " -cover -coverprofile coverage.txt"

	}
}

func goBuildCmds(bs *BuildScript) string {
	var cmds []string
	for _, platform := range bs.Platforms {
		m := map[string]interface{}{"os": platform.OS, "arch": platform.Architecture, "app": bs.AppName, "mainfile": bs.MainFile, "verbose": "", "tags": ""}
		if bs.Verbose {
			m["verbose"] = " -x"
		}
		if len(bs.Tags) > 0 {
			m["tags"] = " -tags " + strings.Join(bs.Tags, ",")
		}
		cmds = append(cmds, bs.renderCompileCommand(m))
	}
	coverage := coverage(bs.NoCoverage, bs.CoverageMode)
	m := map[string]interface{}{"verbose": "", "tags": "", "cgo": 0, "coverage": coverage}
	if bs.Verbose {
		m["verbose"] = " -v"
	}
	if len(bs.Tags) > 0 {
		m["tags"] = " -tags " + strings.Join(bs.Tags, ",")
	}
	cmds = append(cmds, renderTestCommand(bs, m))
	return strings.Join(cmds, "\n")
}
