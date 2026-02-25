package grpc

import (
	"context"

	balancev1 "github.com/nhassl3/servicehub-contracts/pkg/pb/balance/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
)

// BalanceHandler implements balancev1.BalanceServiceServer.
//
// Implemented RPC methods:
//   - GetBalance
//   - Deposit
//   - GetTransactionHistory
type BalanceHandler struct {
	balancev1.UnimplementedBalanceServiceServer
	svc *service.BalanceService
}

func NewBalanceHandler(svc *service.BalanceService) *BalanceHandler {
	return &BalanceHandler{svc: svc}
}

func (h *BalanceHandler) GetBalance(ctx context.Context, _ *balancev1.GetBalanceRequest) (*balancev1.GetBalanceResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	bal, err := h.svc.GetBalance(ctx, username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &balancev1.GetBalanceResponse{Amount: bal.Amount}, nil
}

func (h *BalanceHandler) Deposit(ctx context.Context, req *balancev1.DepositRequest) (*balancev1.DepositResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	bal, err := h.svc.Deposit(ctx, username, req.Amount)
	if err != nil {
		return nil, domainErr(err)
	}
	return &balancev1.DepositResponse{Amount: bal.Amount}, nil
}

func (h *BalanceHandler) GetTransactionHistory(ctx context.Context, req *balancev1.GetTransactionHistoryRequest) (*balancev1.GetTransactionHistoryResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	txs, total, err := h.svc.GetTransactionHistory(ctx, domain.ListTransactionsParams{
		Username: username,
		Limit:    req.Limit,
		Offset:   req.Offset,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	proto := make([]*balancev1.BalanceTransaction, len(txs))
	for i, tx := range txs {
		proto[i] = &balancev1.BalanceTransaction{
			Id:        tx.ID,
			Type:      tx.Type,
			Amount:    tx.Amount,
			Comment:   tx.Comment,
			CreatedAt: safeTimestamp(tx.CreatedAt),
		}
	}
	return &balancev1.GetTransactionHistoryResponse{Transactions: proto, Total: total}, nil
}
