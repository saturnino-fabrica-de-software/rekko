package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_CheckSearchLimit(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  uuid.UUID
		limit     int
		mockCount int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "within limit",
			tenantID:  uuid.New(),
			limit:     30,
			mockCount: 10,
			wantErr:   false,
		},
		{
			name:      "at limit boundary",
			tenantID:  uuid.New(),
			limit:     30,
			mockCount: 30,
			wantErr:   false,
		},
		{
			name:      "exceeds limit",
			tenantID:  uuid.New(),
			limit:     30,
			mockCount: 31,
			wantErr:   true,
			errMsg:    "rate limit exceeded: 31/30 requests in window",
		},
		{
			name:      "no limit configured",
			tenantID:  uuid.New(),
			limit:     0,
			mockCount: 1000,
			wantErr:   false,
		},
		{
			name:      "negative limit",
			tenantID:  uuid.New(),
			limit:     -1,
			mockCount: 1000,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			rl := NewRateLimiterWithDB(mock, time.Minute)

			ctx := context.Background()

			// If limit is configured, expect query
			if tt.limit > 0 {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(tt.mockCount)
				mock.ExpectQuery("WITH current_count AS").
					WithArgs(
						pgxmock.AnyArg(), // key
						pgxmock.AnyArg(), // window_start
						pgxmock.AnyArg(), // window_end (now)
						tt.tenantID,      // tenant_id
					).
					WillReturnRows(rows)
			}

			err = rl.CheckSearchLimit(ctx, tt.tenantID, tt.limit)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}

			if tt.limit > 0 {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

func TestRateLimiter_CleanupExpired(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rl := NewRateLimiterWithDB(mock, time.Minute)

	ctx := context.Background()

	// Expect cleanup query to delete 5 expired entries
	mock.ExpectExec("DELETE FROM rate_limit_counters").
		WillReturnResult(pgxmock.NewResult("DELETE", 5))

	deleted, err := rl.CleanupExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(5), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRateLimiter_GetCurrentCount(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  uuid.UUID
		mockCount int
		mockErr   error
		wantCount int
		wantErr   bool
	}{
		{
			name:      "existing counter",
			tenantID:  uuid.New(),
			mockCount: 15,
			wantCount: 15,
			wantErr:   false,
		},
		{
			name:      "no counter exists",
			tenantID:  uuid.New(),
			mockErr:   pgx.ErrNoRows, // Simulate no rows
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			rl := NewRateLimiterWithDB(mock, time.Minute)

			ctx := context.Background()

			if tt.mockErr != nil {
				mock.ExpectQuery("SELECT count").
					WithArgs(
						pgxmock.AnyArg(), // key
						pgxmock.AnyArg(), // window_start
					).
					WillReturnError(tt.mockErr)
			} else {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(tt.mockCount)
				mock.ExpectQuery("SELECT count").
					WithArgs(
						pgxmock.AnyArg(), // key
						pgxmock.AnyArg(), // window_start
					).
					WillReturnRows(rows)
			}

			count, err := rl.GetCurrentCount(ctx, tt.tenantID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCount, count)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRateLimiter_ResetLimit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	rl := NewRateLimiterWithDB(mock, time.Minute)

	ctx := context.Background()
	tenantID := uuid.New()

	mock.ExpectExec("DELETE FROM rate_limit_counters").
		WithArgs(pgxmock.AnyArg()). // key
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = rl.ResetLimit(ctx, tenantID)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
