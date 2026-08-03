package clickhouse

import (
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/clickhouse"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/nhassl3/servicehub-backend/internal/config"
	"github.com/nhassl3/servicehub-backend/pkg/clickhouse"
)

// EnsureSchema applies idempotent DDL (database + fact table + aggregate
// materialized views). It is safe to call on every startup.
func EnsureSchema(cfg config.ClickhouseConfig, path string) error {
	dbURL := clickhouse.DSN(cfg.Username, cfg.Password, cfg.Hosts[0], cfg.Database)

	m, err := migrate.New("file://"+path, dbURL)
	if err != nil {
		log.Fatalf("Migration initialization failed: %v", err)
	}

	// Apply all up migrations
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations applied successfully!")

	return nil
}
