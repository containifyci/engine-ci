package filesystem

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileExists(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		filename := "file_test.go"
		exists := FileExists(filename)
		assert.True(t, exists)
	})

	t.Run("file does not exist", func(t *testing.T) {
		filename := "file_not_exists.go"
		exists := FileExists(filename)
		assert.False(t, exists)
	})

	osStat = func(name string) (os.FileInfo, error) {
		return nil, os.ErrClosed
	}
	t.Run("file is not accesible", func(t *testing.T) {
		filename := "file_not_exists.go"
		exists := FileExists(filename)
		assert.False(t, exists)
	})
}
