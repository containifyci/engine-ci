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
