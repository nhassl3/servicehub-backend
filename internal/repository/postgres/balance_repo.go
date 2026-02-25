package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type BalanceRepo struct {
	store *db.Store
}

func NewBalanceRepo(store *db.Store) *BalanceRepo {
	return &BalanceRepo{store: store}
}

func (r *BalanceRepo) GetOrCreate(ctx context.Context, username string) (*domain.Balance, error) {
	row, err := r.store.UpsertBalance(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("balance_repo.GetOrCreate: %w", err)
	}
	return &domain.Balance{Username: row.Username, Amount: row.Amount}, nil
}

func (r *BalanceRepo) Deposit(ctx context.Context, username string, amount float64) (*domain.Balance, error) {
	var bal *domain.Balance

	err := r.store.ExecTx(ctx, func(q *db.Queries) error {
		row, err := q.AddToBalance(ctx, db.AddToBalanceParams{
			Username: username,
			Amount:   amount,
		})
		if err != nil {
			return err
		}
		bal = &domain.Balance{Username: row.Username, Amount: row.Amount}

		_, err = q.CreateBalanceTx(ctx, db.CreateBalanceTxParams{
			Username: username,
			Type:     domain.TxTypeDeposit,
			Amount:   amount,
			Comment:  "Manual deposit",
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("balance_repo.Deposit: %w", err)
	}
	return bal, nil
}

func (r *BalanceRepo) Withdraw(ctx context.Context, username string, amount float64) (*domain.Balance, error) {
	var bal *domain.Balance

	err := r.store.ExecTx(ctx, func(q *db.Queries) error {
		current, err := q.GetBalanceForUpdate(ctx, username)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrInsufficientFunds
			}
			return err
		}
		if current.Amount < amount {
			return domain.ErrInsufficientFunds
		}

		row, err := q.DeductFromBalance(ctx, db.DeductFromBalanceParams{
			Username: username,
			Amount:   amount,
		})
		if err != nil {
			return err
		}
		bal = &domain.Balance{Username: row.Username, Amount: row.Amount}

		_, err = q.CreateBalanceTx(ctx, db.CreateBalanceTxParams{
			Username: username,
			Type:     domain.TxTypeWithdraw,
			Amount:   amount,
			Comment:  "Withdrawal",
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("balance_repo.Withdraw: %w", err)
	}
	return bal, nil
}

func (r *BalanceRepo) ListTransactions(ctx context.Context, params domain.ListTransactionsParams) ([]domain.BalanceTransaction, int64, error) {
	total, err := r.store.CountBalanceTxByUsername(ctx, params.Username)
	if err != nil {
		return nil, 0, fmt.Errorf("balance_repo.ListTransactions count: %w", err)
	}

	rows, err := r.store.ListBalanceTxByUsername(ctx, db.ListBalanceTxByUsernameParams{
		Username: params.Username,
		Limit:    params.Limit,
		Offset:   params.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("balance_repo.ListTransactions: %w", err)
	}

	txs := make([]domain.BalanceTransaction, len(rows))
	for i, row := range rows {
		txs[i] = domain.BalanceTransaction{
			ID:        row.ID,
			Username:  row.Username,
			Type:      row.Type,
			Amount:    row.Amount,
			Comment:   row.Comment,
			CreatedAt: pgTimeTZ(row.CreatedAt, time.UTC),
		}
	}
	return txs, total, nil
}
