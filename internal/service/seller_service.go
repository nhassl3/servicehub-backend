package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/nhassl3/servicehub/internal/domain"
)

type SellerService struct {
	sellerRepo domain.SellerRepository
}

func NewSellerService(sellerRepo domain.SellerRepository) *SellerService {
	return &SellerService{sellerRepo: sellerRepo}
}

func (s *SellerService) CreateSeller(ctx context.Context, params domain.CreateSellerParams) (*domain.Seller, error) {
	exists, err := s.sellerRepo.ExistsByUsername(ctx, params.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrAlreadyExists
	}
	return s.sellerRepo.Create(ctx, params)
}

func (s *SellerService) GetSellerProfile(ctx context.Context, params domain.GetSellerProfileParams) (*domain.Seller, error) {
	if params.SellerId == nil && params.Username == nil {
		return nil, domain.ErrInvalidInput
	}
	if params.SellerId != nil {
		if _, err := uuid.Parse(*params.SellerId); err != nil {
			return nil, domain.ErrInvalidInput
		}
	}
	return s.sellerRepo.GetSeller(ctx, params)
}

func (s *SellerService) UpdateSeller(ctx context.Context, params domain.UpdateSellerParams) (*domain.Seller, error) {
	return s.sellerRepo.Update(ctx, params)
}
