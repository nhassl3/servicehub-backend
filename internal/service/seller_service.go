package service

import (
	"context"

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

func (s *SellerService) GetSellerProfile(ctx context.Context, username string) (*domain.Seller, error) {
	return s.sellerRepo.GetByUsername(ctx, username)
}

func (s *SellerService) UpdateSeller(ctx context.Context, params domain.UpdateSellerParams) (*domain.Seller, error) {
	return s.sellerRepo.Update(ctx, params)
}
