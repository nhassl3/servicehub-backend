package clickhouse

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func Connect(
	ctx context.Context,
	hosts []string,
	username, database, password, product, version string,
	tlsEnable bool,
) (driver.Conn, error) {
	var tlsConfig *tls.Config
	if tlsEnable {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: hosts,
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		ClientInfo: clickhouse.ClientInfo{
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: product, Version: version},
			},
		},
		Logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)), // new slog logger with JSON handler
		TLS:    tlsConfig,
	})
	if err != nil {
		return nil, err
	}

	if err = conn.Ping(ctx); err != nil {
		if exception, ok := errors.AsType[*clickhouse.Exception](err); ok {
			fmt.Printf("Exception [%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, err
	}

	return conn, nil
}

func DSN(username, password, host, database string) string {
	return fmt.Sprintf(
		"clickhouse://%s:%s@%s/%s?x-multi-statement=true&x-migrations-table-engine=MergeTree",
		username, password, host, database,
	)
}
