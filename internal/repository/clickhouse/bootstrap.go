package clickhouse

import (
	"errors"
	"fmt"

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
		return fmt.Errorf("failed to load golang-migrate: %w", err)
	}
	defer func(m *migrate.Migrate) {
		_, _ = m.Close()
	}(m) //nolint:errcheck

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
