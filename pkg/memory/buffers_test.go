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
		{"Small", SmallBuffer, SmallBufferSize},
		{"Medium", MediumBuffer, MediumBufferSize},
		{"Large", LargeBuffer, LargeBufferSize},
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
		buffer := pool.Get(MediumBuffer)
		pool.Put(buffer, MediumBuffer)
	}

	metrics := pool.GetMetrics()

	// Should have some hits and misses
	if metrics.MediumHits == 0 {
		t.Error("Expected some medium hits")
	}

	if metrics.MediumMisses == 0 {
		t.Error("Expected some medium misses")
	}

	// Test hit rate calculation
	hitRate := metrics.HitRate()
	if hitRate < 0 || hitRate > 1 {
		t.Errorf("Hit rate should be between 0 and 1, got %f", hitRate)
	}
}

// TestWithBuffer tests the convenience function
func TestWithBuffer(t *testing.T) {
	var usedBuffer []byte

	WithBuffer(SmallBuffer, func(buffer []byte) {
		if len(buffer) != SmallBufferSize {
			t.Errorf("Expected buffer size %d, got %d", SmallBufferSize, len(buffer))
		}
		usedBuffer = buffer
		buffer[0] = 42
	})

	// Buffer should have been used
	if usedBuffer == nil {
		t.Error("Buffer was not used")
	}
}

// TestWithBufferReturn tests the convenience function with return value
func TestWithBufferReturn(t *testing.T) {
	result := WithBufferReturn(MediumBuffer, func(buffer []byte) int {
		if len(buffer) != MediumBufferSize {
			t.Errorf("Expected buffer size %d, got %d", MediumBufferSize, len(buffer))
		}
		return len(buffer)
	})

	if result != MediumBufferSize {
		t.Errorf("Expected result %d, got %d", MediumBufferSize, result)
	}
}

// TestEstimateBufferSize tests buffer size estimation
func TestEstimateBufferSize(t *testing.T) {
	testCases := []struct {
		dataSize int
		expected BufferSize
	}{
		{1000, SmallBuffer},
		{10000, MediumBuffer},
		{50000, LargeBuffer},
		{200000, LargeBuffer},
	}

	for _, tc := range testCases {
		actual := EstimateBufferSize(tc.dataSize)
		if actual != tc.expected {
			t.Errorf("For data size %d, expected %v, got %v", tc.dataSize, tc.expected, actual)
		}
	}
}

// BenchmarkBufferPool benchmarks buffer pool performance
func BenchmarkBufferPool(b *testing.B) {
	pool := NewBufferPool()

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buffer := pool.Get(MediumBuffer)
			buffer[0] = byte(i % 256)
			buffer[1] = byte((i * 2) % 256)
			pool.Put(buffer, MediumBuffer)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buffer := make([]byte, MediumBufferSize)
			buffer[0] = byte(i % 256)
			buffer[1] = byte((i * 2) % 256)
			_ = buffer
		}
	})
}

// BenchmarkWithBuffer benchmarks the convenience function
func BenchmarkWithBuffer(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		WithBuffer(MediumBuffer, func(buffer []byte) {
			buffer[0] = byte(i % 256)
			buffer[1] = byte((i * 2) % 256)
		})
	}
}
