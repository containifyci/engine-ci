package memory

import (
	"testing"
)

// TestBufferPool tests the TAR buffer pool functionality
func TestBufferPool(t *testing.T) {
	pool := NewBufferPool()

	t.Run("TarBuffer", func(t *testing.T) {
		// Get a buffer
		buffer := pool.Get(TarBuffer)
		if buffer == nil {
			t.Fatal("Expected non-nil buffer")
		}

		if len(buffer) != TarBufferSize {
			t.Errorf("Expected buffer size %d, got %d", TarBufferSize, len(buffer))
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
		pool.Put(buffer, TarBuffer)

		// Get another buffer and verify it's zeroed
		buffer2 := pool.Get(TarBuffer)
		if buffer2[0] != 0 || buffer2[1] != 0 {
			t.Error("Expected zeroed buffer from pool")
		}

		pool.Put(buffer2, TarBuffer)
	})
}

// TestBufferPoolMetrics tests buffer pool metrics collection
func TestBufferPoolMetrics(t *testing.T) {
	pool := NewBufferPool()

	// Get and put some buffers to generate metrics
	for i := 0; i < 5; i++ {
		buffer := pool.Get(TarBuffer)
		pool.Put(buffer, TarBuffer)
	}

	metrics := pool.GetMetrics()

	// Should have some hits and misses
	if metrics.TarHits == 0 {
		t.Error("Expected some tar hits")
	}

	if metrics.TarMisses == 0 {
		t.Error("Expected some tar misses")
	}

	// Test hit rate calculation
	hitRate := metrics.HitRate()
	if hitRate < 0 || hitRate > 1 {
		t.Errorf("Hit rate should be between 0 and 1, got %f", hitRate)
	}
}

// TestWithBuffer tests the convenience function
func TestWithBuffer(t *testing.T) {
	WithBuffer(TarBuffer, func(buffer []byte) {
		if len(buffer) != TarBufferSize {
			t.Errorf("Expected buffer size %d, got %d", TarBufferSize, len(buffer))
		}
		buffer[0] = 42
	})

	// After WithBuffer, the buffer should be returned to pool
	// and the next use should get a zeroed buffer
	WithBuffer(TarBuffer, func(buffer []byte) {
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
	pool.Put(oversized, TarBuffer)

	// The next get should allocate a new buffer, not return the oversized one
	buffer := pool.Get(TarBuffer)
	if len(buffer) != TarBufferSize {
		t.Errorf("Expected buffer size %d, got %d", TarBufferSize, len(buffer))
	}
	if cap(buffer) > MaxRetainedBufferSize {
		t.Error("Pool returned an oversized buffer")
	}
}

// BenchmarkBufferPool benchmarks TAR buffer pool performance
func BenchmarkBufferPool(b *testing.B) {
	pool := NewBufferPool()

	b.Run("Get/Put", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buffer := pool.Get(TarBuffer)
			// Simulate some work
			buffer[0] = byte(i)
			pool.Put(buffer, TarBuffer)
		}
	})

	b.Run("WithBuffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			WithBuffer(TarBuffer, func(buffer []byte) {
				// Simulate some work
				buffer[0] = byte(i)
			})
		}
	})
}

// BenchmarkBufferPoolParallel benchmarks concurrent TAR buffer pool usage
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
