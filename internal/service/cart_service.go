package service

import (
	"context"
	"fmt"

	"github.com/nhassl3/servicehub/internal/domain"
)

type CartService struct {
	cartRepo domain.CartRepository
}

func NewCartService(cartRepo domain.CartRepository) *CartService {
	return &CartService{cartRepo: cartRepo}
}

func (s *CartService) GetCart(ctx context.Context, username string) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetOrCreate(ctx, username)
	if err != nil {
		return nil, err
	}
	items, err := s.cartRepo.GetItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	cart.Items = items
	return cart, nil
}

func (s *CartService) AddItem(ctx context.Context, username, productID string, qty int) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetOrCreate(ctx, username)
	if err != nil {
		return nil, err
	}

	if _, err := s.cartRepo.AddItem(ctx, cart.ID, productID, qty); err != nil {
		return nil, fmt.Errorf("cart_service.AddItem: %w", err)
	}

	return s.GetCart(ctx, username)
}

func (s *CartService) RemoveItem(ctx context.Context, username, productID string) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetOrCreate(ctx, username)
	if err != nil {
		return nil, err
	}

	if err := s.cartRepo.RemoveItem(ctx, cart.ID, productID); err != nil {
		return nil, fmt.Errorf("cart_service.RemoveItem: %w", err)
	}

	return s.GetCart(ctx, username)
}

func (s *CartService) UpdateItemQty(ctx context.Context, username, productID string, qty int) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetOrCreate(ctx, username)
	if err != nil {
		return nil, err
	}

	if qty <= 0 {
		if err := s.cartRepo.RemoveItem(ctx, cart.ID, productID); err != nil {
			return nil, err
		}
	} else {
		if _, err := s.cartRepo.UpdateItemQty(ctx, cart.ID, productID, qty); err != nil {
			return nil, fmt.Errorf("cart_service.UpdateItemQty: %w", err)
		}
	}

	return s.GetCart(ctx, username)
}

func (s *CartService) ClearCart(ctx context.Context, username string) error {
	cart, err := s.cartRepo.GetOrCreate(ctx, username)
	if err != nil {
		return err
	}
	return s.cartRepo.Clear(ctx, cart.ID)
}
