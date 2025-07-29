package memory

import (
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// StringBuilderSize is the default size for string operations
	StringBuilderSize = 1024 // For string operations (like AsFlags)
	DefaultBufferSize = 8192 // Default buffer size for I/O operations
)

// StringBuilderPool manages reusable string.Builder instances
type StringBuilderPool struct {
	pool sync.Pool

	// Metrics for monitoring pool efficiency
	hits   int64
	misses int64
}

// Global string builder pool instance
var defaultStringBuilderPool = NewStringBuilderPool()

// NewStringBuilderPool creates a new StringBuilderPool
func NewStringBuilderPool() *StringBuilderPool {
	pool := &StringBuilderPool{}

	// Initialize pool
	pool.pool = sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pool.misses, 1)
			builder := &strings.Builder{}
			builder.Grow(StringBuilderSize)
			return builder
		},
	}

	return pool
}

// Get retrieves a string.Builder from the pool
func (p *StringBuilderPool) Get() *strings.Builder {
	atomic.AddInt64(&p.hits, 1)
	builder := p.pool.Get().(*strings.Builder)
	// Reset the builder before returning
	builder.Reset()
	return builder
}

// Put returns a string.Builder to the pool
func (p *StringBuilderPool) Put(builder *strings.Builder) {
	// Only return to pool if capacity is reasonable to avoid memory bloat
	const maxRetainedCapacity = StringBuilderSize * 2

	if builder.Cap() > maxRetainedCapacity {
		// Don't return oversized builders to pool
		return
	}

	p.pool.Put(builder)
}

// GetMetrics returns pool efficiency metrics
func (p *StringBuilderPool) GetMetrics() PoolMetrics {
	return PoolMetrics{
		Hits:   atomic.LoadInt64(&p.hits),
		Misses: atomic.LoadInt64(&p.misses),
	}
}

// PoolMetrics contains statistics about pool usage
type PoolMetrics struct {
	Hits   int64
	Misses int64
}

// HitRate calculates the hit rate
func (m PoolMetrics) HitRate() float64 {
	total := m.Hits + m.Misses
	if total == 0 {
		return 0.0
	}
	return float64(m.Hits) / float64(total)
}

// Convenience functions for the default pool

// GetStringBuilder gets a string builder from the default pool
func GetStringBuilder() *strings.Builder {
	return defaultStringBuilderPool.Get()
}

// PutStringBuilder returns a string builder to the default pool
func PutStringBuilder(builder *strings.Builder) {
	defaultStringBuilderPool.Put(builder)
}

// GetPoolMetrics returns metrics for the default pool
func GetPoolMetrics() PoolMetrics {
	return defaultStringBuilderPool.GetMetrics()
}

// WithStringBuilder executes a function with a pooled string builder
// This ensures proper cleanup even if the function panics
func WithStringBuilder(fn func(*strings.Builder) string) string {
	builder := GetStringBuilder()
	defer PutStringBuilder(builder)
	return fn(builder)
}