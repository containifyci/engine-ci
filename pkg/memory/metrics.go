package memory

import "sync/atomic"

// MemoryTracker tracks basic memory usage metrics for pools
type MemoryTracker struct {
	enabled int64 // Use atomic for thread-safe enable/disable

	// Pool reuse metrics
	bufferReuseCount int64
}

// Global memory tracker instance
var defaultMemoryTracker = NewMemoryTracker()

// NewMemoryTracker creates a new memory tracker instance
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		enabled: 1, // Start enabled by default
	}
}

// TrackBufferReuse increments buffer reuse counter
func (mt *MemoryTracker) TrackBufferReuse() {
	if atomic.LoadInt64(&mt.enabled) != 1 {
		return
	}

	atomic.AddInt64(&mt.bufferReuseCount, 1)
}

// TrackBufferReuse increments buffer reuse counter using the default tracker
func TrackBufferReuse() {
	defaultMemoryTracker.TrackBufferReuse()
}
