package clickhouse

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// EnsureSchema applies idempotent DDL (database + fact table + aggregate
// materialized views). It is safe to call on every startup.
func EnsureSchema(ctx context.Context, conn driver.Conn) error {
	entries, err := os.ReadDir("internal/repository/clickhouse/migrations")
	if err != nil {
		return err
	}

	for _, e := range entries {
		sql, err := os.ReadFile(filepath.Join("internal/clickhouse/migrations", e.Name()))
		if err != nil {
			return err
		}

		if err := conn.Exec(ctx, string(sql)); err != nil {
			return err
		}
	}
	for _, stmt := range statements {
		if err := conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("clickhouse.EnsureSchema: %w", err)
		}
	}
	return nil
}
