package buildscript

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/with_generate.go.txt
var fixtureWithGenerate []byte

//go:embed testdata/without_generate.go.txt
var fixtureWithoutGenerate []byte

//go:embed testdata/vendor_file.go.txt
var fixtureVendorFile []byte

//go:embed testdata/hidden_file.go.txt
var fixtureHiddenFile []byte

func TestVerboseScript(t *testing.T) {
	bs := NewBuildScript("test", "/src/main.go", "", []string{"build_tag"}, true, true, CoverageMode("text"), "disabled", types.ParsePlatform("linux/amd64"), types.ParsePlatform("darwin/arm64"))
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
	bs := NewBuildScript("test", "/src/main.go", "src", nil, false, false, CoverageMode(""), "disabled", types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/amd64"))

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
	bs := NewBuildScript("test", "/src/main.go", "src", nil, false, false, CoverageMode("binary"), "disabled", types.ParsePlatform("darwin/arm64"), types.ParsePlatform("linux/amd64"))

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

func TestGoGenerateEnabled(t *testing.T) {
	bs := NewBuildScript("test", "/src/main.go", ".", nil, false, false, CoverageMode(""), "enabled", types.ParsePlatform("linux/amd64"))

	expected := `#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
cd .
go generate ./...
env GOOS=linux GOARCH=amd64 go build -o /src/test-linux-amd64 /src/main.go
go test -timeout 120s -cover -coverprofile coverage.txt ./...
`
	script := bs.String()
	assert.Equal(t, expected, script)
	assert.True(t, bs.ShouldGenerate, "ShouldGenerate should be true when mode is enabled")
}

func TestGoGenerateDisabled(t *testing.T) {
	bs := NewBuildScript("test", "/src/main.go", ".", nil, false, false, CoverageMode(""), "disabled", types.ParsePlatform("linux/amd64"))

	expected := `#!/bin/sh
set -xe
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
git config --global url."ssh://git@github.com/.insteadOf" "https://github.com/"
cd .
env GOOS=linux GOARCH=amd64 go build -o /src/test-linux-amd64 /src/main.go
go test -timeout 120s -cover -coverprofile coverage.txt ./...
`
	script := bs.String()
	assert.Equal(t, expected, script)
	assert.False(t, bs.ShouldGenerate, "ShouldGenerate should be false when mode is disabled")
}

func TestGoGenerateAutoDetectionWithDirective(t *testing.T) {
	// Create temp directory with go file containing //go:generate
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, fixtureWithGenerate, 0644)
	require.NoError(t, err)

	bs := NewBuildScript("test", "main.go", tmpDir, nil, false, false, CoverageMode(""), "auto", types.ParsePlatform("linux/amd64"))

	assert.True(t, bs.ShouldGenerate, "ShouldGenerate should be true when //go:generate directive is found")
	script := bs.String()
	assert.Contains(t, script, "go generate ./...", "Script should contain go generate command")
}

func TestGoGenerateAutoDetectionWithoutDirective(t *testing.T) {
	// Create temp directory with go file WITHOUT //go:generate
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, fixtureWithoutGenerate, 0644)
	require.NoError(t, err)

	bs := NewBuildScript("test", "main.go", tmpDir, nil, false, false, CoverageMode(""), "auto", types.ParsePlatform("linux/amd64"))

	assert.False(t, bs.ShouldGenerate, "ShouldGenerate should be false when no //go:generate directive is found")
	script := bs.String()
	assert.NotContains(t, script, "go generate ./...", "Script should not contain go generate command")
}

func TestGoGenerateAutoDetectionSkipsVendor(t *testing.T) {
	// Create temp directory with vendor directory containing //go:generate
	tmpDir := t.TempDir()
	vendorDir := filepath.Join(tmpDir, "vendor")
	err := os.MkdirAll(vendorDir, 0755)
	require.NoError(t, err)

	// Create go file in vendor with //go:generate (should be ignored)
	vendorFile := filepath.Join(vendorDir, "vendor.go")
	err = os.WriteFile(vendorFile, fixtureVendorFile, 0644)
	require.NoError(t, err)

	// Create main go file without //go:generate
	mainFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainFile, fixtureWithoutGenerate, 0644)
	require.NoError(t, err)

	bs := NewBuildScript("test", "main.go", tmpDir, nil, false, false, CoverageMode(""), "auto", types.ParsePlatform("linux/amd64"))

	assert.False(t, bs.ShouldGenerate, "ShouldGenerate should be false when //go:generate is only in vendor directory")
}

func TestGoGenerateAutoDetectionSkipsHiddenDirs(t *testing.T) {
	// Create temp directory with hidden directory containing //go:generate
	tmpDir := t.TempDir()
	hiddenDir := filepath.Join(tmpDir, ".git")
	err := os.MkdirAll(hiddenDir, 0755)
	require.NoError(t, err)

	// Create go file in hidden dir with //go:generate (should be ignored)
	hiddenFile := filepath.Join(hiddenDir, "hidden.go")
	err = os.WriteFile(hiddenFile, fixtureHiddenFile, 0644)
	require.NoError(t, err)

	// Create main go file without //go:generate
	mainFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainFile, fixtureWithoutGenerate, 0644)
	require.NoError(t, err)

	bs := NewBuildScript("test", "main.go", tmpDir, nil, false, false, CoverageMode(""), "auto", types.ParsePlatform("linux/amd64"))

	assert.False(t, bs.ShouldGenerate, "ShouldGenerate should be false when //go:generate is only in hidden directories")
}

func TestGoGenerateInvalidModeDefaultsToAuto(t *testing.T) {
	// Create temp directory with go file WITHOUT //go:generate
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "main.go")
	err := os.WriteFile(goFile, fixtureWithoutGenerate, 0644)
	require.NoError(t, err)

	// Pass invalid mode, should default to auto
	bs := NewBuildScript("test", "main.go", tmpDir, nil, false, false, CoverageMode(""), "invalid_mode", types.ParsePlatform("linux/amd64"))

	assert.False(t, bs.ShouldGenerate, "Invalid mode should default to auto, which should not detect directives in this case")
}
