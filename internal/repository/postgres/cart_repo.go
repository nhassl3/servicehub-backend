package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type CartRepo struct {
	store *db.Store
}

func NewCartRepo(store *db.Store) *CartRepo {
	return &CartRepo{store: store}
}

func (r *CartRepo) GetOrCreate(ctx context.Context, username string) (*domain.Cart, error) {
	row, err := r.store.UpsertCart(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("cart_repo.GetOrCreate: %w", err)
	}
	return &domain.Cart{ID: row.ID, Username: row.Username}, nil
}

func (r *CartRepo) GetItems(ctx context.Context, cartID int64) ([]domain.CartItem, error) {
	rows, err := r.store.GetCartItems(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("cart_repo.GetItems: %w", err)
	}
	items := make([]domain.CartItem, len(rows))
	for i, row := range rows {
		items[i] = domain.CartItem{
			ID:        row.ID,
			CartID:    row.CartID,
			ProductID: row.ProductID.String(),
			Quantity:  int(row.Quantity),
			UnitPrice: row.UnitPrice,
		}
	}
	return items, nil
}

func (r *CartRepo) AddItem(ctx context.Context, cartID int64, productID string, qty int) (*domain.CartItem, error) {
	uid, err := parseUUID(productID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	price, err := r.store.GetProductPrice(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("cart_repo.AddItem get price: %w", err)
	}

	row, err := r.store.UpsertCartItem(ctx, db.UpsertCartItemParams{
		CartID:    cartID,
		ProductID: uid,
		Quantity:  int32(qty),
		UnitPrice: price,
	})
	if err != nil {
		return nil, fmt.Errorf("cart_repo.AddItem: %w", err)
	}
	return &domain.CartItem{
		ID:        row.ID,
		CartID:    row.CartID,
		ProductID: row.ProductID.String(),
		Quantity:  int(row.Quantity),
		UnitPrice: row.UnitPrice,
	}, nil
}

func (r *CartRepo) UpdateItemQty(ctx context.Context, cartID int64, productID string, qty int) (*domain.CartItem, error) {
	uid, err := parseUUID(productID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	row, err := r.store.UpdateCartItemQty(ctx, db.UpdateCartItemQtyParams{
		CartID:    cartID,
		ProductID: uid,
		Quantity:  int32(qty),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("cart_repo.UpdateItemQty: %w", err)
	}
	return &domain.CartItem{
		ID:        row.ID,
		CartID:    row.CartID,
		ProductID: row.ProductID.String(),
		Quantity:  int(row.Quantity),
		UnitPrice: row.UnitPrice,
	}, nil
}

func (r *CartRepo) RemoveItem(ctx context.Context, cartID int64, productID string) error {
	uid, err := parseUUID(productID)
	if err != nil {
		return domain.ErrNotFound
	}
	affected, err := r.store.DeleteCartItem(ctx, db.DeleteCartItemParams{
		CartID:    cartID,
		ProductID: uid,
	})
	if err != nil {
		return fmt.Errorf("cart_repo.RemoveItem: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CartRepo) Clear(ctx context.Context, cartID int64) error {
	if err := r.store.ClearCart(ctx, cartID); err != nil {
		return fmt.Errorf("cart_repo.Clear: %w", err)
	}
	return nil
}
