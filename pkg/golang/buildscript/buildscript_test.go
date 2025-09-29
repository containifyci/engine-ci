package buildscript

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"

	"github.com/stretchr/testify/assert"
)

func TestVerboseScript(t *testing.T) {
	bs := NewBuildScript("test", "/src/main.go", "", []string{"build_tag"}, true, true, CoverageMode("text"), types.ParsePlatform("linux/amd64"), types.ParsePlatform("darwin/arm64"))
	expected := `#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
cd .
env GOOS=linux GOARCH=amd64 go build -tags build_tag -x -o /src/test-linux-amd64 /src/main.go
env GOOS=darwin GOARCH=arm64 go build -tags build_tag -x -o /src/test-darwin-arm64 /src/main.go
go test -v -timeout 120s -tags build_tag ./...
`
	script := bs.String()
	assert.Equal(t, expected, script)
	assert.Equal(t, []string{"test-linux-amd64", "test-darwin-arm64"}, bs.Artifacts)
}

func TestSimpleScript(t *testing.T) {
	bs := NewBuildScript("test", "/src/main.go", "src", nil, false, false, CoverageMode(""), types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/amd64"))

	expected := `#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
cd src
env GOOS=darwin GOARCH=arm64 go build -o /src/src/test-darwin-arm64 /src/main.go
env GOOS=linux GOARCH=amd64 go build -o /src/src/test-linux-amd64 /src/main.go
go test -timeout 120s -cover -coverprofile coverage.txt ./...
`
	script := bs.String()
	assert.Equal(t, expected, script)
	assert.Equal(t, []string{"test-darwin-arm64", "test-linux-amd64"}, bs.Artifacts)
}

func TestSimpleScriptCoverageBinary(t *testing.T) {
	bs := NewBuildScript("test", "/src/main.go", "src", nil, false, false, CoverageMode("binary"), types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/amd64"))

	expected := `#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
cd src
env GOOS=darwin GOARCH=arm64 go build -o /src/src/test-darwin-arm64 /src/main.go
env GOOS=linux GOARCH=amd64 go build -o /src/src/test-linux-amd64 /src/main.go
mkdir -p ${PWD}/.coverdata/unit
go test -timeout 120s -cover ./... -args -test.gocoverdir=${PWD}/.coverdata/unit
`
	script := bs.String()
	assert.Equal(t, expected, script)
	assert.Equal(t, []string{"test-darwin-arm64", "test-linux-amd64"}, bs.Artifacts)
}
