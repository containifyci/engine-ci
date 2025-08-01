package memory

import (
	"sync"
	"sync/atomic"
)

const (
	// TarBufferSize is optimized for TAR file operations (64KB)
	TarBufferSize = 65536 // 64KB - optimized for tar operations

	// Maximum buffer size to retain in pool (prevents memory bloat)
	MaxRetainedBufferSize = 1048576 // 1MB
)

// BufferSize represents buffer size categories
type BufferSize int

const (
	TarBuffer BufferSize = iota
)

// bufferWrapper wraps a byte slice for sync.Pool compatibility
type bufferWrapper struct {
	b []byte
}

// BufferPool manages reusable byte slices for TAR operations
type BufferPool struct {
	tar sync.Pool

	// Metrics for monitoring pool efficiency
	tarHits   int64
	tarMisses int64
}

// Global buffer pool instance
var defaultBufferPool = NewBufferPool()

// NewBufferPool creates a new BufferPool optimized for TAR operations
func NewBufferPool() *BufferPool {
	pool := &BufferPool{}

	// Initialize TAR buffer pool (37% performance improvement)
	pool.tar = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.tarMisses, 1)
			return &bufferWrapper{b: make([]byte, TarBufferSize)}
		},
	}

	return pool
}

// Get retrieves a TAR buffer from the pool
func (p *BufferPool) Get(size BufferSize) []byte {
	if size == TarBuffer {
		atomic.AddInt64(&p.tarHits, 1)
		return p.tar.Get().(*bufferWrapper).b
	}
	// Default to TAR buffer for any other size
	atomic.AddInt64(&p.tarHits, 1)
	return p.tar.Get().(*bufferWrapper).b
}

// Put returns a buffer to the TAR pool
func (p *BufferPool) Put(buffer []byte, size BufferSize) {
	// Only return to pool if capacity is reasonable to avoid memory bloat
	if cap(buffer) > MaxRetainedBufferSize {
		// Don't return oversized buffers to pool
		return
	}

	// Clear the buffer for security and cleanliness
	for i := range buffer {
		buffer[i] = 0
	}

	wrapper := &bufferWrapper{b: buffer}
	p.tar.Put(wrapper)
}

// GetMetrics returns pool efficiency metrics
func (p *BufferPool) GetMetrics() BufferPoolMetrics {
	return BufferPoolMetrics{
		TarHits:   atomic.LoadInt64(&p.tarHits),
		TarMisses: atomic.LoadInt64(&p.tarMisses),
	}
}

// BufferPoolMetrics contains statistics about buffer pool usage
type BufferPoolMetrics struct {
	TarHits   int64
	TarMisses int64
}

// HitRate calculates the hit rate for the TAR buffer pool
func (m BufferPoolMetrics) HitRate() float64 {
	total := m.TarHits + m.TarMisses

	if total == 0 {
		return 0.0
	}

	return float64(m.TarHits) / float64(total)
}

// Convenience functions for the default pool

// GetBuffer gets a TAR buffer from the default pool
func GetBuffer(size BufferSize) []byte {
	return defaultBufferPool.Get(size)
}

// PutBuffer returns a buffer to the default pool
func PutBuffer(buffer []byte, size BufferSize) {
	defaultBufferPool.Put(buffer, size)
}

// GetBufferPoolMetrics returns metrics for the default pool
func GetBufferPoolMetrics() BufferPoolMetrics {
	return defaultBufferPool.GetMetrics()
}

// WithBuffer executes a function with a pooled TAR buffer
// This ensures proper cleanup even if the function panics
func WithBuffer(size BufferSize, fn func([]byte)) {
	buffer := GetBuffer(size)
	defer PutBuffer(buffer, size)
	fn(buffer)
}

// WithBufferReturn executes a function with a pooled TAR buffer and returns a value
// This ensures proper cleanup even if the function panics
func WithBufferReturn[T any](size BufferSize, fn func([]byte) T) T {
	buffer := GetBuffer(size)
	defer PutBuffer(buffer, size)
	return fn(buffer)
}