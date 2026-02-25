package domain

import "context"

type Cart struct {
	ID       int64      `db:"id"`
	Username string     `db:"username"`
	Items    []CartItem `db:"-"`
}

type CartItem struct {
	ID        int64   `db:"id"`
	CartID    int64   `db:"cart_id"`
	ProductID string  `db:"product_id"`
	Quantity  int     `db:"quantity"`
	UnitPrice float64 `db:"unit_price"`
}

//go:generate mockgen -source=cart.go -destination=../repository/mock/cart_repo_mock.go -package=mockrepo
type CartRepository interface {
	GetOrCreate(ctx context.Context, username string) (*Cart, error)
	GetItems(ctx context.Context, cartID int64) ([]CartItem, error)
	AddItem(ctx context.Context, cartID int64, productID string, qty int) (*CartItem, error)
	UpdateItemQty(ctx context.Context, cartID int64, productID string, qty int) (*CartItem, error)
	RemoveItem(ctx context.Context, cartID int64, productID string) error
	Clear(ctx context.Context, cartID int64) error
}
