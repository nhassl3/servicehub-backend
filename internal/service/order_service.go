package service

import (
	"context"

	"github.com/nhassl3/servicehub/internal/domain"
)

type OrderService struct {
	orderRepo domain.OrderRepository
}

func NewOrderService(orderRepo domain.OrderRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo}
}

func (s *OrderService) CreateOrder(ctx context.Context, username string) (*domain.Order, error) {
	return s.orderRepo.Checkout(ctx, username)
}

func (s *OrderService) GetOrder(ctx context.Context, username, id string) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order.Username != username {
		return nil, domain.ErrForbidden
	}
	return order, nil
}

func (s *OrderService) ListOrders(ctx context.Context, params domain.ListOrdersParams) ([]domain.Order, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	return s.orderRepo.List(ctx, params)
}

func (s *OrderService) CancelOrder(ctx context.Context, username, id string) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order.Username != username {
		return nil, domain.ErrForbidden
	}
	if order.Status != domain.OrderStatusPending && order.Status != domain.OrderStatusPaid {
		return nil, domain.ErrInvalidInput
	}
	return s.orderRepo.UpdateStatus(ctx, id, domain.OrderStatusCancelled)
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id, status string) (*domain.Order, error) {
	return s.orderRepo.UpdateStatus(ctx, id, status)
}
