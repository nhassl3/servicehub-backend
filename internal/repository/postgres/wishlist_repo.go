package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type WishlistRepo struct {
	store *db.Store
}

func NewWishlistRepo(store *db.Store) *WishlistRepo {
	return &WishlistRepo{store: store}
}

func (r *WishlistRepo) GetItems(ctx context.Context, username string) ([]domain.WishlistItem, error) {
	rows, err := r.store.GetWishlistItems(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("wishlist_repo.GetItems: %w", err)
	}
	items := make([]domain.WishlistItem, len(rows))
	for i, row := range rows {
		items[i] = domain.WishlistItem{
			ID:        row.ID,
			Username:  row.Username,
			ProductID: row.ProductID.String(),
			CreatedAt: pgTimeTZ(row.CreatedAt, time.UTC),
		}
	}
	return items, nil
}

func (r *WishlistRepo) AddItem(ctx context.Context, username, productID string) (*domain.WishlistItem, error) {
	uid, err := parseUUID(productID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	row, err := r.store.AddWishlistItem(ctx, db.AddWishlistItemParams{
		Username:  username,
		ProductID: uid,
	})
	if err != nil {
		return nil, fmt.Errorf("wishlist_repo.AddItem: %w", err)
	}
	return &domain.WishlistItem{
		ID:        row.ID,
		Username:  row.Username,
		ProductID: row.ProductID.String(),
		CreatedAt: pgTimeTZ(row.CreatedAt, time.UTC),
	}, nil
}

func (r *WishlistRepo) RemoveItem(ctx context.Context, username, productID string) error {
	uid, err := parseUUID(productID)
	if err != nil {
		return domain.ErrNotFound
	}
	affected, err := r.store.RemoveWishlistItem(ctx, db.RemoveWishlistItemParams{
		Username:  username,
		ProductID: uid,
	})
	if err != nil {
		return fmt.Errorf("wishlist_repo.RemoveItem: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *WishlistRepo) Exists(ctx context.Context, username, productID string) (bool, error) {
	uid, err := parseUUID(productID)
	if err != nil {
		return false, nil
	}
	exists, err := r.store.WishlistItemExists(ctx, db.WishlistItemExistsParams{
		Username:  username,
		ProductID: uid,
	})
	if err != nil {
		return false, fmt.Errorf("wishlist_repo.Exists: %w", err)
	}
	return exists, nil
}
