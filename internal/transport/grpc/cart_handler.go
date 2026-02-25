package grpc

import (
	"context"

	cartv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/cart/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
	"github.com/nhassl3/servicehub/internal/transport/grpc/interceptors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CartHandler implements cartv1.CartServiceServer.
//
// Implemented RPC methods:
//   - GetCart
//   - AddItem
//   - RemoveItem
//   - UpdateItemQty
//   - ClearCart
type CartHandler struct {
	cartv1.UnimplementedCartServiceServer
	svc *service.CartService
}

func NewCartHandler(svc *service.CartService) *CartHandler {
	return &CartHandler{svc: svc}
}

func (h *CartHandler) GetCart(ctx context.Context, _ *cartv1.GetCartRequest) (*cartv1.GetCartResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	cart, err := h.svc.GetCart(ctx, username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &cartv1.GetCartResponse{Cart: protoCart(cart)}, nil
}

func (h *CartHandler) AddItem(ctx context.Context, req *cartv1.AddItemRequest) (*cartv1.AddItemResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	cart, err := h.svc.AddItem(ctx, username, req.ProductId, int(req.Quantity))
	if err != nil {
		return nil, domainErr(err)
	}
	return &cartv1.AddItemResponse{Cart: protoCart(cart)}, nil
}

func (h *CartHandler) RemoveItem(ctx context.Context, req *cartv1.RemoveItemRequest) (*cartv1.RemoveItemResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	cart, err := h.svc.RemoveItem(ctx, username, req.ProductId)
	if err != nil {
		return nil, domainErr(err)
	}
	return &cartv1.RemoveItemResponse{Cart: protoCart(cart)}, nil
}

func (h *CartHandler) UpdateItemQty(ctx context.Context, req *cartv1.UpdateItemQtyRequest) (*cartv1.UpdateItemQtyResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	cart, err := h.svc.UpdateItemQty(ctx, username, req.ProductId, int(req.Quantity))
	if err != nil {
		return nil, domainErr(err)
	}
	return &cartv1.UpdateItemQtyResponse{Cart: protoCart(cart)}, nil
}

func (h *CartHandler) ClearCart(ctx context.Context, _ *cartv1.ClearCartRequest) (*cartv1.ClearCartResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.svc.ClearCart(ctx, username); err != nil {
		return nil, domainErr(err)
	}
	return &cartv1.ClearCartResponse{Success: true}, nil
}

// ── Proto mapper ─────────────────────────────────────────────────────────────

func protoCart(c *domain.Cart) *cartv1.Cart {
	items := make([]*cartv1.CartItem, len(c.Items))
	var subtotal float64
	for i, ci := range c.Items {
		total := float64(ci.Quantity) * ci.UnitPrice
		subtotal += total
		items[i] = &cartv1.CartItem{
			Id:         ci.ID,
			ProductId:  ci.ProductID,
			Quantity:   int32(ci.Quantity),
			UnitPrice:  ci.UnitPrice,
			TotalPrice: total,
		}
	}
	return &cartv1.Cart{
		Id:       c.ID,
		Username: c.Username,
		Items:    items,
		Subtotal: subtotal,
	}
}

// mustUsername extracts the authenticated username from context or returns
// an Unauthenticated gRPC status error.
func mustUsername(ctx context.Context) (string, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing auth payload")
	}
	return payload.Username, nil
}
