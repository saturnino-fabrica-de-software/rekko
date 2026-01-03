package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

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
		floats := make([]float32, len(face.Embedding))
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
