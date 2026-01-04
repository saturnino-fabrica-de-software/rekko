package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

type SearchAuditRepository struct {
	pool PgxPool
}

func NewSearchAuditRepository(pool PgxPool) *SearchAuditRepository {
	return &SearchAuditRepository{pool: pool}
}

// Create inserts a new search audit record
func (r *SearchAuditRepository) Create(ctx context.Context, audit *domain.SearchAudit) error {
	query := `
		INSERT INTO search_audits (
			id, tenant_id, results_count, top_match_external_id,
			top_match_similarity, threshold, max_results, latency_ms, client_ip, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING created_at
	`

	if audit.ID == uuid.Nil {
		audit.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		audit.ID,
		audit.TenantID,
		audit.ResultsCount,
		audit.TopMatchExternalID,
		audit.TopMatchSimilarity,
		audit.Threshold,
		audit.MaxResults,
		audit.LatencyMs,
		audit.ClientIP,
	).Scan(&audit.CreatedAt)

	if err != nil {
		return fmt.Errorf("create search audit: %w", err)
	}

	return nil
}
