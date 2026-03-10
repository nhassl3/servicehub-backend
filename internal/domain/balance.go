package domain

import (
	"context"
	"time"
)

const (
	TxTypeDeposit    = "deposit"
	TxTypeWithdraw   = "withdraw"
	TxTypeProfit     = "profit"
	TxTypeCommission = "commission"
)

type Balance struct {
	Username string  `db:"username"`
	Amount   float64 `db:"amount"`
}

type BalanceTransaction struct {
	ID        int64     `db:"id"`
	Username  string    `db:"username"`
	Type      string    `db:"type"`
	Amount    float64   `db:"amount"`
	Comment   string    `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
}

type ListTransactionsParams struct {
	Username string
	Limit    int32
	Offset   int32
}

//go:generate mockgen -source=balance.go -destination=../repository/mock/balance_repo_mock.go -package=mockrepo
type BalanceRepository interface {
	GetOrCreate(ctx context.Context, username string) (*Balance, error)
	Deposit(ctx context.Context, username string, amount float64) (*Balance, error)
	Withdraw(ctx context.Context, username string, amount float64) (*Balance, error)
	ListTransactions(ctx context.Context, params ListTransactionsParams) ([]BalanceTransaction, int64, error)
}
