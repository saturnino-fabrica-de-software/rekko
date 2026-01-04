package database_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saturnino-fabrica-de-software/rekko/internal/database"
)

// TestMigratorIntegration tests the migration functionality
func TestMigratorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Setup test database connection
	dsn := "postgres://rekko:rekko_dev_pass@localhost:5432/rekko_test?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx))

	// Clean up test database before running tests
	cleanupDatabase(t, db)

	t.Run("NewMigrator creates migrator successfully", func(t *testing.T) {
		migrator, err := database.NewMigrator(db, "rekko_test")
		require.NoError(t, err)
		require.NotNil(t, migrator)
		defer func() { _ = migrator.Close() }()
	})

	t.Run("Up runs migrations successfully", func(t *testing.T) {
		migrator, err := database.NewMigrator(db, "rekko_test")
		require.NoError(t, err)
		defer func() { _ = migrator.Close() }()

		// Run migrations
		err = migrator.Up()
		require.NoError(t, err)

		// Verify tables exist
		assertTableExists(t, db, "tenants")
		assertTableExists(t, db, "api_keys")
	})

	t.Run("Version returns current version", func(t *testing.T) {
		migrator, err := database.NewMigrator(db, "rekko_test")
		require.NoError(t, err)
		defer func() { _ = migrator.Close() }()

		version, dirty, err := migrator.Version()
		require.NoError(t, err)
		assert.False(t, dirty, "migration should not be dirty")
		assert.Equal(t, uint(1), version, "should be at version 1")
	})

	t.Run("Schema validation after migration", func(t *testing.T) {
		// Test tenants table schema
		t.Run("tenants table has correct columns", func(t *testing.T) {
			columns := getTableColumns(t, db, "tenants")
			expectedColumns := []string{
				"id", "name", "slug", "is_active", "plan",
				"settings", "created_at", "updated_at",
			}
			for _, col := range expectedColumns {
				assert.Contains(t, columns, col, "tenants should have column %s", col)
			}
		})

		// Test api_keys table schema
		t.Run("api_keys table has correct columns", func(t *testing.T) {
			columns := getTableColumns(t, db, "api_keys")
			expectedColumns := []string{
				"id", "tenant_id", "name", "key_hash", "key_prefix",
				"environment", "is_active", "last_used_at", "created_at",
			}
			for _, col := range expectedColumns {
				assert.Contains(t, columns, col, "api_keys should have column %s", col)
			}
		})

		// Test indexes exist
		t.Run("indexes are created", func(t *testing.T) {
			indexes := getTableIndexes(t, db, "tenants")
			assert.Contains(t, indexes, "idx_tenants_slug")
			assert.Contains(t, indexes, "idx_tenants_active")

			apiKeyIndexes := getTableIndexes(t, db, "api_keys")
			assert.Contains(t, apiKeyIndexes, "idx_api_keys_hash")
			assert.Contains(t, apiKeyIndexes, "idx_api_keys_tenant")
		})
	})

	t.Run("Data insertion works", func(t *testing.T) {
		// Insert tenant
		var tenantID string
		err := db.QueryRow(`
			INSERT INTO tenants (name, slug, plan, settings)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, "Test Org", "test-org", "pro", `{"max_faces": 1000}`).Scan(&tenantID)
		require.NoError(t, err)
		assert.NotEmpty(t, tenantID)

		// Insert API key
		var apiKeyID string
		err = db.QueryRow(`
			INSERT INTO api_keys (tenant_id, name, key_hash, key_prefix, environment)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, tenantID, "Test Key", "hash123", "rekko_test_abc1", "test").Scan(&apiKeyID)
		require.NoError(t, err)
		assert.NotEmpty(t, apiKeyID)

		// Verify cascade delete
		_, err = db.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
		require.NoError(t, err)

		// API key should be deleted automatically
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE id = $1", apiKeyID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "API key should be deleted via CASCADE")
	})

	// Clean up after all tests
	t.Cleanup(func() {
		cleanupDatabase(t, db)
	})
}

// Helper functions

func cleanupDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	// Drop all tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS api_keys;
		DROP TABLE IF EXISTS tenants;
		DROP TABLE IF EXISTS schema_migrations;
	`)
	if err != nil {
		t.Logf("cleanup warning: %v", err)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, tableName string) {
	t.Helper()

	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`, tableName).Scan(&exists)

	require.NoError(t, err)
	assert.True(t, exists, "table %s should exist", tableName)
}

func getTableColumns(t *testing.T, db *sql.DB, tableName string) []string {
	t.Helper()

	rows, err := db.Query(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = $1
		ORDER BY ordinal_position
	`, tableName)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var columns []string
	for rows.Next() {
		var col string
		require.NoError(t, rows.Scan(&col))
		columns = append(columns, col)
	}

	return columns
}

func getTableIndexes(t *testing.T, db *sql.DB, tableName string) []string {
	t.Helper()

	rows, err := db.Query(`
		SELECT indexname
		FROM pg_indexes
		WHERE schemaname = 'public'
		AND tablename = $1
	`, tableName)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var indexes []string
	for rows.Next() {
		var idx string
		require.NoError(t, rows.Scan(&idx))
		indexes = append(indexes, idx)
	}

	return indexes
}
