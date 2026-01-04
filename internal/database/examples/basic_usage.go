package examples

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/saturnino-fabrica-de-software/rekko/internal/database"
)

const defaultDSN = "postgres://rekko:rekko_dev_pass@localhost:5432/rekko_dev?sslmode=disable"

// ExampleBasicMigration demonstrates basic migration usage
func ExampleBasicMigration() {
	// Connect to database
	dsn := defaultDSN
	cfg := database.DefaultPoolConfig(dsn)
	db, err := database.NewPool(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	// Run migrations
	migrator, err := database.NewMigrator(db, "rekko_dev")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = migrator.Close() }()

	if err := migrator.Up(); err != nil {
		log.Fatal(err)
	}

	log.Println("Migrations completed successfully")
}

// ExampleInsertTenant demonstrates inserting a tenant
func ExampleInsertTenant() {
	dsn := defaultDSN
	cfg := database.DefaultPoolConfig(dsn)
	db, err := database.NewPool(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Insert tenant
	var tenantID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO tenants (name, slug, plan, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Acme Corp", "acme-corp", "enterprise", `{
		"max_faces": 10000,
		"retention_days": 365,
		"features": ["liveness", "encryption"]
	}`).Scan(&tenantID)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Tenant created: %s\n", tenantID)
}

// ExampleInsertAPIKey demonstrates inserting an API key
func ExampleInsertAPIKey() {
	dsn := defaultDSN
	cfg := database.DefaultPoolConfig(dsn)
	db, err := database.NewPool(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Assume tenant exists
	tenantID := "00000000-0000-0000-0000-000000000001"

	// Insert API key
	var apiKeyID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO api_keys (tenant_id, name, key_hash, key_prefix, environment)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, tenantID, "Production Key", "sha256_hash_here", "rekko_live_abc1", "live").Scan(&apiKeyID)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("API Key created: %s\n", apiKeyID)
}

// ExampleQueryTenant demonstrates querying a tenant by slug
func ExampleQueryTenant() {
	dsn := defaultDSN
	cfg := database.DefaultPoolConfig(dsn)
	db, err := database.NewPool(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Query tenant
	var (
		id       string
		name     string
		plan     string
		settings string
	)

	err = db.QueryRowContext(ctx, `
		SELECT id, name, plan, settings
		FROM tenants
		WHERE slug = $1 AND is_active = true
	`, "acme-corp").Scan(&id, &name, &plan, &settings)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("Tenant not found")
			return
		}
		log.Fatal(err)
	}

	fmt.Printf("Tenant: %s (Plan: %s)\n", name, plan)
	fmt.Printf("Settings: %s\n", settings)
}

// ExampleVerifyAPIKey demonstrates API key verification
func ExampleVerifyAPIKey() {
	dsn := defaultDSN
	cfg := database.DefaultPoolConfig(dsn)
	db, err := database.NewPool(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Hash of the API key to verify
	keyHash := "sha256_hash_here"

	// Verify and get tenant
	var (
		tenantID    string
		tenantName  string
		environment string
	)

	err = db.QueryRowContext(ctx, `
		SELECT
			t.id,
			t.name,
			ak.environment
		FROM api_keys ak
		INNER JOIN tenants t ON t.id = ak.tenant_id
		WHERE ak.key_hash = $1
		  AND ak.is_active = true
		  AND t.is_active = true
		  AND (ak.last_used_at IS NULL OR ak.last_used_at < NOW() - INTERVAL '1 minute')
	`, keyHash).Scan(&tenantID, &tenantName, &environment)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("Invalid or inactive API key")
			return
		}
		log.Fatal(err)
	}

	// Update last_used_at
	_, err = db.ExecContext(ctx, `
		UPDATE api_keys
		SET last_used_at = NOW()
		WHERE key_hash = $1
	`, keyHash)

	if err != nil {
		log.Printf("Warning: failed to update last_used_at: %v", err)
	}

	fmt.Printf("Authenticated: %s (Environment: %s)\n", tenantName, environment)
}

// ExampleHealthCheck demonstrates database health checking
func ExampleHealthCheck() {
	dsn := defaultDSN
	cfg := database.DefaultPoolConfig(dsn)
	db, err := database.NewPool(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Health check
	if err := database.HealthCheck(ctx, db); err != nil {
		log.Printf("Database unhealthy: %v", err)
		return
	}

	// Get pool stats
	stats := database.Stats(db)
	fmt.Printf("Pool stats:\n")
	fmt.Printf("  Open connections: %d\n", stats.OpenConnections)
	fmt.Printf("  In use: %d\n", stats.InUse)
	fmt.Printf("  Idle: %d\n", stats.Idle)
	fmt.Printf("  Wait count: %d\n", stats.WaitCount)
}

// ExampleTransaction demonstrates a transaction with tenant creation and API key
func ExampleTransaction() {
	dsn := defaultDSN
	cfg := database.DefaultPoolConfig(dsn)
	db, err := database.NewPool(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Begin transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }() // Rollback if not committed

	// Insert tenant
	var tenantID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO tenants (name, slug, plan, settings)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "New Company", "new-company", "starter", `{}`).Scan(&tenantID)

	if err != nil {
		log.Fatal(err)
	}

	// Insert API key
	_, err = tx.ExecContext(ctx, `
		INSERT INTO api_keys (tenant_id, name, key_hash, key_prefix, environment)
		VALUES ($1, $2, $3, $4, $5)
	`, tenantID, "Default Key", "hash123", "rekko_test_xyz", "test")

	if err != nil {
		log.Fatal(err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Tenant and API key created in transaction: %s\n", tenantID)
}
