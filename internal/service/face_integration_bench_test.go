//go:build integration

package service

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
	"github.com/saturnino-fabrica-de-software/rekko/internal/ratelimit"
	"github.com/saturnino-fabrica-de-software/rekko/internal/repository"
)

// benchmarkProvider is a minimal mock provider for benchmarks
type benchmarkProvider struct {
	embedding []float64
}

func newBenchmarkProvider() *benchmarkProvider {
	return &benchmarkProvider{
		embedding: generateRandomEmbedding(512),
	}
}

func (p *benchmarkProvider) DetectFaces(_ context.Context, _ []byte) ([]provider.DetectedFace, error) {
	return []provider.DetectedFace{{Confidence: 0.99, QualityScore: 0.95}}, nil
}

func (p *benchmarkProvider) IndexFace(_ context.Context, _ []byte) (string, []float64, error) {
	return uuid.New().String(), p.embedding, nil
}

func (p *benchmarkProvider) CompareFaces(_ context.Context, _, _ []float64) (float64, error) {
	return 0.95, nil
}

func (p *benchmarkProvider) CheckLiveness(_ context.Context, _ []byte, _ float64) (*provider.LivenessResult, error) {
	return &provider.LivenessResult{IsLive: true, Confidence: 0.99}, nil
}

func (p *benchmarkProvider) DeleteFace(_ context.Context, _ string) error {
	return nil
}

// setupBenchmarkDB creates a PostgreSQL container with pgvector for benchmarks
func setupBenchmarkDB(b *testing.B) (*pgxpool.Pool, func()) {
	b.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "pgvector/pgvector:pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "bench",
			"POSTGRES_PASSWORD": "bench",
			"POSTGRES_DB":       "rekko_bench",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		b.Fatalf("failed to start container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		b.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		b.Fatalf("failed to get container port: %v", err)
	}

	connStr := fmt.Sprintf("postgres://bench:bench@%s:%s/rekko_bench?sslmode=disable", host, port.Port())

	db, err := pgxpool.New(ctx, connStr)
	if err != nil {
		b.Fatalf("failed to connect to database: %v", err)
	}

	// Create schema
	if err := createBenchmarkSchema(ctx, db); err != nil {
		b.Fatalf("failed to create schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		if err := container.Terminate(ctx); err != nil {
			b.Logf("Failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

func createBenchmarkSchema(ctx context.Context, db *pgxpool.Pool) error {
	schema := `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE EXTENSION IF NOT EXISTS "vector";

		-- Tenants table
		CREATE TABLE IF NOT EXISTS tenants (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			name VARCHAR(255) NOT NULL,
			slug VARCHAR(255) UNIQUE NOT NULL,
			settings JSONB DEFAULT '{}',
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		-- Faces table
		CREATE TABLE IF NOT EXISTS faces (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			external_id VARCHAR(255) NOT NULL,
			embedding vector(512),
			metadata JSONB,
			quality_score FLOAT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE(tenant_id, external_id)
		);

		-- Verifications table
		CREATE TABLE IF NOT EXISTS verifications (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			face_id UUID REFERENCES faces(id) ON DELETE SET NULL,
			external_id VARCHAR(255) NOT NULL,
			verified BOOLEAN NOT NULL,
			confidence FLOAT NOT NULL,
			latency_ms BIGINT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		-- Search audit table
		CREATE TABLE IF NOT EXISTS search_audits (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			results_count INTEGER NOT NULL DEFAULT 0,
			threshold FLOAT NOT NULL,
			max_results INTEGER NOT NULL,
			latency_ms BIGINT NOT NULL,
			client_ip VARCHAR(45),
			top_match_external_id VARCHAR(255),
			top_match_similarity FLOAT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);

		-- Rate limit counters table
		CREATE TABLE IF NOT EXISTS rate_limit_counters (
			key VARCHAR(255) PRIMARY KEY,
			count INTEGER NOT NULL DEFAULT 0,
			window_start TIMESTAMP WITH TIME ZONE NOT NULL,
			window_end TIMESTAMP WITH TIME ZONE NOT NULL,
			tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		-- Indexes
		CREATE INDEX IF NOT EXISTS idx_faces_tenant_id ON faces(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_faces_embedding ON faces USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
		CREATE INDEX IF NOT EXISTS idx_rate_limit_tenant ON rate_limit_counters(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_rate_limit_window_end ON rate_limit_counters(window_end);
	`

	_, err := db.Exec(ctx, schema)
	return err
}

// seedFaces inserts n random faces for the given tenant
func seedFaces(ctx context.Context, db *pgxpool.Pool, tenantID uuid.UUID, count int) error {
	for i := 0; i < count; i++ {
		embedding := generateRandomEmbedding(512)
		embeddingStr := embeddingToVector(embedding)
		_, err := db.Exec(ctx, `
			INSERT INTO faces (tenant_id, external_id, embedding, quality_score)
			VALUES ($1, $2, $3::vector, $4)
		`, tenantID, fmt.Sprintf("user-%d", i), embeddingStr, 0.9)
		if err != nil {
			return fmt.Errorf("insert face %d: %w", i, err)
		}
	}
	return nil
}

// embeddingToVector converts a float64 slice to pgvector format string
func embeddingToVector(embedding []float64) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, v := range embedding {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("%f", v))
	}
	sb.WriteByte(']')
	return sb.String()
}

// generateRandomEmbedding creates a normalized random embedding
func generateRandomEmbedding(dim int) []float64 {
	embedding := make([]float64, dim)
	var magnitude float64
	for i := range embedding {
		embedding[i] = rand.Float64()*2 - 1
		magnitude += embedding[i] * embedding[i]
	}
	// Normalize
	magnitude = 1.0 / (magnitude + 1e-10)
	for i := range embedding {
		embedding[i] *= magnitude
	}
	return embedding
}

// BenchmarkFaceService_Search_Integration benchmarks Search with real PostgreSQL
// Tests various database sizes to measure pgvector performance
func BenchmarkFaceService_Search_Integration(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	ctx := context.Background()

	// Create tenant
	tenantID := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, settings, is_active)
		VALUES ($1, $2, $3, $4, true)
	`, tenantID, "Benchmark Tenant", "bench-tenant", map[string]interface{}{
		"search_enabled":    true,
		"search_rate_limit": 10000, // High limit for benchmark
	})
	if err != nil {
		b.Fatalf("failed to create tenant: %v", err)
	}

	tenant := &domain.Tenant{
		ID:       tenantID,
		Name:     "Benchmark Tenant",
		Slug:     "bench-tenant",
		IsActive: true,
		Settings: map[string]interface{}{
			"search_enabled":    true,
			"search_rate_limit": float64(10000),
		},
	}

	// Create repositories
	faceRepo := repository.NewFaceRepository(db)
	verificationRepo := repository.NewVerificationRepository(db)
	searchAuditRepo := repository.NewSearchAuditRepository(db)
	rateLimiter := ratelimit.NewRateLimiter(db, time.Minute)

	// Create benchmark provider (we're testing service + DB, not provider)
	faceProvider := newBenchmarkProvider()

	// Create service
	svc := NewFaceService(faceRepo, verificationRepo, searchAuditRepo, faceProvider, rateLimiter)

	// Test with different database sizes
	sizes := []int{100, 1000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("faces_%d", size), func(b *testing.B) {
			// Clear and seed faces
			_, _ = db.Exec(ctx, "DELETE FROM faces WHERE tenant_id = $1", tenantID)
			_, _ = db.Exec(ctx, "DELETE FROM rate_limit_counters WHERE tenant_id = $1", tenantID)

			if err := seedFaces(ctx, db, tenantID, size); err != nil {
				b.Fatalf("failed to seed faces: %v", err)
			}

			// Generate query embedding
			queryEmbedding := generateRandomEmbedding(512)
			imageBytes := []byte("fake-image-data")

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				// Mock provider returns our query embedding
				result, err := svc.Search(ctx, tenant, imageBytes, 0.7, 10, "127.0.0.1")
				if err != nil {
					b.Fatalf("search failed: %v", err)
				}
				_ = result
				_ = queryEmbedding
			}
		})
	}
}

// BenchmarkSearchByEmbedding_Integration benchmarks just the repository layer
func BenchmarkSearchByEmbedding_Integration(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	ctx := context.Background()

	// Create tenant
	tenantID := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, is_active)
		VALUES ($1, $2, $3, true)
	`, tenantID, "Benchmark Tenant", "bench-tenant")
	if err != nil {
		b.Fatalf("failed to create tenant: %v", err)
	}

	repo := repository.NewFaceRepository(db)

	sizes := []int{100, 1000, 5000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("faces_%d", size), func(b *testing.B) {
			// Clear and seed
			_, _ = db.Exec(ctx, "DELETE FROM faces WHERE tenant_id = $1", tenantID)

			if err := seedFaces(ctx, db, tenantID, size); err != nil {
				b.Fatalf("failed to seed faces: %v", err)
			}

			queryEmbedding := generateRandomEmbedding(512)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				matches, err := repo.SearchByEmbedding(ctx, tenantID, queryEmbedding, 0.7, 10)
				if err != nil {
					b.Fatalf("search failed: %v", err)
				}
				_ = matches
			}
		})
	}
}

// BenchmarkRateLimiter_Integration benchmarks PostgreSQL rate limiter
func BenchmarkRateLimiter_Integration(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	ctx := context.Background()

	// Create tenant
	tenantID := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, is_active)
		VALUES ($1, $2, $3, true)
	`, tenantID, "Benchmark Tenant", "bench-tenant")
	if err != nil {
		b.Fatalf("failed to create tenant: %v", err)
	}

	rateLimiter := ratelimit.NewRateLimiter(db, time.Minute)

	b.Run("check_and_increment", func(b *testing.B) {
		// Clear rate limit counters
		_, _ = db.Exec(ctx, "DELETE FROM rate_limit_counters WHERE tenant_id = $1", tenantID)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			err := rateLimiter.CheckSearchLimit(ctx, tenantID, 1000000) // Very high limit
			if err != nil {
				b.Fatalf("rate limit check failed: %v", err)
			}
		}
	})
}

// BenchmarkCountByTenant_Integration benchmarks face counting
func BenchmarkCountByTenant_Integration(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	ctx := context.Background()

	// Create tenant
	tenantID := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, is_active)
		VALUES ($1, $2, $3, true)
	`, tenantID, "Benchmark Tenant", "bench-tenant")
	if err != nil {
		b.Fatalf("failed to create tenant: %v", err)
	}

	repo := repository.NewFaceRepository(db)

	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("faces_%d", size), func(b *testing.B) {
			_, _ = db.Exec(ctx, "DELETE FROM faces WHERE tenant_id = $1", tenantID)

			if err := seedFaces(ctx, db, tenantID, size); err != nil {
				b.Fatalf("failed to seed faces: %v", err)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				count, err := repo.CountByTenant(ctx, tenantID)
				if err != nil {
					b.Fatalf("count failed: %v", err)
				}
				if count != size {
					b.Fatalf("expected %d, got %d", size, count)
				}
			}
		})
	}
}

// BenchmarkFullSearchFlow_Integration benchmarks complete search including audit
func BenchmarkFullSearchFlow_Integration(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b)
	defer cleanup()

	ctx := context.Background()

	// Create tenant
	tenantID := uuid.New()
	_, err := db.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, settings, is_active)
		VALUES ($1, $2, $3, $4, true)
	`, tenantID, "Benchmark Tenant", "bench-tenant", map[string]interface{}{
		"search_enabled":    true,
		"search_rate_limit": 100000,
	})
	if err != nil {
		b.Fatalf("failed to create tenant: %v", err)
	}

	tenant := &domain.Tenant{
		ID:       tenantID,
		Name:     "Benchmark Tenant",
		Slug:     "bench-tenant",
		IsActive: true,
		Settings: map[string]interface{}{
			"search_enabled":    true,
			"search_rate_limit": float64(100000),
		},
	}

	// Seed 1000 faces
	if err := seedFaces(ctx, db, tenantID, 1000); err != nil {
		b.Fatalf("failed to seed faces: %v", err)
	}

	faceRepo := repository.NewFaceRepository(db)
	verificationRepo := repository.NewVerificationRepository(db)
	searchAuditRepo := repository.NewSearchAuditRepository(db)
	rateLimiter := ratelimit.NewRateLimiter(db, time.Minute)
	faceProvider := newBenchmarkProvider()

	svc := NewFaceService(faceRepo, verificationRepo, searchAuditRepo, faceProvider, rateLimiter)

	imageBytes := []byte("benchmark-image")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := svc.Search(ctx, tenant, imageBytes, 0.7, 10, "127.0.0.1")
		if err != nil {
			b.Fatalf("search failed: %v", err)
		}
		_ = result
	}

	b.StopTimer()

	// Report latency stats
	var totalLatency int64
	rows, _ := db.Query(ctx, "SELECT latency_ms FROM search_audits WHERE tenant_id = $1 LIMIT 100", tenantID)
	count := 0
	for rows.Next() {
		var latency int64
		_ = rows.Scan(&latency)
		totalLatency += latency
		count++
	}
	rows.Close()

	if count > 0 {
		b.ReportMetric(float64(totalLatency)/float64(count), "avg_latency_ms")
	}
}
