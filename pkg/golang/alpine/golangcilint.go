package alpine

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/containifyci/engine-ci/pkg/filesystem"
	"gopkg.in/yaml.v3"
)

type customGCL struct {
	Destination string `yaml:"destination"`
	Name        string `yaml:"name"`
}

type CustomGCLReader struct {
	FileReader
}

type FileReader interface {
	FileExists(filename string) bool
	ReadFile(name string) ([]byte, error)
}

func (c CustomGCLReader) FileExists(filename string) bool {
	return c.FileReader.FileExists(filename)
}

func (c CustomGCLReader) ReadFile(name string) ([]byte, error) {
	return c.FileReader.ReadFile(name)
}

type CustomGCLFileReader struct{}

func (c CustomGCLFileReader) FileExists(filename string) bool {
	return filesystem.FileExists(filename)
}

func (c CustomGCLFileReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

type GolangCiLint struct {
	reader CustomGCLReader
}

func NewGolangCiLint() GolangCiLint {
	return GolangCiLint{
		reader: CustomGCLReader{
			FileReader: CustomGCLFileReader{},
		},
	}
}

func (cg *customGCL) Defaults() {
	if cg.Destination == "" {
		cg.Destination = "."
	} else {
		cg.Destination = strings.TrimSuffix(cg.Destination, "/")
	}
	if cg.Name == "" {
		cg.Name = "custom-gcl"
	}
}

func (c GolangCiLint) Command(tags string, folder string) string {
	cmd := fmt.Sprintf("golangci-lint -v run %s --timeout=5m", tags)
	if !c.reader.FileExists(filepath.Join(folder, ".golangci.yml")) {
		cmd = fmt.Sprintf("golangci-lint -v run %s --timeout=5m", tags)
	}

	if c.reader.FileExists(filepath.Join(folder, ".custom-gcl.yml")) {
		cnt, err := c.reader.ReadFile(filepath.Join(folder, ".custom-gcl.yml"))
		if err != nil {
			slog.Error("Failed to read .custom-gcl.yml file", "error", err)
			os.Exit(1)
		}
		var cGCL customGCL
		err = yaml.Unmarshal(cnt, &cGCL)
		if err != nil {
			slog.Error("Failed to parse .custom-gcl.yml file", "error", err)
			os.Exit(1)
		}
		cGCL.Defaults()

		cmd = fmt.Sprintf(
			`golangci-lint custom
%s/%s run %s`, cGCL.Destination, cGCL.Name, tags)
	}
	return cmd
}

func (c GolangCiLint) LintScript(tags []string, folder string) string {
	_tags := ""
	if len(tags) > 0 {
		_tags = "--build-tags " + strings.Join(tags, ",")
	}

	cmd := c.Command(_tags, folder)
	//TODO: add suport for custom-gcl in the future
	script := fmt.Sprintf(`#!/bin/sh
set -x
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
%s`, cmd)
	return script
}
