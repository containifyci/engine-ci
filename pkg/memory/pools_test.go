package memory

import (
	"strings"
	"testing"
)

// TestStringBuilderPool tests the string builder pool functionality
func TestStringBuilderPool(t *testing.T) {
	pool := NewStringBuilderPool()

	// Test getting and putting builders of different sizes
	testCases := []struct {
		name string
		size PoolSize
	}{
		{"Small", Small},
		{"Medium", Medium},
		{"Large", Large},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get a builder
			builder := pool.Get(tc.size)
			if builder == nil {
				t.Fatal("Expected non-nil builder")
			}

			// Test that builder is reset
			if builder.Len() != 0 {
				t.Error("Expected empty builder")
			}

			// Use the builder
			builder.WriteString("test content")
			if builder.String() != "test content" {
				t.Error("Builder not working correctly")
			}

			// Put it back
			pool.Put(builder, tc.size)

			// Get another builder and verify it's reset
			builder2 := pool.Get(tc.size)
			if builder2.Len() != 0 {
				t.Error("Expected reset builder from pool")
			}

			pool.Put(builder2, tc.size)
		})
	}
}

// TestStringBuilderPoolMetrics tests pool metrics collection
func TestStringBuilderPoolMetrics(t *testing.T) {
	pool := NewStringBuilderPool()

	// Get and put some builders to generate metrics
	for i := 0; i < 5; i++ {
		builder := pool.Get(Small)
		pool.Put(builder, Small)
	}

	metrics := pool.GetMetrics()

	// Should have some hits and misses
	if metrics.SmallHits == 0 {
		t.Error("Expected some small hits")
	}

	if metrics.SmallMisses == 0 {
		t.Error("Expected some small misses")
	}

	// Test hit rate calculation
	hitRate := metrics.HitRate()
	if hitRate < 0 || hitRate > 1 {
		t.Errorf("Hit rate should be between 0 and 1, got %f", hitRate)
	}
}

// TestWithStringBuilder tests the convenience function
func TestWithStringBuilder(t *testing.T) {
	result := WithStringBuilder(Medium, func(builder *strings.Builder) string {
		builder.WriteString("Hello, ")
		builder.WriteString("World!")
		return builder.String()
	})

	if result != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got '%s'", result)
	}
}

// TestEstimateSize tests size estimation
func TestEstimateSize(t *testing.T) {
	testCases := []struct {
		length   int
		expected PoolSize
	}{
		{100, Small},
		{500, Medium},
		{2000, Large},
		{10000, Large},
	}

	for _, tc := range testCases {
		actual := EstimateSize(tc.length)
		if actual != tc.expected {
			t.Errorf("For length %d, expected %v, got %v", tc.length, tc.expected, actual)
		}
	}
}

// BenchmarkStringBuilderPool benchmarks pool performance
func BenchmarkStringBuilderPool(b *testing.B) {
	pool := NewStringBuilderPool()

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			builder := pool.Get(Medium)
			builder.WriteString("test content that is moderately long")
			_ = builder.String()
			pool.Put(builder, Medium)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			builder := &strings.Builder{}
			builder.Grow(MediumStringBuilderSize)
			builder.WriteString("test content that is moderately long")
			_ = builder.String()
		}
	})
}

// BenchmarkWithStringBuilder benchmarks the convenience function
func BenchmarkWithStringBuilder(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		result := WithStringBuilder(Medium, func(builder *strings.Builder) string {
			builder.WriteString("Hello, ")
			builder.WriteString("World!")
			return builder.String()
		})
		_ = result
	}
}
