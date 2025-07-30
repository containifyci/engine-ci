package memory

import (
	"sync/atomic"
	"time"
)

// MemoryTracker tracks basic memory usage metrics for pools
type MemoryTracker struct {
	enabled int64 // Use atomic for thread-safe enable/disable

	// Basic allocation tracking
	totalAllocations int64
	totalAllocBytes  int64

	// Performance tracking
	operationCount int64
	totalDuration  int64 // in nanoseconds

	// Pool reuse metrics
	bufferReuseCount int64
	stringReuseCount int64
}

// Global memory tracker instance
var defaultMemoryTracker = NewMemoryTracker()

// NewMemoryTracker creates a new memory tracker instance
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		enabled: 1, // Start enabled by default
	}
}

// TrackAllocation records an allocation event
func (mt *MemoryTracker) TrackAllocation(bytes int64) {
	if atomic.LoadInt64(&mt.enabled) != 1 {
		return
	}

	atomic.AddInt64(&mt.totalAllocations, 1)
	atomic.AddInt64(&mt.totalAllocBytes, bytes)
}

// TrackOperation records the duration of an operation
func (mt *MemoryTracker) TrackOperation(duration time.Duration) {
	if atomic.LoadInt64(&mt.enabled) != 1 {
		return
	}

	atomic.AddInt64(&mt.operationCount, 1)
	atomic.AddInt64(&mt.totalDuration, duration.Nanoseconds())
}

// TrackBufferReuse increments buffer reuse counter
func (mt *MemoryTracker) TrackBufferReuse() {
	if atomic.LoadInt64(&mt.enabled) != 1 {
		return
	}

	atomic.AddInt64(&mt.bufferReuseCount, 1)
}

// TrackStringReuse increments string reuse counter
func (mt *MemoryTracker) TrackStringReuse() {
	if atomic.LoadInt64(&mt.enabled) != 1 {
		return
	}

	atomic.AddInt64(&mt.stringReuseCount, 1)
}

// Convenience functions for the default tracker

// TrackAllocation records an allocation using the default tracker
func TrackAllocation(bytes int64) {
	defaultMemoryTracker.TrackAllocation(bytes)
}

// TrackOperation records an operation duration using the default tracker
func TrackOperation(duration time.Duration) {
	defaultMemoryTracker.TrackOperation(duration)
}

// TrackBufferReuse increments buffer reuse counter using the default tracker
func TrackBufferReuse() {
	defaultMemoryTracker.TrackBufferReuse()
}

// TrackStringReuse increments string reuse counter using the default tracker
func TrackStringReuse() {
	defaultMemoryTracker.TrackStringReuse()
}