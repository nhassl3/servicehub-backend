package domain

import (
	"context"
	"time"
)

type WishlistItem struct {
	ID        int64     `db:"id"`
	Username  string    `db:"username"`
	ProductID string    `db:"product_id"`
	CreatedAt time.Time `db:"created_at"`
}

//go:generate mockgen -source=wishlist.go -destination=../repository/mock/wishlist_repo_mock.go -package=mockrepo
type WishlistRepository interface {
	GetItems(ctx context.Context, username string) ([]WishlistItem, error)
	AddItem(ctx context.Context, username, productID string) (*WishlistItem, error)
	RemoveItem(ctx context.Context, username, productID string) error
	Exists(ctx context.Context, username, productID string) (bool, error)
}
