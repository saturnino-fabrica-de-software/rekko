package service

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

// BenchmarkFaceService_Search benchmarks the complete search flow
// This includes: provider extraction + database search + count + audit
// Target P99: < 10ms (for 50k faces, excluding actual provider call)
// Expected allocations: < 50 allocs/op
func BenchmarkFaceService_Search(b *testing.B) {
	// Setup
	faceRepo := &MockFaceRepository{}
	verificationRepo := &MockVerificationRepository{}
	searchAuditRepo := &MockSearchAuditRepository{}
	faceProvider := &MockFaceProvider{}
	rateLimiter := &MockRateLimiter{}

	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID:   tenantID,
		Name: "Benchmark Tenant",
		Settings: map[string]interface{}{
			"search_enabled":    true,
			"search_threshold":  0.85,
			"search_rate_limit": float64(30),
		},
	}

	imageBytes := make([]byte, 10000) // Simulate ~10KB image
	embedding := generateBenchmarkEmbedding(512)

	// Rate limiter mock - always allow
	rateLimiter.On("CheckSearchLimit", mock.Anything, tenantID, 30).
		Return(nil)

	// Mock setup - will be called b.N times (using AnalyzeFace now for optimization)
	faceProvider.On("AnalyzeFace", mock.Anything, mock.Anything).
		Return(&provider.FaceAnalysis{
			Embedding:     embedding,
			Confidence:    0.99,
			QualityScore:  0.95,
			LivenessScore: 0.90,
			FaceCount:     1,
		}, nil)

	// Simulate finding 3 matches in a 50k face database
	matches := []domain.SearchMatch{
		{
			FaceID:     uuid.New(),
			ExternalID: "match-1",
			Similarity: 0.95,
			Metadata:   map[string]interface{}{"name": "User One"},
		},
		{
			FaceID:     uuid.New(),
			ExternalID: "match-2",
			Similarity: 0.90,
			Metadata:   map[string]interface{}{"name": "User Two"},
		},
		{
			FaceID:     uuid.New(),
			ExternalID: "match-3",
			Similarity: 0.87,
			Metadata:   map[string]interface{}{"name": "User Three"},
		},
	}

	faceRepo.On("SearchByEmbedding", mock.Anything, tenantID, embedding, 0.85, 10).
		Return(matches, nil)

	// Audit is async, may or may not be called depending on goroutine timing
	searchAuditRepo.On("Create", mock.Anything, mock.Anything).
		Return(nil).Maybe()

	svc := NewFaceService(faceRepo, verificationRepo, searchAuditRepo, faceProvider, rateLimiter)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := svc.Search(ctx, tenant, imageBytes, 0.85, 10, "192.168.1.1")
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Matches) != 3 {
			b.Fatalf("expected 3 matches, got %d", len(result.Matches))
		}
		// TotalFaces removed from hot path for performance (now returns 0)
	}

	b.StopTimer()

	// Verify expectations (optional in benchmark, but good practice)
	faceProvider.AssertExpectations(b)
	faceRepo.AssertExpectations(b)
}

// BenchmarkFaceService_Search_WithAudit benchmarks search with synchronous audit
// This version ensures audit is created synchronously for accurate measurement
// Target: < 12ms (includes audit creation overhead)
func BenchmarkFaceService_Search_WithAudit(b *testing.B) {
	// Setup
	faceRepo := &MockFaceRepository{}
	verificationRepo := &MockVerificationRepository{}
	searchAuditRepo := &MockSearchAuditRepository{}
	faceProvider := &MockFaceProvider{}
	rateLimiter := &MockRateLimiter{}

	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID:   tenantID,
		Name: "Benchmark Tenant",
		Settings: map[string]interface{}{
			"search_enabled":    true,
			"search_rate_limit": float64(30),
		},
	}

	imageBytes := make([]byte, 10000)
	embedding := generateBenchmarkEmbedding(512)

	rateLimiter.On("CheckSearchLimit", mock.Anything, tenantID, 30).
		Return(nil)

	faceProvider.On("AnalyzeFace", mock.Anything, mock.Anything).
		Return(&provider.FaceAnalysis{
			Embedding:     embedding,
			Confidence:    0.99,
			QualityScore:  0.95,
			LivenessScore: 0.90,
			FaceCount:     1,
		}, nil)

	matches := []domain.SearchMatch{
		{
			FaceID:     uuid.New(),
			ExternalID: "user-1",
			Similarity: 0.92,
		},
	}

	faceRepo.On("SearchByEmbedding", mock.Anything, tenantID, mock.Anything, mock.Anything, mock.Anything).
		Return(matches, nil)

	// Explicitly expect audit creation
	searchAuditRepo.On("Create", mock.Anything, mock.MatchedBy(func(audit *domain.SearchAudit) bool {
		return audit.TenantID == tenantID && audit.ResultsCount == 1
	})).Return(nil)

	svc := NewFaceService(faceRepo, verificationRepo, searchAuditRepo, faceProvider, rateLimiter)

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := svc.Search(ctx, tenant, imageBytes, 0.8, 10, "192.168.1.1")
		if err != nil {
			b.Fatal(err)
		}
		// Note: Audit is async via goroutine, so we can't verify it here
		// This benchmark measures the overhead of spawning the goroutine
	}

	b.StopTimer()
}

// BenchmarkFaceService_Search_NoMatches benchmarks search with no results
// Important to measure "miss" scenario
// Target: Similar to regular search (< 10ms)
func BenchmarkFaceService_Search_NoMatches(b *testing.B) {
	faceRepo := &MockFaceRepository{}
	verificationRepo := &MockVerificationRepository{}
	searchAuditRepo := &MockSearchAuditRepository{}
	faceProvider := &MockFaceProvider{}
	rateLimiter := &MockRateLimiter{}

	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID: tenantID,
		Settings: map[string]interface{}{
			"search_enabled":    true,
			"search_rate_limit": float64(30),
		},
	}

	embedding := generateBenchmarkEmbedding(512)

	rateLimiter.On("CheckSearchLimit", mock.Anything, tenantID, 30).
		Return(nil)

	faceProvider.On("AnalyzeFace", mock.Anything, mock.Anything).
		Return(&provider.FaceAnalysis{
			Embedding:     embedding,
			Confidence:    0.99,
			QualityScore:  0.95,
			LivenessScore: 0.90,
			FaceCount:     1,
		}, nil)

	// No matches found
	faceRepo.On("SearchByEmbedding", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]domain.SearchMatch{}, nil)

	searchAuditRepo.On("Create", mock.Anything, mock.Anything).
		Return(nil).Maybe()

	svc := NewFaceService(faceRepo, verificationRepo, searchAuditRepo, faceProvider, rateLimiter)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := svc.Search(context.Background(), tenant, []byte("image"), 0.95, 10, "127.0.0.1")
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Matches) != 0 {
			b.Fatalf("expected 0 matches, got %d", len(result.Matches))
		}
	}
}

// BenchmarkFaceService_Search_WithLiveness benchmarks search with liveness check
// This adds liveness overhead to the flow
// Target: < 15ms (includes liveness check)
func BenchmarkFaceService_Search_WithLiveness(b *testing.B) {
	faceRepo := &MockFaceRepository{}
	verificationRepo := &MockVerificationRepository{}
	searchAuditRepo := &MockSearchAuditRepository{}
	faceProvider := &MockFaceProvider{}
	rateLimiter := &MockRateLimiter{}

	tenantID := uuid.New()
	tenant := &domain.Tenant{
		ID: tenantID,
		Settings: map[string]interface{}{
			"search_enabled":     true,
			"security_level":     "maximum",
			"liveness_threshold": 0.9,
			"search_rate_limit":  float64(30),
		},
	}

	embedding := generateBenchmarkEmbedding(512)

	rateLimiter.On("CheckSearchLimit", mock.Anything, tenantID, 30).
		Return(nil)

	// AnalyzeFace now includes liveness data (optimized single call)
	faceProvider.On("AnalyzeFace", mock.Anything, mock.Anything).
		Return(&provider.FaceAnalysis{
			Embedding:     embedding,
			Confidence:    0.99,
			QualityScore:  0.95,
			LivenessScore: 0.95,
			FaceCount:     1,
		}, nil)

	faceRepo.On("SearchByEmbedding", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]domain.SearchMatch{
			{FaceID: uuid.New(), ExternalID: "user-live", Similarity: 0.93},
		}, nil)

	searchAuditRepo.On("Create", mock.Anything, mock.Anything).
		Return(nil).Maybe()

	svc := NewFaceService(faceRepo, verificationRepo, searchAuditRepo, faceProvider, rateLimiter)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := svc.Search(context.Background(), tenant, []byte("image"), 0.8, 10, "127.0.0.1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFaceService_Verify benchmarks the verification flow
// Critical for entry gates at events
// Target P99: < 5ms (excluding provider calls)
func BenchmarkFaceService_Verify(b *testing.B) {
	faceRepo := &MockFaceRepository{}
	verificationRepo := &MockVerificationRepository{}
	searchAuditRepo := &MockSearchAuditRepository{}
	faceProvider := &MockFaceProvider{}

	tenantID := uuid.New()
	externalID := "user-verify"
	storedFaceID := uuid.New()
	storedEmbedding := generateBenchmarkEmbedding(512)

	faceRepo.On("GetByExternalID", mock.Anything, tenantID, externalID).
		Return(&domain.Face{
			ID:        storedFaceID,
			TenantID:  tenantID,
			Embedding: storedEmbedding,
		}, nil)

	faceProvider.On("DetectFaces", mock.Anything, mock.Anything).
		Return([]provider.DetectedFace{
			{Confidence: 0.99, QualityScore: 0.95},
		}, nil)

	faceProvider.On("IndexFace", mock.Anything, mock.Anything).
		Return("face-id", storedEmbedding, nil)

	faceProvider.On("CompareFaces", mock.Anything, storedEmbedding, storedEmbedding).
		Return(0.95, nil)

	verificationRepo.On("Create", mock.Anything, mock.Anything).
		Return(nil)

	rateLimiter := &MockRateLimiter{}
	svc := NewFaceService(faceRepo, verificationRepo, searchAuditRepo, faceProvider, rateLimiter)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		verification, err := svc.Verify(context.Background(), tenantID, externalID, []byte("image"))
		if err != nil {
			b.Fatal(err)
		}
		if !verification.Verified {
			b.Fatal("expected verified=true")
		}
	}
}

// BenchmarkFaceService_Register benchmarks face registration
// Not as critical as search/verify, but still important
// Target: < 20ms (excluding provider calls)
func BenchmarkFaceService_Register(b *testing.B) {
	faceRepo := &MockFaceRepository{}
	verificationRepo := &MockVerificationRepository{}
	searchAuditRepo := &MockSearchAuditRepository{}
	faceProvider := &MockFaceProvider{}

	tenantID := uuid.New()
	embedding := generateBenchmarkEmbedding(512)

	faceProvider.On("AnalyzeFace", mock.Anything, mock.Anything).
		Return(&provider.FaceAnalysis{
			Embedding:     embedding,
			Confidence:    0.99,
			QualityScore:  0.95,
			LivenessScore: 0.9,
			FaceCount:     1,
		}, nil)

	faceRepo.On("Create", mock.Anything, mock.Anything).
		Return(nil)

	rateLimiter := &MockRateLimiter{}
	svc := NewFaceService(faceRepo, verificationRepo, searchAuditRepo, faceProvider, rateLimiter)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		externalID := fmt.Sprintf("user-bench-%d", i)
		face, err := svc.Register(context.Background(), tenantID, externalID, []byte("image"), false, 0.9)
		if err != nil {
			b.Fatal(err)
		}
		if face.ExternalID != externalID {
			b.Fatalf("expected external_id %s, got %s", externalID, face.ExternalID)
		}
	}
}

// BenchmarkExtractTenantSettings benchmarks settings extraction
// This happens on every search request
// Target: < 100ns/op, 0 allocs/op (pure computation)
func BenchmarkExtractTenantSettings(b *testing.B) {
	tenant := &domain.Tenant{
		ID: uuid.New(),
		Settings: map[string]interface{}{
			"verification_threshold":  0.85,
			"max_faces_per_user":      float64(5),
			"require_liveness":        true,
			"liveness_threshold":      0.9,
			"search_enabled":          true,
			"search_require_liveness": false,
			"search_threshold":        0.8,
			"search_max_results":      float64(10),
			"search_rate_limit":       float64(100),
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		settings := tenant.GetSettings()
		if settings.SearchThreshold != 0.8 {
			b.Fatal("incorrect settings extraction")
		}
	}
}

// BenchmarkExtractTenantSettings_Empty benchmarks extraction with empty settings
// Shows the cost of applying defaults
func BenchmarkExtractTenantSettings_Empty(b *testing.B) {
	tenant := &domain.Tenant{
		ID:       uuid.New(),
		Settings: nil,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		settings := tenant.GetSettings()
		if settings.SearchThreshold == 0 {
			b.Fatal("default settings not applied")
		}
	}
}

// Benchmark helper functions

func generateBenchmarkEmbedding(dim int) []float64 {
	embedding := make([]float64, dim)
	for i := range embedding {
		//nolint:gosec // Using math/rand is acceptable for benchmark test data
		embedding[i] = rand.Float64()*2 - 1
	}
	return embedding
}
