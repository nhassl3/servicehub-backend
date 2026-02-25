package grpc

import (
	"context"

	sellerv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/seller/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
)

// SellerHandler implements sellerv1.SellerServiceServer.
//
// Implemented RPC methods:
//   - CreateSeller
//   - GetSellerProfile
//   - UpdateSeller
type SellerHandler struct {
	sellerv1.UnimplementedSellerServiceServer
	svc *service.SellerService
}

func NewSellerHandler(svc *service.SellerService) *SellerHandler {
	return &SellerHandler{svc: svc}
}

func (h *SellerHandler) CreateSeller(ctx context.Context, req *sellerv1.CreateSellerRequest) (*sellerv1.CreateSellerResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	seller, err := h.svc.CreateSeller(ctx, domain.CreateSellerParams{
		Username:    username,
		DisplayName: req.DisplayName,
		Description: req.Description,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &sellerv1.CreateSellerResponse{Seller: protoSeller(seller)}, nil
}

func (h *SellerHandler) GetSellerProfile(ctx context.Context, req *sellerv1.GetSellerProfileRequest) (*sellerv1.GetSellerProfileResponse, error) {
	seller, err := h.svc.GetSellerProfile(ctx, req.Username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &sellerv1.GetSellerProfileResponse{Seller: protoSeller(seller)}, nil
}

func (h *SellerHandler) UpdateSeller(ctx context.Context, req *sellerv1.UpdateSellerRequest) (*sellerv1.UpdateSellerResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	seller, err := h.svc.UpdateSeller(ctx, domain.UpdateSellerParams{
		Username:    username,
		DisplayName: req.DisplayName,
		Description: req.Description,
		AvatarURL:   req.AvatarUrl,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &sellerv1.UpdateSellerResponse{Seller: protoSeller(seller)}, nil
}

// ── Proto mapper ─────────────────────────────────────────────────────────────

func protoSeller(s *domain.Seller) *sellerv1.SellerProfile {
	return &sellerv1.SellerProfile{
		Id:          s.ID,
		Username:    s.Username,
		DisplayName: s.DisplayName,
		Description: s.Description,
		AvatarUrl:   s.AvatarURL,
		Rating:      s.Rating,
		TotalSales:  int32(s.TotalSales),
		CreatedAt:   safeTimestamp(s.CreatedAt),
		UpdatedAt:   safeTimestamp(s.UpdatedAt),
	}
}
