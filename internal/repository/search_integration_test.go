//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

var integrationTestDB *pgxpool.Pool

func setupIntegrationTest(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "pgvector/pgvector:pg16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "rekko_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	connStr := fmt.Sprintf("postgres://test:test@%s:%s/rekko_test?sslmode=disable", host, port.Port())

	db, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE EXTENSION IF NOT EXISTS "vector";

		CREATE TABLE IF NOT EXISTS faces (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			tenant_id UUID NOT NULL,
			external_id VARCHAR(255) NOT NULL,
			embedding vector(512),
			metadata JSONB,
			quality_score FLOAT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE(tenant_id, external_id)
		);

		CREATE INDEX IF NOT EXISTS idx_faces_tenant_id ON faces(tenant_id);
		CREATE INDEX IF NOT EXISTS idx_faces_embedding ON faces USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
	`)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}

	return db, cleanup
}

func TestSearchByEmbedding_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewFaceRepository(db)
	tenantID := uuid.New()

	// Insert test faces with known embeddings
	testFaces := []struct {
		externalID string
		embedding  []float64
		metadata   map[string]interface{}
	}{
		{
			externalID: "user-identical",
			embedding:  createNormalizedEmbedding([]float64{1.0, 0.0, 0.0}),
			metadata:   map[string]interface{}{"name": "Identical Match"},
		},
		{
			externalID: "user-very-similar",
			embedding:  createNormalizedEmbedding([]float64{0.95, 0.05, 0.0}),
			metadata:   map[string]interface{}{"name": "Very Similar"},
		},
		{
			externalID: "user-similar",
			embedding:  createNormalizedEmbedding([]float64{0.8, 0.2, 0.0}),
			metadata:   map[string]interface{}{"name": "Similar"},
		},
		{
			externalID: "user-different",
			embedding:  createNormalizedEmbedding([]float64{0.0, 1.0, 0.0}),
			metadata:   map[string]interface{}{"name": "Different"},
		},
		{
			externalID: "user-opposite",
			embedding:  createNormalizedEmbedding([]float64{-1.0, 0.0, 0.0}),
			metadata:   map[string]interface{}{"name": "Opposite"},
		},
	}

	for _, tf := range testFaces {
		face := &domain.Face{
			TenantID:     tenantID,
			ExternalID:   tf.externalID,
			Embedding:    tf.embedding,
			Metadata:     tf.metadata,
			QualityScore: 0.9,
		}
		err := repo.Create(ctx, face)
		require.NoError(t, err, "failed to insert test face: %s", tf.externalID)
	}

	t.Run("search with high similarity threshold", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding([]float64{1.0, 0.0, 0.0})
		threshold := 0.95
		limit := 10

		matches, err := repo.SearchByEmbedding(ctx, tenantID, queryEmbedding, threshold, limit)
		require.NoError(t, err)

		// Should find identical and very similar
		assert.GreaterOrEqual(t, len(matches), 1, "should find at least the identical match")
		assert.LessOrEqual(t, len(matches), 2, "should not find too many matches")

		// First match should be identical with ~1.0 similarity
		if len(matches) > 0 {
			assert.Equal(t, "user-identical", matches[0].ExternalID)
			assert.InDelta(t, 1.0, matches[0].Similarity, 0.01, "identical face should have ~1.0 similarity")
		}

		// Results should be ordered by similarity (highest first)
		for i := 1; i < len(matches); i++ {
			assert.GreaterOrEqual(t, matches[i-1].Similarity, matches[i].Similarity, "results should be ordered by similarity")
		}

		// All matches should be above threshold
		for _, match := range matches {
			assert.GreaterOrEqual(t, match.Similarity, threshold, "all matches should be above threshold")
		}
	})

	t.Run("search with medium similarity threshold", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding([]float64{1.0, 0.0, 0.0})
		threshold := 0.7
		limit := 10

		matches, err := repo.SearchByEmbedding(ctx, tenantID, queryEmbedding, threshold, limit)
		require.NoError(t, err)

		// Should find identical, very similar, and similar
		assert.GreaterOrEqual(t, len(matches), 2, "should find multiple matches")
		assert.LessOrEqual(t, len(matches), 3, "should not find opposite vectors")

		// All matches should be above threshold
		for _, match := range matches {
			assert.GreaterOrEqual(t, match.Similarity, threshold)
			assert.NotEmpty(t, match.ExternalID)
			assert.NotEqual(t, uuid.Nil, match.FaceID)
		}
	})

	t.Run("search with low similarity threshold", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding([]float64{1.0, 0.0, 0.0})
		threshold := 0.0
		limit := 10

		matches, err := repo.SearchByEmbedding(ctx, tenantID, queryEmbedding, threshold, limit)
		require.NoError(t, err)

		// Should find all faces except opposite (which has negative similarity)
		assert.GreaterOrEqual(t, len(matches), 3, "should find most faces")
	})

	t.Run("search with limit", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding([]float64{1.0, 0.0, 0.0})
		threshold := 0.5
		limit := 2

		matches, err := repo.SearchByEmbedding(ctx, tenantID, queryEmbedding, threshold, limit)
		require.NoError(t, err)

		// Should respect limit
		assert.LessOrEqual(t, len(matches), limit, "should respect limit parameter")
	})

	t.Run("search with no matches", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding([]float64{0.0, 0.0, 1.0})
		threshold := 0.95
		limit := 10

		matches, err := repo.SearchByEmbedding(ctx, tenantID, queryEmbedding, threshold, limit)
		require.NoError(t, err)

		// Should return empty slice, not nil
		assert.NotNil(t, matches, "should return empty slice, not nil")
		assert.Equal(t, 0, len(matches), "should find no matches")
	})

	t.Run("search respects tenant isolation", func(t *testing.T) {
		otherTenantID := uuid.New()

		// Insert face for different tenant
		otherFace := &domain.Face{
			TenantID:     otherTenantID,
			ExternalID:   "other-tenant-user",
			Embedding:    createNormalizedEmbedding([]float64{1.0, 0.0, 0.0}),
			QualityScore: 0.9,
		}
		err := repo.Create(ctx, otherFace)
		require.NoError(t, err)

		// Search should only find faces from the queried tenant
		queryEmbedding := createNormalizedEmbedding([]float64{1.0, 0.0, 0.0})
		matches, err := repo.SearchByEmbedding(ctx, tenantID, queryEmbedding, 0.5, 10)
		require.NoError(t, err)

		for _, match := range matches {
			assert.NotEqual(t, "other-tenant-user", match.ExternalID, "should not find faces from other tenants")
		}
	})

	t.Run("search returns metadata", func(t *testing.T) {
		queryEmbedding := createNormalizedEmbedding([]float64{1.0, 0.0, 0.0})
		threshold := 0.95
		limit := 10

		matches, err := repo.SearchByEmbedding(ctx, tenantID, queryEmbedding, threshold, limit)
		require.NoError(t, err)
		require.Greater(t, len(matches), 0, "need at least one match to validate metadata")

		// Check that metadata is returned
		firstMatch := matches[0]
		assert.NotNil(t, firstMatch.Metadata, "metadata should not be nil")
		if name, ok := firstMatch.Metadata["name"].(string); ok {
			assert.NotEmpty(t, name, "metadata should contain name field")
		}
	})
}

func TestCountByTenant_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupIntegrationTest(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewFaceRepository(db)
	tenantID := uuid.New()

	t.Run("count with no faces", func(t *testing.T) {
		count, err := repo.CountByTenant(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("count with multiple faces", func(t *testing.T) {
		// Insert 5 test faces
		for i := 0; i < 5; i++ {
			face := &domain.Face{
				TenantID:     tenantID,
				ExternalID:   fmt.Sprintf("user-%d", i),
				Embedding:    createNormalizedEmbedding([]float64{float64(i) / 10.0, 0.0, 0.0}),
				QualityScore: 0.9,
			}
			err := repo.Create(ctx, face)
			require.NoError(t, err)
		}

		count, err := repo.CountByTenant(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("count respects tenant isolation", func(t *testing.T) {
		otherTenantID := uuid.New()

		// Insert faces for different tenant
		for i := 0; i < 3; i++ {
			face := &domain.Face{
				TenantID:     otherTenantID,
				ExternalID:   fmt.Sprintf("other-user-%d", i),
				Embedding:    createNormalizedEmbedding([]float64{0.0, float64(i) / 10.0, 0.0}),
				QualityScore: 0.9,
			}
			err := repo.Create(ctx, face)
			require.NoError(t, err)
		}

		// Original tenant should still have 5 faces
		count, err := repo.CountByTenant(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 5, count)

		// Other tenant should have 3 faces
		otherCount, err := repo.CountByTenant(ctx, otherTenantID)
		require.NoError(t, err)
		assert.Equal(t, 3, otherCount)
	})
}

// createNormalizedEmbedding creates a 512-dimensional normalized embedding
// from a smaller input vector by padding with zeros
func createNormalizedEmbedding(values []float64) []float64 {
	embedding := make([]float64, 512)

	// Copy input values
	for i, v := range values {
		if i < 512 {
			embedding[i] = v
		}
	}

	// Normalize the vector to unit length
	var magnitude float64
	for _, v := range embedding {
		magnitude += v * v
	}
	magnitude = 1.0 / (magnitude + 1e-10) // Add small epsilon to prevent division by zero

	for i := range embedding {
		embedding[i] *= magnitude
	}

	return embedding
}

func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
}
