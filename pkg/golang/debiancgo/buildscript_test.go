package debiancgo

import (
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"

	"github.com/stretchr/testify/assert"
)

func TestVerboseScript(t *testing.T) {
	bs := NewBuildScript("test", "/src/main.go", []string{"build_tag"}, true, types.ParsePlatform("linux/amd64"), types.ParsePlatform("darwin/arm64"))
	expected := `#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
env GOOS=linux GOARCH=amd64 go build -tags build_tag -x -o /src/test-linux-amd64 /src/main.go
env GOOS=darwin GOARCH=arm64 go build -tags build_tag -x -o /src/test-darwin-arm64 /src/main.go
go test -v -timeout 120s -tags build_tag -cover -coverprofile coverage.txt ./...
`
	script := Script(bs)
	assert.Equal(t, expected, script)
}

func TestSimpleScript(t *testing.T) {
	bs := NewBuildScript("test", "/src/main.go", nil, false, types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/amd64"))

	expected := `#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
env GOOS=darwin GOARCH=arm64 go build -o /src/test-darwin-arm64 /src/main.go
env GOOS=linux GOARCH=amd64 go build -o /src/test-linux-amd64 /src/main.go
go test -timeout 120s -cover -coverprofile coverage.txt ./...
`
	script := Script(bs)
	assert.Equal(t, expected, script)
}
