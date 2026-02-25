package service

import (
	"context"

	"github.com/nhassl3/servicehub/internal/domain"
)

type WishlistService struct {
	repo domain.WishlistRepository
}

func NewWishlistService(repo domain.WishlistRepository) *WishlistService {
	return &WishlistService{repo: repo}
}

func (s *WishlistService) GetWishlist(ctx context.Context, username string) ([]domain.WishlistItem, error) {
	return s.repo.GetItems(ctx, username)
}

func (s *WishlistService) AddItem(ctx context.Context, username, productID string) (*domain.WishlistItem, error) {
	return s.repo.AddItem(ctx, username, productID)
}

func (s *WishlistService) RemoveItem(ctx context.Context, username, productID string) error {
	return s.repo.RemoveItem(ctx, username, productID)
}
