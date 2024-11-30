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

type Image string

type BuildScript struct {
	Tags       []string
	AppName    string
	MainFile   string
	Folder     string
	Output     string
	Platforms  []*types.PlatformSpec
	Verbose    bool
	NoCoverage bool
}

func NewBuildScript(appName, mainfile string, folder string, tags []string, verbose bool, nocoverage bool, platforms ...*types.PlatformSpec) *BuildScript {
	output := "-o /src/{{.app}}-{{.os}}-{{.arch}}"
	if mainfile == "" {
		mainfile = "./..."
		output = ""
	}
	if folder == "" {
		folder = "."
	}
	script := &BuildScript{
		AppName:    appName,
		MainFile:   mainfile,
		Folder:     folder,
		NoCoverage: nocoverage,
		Output:     output,
		Platforms:  platforms,
		Tags:       tags,
		Verbose:    verbose,
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

func renderTestCommand(m map[string]interface{}) string {
	t := template.Must(template.New("").
		Parse(trim(`
go test {{- .verbose }} -timeout 120s {{- .tags }} {{- .coverage}} ./...
`)))
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
	return buf.String()
}

func coverage(nocoverage bool) string {
	if nocoverage {
		return ""
	}
	return " -cover -coverprofile coverage.txt"
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
	coverage := coverage(bs.NoCoverage)
	m := map[string]interface{}{"verbose": "", "tags": "", "cgo": 0, "coverage": coverage}
	if bs.Verbose {
		m["verbose"] = " -v"
	}
	if len(bs.Tags) > 0 {
		m["tags"] = " -tags " + strings.Join(bs.Tags, ",")
	}
	cmds = append(cmds, renderTestCommand(m))
	return strings.Join(cmds, "\n")
}
