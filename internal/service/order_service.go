package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"go.uber.org/zap"
)

type OrderService struct {
	eventPublisher domain.EventPublisher
	orderRepo      domain.OrderRepository
	productRepo    domain.ProductRepository
	userRedis      domain.UserRedis
	log            *zap.Logger
}

func NewOrderService(orderRepo domain.OrderRepository, productRepo domain.ProductRepository, publisher domain.EventPublisher, userRedis domain.UserRedis, logger *zap.Logger) *OrderService {
	return &OrderService{
		eventPublisher: publisher,
		orderRepo:      orderRepo,
		productRepo:    productRepo,
		userRedis:      userRedis,
		log:            logger,
	}
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
			OrderUID: order.UID,
			Username: username,
			Total:    order.TotalAmount,
		}); err != nil {
			s.log.Error("failed to publish order.created event", zap.Error(err))
		}
	}

	s.publishOrderItems(ctx, order)

	return order, nil
}

// publishOrderItems emits an order_item_created fact for every line item so the
// analytics layer can aggregate per-category sales. Best-effort (Postgres is
// the source of truth).
func (s *OrderService) publishOrderItems(ctx context.Context, order *domain.Order) {
	occurredAt := time.Now()
	for _, item := range order.Items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			s.log.Warn("order_service: failed to load product for analytics event", zap.Error(err))
			continue
		}
		if err := s.eventPublisher.PublishOrderItemCreated(ctx, domain.OrderItemCreatedPayload{
			OrderID:    order.ID,
			ProductID:  item.ProductID,
			Title:      product.Title,
			CategoryID: product.CategoryID,
			SellerID:   product.SellerID,
			Qty:        item.Quantity,
			Total:      item.TotalPrice,
			OccurredAt: occurredAt,
		}); err != nil {
			s.log.Warn("(Kafka) analytics: failed to publish order item created", zap.Error(err))
		}
	}
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
	return s.UpdateOrderStatus(ctx, id, domain.OrderStatusCancelled)
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, id, status string) (*domain.Order, error) {
	order, err := s.orderRepo.UpdateStatus(ctx, id, status)
	if err != nil {
		return nil, fmt.Errorf("order_service.UpdateOrderStatus: failed to update order: %w", err)
	}

	{
		var email string
		payload, ok := interceptors.PayloadFromContext(ctx)
		if !ok {
			user, err := s.userRedis.Profile(ctx, id)
			if err != nil {
				s.log.Error("order_service.UpdateOrderStatus: failed to get user from redis", zap.Error(err))
			}
			email = user.Email
		} else {
			email = payload.Email
		}

		if err = s.eventPublisher.PublishOrderStatusChanged(ctx, domain.OrderStatusChangedPayload{
			Email:     email,
			OrderUID:  order.Order.UID,
			OldStatus: order.OldStatus,
			NewStatus: status,
		}); err != nil {
			s.log.Error("failed to publish order.status event", zap.Error(err))
		}
	}

	return order.Order, err
}
