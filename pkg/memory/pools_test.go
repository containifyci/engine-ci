package memory

import (
	"strings"
	"testing"
)

// TestStringBuilderPool tests the string builder pool functionality
func TestStringBuilderPool(t *testing.T) {
	pool := NewStringBuilderPool()

	// Get a builder
	builder := pool.Get()
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
	pool.Put(builder)

	// Get another builder and verify it's reset
	builder2 := pool.Get()
	if builder2.Len() != 0 {
		t.Error("Expected reset builder from pool")
	}

	pool.Put(builder2)
}

// TestStringBuilderPoolMetrics tests string builder pool metrics collection
func TestStringBuilderPoolMetrics(t *testing.T) {
	pool := NewStringBuilderPool()

	// Get and put some builders to generate metrics
	for i := 0; i < 5; i++ {
		builder := pool.Get()
		pool.Put(builder)
	}

	metrics := pool.GetMetrics()

	// Should have some hits and misses
	if metrics.Hits == 0 {
		t.Error("Expected some hits")
	}

	if metrics.Misses == 0 {
		t.Error("Expected some misses")
	}

	// Test hit rate calculation
	hitRate := metrics.HitRate()
	if hitRate < 0 || hitRate > 1 {
		t.Errorf("Hit rate should be between 0 and 1, got %f", hitRate)
	}
}

// TestWithStringBuilder tests the convenience function
func TestWithStringBuilder(t *testing.T) {
	result := WithStringBuilder(func(builder *strings.Builder) string {
		builder.WriteString("Hello, ")
		builder.WriteString("World!")
		return builder.String()
	})

	if result != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got %s", result)
	}

	// Test that panic is handled and builder is still returned to pool
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic to be propagated")
			}
		}()

		WithStringBuilder(func(builder *strings.Builder) string {
			panic("test panic")
		})
	}()
}

// TestOversizedStringBuilder tests that oversized builders are not pooled
func TestOversizedStringBuilder(t *testing.T) {
	pool := NewStringBuilderPool()

	// Get a builder and grow it beyond the max retained capacity
	builder := pool.Get()
	for i := 0; i < StringBuilderSize*3; i++ {
		builder.WriteByte('x')
	}

	// Put it back
	pool.Put(builder)

	// The next get should return a new builder with normal capacity
	builder2 := pool.Get()
	if builder2.Cap() > StringBuilderSize*2 {
		t.Error("Pool returned an oversized builder")
	}
	pool.Put(builder2)
}

// BenchmarkStringBuilderPool benchmarks string builder pool performance
func BenchmarkStringBuilderPool(b *testing.B) {
	pool := NewStringBuilderPool()

	b.Run("Get/Put", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := pool.Get()
			builder.WriteString("test")
			pool.Put(builder)
		}
	})

	b.Run("WithStringBuilder", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = WithStringBuilder(func(builder *strings.Builder) string {
				builder.WriteString("test")
				return builder.String()
			})
		}
	})

	b.Run("NoPool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			builder := &strings.Builder{}
			builder.WriteString("test")
			_ = builder.String()
		}
	})
}

// BenchmarkStringBuilderPoolParallel benchmarks concurrent string builder pool usage
func BenchmarkStringBuilderPoolParallel(b *testing.B) {
	pool := NewStringBuilderPool()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			builder := pool.Get()
			builder.WriteString("parallel test")
			_ = builder.String()
			pool.Put(builder)
		}
	})
}