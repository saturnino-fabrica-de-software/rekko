package repository

import (
	"testing"

	"github.com/pgvector/pgvector-go"
)

// BenchmarkFloat32Conversion benchmarks the conversion overhead
func BenchmarkFloat32Conversion(b *testing.B) {
	embedding := make([]float64, embeddingSize)
	for i := range embedding {
		embedding[i] = float64(i) / float64(embeddingSize)
	}

	b.Run("WithPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			floatsPtr := float32Pool.Get().(*[]float32)
			floats := (*floatsPtr)[:embeddingSize]

			for j, v := range embedding {
				floats[j] = float32(v)
			}

			_ = pgvector.NewVector(floats)
			float32Pool.Put(floatsPtr)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			floats := make([]float32, embeddingSize)

			for j, v := range embedding {
				floats[j] = float32(v)
			}

			_ = pgvector.NewVector(floats)
		}
	})
}

// BenchmarkParallelFloat32Conversion tests concurrent pool access
func BenchmarkParallelFloat32Conversion(b *testing.B) {
	embedding := make([]float64, embeddingSize)
	for i := range embedding {
		embedding[i] = float64(i) / float64(embeddingSize)
	}

	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			floatsPtr := float32Pool.Get().(*[]float32)
			floats := (*floatsPtr)[:embeddingSize]

			for j, v := range embedding {
				floats[j] = float32(v)
			}

			_ = pgvector.NewVector(floats)
			float32Pool.Put(floatsPtr)
		}
	})
}
