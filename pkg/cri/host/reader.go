package host

import (
	"io"
	"sync"
)

// WriterToReadCloser wraps an io.Writer and allows reading the written data as an io.ReadCloser.
type WriterToReadCloser struct {
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	closeOnce  sync.Once
}

// NewWriterToReadCloser creates a new WriterToReadCloser.
func NewWriterToReadCloser() *WriterToReadCloser {
	reader, writer := io.Pipe()
	return &WriterToReadCloser{
		pipeReader: reader,
		pipeWriter: writer,
	}
}

// Write writes data to the underlying io.PipeWriter.
func (wtrc *WriterToReadCloser) Write(p []byte) (int, error) {
	return wtrc.pipeWriter.Write(p)
}

// Read reads data from the underlying io.PipeReader.
func (wtrc *WriterToReadCloser) Read(p []byte) (int, error) {
	return wtrc.pipeReader.Read(p)
}

// Close closes both the writer and reader ends of the pipe.
func (wtrc *WriterToReadCloser) Close() error {
	wtrc.closeOnce.Do(func() {
		wtrc.pipeWriter.Close()
		wtrc.pipeReader.Close()
	})
	return nil
}
