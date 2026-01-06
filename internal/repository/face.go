package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// embeddingSize is the standard face recognition embedding dimension
const embeddingSize = 512

// float32Pool reuses float32 slices to reduce allocations in hot paths
// Each slice is pre-allocated to embeddingSize (512 float32)
var float32Pool = sync.Pool{
	New: func() interface{} {
		slice := make([]float32, embeddingSize)
		return &slice
	},
}

type FaceRepository struct {
	pool PgxPool
}

func NewFaceRepository(pool PgxPool) *FaceRepository {
	return &FaceRepository{pool: pool}
}

func (r *FaceRepository) Create(ctx context.Context, face *domain.Face) error {
	query := `
		INSERT INTO faces (id, tenant_id, external_id, embedding, metadata, quality_score, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	if face.ID == uuid.Nil {
		face.ID = uuid.New()
	}

	var embedding *pgvector.Vector
	if len(face.Embedding) > 0 {
		var floats []float32

		// Use pool for standard embedding size (zero-allocation path)
		// For non-standard sizes (tests), allocate normally
		if len(face.Embedding) == embeddingSize {
			floatsPtr := float32Pool.Get().(*[]float32)
			defer float32Pool.Put(floatsPtr)
			floats = (*floatsPtr)[:len(face.Embedding)]
		} else {
			floats = make([]float32, len(face.Embedding))
		}

		// Convert float64 to float32
		for i, v := range face.Embedding {
			floats[i] = float32(v)
		}

		vec := pgvector.NewVector(floats)
		embedding = &vec
	}

	err := r.pool.QueryRow(ctx, query,
		face.ID,
		face.TenantID,
		face.ExternalID,
		embedding,
		face.Metadata,
		face.QualityScore,
	).Scan(&face.CreatedAt, &face.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrFaceExists
		}
		return fmt.Errorf("create face: %w", err)
	}

	return nil
}

// Update updates an existing face's embedding and quality score
func (r *FaceRepository) Update(ctx context.Context, face *domain.Face) error {
	query := `
		UPDATE faces
		SET embedding = $1, quality_score = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
		RETURNING updated_at
	`

	var embedding *pgvector.Vector
	if len(face.Embedding) > 0 {
		floats := make([]float32, len(face.Embedding))
		for i, v := range face.Embedding {
			floats[i] = float32(v)
		}
		vec := pgvector.NewVector(floats)
		embedding = &vec
	}

	err := r.pool.QueryRow(ctx, query,
		embedding,
		face.QualityScore,
		face.ID,
		face.TenantID,
	).Scan(&face.UpdatedAt)

	if err != nil {
		return fmt.Errorf("update face: %w", err)
	}

	return nil
}

func (r *FaceRepository) GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Face, error) {
	query := `
		SELECT id, tenant_id, external_id, embedding, metadata, quality_score, created_at, updated_at
		FROM faces
		WHERE tenant_id = $1 AND external_id = $2
	`

	var face domain.Face
	var embedding *pgvector.Vector

	err := r.pool.QueryRow(ctx, query, tenantID, externalID).Scan(
		&face.ID,
		&face.TenantID,
		&face.ExternalID,
		&embedding,
		&face.Metadata,
		&face.QualityScore,
		&face.CreatedAt,
		&face.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrFaceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get face by external_id: %w", err)
	}

	if embedding != nil && embedding.Slice() != nil {
		face.Embedding = make([]float64, len(embedding.Slice()))
		for i, v := range embedding.Slice() {
			face.Embedding[i] = float64(v)
		}
	}

	return &face, nil
}

func (r *FaceRepository) Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error {
	query := `
		DELETE FROM faces
		WHERE tenant_id = $1 AND external_id = $2
	`

	result, err := r.pool.Exec(ctx, query, tenantID, externalID)
	if err != nil {
		return fmt.Errorf("delete face: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrFaceNotFound
	}

	return nil
}

// SearchByEmbedding searches for similar faces using cosine distance
// Returns matches above threshold, ordered by similarity (highest first)
func (r *FaceRepository) SearchByEmbedding(ctx context.Context, tenantID uuid.UUID, embedding []float64, threshold float64, limit int) ([]domain.SearchMatch, error) {
	var floats []float32

	// Use pool for standard embedding size (zero-allocation hot path)
	// For non-standard sizes (tests), allocate normally
	if len(embedding) == embeddingSize {
		floatsPtr := float32Pool.Get().(*[]float32)
		defer float32Pool.Put(floatsPtr)
		floats = (*floatsPtr)[:len(embedding)]
	} else {
		floats = make([]float32, len(embedding))
	}

	// Convert float64 to float32
	for i, v := range embedding {
		floats[i] = float32(v)
	}

	// Create vector
	vec := pgvector.NewVector(floats)

	// Query usando cosine distance (<=>)
	// pgvector retorna distância (0 = idêntico, 2 = oposto)
	// Convertemos para similarity: 1 - (distance / 2)
	query := `
		SELECT id, external_id, metadata,
		       1 - (embedding <=> $1) / 2 as similarity
		FROM faces
		WHERE tenant_id = $2
		  AND embedding IS NOT NULL
		  AND 1 - (embedding <=> $1) / 2 >= $3
		ORDER BY embedding <=> $1
		LIMIT $4
	`

	rows, err := r.pool.Query(ctx, query, vec, tenantID, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("search faces by embedding: %w", err)
	}
	defer rows.Close()

	var matches []domain.SearchMatch
	for rows.Next() {
		var match domain.SearchMatch
		var metadata map[string]interface{}

		err := rows.Scan(
			&match.FaceID,
			&match.ExternalID,
			&metadata,
			&match.Similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("scan search match: %w", err)
		}

		match.Metadata = metadata
		matches = append(matches, match)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}

	// Retornar slice vazio ao invés de nil se não houver resultados
	if matches == nil {
		matches = []domain.SearchMatch{}
	}

	return matches, nil
}

// CountByTenant returns the total number of faces for a tenant
func (r *FaceRepository) CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM faces WHERE tenant_id = $1`

	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count faces by tenant: %w", err)
	}

	return count, nil
}

// List returns all faces for a tenant with pagination
func (r *FaceRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Face, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT id, tenant_id, external_id, embedding, metadata, quality_score, created_at, updated_at
		FROM faces
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list faces: %w", err)
	}
	defer rows.Close()

	var faces []*domain.Face
	for rows.Next() {
		face := &domain.Face{}
		var embedding *pgvector.Vector

		if err := rows.Scan(
			&face.ID,
			&face.TenantID,
			&face.ExternalID,
			&embedding,
			&face.Metadata,
			&face.QualityScore,
			&face.CreatedAt,
			&face.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan face row: %w", err)
		}

		if embedding != nil && embedding.Slice() != nil {
			face.Embedding = make([]float64, len(embedding.Slice()))
			for i, v := range embedding.Slice() {
				face.Embedding[i] = float64(v)
			}
		}

		faces = append(faces, face)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate face rows: %w", err)
	}

	return faces, nil
}
