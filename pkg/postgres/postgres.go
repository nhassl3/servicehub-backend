package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultMaxConns          = 25
	defaultMinConns          = 5
	defaultMaxConnLifetime   = time.Hour
	defaultMaxConnIdleTime   = 30 * time.Minute
	defaultHealthCheckPeriod = time.Minute
	defaultConnectTimeout    = 5 * time.Second
)

type Options struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration
}

func New(ctx context.Context, dsn string, opts ...Options) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}

	o := Options{
		MaxConns:          defaultMaxConns,
		MinConns:          defaultMinConns,
		MaxConnLifetime:   defaultMaxConnLifetime,
		MaxConnIdleTime:   defaultMaxConnIdleTime,
		HealthCheckPeriod: defaultHealthCheckPeriod,
		ConnectTimeout:    defaultConnectTimeout,
	}
	if len(opts) > 0 {
		o = opts[0]
	}

	cfg.MaxConns = o.MaxConns
	cfg.MinConns = o.MinConns
	cfg.MaxConnLifetime = o.MaxConnLifetime
	cfg.MaxConnIdleTime = o.MaxConnIdleTime
	cfg.HealthCheckPeriod = o.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}

	return pool, nil
}

func DSN(host string, port int, user, password, dbName, sslMode string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password, host, port, dbName, sslMode)
}
