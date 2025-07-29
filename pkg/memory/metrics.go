package memory

import (
	"runtime"
	"sync/atomic"
	"time"
)

// MemoryTracker tracks memory usage and performance metrics
type MemoryTracker struct {
	enabled int64 // Use atomic for thread-safe enable/disable

	// Allocation tracking
	totalAllocations int64
	totalAllocBytes  int64
	peakAllocBytes   int64

	// Performance tracking
	operationCount  int64
	totalDuration   int64 // in nanoseconds
	lastGCTimestamp int64

	// Memory optimization metrics
	poolHitRate      int64 // stored as percentage * 100 for atomic operations
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

// Enable enables memory tracking
func (mt *MemoryTracker) Enable() {
	atomic.StoreInt64(&mt.enabled, 1)
}

// Disable disables memory tracking
func (mt *MemoryTracker) Disable() {
	atomic.StoreInt64(&mt.enabled, 0)
}

// IsEnabled returns whether memory tracking is enabled
func (mt *MemoryTracker) IsEnabled() bool {
	return atomic.LoadInt64(&mt.enabled) == 1
}

// TrackAllocation records an allocation event
func (mt *MemoryTracker) TrackAllocation(bytes int64) {
	if !mt.IsEnabled() {
		return
	}

	atomic.AddInt64(&mt.totalAllocations, 1)
	atomic.AddInt64(&mt.totalAllocBytes, bytes)

	// Update peak allocation if necessary
	for {
		current := atomic.LoadInt64(&mt.peakAllocBytes)
		if bytes <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&mt.peakAllocBytes, current, bytes) {
			break
		}
	}
}

// TrackOperation records the duration of an operation
func (mt *MemoryTracker) TrackOperation(duration time.Duration) {
	if !mt.IsEnabled() {
		return
	}

	atomic.AddInt64(&mt.operationCount, 1)
	atomic.AddInt64(&mt.totalDuration, duration.Nanoseconds())
}

// TrackPoolHitRate updates the pool hit rate
func (mt *MemoryTracker) TrackPoolHitRate(hitRate float64) {
	if !mt.IsEnabled() {
		return
	}

	// Store as percentage * 100 for atomic operations
	atomic.StoreInt64(&mt.poolHitRate, int64(hitRate*10000))
}

// TrackBufferReuse increments buffer reuse counter
func (mt *MemoryTracker) TrackBufferReuse() {
	if !mt.IsEnabled() {
		return
	}

	atomic.AddInt64(&mt.bufferReuseCount, 1)
}

// TrackStringReuse increments string reuse counter
func (mt *MemoryTracker) TrackStringReuse() {
	if !mt.IsEnabled() {
		return
	}

	atomic.AddInt64(&mt.stringReuseCount, 1)
}

// UpdateGCTimestamp updates the last garbage collection timestamp
func (mt *MemoryTracker) UpdateGCTimestamp() {
	if !mt.IsEnabled() {
		return
	}

	atomic.StoreInt64(&mt.lastGCTimestamp, time.Now().UnixNano())
}

// GetMetrics returns current memory metrics
func (mt *MemoryTracker) GetMetrics() MemoryMetrics {
	return MemoryMetrics{
		TotalAllocations: atomic.LoadInt64(&mt.totalAllocations),
		TotalAllocBytes:  atomic.LoadInt64(&mt.totalAllocBytes),
		PeakAllocBytes:   atomic.LoadInt64(&mt.peakAllocBytes),
		OperationCount:   atomic.LoadInt64(&mt.operationCount),
		TotalDuration:    time.Duration(atomic.LoadInt64(&mt.totalDuration)),
		LastGCTimestamp:  time.Unix(0, atomic.LoadInt64(&mt.lastGCTimestamp)),
		PoolHitRate:      float64(atomic.LoadInt64(&mt.poolHitRate)) / 10000,
		BufferReuseCount: atomic.LoadInt64(&mt.bufferReuseCount),
		StringReuseCount: atomic.LoadInt64(&mt.stringReuseCount),
	}
}

// Reset resets all tracked metrics
func (mt *MemoryTracker) Reset() {
	atomic.StoreInt64(&mt.totalAllocations, 0)
	atomic.StoreInt64(&mt.totalAllocBytes, 0)
	atomic.StoreInt64(&mt.peakAllocBytes, 0)
	atomic.StoreInt64(&mt.operationCount, 0)
	atomic.StoreInt64(&mt.totalDuration, 0)
	atomic.StoreInt64(&mt.lastGCTimestamp, 0)
	atomic.StoreInt64(&mt.poolHitRate, 0)
	atomic.StoreInt64(&mt.bufferReuseCount, 0)
	atomic.StoreInt64(&mt.stringReuseCount, 0)
}

// MemoryMetrics contains memory usage and performance statistics
type MemoryMetrics struct {
	LastGCTimestamp  time.Time
	TotalAllocations int64
	TotalAllocBytes  int64
	PeakAllocBytes   int64
	OperationCount   int64
	TotalDuration    time.Duration
	PoolHitRate      float64
	BufferReuseCount int64
	StringReuseCount int64
}

// AverageOperationDuration calculates the average duration per operation
func (m MemoryMetrics) AverageOperationDuration() time.Duration {
	if m.OperationCount == 0 {
		return 0
	}
	return m.TotalDuration / time.Duration(m.OperationCount)
}

// AverageAllocationSize calculates the average allocation size
func (m MemoryMetrics) AverageAllocationSize() int64 {
	if m.TotalAllocations == 0 {
		return 0
	}
	return m.TotalAllocBytes / m.TotalAllocations
}

// ReuseEfficiency calculates the efficiency of resource reuse
func (m MemoryMetrics) ReuseEfficiency() float64 {
	totalReuse := m.BufferReuseCount + m.StringReuseCount
	if m.TotalAllocations == 0 {
		return 0.0
	}
	return float64(totalReuse) / float64(m.TotalAllocations)
}

// SystemMemoryStats returns current system memory statistics
type SystemMemoryStats struct {
	LastGC        time.Time
	HeapAlloc     uint64
	Sys           uint64
	Lookups       uint64
	Mallocs       uint64
	Frees         uint64
	Alloc         uint64
	HeapSys       uint64
	HeapIdle      uint64
	HeapInuse     uint64
	HeapReleased  uint64
	GCCPUFraction float64
	TotalAlloc    uint64
	NumGC         uint32
}

// GetSystemMemoryStats returns current system memory statistics
func GetSystemMemoryStats() SystemMemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMemoryStats{
		Alloc:         m.Alloc,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		Lookups:       m.Lookups,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		HeapReleased:  m.HeapReleased,
		GCCPUFraction: m.GCCPUFraction,
		NumGC:         m.NumGC,
		LastGC:        time.Unix(0, int64(m.LastGC)),
	}
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

// TrackPoolHitRate updates pool hit rate using the default tracker
func TrackPoolHitRate(hitRate float64) {
	defaultMemoryTracker.TrackPoolHitRate(hitRate)
}

// TrackBufferReuse increments buffer reuse counter using the default tracker
func TrackBufferReuse() {
	defaultMemoryTracker.TrackBufferReuse()
}

// TrackStringReuse increments string reuse counter using the default tracker
func TrackStringReuse() {
	defaultMemoryTracker.TrackStringReuse()
}

// GetMemoryMetrics returns current memory metrics from the default tracker
func GetMemoryMetrics() MemoryMetrics {
	return defaultMemoryTracker.GetMetrics()
}

// ResetMemoryMetrics resets all memory metrics in the default tracker
func ResetMemoryMetrics() {
	defaultMemoryTracker.Reset()
}

// EnableMemoryTracking enables memory tracking in the default tracker
func EnableMemoryTracking() {
	defaultMemoryTracker.Enable()
}

// DisableMemoryTracking disables memory tracking in the default tracker
func DisableMemoryTracking() {
	defaultMemoryTracker.Disable()
}

// IsMemoryTrackingEnabled returns whether memory tracking is enabled in the default tracker
func IsMemoryTrackingEnabled() bool {
	return defaultMemoryTracker.IsEnabled()
}

// WithMemoryTracking executes a function with memory tracking enabled
func WithMemoryTracking(fn func()) MemoryMetrics {
	before := GetMemoryMetrics()
	start := time.Now()

	fn()

	duration := time.Since(start)
	TrackOperation(duration)

	after := GetMemoryMetrics()

	// Return the delta metrics
	return MemoryMetrics{
		TotalAllocations: after.TotalAllocations - before.TotalAllocations,
		TotalAllocBytes:  after.TotalAllocBytes - before.TotalAllocBytes,
		PeakAllocBytes:   after.PeakAllocBytes,
		OperationCount:   1,
		TotalDuration:    duration,
		LastGCTimestamp:  after.LastGCTimestamp,
		PoolHitRate:      after.PoolHitRate,
		BufferReuseCount: after.BufferReuseCount - before.BufferReuseCount,
		StringReuseCount: after.StringReuseCount - before.StringReuseCount,
	}
}
