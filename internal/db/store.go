package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store combines the SQLC Queries with a connection pool, adding support
// for transactional operations via ExecTx.
type Store struct {
	*Queries
	pool *pgxpool.Pool
}

// NewStore creates a new Store backed by the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		Queries: New(pool),
		pool:    pool,
	}
}

// ExecTx executes fn inside a database transaction.
// The transaction is committed if fn returns nil, rolled back otherwise.
func (s *Store) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	// Create new connection from pool
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed when acquire db connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err = fn(s.WithTx(tx)); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err : %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}

// Pool returns the underlying connection pool for operations that require
// direct pool access (e.g. advisory locks, COPY protocol).
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// BeginTx starts a transaction and returns a Queries scoped to it.
// The caller is responsible for committing or rolling back the transaction.
func (s *Store) BeginTx(ctx context.Context) (*Queries, pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	return s.WithTx(tx), tx, nil
}
