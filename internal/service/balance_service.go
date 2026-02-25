package service

import (
	"context"

	"github.com/nhassl3/servicehub/internal/domain"
)

type BalanceService struct {
	repo domain.BalanceRepository
}

func NewBalanceService(repo domain.BalanceRepository) *BalanceService {
	return &BalanceService{repo: repo}
}

func (s *BalanceService) GetBalance(ctx context.Context, username string) (*domain.Balance, error) {
	return s.repo.GetOrCreate(ctx, username)
}

func (s *BalanceService) Deposit(ctx context.Context, username string, amount float64) (*domain.Balance, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidInput
	}
	return s.repo.Deposit(ctx, username, amount)
}

func (s *BalanceService) GetTransactionHistory(ctx context.Context, params domain.ListTransactionsParams) ([]domain.BalanceTransaction, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	return s.repo.ListTransactions(ctx, params)
}
