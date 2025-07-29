package memory

import (
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// Pool size thresholds for different operation types
	SmallStringBuilderSize  = 256  // For small string operations
	MediumStringBuilderSize = 1024 // For medium string operations (like AsFlags)
	LargeStringBuilderSize  = 4096 // For large string operations
	DefaultBufferSize       = 8192 // Default buffer size for I/O operations
)

// StringBuilderPool manages reusable string.Builder instances
type StringBuilderPool struct {
	small  sync.Pool
	medium sync.Pool
	large  sync.Pool

	// Metrics for monitoring pool efficiency
	smallHits    int64
	mediumHits   int64
	largeHits    int64
	smallMisses  int64
	mediumMisses int64
	largeMisses  int64
}

// PoolSize represents different pool size categories
type PoolSize int

const (
	Small PoolSize = iota
	Medium
	Large
)

// Global string builder pool instance
var defaultStringBuilderPool = NewStringBuilderPool()

// NewStringBuilderPool creates a new StringBuilderPool with optimized configurations
func NewStringBuilderPool() *StringBuilderPool {
	pool := &StringBuilderPool{}

	// Initialize small pool (for image tags, short strings)
	pool.small = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.smallMisses, 1)
			builder := &strings.Builder{}
			builder.Grow(SmallStringBuilderSize)
			return builder
		},
	}

	// Initialize medium pool (for AsFlags, configuration strings)
	pool.medium = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.mediumMisses, 1)
			builder := &strings.Builder{}
			builder.Grow(MediumStringBuilderSize)
			return builder
		},
	}

	// Initialize large pool (for complex operations)
	pool.large = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.largeMisses, 1)
			builder := &strings.Builder{}
			builder.Grow(LargeStringBuilderSize)
			return builder
		},
	}

	return pool
}

// Get retrieves a string.Builder from the appropriate pool
func (p *StringBuilderPool) Get(size PoolSize) *strings.Builder {
	var builder *strings.Builder

	switch size {
	case Small:
		atomic.AddInt64(&p.smallHits, 1)
		builder = p.small.Get().(*strings.Builder)
	case Medium:
		atomic.AddInt64(&p.mediumHits, 1)
		builder = p.medium.Get().(*strings.Builder)
	case Large:
		atomic.AddInt64(&p.largeHits, 1)
		builder = p.large.Get().(*strings.Builder)
	default:
		// Default to medium size
		atomic.AddInt64(&p.mediumHits, 1)
		builder = p.medium.Get().(*strings.Builder)
	}

	// Reset the builder before returning
	builder.Reset()
	return builder
}

// Put returns a string.Builder to the appropriate pool
func (p *StringBuilderPool) Put(builder *strings.Builder, size PoolSize) {
	// Only return to pool if capacity is reasonable to avoid memory bloat
	const maxRetainedCapacity = LargeStringBuilderSize * 2

	if builder.Cap() > maxRetainedCapacity {
		// Don't return oversized builders to pool
		return
	}

	switch size {
	case Small:
		p.small.Put(builder)
	case Medium:
		p.medium.Put(builder)
	case Large:
		p.large.Put(builder)
	default:
		p.medium.Put(builder)
	}
}

// GetMetrics returns pool efficiency metrics
func (p *StringBuilderPool) GetMetrics() PoolMetrics {
	return PoolMetrics{
		SmallHits:    atomic.LoadInt64(&p.smallHits),
		SmallMisses:  atomic.LoadInt64(&p.smallMisses),
		MediumHits:   atomic.LoadInt64(&p.mediumHits),
		MediumMisses: atomic.LoadInt64(&p.mediumMisses),
		LargeHits:    atomic.LoadInt64(&p.largeHits),
		LargeMisses:  atomic.LoadInt64(&p.largeMisses),
	}
}

// PoolMetrics contains statistics about pool usage
type PoolMetrics struct {
	SmallHits    int64
	SmallMisses  int64
	MediumHits   int64
	MediumMisses int64
	LargeHits    int64
	LargeMisses  int64
}

// HitRate calculates the overall hit rate across all pools
func (m PoolMetrics) HitRate() float64 {
	totalHits := m.SmallHits + m.MediumHits + m.LargeHits
	totalMisses := m.SmallMisses + m.MediumMisses + m.LargeMisses
	total := totalHits + totalMisses

	if total == 0 {
		return 0.0
	}

	return float64(totalHits) / float64(total)
}

// Convenience functions for the default pool

// GetStringBuilder gets a string builder from the default pool
func GetStringBuilder(size PoolSize) *strings.Builder {
	return defaultStringBuilderPool.Get(size)
}

// PutStringBuilder returns a string builder to the default pool
func PutStringBuilder(builder *strings.Builder, size PoolSize) {
	defaultStringBuilderPool.Put(builder, size)
}

// GetPoolMetrics returns metrics for the default pool
func GetPoolMetrics() PoolMetrics {
	return defaultStringBuilderPool.GetMetrics()
}

// ResetPoolMetrics resets all pool metrics (useful for testing)
func ResetPoolMetrics() {
	defaultStringBuilderPool = NewStringBuilderPool()
}

// WithStringBuilder executes a function with a pooled string builder
// This ensures proper cleanup even if the function panics
func WithStringBuilder(size PoolSize, fn func(*strings.Builder) string) string {
	builder := GetStringBuilder(size)
	defer PutStringBuilder(builder, size)
	return fn(builder)
}

// EstimateSize estimates the appropriate pool size based on expected content
func EstimateSize(estimatedLength int) PoolSize {
	switch {
	case estimatedLength <= SmallStringBuilderSize:
		return Small
	case estimatedLength <= MediumStringBuilderSize:
		return Medium
	default:
		return Large
	}
}
