package memory

import (
	"sync"
	"sync/atomic"
)

const (
	// Specialized buffer sizes
	HashBufferSize = 65536 // 64KB - optimized for hash operations
	TarBufferSize  = 65536 // 64KB - optimized for tar operations

	// Maximum buffer size to retain in pool (prevents memory bloat)
	MaxRetainedBufferSize = 1048576 // 1MB
)

// BufferSize represents different buffer size categories
type BufferSize int

const (
	HashBuffer BufferSize = iota
	TarBuffer
	// Alias for backward compatibility
	SmallBuffer = HashBuffer
)

// bufferWrapper wraps a byte slice for sync.Pool compatibility
type bufferWrapper struct {
	b []byte
}

// BufferPool manages reusable byte slices for I/O operations
type BufferPool struct {
	hash sync.Pool
	tar  sync.Pool

	// Metrics for monitoring pool efficiency
	hashHits   int64
	tarHits    int64
	hashMisses int64
	tarMisses  int64
}

// Global buffer pool instance
var defaultBufferPool = NewBufferPool()

// NewBufferPool creates a new BufferPool with optimized configurations
func NewBufferPool() *BufferPool {
	pool := &BufferPool{}

	// Initialize specialized hash buffer pool
	pool.hash = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.hashMisses, 1)
			return &bufferWrapper{b: make([]byte, HashBufferSize)}
		},
	}

	// Initialize specialized tar buffer pool
	pool.tar = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.tarMisses, 1)
			return &bufferWrapper{b: make([]byte, TarBufferSize)}
		},
	}

	return pool
}

// Get retrieves a buffer from the appropriate pool
func (p *BufferPool) Get(size BufferSize) []byte {
	switch size {
	case HashBuffer:
		atomic.AddInt64(&p.hashHits, 1)
		return p.hash.Get().(*bufferWrapper).b
	case TarBuffer:
		atomic.AddInt64(&p.tarHits, 1)
		return p.tar.Get().(*bufferWrapper).b
	default:
		// Default to hash buffer
		atomic.AddInt64(&p.hashHits, 1)
		return p.hash.Get().(*bufferWrapper).b
	}
}

// Put returns a buffer to the appropriate pool
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

	switch size {
	case HashBuffer:
		p.hash.Put(wrapper)
	case TarBuffer:
		p.tar.Put(wrapper)
	default:
		p.hash.Put(wrapper)
	}
}

// GetMetrics returns pool efficiency metrics
func (p *BufferPool) GetMetrics() BufferPoolMetrics {
	return BufferPoolMetrics{
		HashHits:   atomic.LoadInt64(&p.hashHits),
		HashMisses: atomic.LoadInt64(&p.hashMisses),
		TarHits:    atomic.LoadInt64(&p.tarHits),
		TarMisses:  atomic.LoadInt64(&p.tarMisses),
	}
}

// BufferPoolMetrics contains statistics about buffer pool usage
type BufferPoolMetrics struct {
	HashHits   int64
	HashMisses int64
	TarHits    int64
	TarMisses  int64
}

// HitRate calculates the overall hit rate across all pools
func (m BufferPoolMetrics) HitRate() float64 {
	totalHits := m.HashHits + m.TarHits
	totalMisses := m.HashMisses + m.TarMisses
	total := totalHits + totalMisses

	if total == 0 {
		return 0.0
	}

	return float64(totalHits) / float64(total)
}

// Convenience functions for the default pool

// GetBuffer gets a buffer from the default pool
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

// WithBuffer executes a function with a pooled buffer
// This ensures proper cleanup even if the function panics
func WithBuffer(size BufferSize, fn func([]byte)) {
	buffer := GetBuffer(size)
	defer PutBuffer(buffer, size)
	fn(buffer)
}

// WithBufferReturn executes a function with a pooled buffer and returns a value
// This ensures proper cleanup even if the function panics
func WithBufferReturn[T any](size BufferSize, fn func([]byte) T) T {
	buffer := GetBuffer(size)
	defer PutBuffer(buffer, size)
	return fn(buffer)
}