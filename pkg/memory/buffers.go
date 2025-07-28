package memory

import (
	"sync"
	"sync/atomic"
)

const (
	// Buffer size categories for different use cases
	SmallBufferSize  = 4096   // 4KB - for small I/O operations
	MediumBufferSize = 32768  // 32KB - for medium I/O operations
	LargeBufferSize  = 131072 // 128KB - for large I/O operations

	// Specialized buffer sizes
	HashBufferSize = 65536 // 64KB - optimized for hash operations
	TarBufferSize  = 65536 // 64KB - optimized for tar operations

	// Maximum buffer size to retain in pool (prevents memory bloat)
	MaxRetainedBufferSize = 1048576 // 1MB
)

// BufferSize represents different buffer size categories
type BufferSize int

const (
	SmallBuffer BufferSize = iota
	MediumBuffer
	LargeBuffer
	HashBuffer
	TarBuffer
)

// BufferPool manages reusable byte slices for I/O operations
type BufferPool struct {
	small  sync.Pool
	medium sync.Pool
	large  sync.Pool
	hash   sync.Pool
	tar    sync.Pool

	// Metrics for monitoring pool efficiency
	smallHits    int64
	mediumHits   int64
	largeHits    int64
	hashHits     int64
	tarHits      int64
	smallMisses  int64
	mediumMisses int64
	largeMisses  int64
	hashMisses   int64
	tarMisses    int64
}

// Global buffer pool instance
var defaultBufferPool = NewBufferPool()

// NewBufferPool creates a new BufferPool with optimized configurations
func NewBufferPool() *BufferPool {
	pool := &BufferPool{}

	// Initialize small buffer pool (for small I/O operations)
	pool.small = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.smallMisses, 1)
			return make([]byte, SmallBufferSize)
		},
	}

	// Initialize medium buffer pool (for typical I/O operations)
	pool.medium = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.mediumMisses, 1)
			return make([]byte, MediumBufferSize)
		},
	}

	// Initialize large buffer pool (for large I/O operations)
	pool.large = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.largeMisses, 1)
			return make([]byte, LargeBufferSize)
		},
	}

	// Initialize specialized hash buffer pool
	pool.hash = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.hashMisses, 1)
			return make([]byte, HashBufferSize)
		},
	}

	// Initialize specialized tar buffer pool
	pool.tar = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.tarMisses, 1)
			return make([]byte, TarBufferSize)
		},
	}

	return pool
}

// Get retrieves a byte slice from the appropriate pool
func (p *BufferPool) Get(size BufferSize) []byte {
	var buffer []byte

	switch size {
	case SmallBuffer:
		atomic.AddInt64(&p.smallHits, 1)
		buffer = p.small.Get().([]byte)
	case MediumBuffer:
		atomic.AddInt64(&p.mediumHits, 1)
		buffer = p.medium.Get().([]byte)
	case LargeBuffer:
		atomic.AddInt64(&p.largeHits, 1)
		buffer = p.large.Get().([]byte)
	case HashBuffer:
		atomic.AddInt64(&p.hashHits, 1)
		buffer = p.hash.Get().([]byte)
	case TarBuffer:
		atomic.AddInt64(&p.tarHits, 1)
		buffer = p.tar.Get().([]byte)
	default:
		// Default to medium size
		atomic.AddInt64(&p.mediumHits, 1)
		buffer = p.medium.Get().([]byte)
	}

	// Clear the buffer before returning (security and correctness)
	for i := range buffer {
		buffer[i] = 0
	}

	return buffer
}

// Put returns a byte slice to the appropriate pool
func (p *BufferPool) Put(buffer []byte, size BufferSize) {
	// Only return to pool if capacity is reasonable to avoid memory bloat
	if len(buffer) > MaxRetainedBufferSize {
		// Don't return oversized buffers to pool
		return
	}

	switch size {
	case SmallBuffer:
		if len(buffer) >= SmallBufferSize {
			p.small.Put(buffer[:SmallBufferSize])
		}
	case MediumBuffer:
		if len(buffer) >= MediumBufferSize {
			p.medium.Put(buffer[:MediumBufferSize])
		}
	case LargeBuffer:
		if len(buffer) >= LargeBufferSize {
			p.large.Put(buffer[:LargeBufferSize])
		}
	case HashBuffer:
		if len(buffer) >= HashBufferSize {
			p.hash.Put(buffer[:HashBufferSize])
		}
	case TarBuffer:
		if len(buffer) >= TarBufferSize {
			p.tar.Put(buffer[:TarBufferSize])
		}
	default:
		if len(buffer) >= MediumBufferSize {
			p.medium.Put(buffer[:MediumBufferSize])
		}
	}
}

// GetMetrics returns buffer pool efficiency metrics
func (p *BufferPool) GetMetrics() BufferPoolMetrics {
	return BufferPoolMetrics{
		SmallHits:    atomic.LoadInt64(&p.smallHits),
		SmallMisses:  atomic.LoadInt64(&p.smallMisses),
		MediumHits:   atomic.LoadInt64(&p.mediumHits),
		MediumMisses: atomic.LoadInt64(&p.mediumMisses),
		LargeHits:    atomic.LoadInt64(&p.largeHits),
		LargeMisses:  atomic.LoadInt64(&p.largeMisses),
		HashHits:     atomic.LoadInt64(&p.hashHits),
		HashMisses:   atomic.LoadInt64(&p.hashMisses),
		TarHits:      atomic.LoadInt64(&p.tarHits),
		TarMisses:    atomic.LoadInt64(&p.tarMisses),
	}
}

// BufferPoolMetrics contains statistics about buffer pool usage
type BufferPoolMetrics struct {
	SmallHits    int64
	SmallMisses  int64
	MediumHits   int64
	MediumMisses int64
	LargeHits    int64
	LargeMisses  int64
	HashHits     int64
	HashMisses   int64
	TarHits      int64
	TarMisses    int64
}

// HitRate calculates the overall hit rate across all buffer pools
func (m BufferPoolMetrics) HitRate() float64 {
	totalHits := m.SmallHits + m.MediumHits + m.LargeHits + m.HashHits + m.TarHits
	totalMisses := m.SmallMisses + m.MediumMisses + m.LargeMisses + m.HashMisses + m.TarMisses
	total := totalHits + totalMisses

	if total == 0 {
		return 0.0
	}

	return float64(totalHits) / float64(total)
}

// Convenience functions for the default buffer pool

// GetBuffer gets a buffer from the default pool
func GetBuffer(size BufferSize) []byte {
	return defaultBufferPool.Get(size)
}

// PutBuffer returns a buffer to the default pool
func PutBuffer(buffer []byte, size BufferSize) {
	defaultBufferPool.Put(buffer, size)
}

// GetBufferPoolMetrics returns metrics for the default buffer pool
func GetBufferPoolMetrics() BufferPoolMetrics {
	return defaultBufferPool.GetMetrics()
}

// ResetBufferPoolMetrics resets all buffer pool metrics (useful for testing)
func ResetBufferPoolMetrics() {
	defaultBufferPool = NewBufferPool()
}

// WithBuffer executes a function with a pooled buffer
// This ensures proper cleanup even if the function panics
func WithBuffer(size BufferSize, fn func([]byte)) {
	buffer := GetBuffer(size)
	defer PutBuffer(buffer, size)
	fn(buffer)
}

// WithBufferReturn executes a function with a pooled buffer and returns a result
// This ensures proper cleanup even if the function panics
func WithBufferReturn[T any](size BufferSize, fn func([]byte) T) T {
	buffer := GetBuffer(size)
	defer PutBuffer(buffer, size)
	return fn(buffer)
}

// EstimateBufferSize estimates the appropriate buffer size based on expected data size
func EstimateBufferSize(estimatedDataSize int) BufferSize {
	switch {
	case estimatedDataSize <= SmallBufferSize:
		return SmallBuffer
	case estimatedDataSize <= MediumBufferSize:
		return MediumBuffer
	case estimatedDataSize <= LargeBufferSize:
		return LargeBuffer
	default:
		return LargeBuffer
	}
}
