package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrator handles database migrations
type Migrator struct {
	m *migrate.Migrate
}

// NewMigrator creates a migrator instance
func NewMigrator(db *sql.DB, dbName string) (*Migrator, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: dbName,
	})
	if err != nil {
		return nil, fmt.Errorf("create postgres driver: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, dbName, driver)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return &Migrator{m: m}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	err := m.m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// Down rolls back the last migration (DEV ONLY)
func (m *Migrator) Down() error {
	if err := m.m.Steps(-1); err != nil {
		return fmt.Errorf("rollback migration: %w", err)
	}
	return nil
}

// Version returns current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get version: %w", err)
	}
	return version, dirty, nil
}

// Force sets the migration version without running migrations (DANGEROUS)
func (m *Migrator) Force(version int) error {
	if err := m.m.Force(version); err != nil {
		return fmt.Errorf("force version: %w", err)
	}
	return nil
}

// Close closes the migrator
func (m *Migrator) Close() error {
	srcErr, dbErr := m.m.Close()
	if srcErr != nil {
		return fmt.Errorf("close source: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("close database: %w", dbErr)
	}
	return nil
}
