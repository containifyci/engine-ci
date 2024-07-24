package utils

import (
	"errors"
	"io"
)

// ChannelReadCloser is a custom type that implements io.ReadCloser.
type ChannelReadCloser struct {
	ch     <-chan string
	buf    []byte
	closed bool
}

// NewChannelReadCloser creates a new ChannelReadCloser.
func NewChannelReadCloser(ch <-chan string) *ChannelReadCloser {
	return &ChannelReadCloser{ch: ch}
}

// Read reads data from the channel into the provided buffer.
func (crc *ChannelReadCloser) Read(p []byte) (n int, err error) {
	if crc.closed && len(crc.buf) == 0 {
		return 0, io.EOF
	}

	for len(crc.buf) == 0 {
		data, ok := <-crc.ch
		if !ok {
			crc.closed = true
			break
		}
		crc.buf = []byte(data)
	}

	if len(crc.buf) == 0 {
		return 0, io.EOF
	}

	n = copy(p, crc.buf)
	crc.buf = crc.buf[n:]
	return n, nil
}

// Close closes the ChannelReadCloser.
func (crc *ChannelReadCloser) Close() error {
	if crc.closed {
		return errors.New("already closed")
	}
	crc.closed = true
	return nil
}

// ReadCloser wraps a bytes.Buffer and implements io.ReadCloser
type ReadCloser struct {
	reader io.Reader
}

// Close implements the io.Closer interface
func (rc *ReadCloser) Close() error {
	// No resources to free, so just return nil
	return nil
}

// Close implements the io.Closer interface
func (rc *ReadCloser) Read(p []byte) (int, error) {
	// No resources to free, so just return nil
	return rc.reader.Read(p)
}

func NewReadCloser(reader io.Reader) *ReadCloser {
	return &ReadCloser{reader}
}
