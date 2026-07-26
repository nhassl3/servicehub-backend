package service

import (
	"context"
	"fmt"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"go.uber.org/zap"
)

type OrderService struct {
	eventPublisher domain.EventPublisher
	orderRepo      domain.OrderRepository
	userRedis      domain.UserRedis
	log            *zap.Logger
}

func NewOrderService(orderRepo domain.OrderRepository, publisher domain.EventPublisher, userRedis domain.UserRedis, logger *zap.Logger) *OrderService {
	return &OrderService{eventPublisher: publisher, orderRepo: orderRepo, userRedis: userRedis, log: logger}
}

// CreateOrder creates order and in parallel send to notifier service message with notifying user action.
// This is mechanism realizing with Apache Kafka data flow broker
func (s *OrderService) CreateOrder(ctx context.Context, username string) (*domain.Order, error) {
	order, err := s.orderRepo.Checkout(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("order_service.CreateOrder: failed to create order (call checkout): %w", err)
	}

	{
		var email string
		payload, ok := interceptors.PayloadFromContext(ctx)
		if !ok {
			user, err := s.userRedis.Profile(ctx, username)
			if err != nil {
				s.log.Error("order_service.CreateOrder: failed to get user from redis", zap.Error(err))
			}
			email = user.Email
		} else {
			email = payload.Email
		}

		if err = s.eventPublisher.PublishOrderCreated(ctx, domain.OrderCreatedPayload{
			Email:    email,
			OrderUID: order.ID,
			Username: username,
			Total:    order.TotalAmount,
		}); err != nil {
			s.log.Error("failed to publish order.created event", zap.Error(err))
		}
	}

	return order, nil
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
