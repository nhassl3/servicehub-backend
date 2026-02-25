package grpc

import (
	"context"

	orderv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/order/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
)

// OrderHandler implements orderv1.OrderServiceServer.
//
// Implemented RPC methods:
//   - CreateOrder
//   - GetOrder
//   - ListOrders
//   - CancelOrder
//   - UpdateOrderStatus
type OrderHandler struct {
	orderv1.UnimplementedOrderServiceServer
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, _ *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	order, err := h.svc.CreateOrder(ctx, username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &orderv1.CreateOrderResponse{Order: protoOrder(order)}, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	order, err := h.svc.GetOrder(ctx, username, req.Id)
	if err != nil {
		return nil, domainErr(err)
	}
	return &orderv1.GetOrderResponse{Order: protoOrder(order)}, nil
}

func (h *OrderHandler) ListOrders(ctx context.Context, req *orderv1.ListOrdersRequest) (*orderv1.ListOrdersResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	orders, total, err := h.svc.ListOrders(ctx, domain.ListOrdersParams{
		Username: username,
		Status:   req.Status,
		Limit:    req.Limit,
		Offset:   req.Offset,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	proto := make([]*orderv1.Order, len(orders))
	for i, o := range orders {
		proto[i] = protoOrder(&o)
	}
	return &orderv1.ListOrdersResponse{Orders: proto, Total: total}, nil
}

func (h *OrderHandler) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.CancelOrderResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	order, err := h.svc.CancelOrder(ctx, username, req.Id)
	if err != nil {
		return nil, domainErr(err)
	}
	return &orderv1.CancelOrderResponse{Order: protoOrder(order)}, nil
}

func (h *OrderHandler) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	order, err := h.svc.UpdateOrderStatus(ctx, req.Id, req.Status)
	if err != nil {
		return nil, domainErr(err)
	}
	return &orderv1.UpdateOrderStatusResponse{Order: protoOrder(order)}, nil
}

// ── Proto mapper ─────────────────────────────────────────────────────────────

func protoOrder(o *domain.Order) *orderv1.Order {
	items := make([]*orderv1.OrderItem, len(o.Items))
	for i, oi := range o.Items {
		items[i] = &orderv1.OrderItem{
			Id:         oi.ID,
			ProductId:  oi.ProductID,
			Quantity:   int32(oi.Quantity),
			UnitPrice:  oi.UnitPrice,
			TotalPrice: oi.TotalPrice,
		}
	}
	return &orderv1.Order{
		Id:          o.ID,
		Uid:         o.UID,
		Username:    o.Username,
		Status:      o.Status,
		TotalAmount: o.TotalAmount,
		Items:       items,
		CreatedAt:   safeTimestamp(o.CreatedAt),
		UpdatedAt:   safeTimestamp(o.UpdatedAt),
	}
}
