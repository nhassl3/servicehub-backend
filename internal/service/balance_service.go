package service

import (
	"context"
	"fmt"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"go.uber.org/zap"
)

type BalanceService struct {
	eventPublisher domain.EventPublisher
	userRedis      domain.UserRedis
	repo           domain.BalanceRepository
	log            *zap.Logger
}

func NewBalanceService(
	repo domain.BalanceRepository,
	eventPublisher domain.EventPublisher,
	userRedis domain.UserRedis,
	log *zap.Logger,
) *BalanceService {
	return &BalanceService{repo: repo, eventPublisher: eventPublisher, userRedis: userRedis, log: log}
}

func (s *BalanceService) GetBalance(ctx context.Context, username string) (*domain.Balance, error) {
	return s.repo.GetOrCreate(ctx, username)
}

func (s *BalanceService) Deposit(ctx context.Context, username string, amount float64) (*domain.Balance, error) {
	if amount <= 0 {
		return nil, domain.ErrInvalidInput
	}
	balance, err := s.repo.Deposit(ctx, username, amount)
	if err != nil {
		return nil, fmt.Errorf("balance_service.Deposit: failed to deposit amount on user balance: %w", err)
	}

	s.log.Info("AMOUNT COMPARE", zap.Float64("amount", amount), zap.Float64("balance", balance.Amount))

	{
		var email string
		payload, ok := interceptors.PayloadFromContext(ctx)
		if !ok {
			user, err := s.userRedis.Profile(ctx, username)
			if err != nil {
				s.log.Warn("balance_service.Deposit: failed to get user profile", zap.Error(err))
			}
			email = user.Email
		} else {
			email = payload.Email
		}
		if err = s.eventPublisher.PublishBalanceUpdated(ctx, domain.BalanceUpdatedPayload{
			Email:      email,
			Username:   username,
			Amount:     amount,
			NewBalance: balance.Amount,
		}); err != nil {
			s.log.Warn("balance_service.Deposit: failed to publish balance updated event", zap.Error(err))
		}
	}

	return balance, nil
}

func (s *BalanceService) GetTransactionHistory(ctx context.Context, params domain.ListTransactionsParams) ([]domain.BalanceTransaction, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	return s.repo.ListTransactions(ctx, params)
}
