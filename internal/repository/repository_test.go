package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// TenantRepository Tests

func TestTenantRepository_GetByAPIKeyHash(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		apiKeyHash string
		mockSetup  func(mock pgxmock.PgxPoolIface)
		want       *domain.Tenant
		wantErr    error
	}{
		{
			name:       "successful retrieval",
			apiKeyHash: "hash_valid_key",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "slug", "is_active", "plan", "settings", "created_at", "updated_at",
				}).AddRow(
					tenantID,
					"Test Tenant",
					"test-tenant",
					true,
					"starter",
					map[string]interface{}{"key": "value"},
					now,
					now,
				)

				mock.ExpectQuery(`SELECT t.id, t.name, t.slug, t.is_active, t.plan, t.settings, t.created_at, t.updated_at FROM tenants t INNER JOIN api_keys ak ON ak.tenant_id = t.id WHERE ak.key_hash = \$1 AND ak.is_active = true AND t.is_active = true`).
					WithArgs("hash_valid_key").
					WillReturnRows(rows)
			},
			want: &domain.Tenant{
				ID:        tenantID,
				Name:      "Test Tenant",
				Slug:      "test-tenant",
				IsActive:  true,
				Plan:      "starter",
				Settings:  map[string]interface{}{"key": "value"},
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: nil,
		},
		{
			name:       "tenant not found",
			apiKeyHash: "hash_nonexistent",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT t.id, t.name, t.slug, t.is_active, t.plan, t.settings, t.created_at, t.updated_at FROM tenants t INNER JOIN api_keys ak ON ak.tenant_id = t.id WHERE ak.key_hash = \$1 AND ak.is_active = true AND t.is_active = true`).
					WithArgs("hash_nonexistent").
					WillReturnError(pgx.ErrNoRows)
			},
			want:    nil,
			wantErr: domain.ErrTenantNotFound,
		},
		{
			name:       "database error",
			apiKeyHash: "hash_error",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT t.id, t.name, t.slug, t.is_active, t.plan, t.settings, t.created_at, t.updated_at FROM tenants t INNER JOIN api_keys ak ON ak.tenant_id = t.id WHERE ak.key_hash = \$1 AND ak.is_active = true AND t.is_active = true`).
					WithArgs("hash_error").
					WillReturnError(errors.New("database connection error"))
			},
			want:    nil,
			wantErr: errors.New("get tenant by api key: database connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewTenantRepository(mock)
			got, err := repo.GetByAPIKeyHash(context.Background(), tt.apiKeyHash)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrTenantNotFound) {
					assert.ErrorIs(t, err, domain.ErrTenantNotFound)
				} else {
					assert.Contains(t, err.Error(), "get tenant by api key")
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.Name, got.Name)
				assert.Equal(t, tt.want.Slug, got.Slug)
				assert.Equal(t, tt.want.Plan, got.Plan)
				assert.Equal(t, tt.want.IsActive, got.IsActive)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTenantRepository_GetByID(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		id        uuid.UUID
		mockSetup func(mock pgxmock.PgxPoolIface)
		want      *domain.Tenant
		wantErr   error
	}{
		{
			name: "successful retrieval",
			id:   tenantID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "name", "slug", "is_active", "plan", "settings", "created_at", "updated_at",
				}).AddRow(
					tenantID,
					"Test Tenant",
					"test-tenant",
					true,
					"pro",
					map[string]interface{}{"key": "value"},
					now,
					now,
				)

				mock.ExpectQuery(`SELECT id, name, slug, is_active, plan, settings, created_at, updated_at FROM tenants WHERE id = \$1`).
					WithArgs(tenantID).
					WillReturnRows(rows)
			},
			want: &domain.Tenant{
				ID:        tenantID,
				Name:      "Test Tenant",
				Slug:      "test-tenant",
				IsActive:  true,
				Plan:      "pro",
				Settings:  map[string]interface{}{"key": "value"},
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: nil,
		},
		{
			name: "tenant not found by id",
			id:   uuid.New(),
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, name, slug, is_active, plan, settings, created_at, updated_at FROM tenants WHERE id = \$1`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
			want:    nil,
			wantErr: domain.ErrTenantNotFound,
		},
		{
			name: "database error on get by id",
			id:   tenantID,
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, name, slug, is_active, plan, settings, created_at, updated_at FROM tenants WHERE id = \$1`).
					WithArgs(tenantID).
					WillReturnError(errors.New("connection lost"))
			},
			want:    nil,
			wantErr: errors.New("get tenant by id: connection lost"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewTenantRepository(mock)
			got, err := repo.GetByID(context.Background(), tt.id)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrTenantNotFound) {
					assert.ErrorIs(t, err, domain.ErrTenantNotFound)
				} else {
					assert.Contains(t, err.Error(), "get tenant by id")
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.Name, got.Name)
				assert.Equal(t, tt.want.Slug, got.Slug)
				assert.Equal(t, tt.want.Plan, got.Plan)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// FaceRepository Tests

func TestFaceRepository_Create(t *testing.T) {
	tenantID := uuid.New()
	faceID := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		face      *domain.Face
		mockSetup func(mock pgxmock.PgxPoolIface)
		wantErr   error
	}{
		{
			name: "successful creation with embedding",
			face: &domain.Face{
				ID:           faceID,
				TenantID:     tenantID,
				ExternalID:   "user-123",
				Embedding:    []float64{0.1, 0.2, 0.3},
				Metadata:     map[string]interface{}{"source": "mobile"},
				QualityScore: 0.95,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"created_at", "updated_at"}).
					AddRow(now, now)

				mock.ExpectQuery(`INSERT INTO faces`).
					WithArgs(
						faceID,
						tenantID,
						"user-123",
						pgxmock.AnyArg(),
						map[string]interface{}{"source": "mobile"},
						0.95,
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "face already exists",
			face: &domain.Face{
				ID:         faceID,
				TenantID:   tenantID,
				ExternalID: "user-duplicate",
				Embedding:  []float64{0.1, 0.2},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`INSERT INTO faces`).
					WithArgs(
						faceID,
						tenantID,
						"user-duplicate",
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(errors.New("duplicate key value violates unique constraint (23505)"))
			},
			wantErr: domain.ErrFaceExists,
		},
		{
			name: "database error on create",
			face: &domain.Face{
				ID:         faceID,
				TenantID:   tenantID,
				ExternalID: "user-error",
				Embedding:  []float64{0.1},
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`INSERT INTO faces`).
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(errors.New("disk full"))
			},
			wantErr: errors.New("create face: disk full"),
		},
		{
			name: "successful creation without id (auto-generate)",
			face: &domain.Face{
				TenantID:     tenantID,
				ExternalID:   "user-autoid",
				Embedding:    []float64{0.5},
				QualityScore: 0.8,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"created_at", "updated_at"}).
					AddRow(now, now)

				mock.ExpectQuery(`INSERT INTO faces`).
					WithArgs(
						pgxmock.AnyArg(),
						tenantID,
						"user-autoid",
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						0.8,
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewFaceRepository(mock)
			err = repo.Create(context.Background(), tt.face)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrFaceExists) {
					assert.ErrorIs(t, err, domain.ErrFaceExists)
				} else {
					assert.Contains(t, err.Error(), "create face")
				}
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tt.face.ID)
				assert.False(t, tt.face.CreatedAt.IsZero())
				assert.False(t, tt.face.UpdatedAt.IsZero())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFaceRepository_GetByExternalID(t *testing.T) {
	tenantID := uuid.New()
	faceID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		tenantID   uuid.UUID
		externalID string
		mockSetup  func(mock pgxmock.PgxPoolIface)
		want       *domain.Face
		wantErr    error
	}{
		{
			name:       "successful retrieval with embedding",
			tenantID:   tenantID,
			externalID: "user-123",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				embedding := pgvector.NewVector([]float32{0.1, 0.2, 0.3})
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "external_id", "embedding", "metadata", "quality_score", "created_at", "updated_at",
				}).AddRow(
					faceID,
					tenantID,
					"user-123",
					&embedding,
					map[string]interface{}{"source": "web"},
					0.92,
					now,
					now,
				)

				mock.ExpectQuery(`SELECT id, tenant_id, external_id, embedding, metadata, quality_score, created_at, updated_at FROM faces WHERE tenant_id = \$1 AND external_id = \$2`).
					WithArgs(tenantID, "user-123").
					WillReturnRows(rows)
			},
			want: &domain.Face{
				ID:           faceID,
				TenantID:     tenantID,
				ExternalID:   "user-123",
				Embedding:    []float64{0.1, 0.2, 0.3},
				Metadata:     map[string]interface{}{"source": "web"},
				QualityScore: 0.92,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			wantErr: nil,
		},
		{
			name:       "face not found",
			tenantID:   tenantID,
			externalID: "user-nonexistent",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, tenant_id, external_id, embedding, metadata, quality_score, created_at, updated_at FROM faces WHERE tenant_id = \$1 AND external_id = \$2`).
					WithArgs(tenantID, "user-nonexistent").
					WillReturnError(pgx.ErrNoRows)
			},
			want:    nil,
			wantErr: domain.ErrFaceNotFound,
		},
		{
			name:       "database error on get",
			tenantID:   tenantID,
			externalID: "user-error",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT id, tenant_id, external_id, embedding, metadata, quality_score, created_at, updated_at FROM faces WHERE tenant_id = \$1 AND external_id = \$2`).
					WithArgs(tenantID, "user-error").
					WillReturnError(errors.New("timeout"))
			},
			want:    nil,
			wantErr: errors.New("get face by external_id: timeout"),
		},
		{
			name:       "successful retrieval with nil embedding",
			tenantID:   tenantID,
			externalID: "user-no-embedding",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "tenant_id", "external_id", "embedding", "metadata", "quality_score", "created_at", "updated_at",
				}).AddRow(
					faceID,
					tenantID,
					"user-no-embedding",
					nil,
					nil,
					0.0,
					now,
					now,
				)

				mock.ExpectQuery(`SELECT id, tenant_id, external_id, embedding, metadata, quality_score, created_at, updated_at FROM faces WHERE tenant_id = \$1 AND external_id = \$2`).
					WithArgs(tenantID, "user-no-embedding").
					WillReturnRows(rows)
			},
			want: &domain.Face{
				ID:           faceID,
				TenantID:     tenantID,
				ExternalID:   "user-no-embedding",
				Embedding:    nil,
				Metadata:     nil,
				QualityScore: 0.0,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewFaceRepository(mock)
			got, err := repo.GetByExternalID(context.Background(), tt.tenantID, tt.externalID)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrFaceNotFound) {
					assert.ErrorIs(t, err, domain.ErrFaceNotFound)
				} else {
					assert.Contains(t, err.Error(), "get face by external_id")
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tt.want.ID, got.ID)
				assert.Equal(t, tt.want.TenantID, got.TenantID)
				assert.Equal(t, tt.want.ExternalID, got.ExternalID)
				assert.Equal(t, tt.want.QualityScore, got.QualityScore)

				if tt.want.Embedding != nil {
					require.NotNil(t, got.Embedding)
					assert.InDeltaSlice(t, tt.want.Embedding, got.Embedding, 0.001)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFaceRepository_Delete(t *testing.T) {
	tenantID := uuid.New()

	tests := []struct {
		name       string
		tenantID   uuid.UUID
		externalID string
		mockSetup  func(mock pgxmock.PgxPoolIface)
		wantErr    error
	}{
		{
			name:       "successful deletion",
			tenantID:   tenantID,
			externalID: "user-delete",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM faces WHERE tenant_id = \$1 AND external_id = \$2`).
					WithArgs(tenantID, "user-delete").
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: nil,
		},
		{
			name:       "face not found on delete",
			tenantID:   tenantID,
			externalID: "user-nonexistent",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM faces WHERE tenant_id = \$1 AND external_id = \$2`).
					WithArgs(tenantID, "user-nonexistent").
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: domain.ErrFaceNotFound,
		},
		{
			name:       "database error on delete",
			tenantID:   tenantID,
			externalID: "user-error",
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM faces WHERE tenant_id = \$1 AND external_id = \$2`).
					WithArgs(tenantID, "user-error").
					WillReturnError(errors.New("constraint violation"))
			},
			wantErr: errors.New("delete face: constraint violation"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewFaceRepository(mock)
			err = repo.Delete(context.Background(), tt.tenantID, tt.externalID)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, domain.ErrFaceNotFound) {
					assert.ErrorIs(t, err, domain.ErrFaceNotFound)
				} else {
					assert.Contains(t, err.Error(), "delete face")
				}
			} else {
				require.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// VerificationRepository Tests

func TestVerificationRepository_Create(t *testing.T) {
	tenantID := uuid.New()
	faceID := uuid.New()
	verificationID := uuid.New()
	now := time.Now()
	livenessPassed := true

	tests := []struct {
		name         string
		verification *domain.Verification
		mockSetup    func(mock pgxmock.PgxPoolIface)
		wantErr      error
	}{
		{
			name: "successful verification creation",
			verification: &domain.Verification{
				ID:             verificationID,
				TenantID:       tenantID,
				FaceID:         &faceID,
				ExternalID:     "user-verify-123",
				Verified:       true,
				Confidence:     0.95,
				LivenessPassed: &livenessPassed,
				LatencyMs:      150,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"created_at"}).
					AddRow(now)

				mock.ExpectQuery(`INSERT INTO verifications`).
					WithArgs(
						verificationID,
						tenantID,
						&faceID,
						"user-verify-123",
						true,
						0.95,
						&livenessPassed,
						int64(150),
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "successful verification without face_id",
			verification: &domain.Verification{
				ID:             verificationID,
				TenantID:       tenantID,
				FaceID:         nil,
				ExternalID:     "user-unknown",
				Verified:       false,
				Confidence:     0.3,
				LivenessPassed: nil,
				LatencyMs:      200,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"created_at"}).
					AddRow(now)

				mock.ExpectQuery(`INSERT INTO verifications`).
					WithArgs(
						verificationID,
						tenantID,
						pgxmock.AnyArg(),
						"user-unknown",
						false,
						0.3,
						pgxmock.AnyArg(),
						int64(200),
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "verification with auto-generated id",
			verification: &domain.Verification{
				TenantID:   tenantID,
				FaceID:     &faceID,
				ExternalID: "user-autoid",
				Verified:   true,
				Confidence: 0.88,
				LatencyMs:  120,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"created_at"}).
					AddRow(now)

				mock.ExpectQuery(`INSERT INTO verifications`).
					WithArgs(
						pgxmock.AnyArg(),
						tenantID,
						&faceID,
						"user-autoid",
						true,
						0.88,
						pgxmock.AnyArg(),
						int64(120),
					).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name: "database error on verification create",
			verification: &domain.Verification{
				ID:         verificationID,
				TenantID:   tenantID,
				ExternalID: "user-error",
				Verified:   false,
				Confidence: 0.0,
				LatencyMs:  0,
			},
			mockSetup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`INSERT INTO verifications`).
					WithArgs(
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
						pgxmock.AnyArg(),
					).
					WillReturnError(errors.New("database unavailable"))
			},
			wantErr: errors.New("create verification: database unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			tt.mockSetup(mock)

			repo := NewVerificationRepository(mock)
			err = repo.Create(context.Background(), tt.verification)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "create verification")
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, tt.verification.ID)
				assert.False(t, tt.verification.CreatedAt.IsZero())
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Helper function to test unique violation detection
func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "postgres error code 23505",
			err:  fmt.Errorf("pq: duplicate key value violates unique constraint (23505)"),
			want: true,
		},
		{
			name: "error contains unique",
			err:  fmt.Errorf("ERROR: unique constraint violated"),
			want: true,
		},
		{
			name: "error contains duplicate key",
			err:  fmt.Errorf("duplicate key value"),
			want: true,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "different error",
			err:  fmt.Errorf("connection timeout"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUniqueViolation(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}
