package postgres

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/nhassl3/servicehub-backend/internal/config"
	"github.com/nhassl3/servicehub-backend/pkg/postgres"
)

func EnsureSchema(cfg config.DBConfig, path string) error {
	dsnURL := postgres.DSN(cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name, cfg.SSLMode)

	m, err := migrate.New("file://"+path, dsnURL)
	if err != nil {
		return fmt.Errorf("create migrate: %w", err)
	}
	defer func(m *migrate.Migrate) {
		_, _ = m.Close()
	}(m) //nolint:errcheck

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}
