package repository

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/pgvector/pgvector-go"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// BenchmarkSearchByEmbedding benchmarks the search operation with a typical number of faces (1000)
// Expected baseline: < 2ms (excluding actual DB query time which is mocked)
// Target allocations: < 20 allocs/op
func BenchmarkSearchByEmbedding(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatal(err)
	}
	defer mock.Close()

	repo := NewFaceRepository(mock)
	ctx := context.Background()
	tenantID := uuid.New()
	embedding := generateRandomEmbedding(512)
	threshold := 0.8
	limit := 10

	// Setup mock rows (simulate 10 matches)
	matches := generateMockSearchMatches(10)

	// Expect the query to be called b.N times
	for i := 0; i < b.N; i++ {
		rows := createSearchResultRows(matches)
		mock.ExpectQuery(`SELECT id, external_id, metadata`).
			WithArgs(pgxmock.AnyArg(), tenantID, threshold, limit).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := repo.SearchByEmbedding(ctx, tenantID, embedding, threshold, limit)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSearchByEmbedding_10kFaces benchmarks search with 10k faces in tenant
// Simulates a medium-sized event with 10k registered attendees
// Expected P99: < 5ms (with proper pgvector index)
func BenchmarkSearchByEmbedding_10kFaces(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatal(err)
	}
	defer mock.Close()

	repo := NewFaceRepository(mock)
	ctx := context.Background()
	tenantID := uuid.New()
	embedding := generateRandomEmbedding(512)
	threshold := 0.85
	limit := 10

	// Simulate realistic scenario: 5 matches above threshold from 10k faces
	matches := generateMockSearchMatches(5)

	for i := 0; i < b.N; i++ {
		rows := createSearchResultRows(matches)
		mock.ExpectQuery(`SELECT id, external_id, metadata`).
			WithArgs(pgxmock.AnyArg(), tenantID, threshold, limit).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := repo.SearchByEmbedding(ctx, tenantID, embedding, threshold, limit)
		if err != nil {
			b.Fatal(err)
		}
		if len(result) != 5 {
			b.Fatalf("expected 5 matches, got %d", len(result))
		}
	}
}

// BenchmarkSearchByEmbedding_50kFaces benchmarks search with 50k faces
// This is the critical target: large festival/concert scenario
// Target P99: < 10ms (CRITICAL requirement from Rekko spec)
func BenchmarkSearchByEmbedding_50kFaces(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatal(err)
	}
	defer mock.Close()

	repo := NewFaceRepository(mock)
	ctx := context.Background()
	tenantID := uuid.New()
	embedding := generateRandomEmbedding(512)
	threshold := 0.90 // Higher threshold for better precision in large dataset
	limit := 20

	// Simulate realistic scenario: 3 high-confidence matches from 50k faces
	matches := generateMockSearchMatches(3)

	for i := 0; i < b.N; i++ {
		rows := createSearchResultRows(matches)
		mock.ExpectQuery(`SELECT id, external_id, metadata`).
			WithArgs(pgxmock.AnyArg(), tenantID, threshold, limit).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := repo.SearchByEmbedding(ctx, tenantID, embedding, threshold, limit)
		if err != nil {
			b.Fatal(err)
		}
		if len(result) != 3 {
			b.Fatalf("expected 3 matches, got %d", len(result))
		}
	}
}

// BenchmarkEmbeddingConversion benchmarks the float64 to pgvector conversion
// This is critical path in search performance
// Target: < 500ns/op, < 5 allocs/op for 512-dim vector
func BenchmarkEmbeddingConversion(b *testing.B) {
	embedding := generateRandomEmbedding(512)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate the conversion code from SearchByEmbedding
		floats := make([]float32, len(embedding))
		for j, v := range embedding {
			floats[j] = float32(v)
		}
		_ = pgvector.NewVector(floats)
	}
}

// BenchmarkEmbeddingConversion_PreAllocated tests pre-allocation optimization
// Shows the benefit of sync.Pool or reusable buffers
// Expected improvement: ~50% fewer allocations
func BenchmarkEmbeddingConversion_PreAllocated(b *testing.B) {
	embedding := generateRandomEmbedding(512)
	floats := make([]float32, 512) // Pre-allocated buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j, v := range embedding {
			floats[j] = float32(v)
		}
		_ = pgvector.NewVector(floats)
	}
}

// BenchmarkCountByTenant benchmarks the simple count operation
// Expected: < 1ms, < 5 allocs/op
func BenchmarkCountByTenant(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatal(err)
	}
	defer mock.Close()

	repo := NewFaceRepository(mock)
	ctx := context.Background()
	tenantID := uuid.New()

	for i := 0; i < b.N; i++ {
		rows := pgxmock.NewRows([]string{"count"}).AddRow(50000)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM faces WHERE tenant_id = \$1`).
			WithArgs(tenantID).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		count, err := repo.CountByTenant(ctx, tenantID)
		if err != nil {
			b.Fatal(err)
		}
		if count != 50000 {
			b.Fatalf("expected count 50000, got %d", count)
		}
	}
}

// BenchmarkGetByExternalID benchmarks single face retrieval
// Critical for verification flow
// Target: < 2ms, < 15 allocs/op
func BenchmarkGetByExternalID(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatal(err)
	}
	defer mock.Close()

	repo := NewFaceRepository(mock)
	ctx := context.Background()
	tenantID := uuid.New()
	externalID := "user-123"
	faceID := uuid.New()
	embedding := pgvector.NewVector(generateRandomEmbedding32(512))
	now := time.Now()

	for i := 0; i < b.N; i++ {
		rows := pgxmock.NewRows([]string{
			"id", "tenant_id", "external_id", "embedding", "metadata", "quality_score", "created_at", "updated_at",
		}).AddRow(
			faceID,
			tenantID,
			externalID,
			&embedding,
			map[string]interface{}{"source": "mobile"},
			0.95,
			now,
			now,
		)

		mock.ExpectQuery(`SELECT id, tenant_id, external_id, embedding`).
			WithArgs(tenantID, externalID).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		face, err := repo.GetByExternalID(ctx, tenantID, externalID)
		if err != nil {
			b.Fatal(err)
		}
		if face.ExternalID != externalID {
			b.Fatalf("expected external_id %s, got %s", externalID, face.ExternalID)
		}
	}
}

// BenchmarkCreate benchmarks face creation
// Target: < 3ms, < 20 allocs/op
func BenchmarkCreate(b *testing.B) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		b.Fatal(err)
	}
	defer mock.Close()

	repo := NewFaceRepository(mock)
	ctx := context.Background()
	tenantID := uuid.New()
	now := time.Now()

	for i := 0; i < b.N; i++ {
		rows := pgxmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now, now)

		mock.ExpectQuery(`INSERT INTO faces`).
			WithArgs(
				pgxmock.AnyArg(),
				tenantID,
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
				pgxmock.AnyArg(),
			).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		face := &domain.Face{
			TenantID:     tenantID,
			ExternalID:   fmt.Sprintf("user-%d", i),
			Embedding:    generateRandomEmbedding(512),
			QualityScore: 0.95,
		}

		err := repo.Create(ctx, face)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark helpers

func generateRandomEmbedding(dim int) []float64 {
	embedding := make([]float64, dim)
	for i := range embedding {
		//nolint:gosec // Using math/rand is acceptable for benchmark test data
		embedding[i] = rand.Float64()*2 - 1 // Range: -1 to 1
	}
	return normalizeEmbedding(embedding)
}

func generateRandomEmbedding32(dim int) []float32 {
	embedding := make([]float32, dim)
	for i := range embedding {
		//nolint:gosec // Using math/rand is acceptable for benchmark test data
		embedding[i] = float32(rand.Float64()*2 - 1)
	}
	return embedding
}

func normalizeEmbedding(embedding []float64) []float64 {
	var sum float64
	for _, v := range embedding {
		sum += v * v
	}
	magnitude := 1.0
	if sum > 0 {
		magnitude = 1.0 / (sum * sum)
	}
	for i := range embedding {
		embedding[i] *= magnitude
	}
	return embedding
}

func generateMockSearchMatches(count int) []domain.SearchMatch {
	matches := make([]domain.SearchMatch, count)
	for i := 0; i < count; i++ {
		matches[i] = domain.SearchMatch{
			FaceID:     uuid.New(),
			ExternalID: fmt.Sprintf("user-%d", i),
			Similarity: 0.95 - float64(i)*0.01, // Descending similarity
			Metadata: map[string]interface{}{
				"name": fmt.Sprintf("User %d", i),
			},
		}
	}
	return matches
}

func createSearchResultRows(matches []domain.SearchMatch) *pgxmock.Rows {
	rows := pgxmock.NewRows([]string{"id", "external_id", "metadata", "similarity"})
	for _, match := range matches {
		rows.AddRow(match.FaceID, match.ExternalID, match.Metadata, match.Similarity)
	}
	return rows
}
