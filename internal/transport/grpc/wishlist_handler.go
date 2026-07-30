package grpc

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/service"
	wishlistv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/wishlist/v1"
)

// WishlistHandler implements wishlistv1.WishlistServiceServer.
//
// Implemented RPC methods:
//   - GetWishlist
//   - AddItem
//   - RemoveItem
type WishlistHandler struct {
	wishlistv1.UnimplementedWishlistServiceServer
	svc *service.WishlistService
}

func NewWishlistHandler(svc *service.WishlistService) *WishlistHandler {
	return &WishlistHandler{svc: svc}
}

func (h *WishlistHandler) GetWishlist(ctx context.Context, _ *wishlistv1.GetWishlistRequest) (*wishlistv1.GetWishlistResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	items, err := h.svc.GetWishlist(ctx, username)
	if err != nil {
		return nil, domainErr(err)
	}
	proto := make([]*wishlistv1.WishlistItem, len(items))
	for i, wi := range items {
		proto[i] = protoWishlistItem(&wi)
	}
	return &wishlistv1.GetWishlistResponse{Items: proto}, nil
}

func (h *WishlistHandler) InWishlist(ctx context.Context, req *wishlistv1.InWishlistRequest) (*wishlistv1.InWishlistResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	ok, err := h.svc.Exists(ctx, username, req.GetProductId())
	if err != nil {
		return nil, domainErr(err)
	}
	return &wishlistv1.InWishlistResponse{InWishlist: ok}, nil
}

func (h *WishlistHandler) ToggleWishlistItem(ctx context.Context, req *wishlistv1.ToggleWishlistItemRequest) (*wishlistv1.ToggleWishlistItemResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	item, err := h.svc.ToggleWishlistItem(ctx, username, req.GetProductId())
	if err != nil {
		return nil, domainErr(err)
	}
	return &wishlistv1.ToggleWishlistItemResponse{Item: protoWishlistItem(item)}, nil
}

func (h *WishlistHandler) RemoveItem(ctx context.Context, req *wishlistv1.RemoveItemRequest) (*wishlistv1.RemoveItemResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.svc.RemoveItem(ctx, username, req.ProductId); err != nil {
		return nil, domainErr(err)
	}
	return &wishlistv1.RemoveItemResponse{Success: true}, nil
}

// ── Proto mapper ─────────────────────────────────────────────────────────────

func protoWishlistItem(wi *domain.WishlistItem) *wishlistv1.WishlistItem {
	return &wishlistv1.WishlistItem{
		Id:        wi.ID,
		ProductId: wi.ProductID,
		CreatedAt: safeTimestamp(wi.CreatedAt),
		Added:     wi.Added,
	}
}
