package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/saturnino-fabrica-de-software/rekko/internal/config"
	"github.com/saturnino-fabrica-de-software/rekko/internal/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Flags
	action := flag.String("action", "up", "Migration action: up, down, version, force")
	steps := flag.Int("steps", 0, "Number of migration steps (for force action)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Convert pgxpool DSN to standard database/sql DSN
	dsn := cfg.DatabaseURL

	// Connect to database using database/sql (required by golang-migrate)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Verify connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to database")

	// Create migrator
	migrator, err := database.NewMigrator(db, "rekko_dev")
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer func() { _ = migrator.Close() }()

	// Execute action
	switch *action {
	case "up":
		log.Println("Running migrations...")
		if err := migrator.Up(); err != nil {
			return fmt.Errorf("migration up failed: %w", err)
		}
		log.Println("✓ Migrations completed successfully")

	case "down":
		log.Println("Rolling back last migration...")
		if err := migrator.Down(); err != nil {
			return fmt.Errorf("migration down failed: %w", err)
		}
		log.Println("✓ Migration rolled back successfully")

	case "version":
		version, dirty, err := migrator.Version()
		if err != nil {
			return fmt.Errorf("failed to get version: %w", err)
		}
		if dirty {
			log.Printf("Current version: %d (DIRTY - migration incomplete)\n", version)
		} else {
			log.Printf("Current version: %d\n", version)
		}

	case "force":
		if *steps == 0 {
			return fmt.Errorf("steps flag is required for force action")
		}
		log.Printf("Forcing migration to version %d...\n", *steps)
		if err := migrator.Force(*steps); err != nil {
			return fmt.Errorf("force migration failed: %w", err)
		}
		log.Println("✓ Migration version forced successfully")

	default:
		return fmt.Errorf("invalid action: %s (use: up, down, version, force)", *action)
	}

	return nil
}
