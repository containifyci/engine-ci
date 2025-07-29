package memory

import (
	"testing"
)

// TestBufferPool tests the buffer pool functionality
func TestBufferPool(t *testing.T) {
	pool := NewBufferPool()

	// Test getting and putting buffers of different sizes
	testCases := []struct {
		name         string
		size         BufferSize
		expectedSize int
	}{
		{"Hash", HashBuffer, HashBufferSize},
		{"Tar", TarBuffer, TarBufferSize},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get a buffer
			buffer := pool.Get(tc.size)
			if buffer == nil {
				t.Fatal("Expected non-nil buffer")
			}

			if len(buffer) != tc.expectedSize {
				t.Errorf("Expected buffer size %d, got %d", tc.expectedSize, len(buffer))
			}

			// Test that buffer is zeroed
			for i, b := range buffer {
				if b != 0 {
					t.Errorf("Expected zero byte at index %d, got %d", i, b)
				}
			}

			// Use the buffer
			buffer[0] = 1
			buffer[1] = 2

			// Put it back
			pool.Put(buffer, tc.size)

			// Get another buffer and verify it's zeroed
			buffer2 := pool.Get(tc.size)
			if buffer2[0] != 0 || buffer2[1] != 0 {
				t.Error("Expected zeroed buffer from pool")
			}

			pool.Put(buffer2, tc.size)
		})
	}
}

// TestBufferPoolMetrics tests buffer pool metrics collection
func TestBufferPoolMetrics(t *testing.T) {
	pool := NewBufferPool()

	// Get and put some buffers to generate metrics
	for i := 0; i < 5; i++ {
		buffer := pool.Get(HashBuffer)
		pool.Put(buffer, HashBuffer)
	}

	metrics := pool.GetMetrics()

	// Should have some hits and misses
	if metrics.HashHits == 0 {
		t.Error("Expected some hash hits")
	}

	if metrics.HashMisses == 0 {
		t.Error("Expected some hash misses")
	}

	// Test hit rate calculation
	hitRate := metrics.HitRate()
	if hitRate < 0 || hitRate > 1 {
		t.Errorf("Hit rate should be between 0 and 1, got %f", hitRate)
	}
}

// TestWithBuffer tests the convenience function
func TestWithBuffer(t *testing.T) {
	WithBuffer(HashBuffer, func(buffer []byte) {
		if len(buffer) != HashBufferSize {
			t.Errorf("Expected buffer size %d, got %d", HashBufferSize, len(buffer))
		}
		buffer[0] = 42
	})

	// After WithBuffer, the buffer should be returned to pool
	// and the next use should get a zeroed buffer
	WithBuffer(HashBuffer, func(buffer []byte) {
		if buffer[0] != 0 {
			t.Error("Expected zeroed buffer from pool")
		}
	})
}

// TestWithBufferReturn tests the WithBufferReturn function
func TestWithBufferReturn(t *testing.T) {
	result := WithBufferReturn(TarBuffer, func(buffer []byte) string {
		if len(buffer) != TarBufferSize {
			t.Errorf("Expected buffer size %d, got %d", TarBufferSize, len(buffer))
		}
		return "test-result"
	})

	if result != "test-result" {
		t.Errorf("Expected 'test-result', got %s", result)
	}
}

// TestOversizedBuffer tests that oversized buffers are not pooled
func TestOversizedBuffer(t *testing.T) {
	pool := NewBufferPool()

	// Create an oversized buffer
	oversized := make([]byte, MaxRetainedBufferSize+1)
	
	// Try to put it back
	pool.Put(oversized, HashBuffer)

	// The next get should allocate a new buffer, not return the oversized one
	buffer := pool.Get(HashBuffer)
	if len(buffer) != HashBufferSize {
		t.Errorf("Expected buffer size %d, got %d", HashBufferSize, len(buffer))
	}
	if cap(buffer) > MaxRetainedBufferSize {
		t.Error("Pool returned an oversized buffer")
	}
}

// BenchmarkBufferPool benchmarks buffer pool performance
func BenchmarkBufferPool(b *testing.B) {
	pool := NewBufferPool()

	b.Run("Get/Put", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buffer := pool.Get(HashBuffer)
			// Simulate some work
			buffer[0] = byte(i)
			pool.Put(buffer, HashBuffer)
		}
	})

	b.Run("WithBuffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			WithBuffer(HashBuffer, func(buffer []byte) {
				// Simulate some work
				buffer[0] = byte(i)
			})
		}
	})
}

// BenchmarkBufferPoolParallel benchmarks concurrent buffer pool usage
func BenchmarkBufferPoolParallel(b *testing.B) {
	pool := NewBufferPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buffer := pool.Get(TarBuffer)
			// Simulate some work
			buffer[0] = 1
			pool.Put(buffer, TarBuffer)
		}
	})
}