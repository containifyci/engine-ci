package alpine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyLintScript(t *testing.T) {
	gcl := NewGolangCiLint()
	script := gcl.LintScript([]string{"build_tag"}, ".")

	assert.Equal(t, `#!/bin/sh
set -x
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
golangci-lint -v run --fix --build-tags build_tag --timeout=5m`, script)
}

func TestCopyLintScriptGCL(t *testing.T) {
	gcl := GolangCiLint{
		reader: CustomGCLReader{
			FileReader: TestFileReader{
				fileExists: func(filename string) bool {
					return true
				},
				readFile: func(filename string) ([]byte, error) {
					return []byte(`destination: build`), nil
				},
			},
		},
	}
	script := gcl.LintScript([]string{"build_tag"}, ".")

	assert.Equal(t, `#!/bin/sh
set -x
mkdir -p ~/.ssh
ssh-keyscan github.com >> ~/.ssh/known_hosts
golangci-lint custom
build/custom-gcl run --fix --build-tags build_tag`, script)
}

// test utilities

// Implements Thing interface
type TestFileReader struct {
	fileExists func(string) bool
	readFile   func(string) ([]byte, error)
}

func (tf TestFileReader) FileExists(filename string) bool          { return tf.fileExists(filename) }
func (tf TestFileReader) ReadFile(filename string) ([]byte, error) { return tf.readFile(filename) }
