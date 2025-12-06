package zig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var platforms = []*types.PlatformSpec{types.ParsePlatform("linux/amd64"), types.ParsePlatform("darwin/arm64")}

func TestBuildScript_SimpleScript(t *testing.T) {
	bs := NewBuildScript("/src", "", "", false, "/root/.cache/zig", platforms)

	script := bs.Script()

	assert.Contains(t, script, "#!/bin/sh")
	assert.Contains(t, script, "set -e")
	assert.Contains(t, script, "zig build --color off --summary all")
	assert.Contains(t, script, "export ZIG_GLOBAL_CACHE_DIR=/root/.cache/zig")
	assert.NotContains(t, script, "set -xe") // verbose mode off
}

func TestBuildScript_VerboseScript(t *testing.T) {
	bs := NewBuildScript("/src", "ReleaseSafe", "", true, "/root/.cache/zig", platforms)

	script := bs.Script()

	assert.Contains(t, script, "#!/bin/sh")
	assert.Contains(t, script, "set -xe") // verbose mode on
	assert.Contains(t, script, "zig build --color off --summary all -Doptimize=ReleaseSafe --verbose")
}

func TestBuildScript_WithoutCacheDir(t *testing.T) {
	bs := NewBuildScript("/src", "", "", false, "", platforms)

	script := bs.Script()

	assert.NotContains(t, script, "export ZIG_GLOBAL_CACHE_DIR")
}

func TestBuildScript_WithOptimization(t *testing.T) {
	tests := []struct {
		name     string
		optimize string
		expected string
	}{
		{
			name:     "ReleaseSafe",
			optimize: "ReleaseSafe",
			expected: "zig build --color off --summary all -Doptimize=ReleaseSafe",
		},
		{
			name:     "ReleaseFast",
			optimize: "ReleaseFast",
			expected: "zig build --color off --summary all -Doptimize=ReleaseFast",
		},
		{
			name:     "ReleaseSmall",
			optimize: "ReleaseSmall",
			expected: "zig build --color off --summary all -Doptimize=ReleaseSmall",
		},
		{
			name:     "Debug",
			optimize: "Debug",
			expected: "zig build --color off --summary all -Doptimize=Debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBuildScript("/src", tt.optimize, "", false, "", platforms)
			script := bs.Script()
			assert.Contains(t, script, tt.expected)
		})
	}
}

func TestBuildScript_WithTarget(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "x86_64-linux-musl",
			target:   "x86_64-linux-musl",
			expected: "zig build --color off --summary all -Dtarget=x86_64-linux-musl",
		},
		{
			name:     "aarch64-linux-gnu",
			target:   "aarch64-linux-gnu",
			expected: "zig build --color off --summary all -Dtarget=aarch64-linux-gnu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBuildScript("/src", "", tt.target, false, "", platforms)
			script := bs.Script()
			assert.Contains(t, script, tt.expected)
		})
	}
}

func TestBuildScript_WithOptimizeAndTarget(t *testing.T) {
	bs := NewBuildScript("/src", "ReleaseFast", "x86_64-linux-musl", false, "", platforms)

	script := bs.Script()

	assert.Contains(t, script, "zig build --color off --summary all -Doptimize=ReleaseFast -Dtarget=x86_64-linux-musl")
	assert.Contains(t, script, "zig build test --summary all -Dtarget=x86_64-linux-musl")
}

func TestBuildScript_WithBuildZon(t *testing.T) {
	// Create temp directory with build.zig.zon
	tempDir := t.TempDir()
	buildZonPath := filepath.Join(tempDir, "build.zig.zon")
	err := os.WriteFile(buildZonPath, []byte(".{.name = \"myapp\"}"), 0644)
	require.NoError(t, err)

	bs := NewBuildScript(tempDir, "", "", false, "", platforms)

	script := bs.Script()

	assert.Contains(t, script, "zig build --color off --summary all")
	// zig fetch should come before zig build
	assert.True(t, strings.Index(script, "zig fetch") < strings.Index(script, "zig build"))
}

func TestBuildScript_WithoutBuildZon(t *testing.T) {
	tempDir := t.TempDir()

	bs := NewBuildScript(tempDir, "", "", false, "", platforms)

	script := bs.Script()

	assert.Contains(t, script, "zig build --color off --summary all")
}
