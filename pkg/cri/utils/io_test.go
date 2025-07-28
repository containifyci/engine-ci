package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewChannelReadCloser(t *testing.T) {
	ch := make(<-chan string)
	crc := NewChannelReadCloser(ch)
	assert.NotNil(t, crc)
	assert.Equal(t, ch, crc.ch)
}

func TestChannelReadCloser_Read(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected []string
	}{
		{"Single message", []string{"hello"}, []string{"hello"}},
		{"Multiple messages", []string{"hello", "world"}, []string{"hello", "world"}},
		{"Empty channel", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan string)
			crc := NewChannelReadCloser(ch)

			go func() {
				for _, input := range tt.inputs {
					ch <- input
				}
				close(ch)
			}()

			var result []string
			buf := make([]byte, 1024) // Buffer size can be adjusted to be bigger than expected message length
			for {
				n, err := crc.Read(buf)
				if n > 0 {
					result = append(result, string(buf[:n]))
				}
				if err != nil {
					break
				}
			}

			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestChannelReadCloser_Close(t *testing.T) {
	ch := make(chan string)
	crc := NewChannelReadCloser(ch)
	assert.NoError(t, crc.Close())

	// Try closing again, expect an error
	err := crc.Close()
	assert.Error(t, err)
	assert.Equal(t, "already closed", err.Error())
}

func TestNewReadCloser(t *testing.T) {
	reader := strings.NewReader("test data")
	rc := NewReadCloser(reader)
	assert.NotNil(t, rc)
}

func TestReadCloser_Read(t *testing.T) {
	reader := strings.NewReader("test data")
	rc := NewReadCloser(reader)

	buf := make([]byte, 4)
	n, err := rc.Read(buf)

	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "test", string(buf))
}

func TestReadCloser_Close(t *testing.T) {
	reader := strings.NewReader("test data")
	rc := NewReadCloser(reader)

	// Close should not return an error
	assert.NoError(t, rc.Close())
}
