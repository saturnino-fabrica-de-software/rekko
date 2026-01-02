package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

type VerificationRepository struct {
	pool PgxPool
}

func NewVerificationRepository(pool PgxPool) *VerificationRepository {
	return &VerificationRepository{pool: pool}
}

func (r *VerificationRepository) Create(ctx context.Context, v *domain.Verification) error {
	query := `
		INSERT INTO verifications (id, tenant_id, face_id, external_id, verified, confidence, liveness_passed, latency_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		RETURNING created_at
	`

	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		v.ID,
		v.TenantID,
		v.FaceID,
		v.ExternalID,
		v.Verified,
		v.Confidence,
		v.LivenessPassed,
		v.LatencyMs,
	).Scan(&v.CreatedAt)

	if err != nil {
		return fmt.Errorf("create verification: %w", err)
	}

	return nil
}
