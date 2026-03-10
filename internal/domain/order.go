package domain

import (
	"context"
	"time"
)

const (
	OrderStatusPending   = "pending"
	OrderStatusPaid      = "paid"
	OrderStatusDelivered = "delivered"
	OrderStatusCancelled = "cancelled"
)

type Order struct {
	ID          string      `db:"id"`
	UID         string      `db:"uid"`
	Username    string      `db:"username"`
	Status      string      `db:"status"`
	TotalAmount float64     `db:"total_amount"`
	Items       []OrderItem `db:"-"`
	CreatedAt   time.Time   `db:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at"`
}

type OrderItem struct {
	ID         int64   `db:"id"`
	OrderID    string  `db:"order_id"`
	ProductID  string  `db:"product_id"`
	Quantity   int     `db:"quantity"`
	UnitPrice  float64 `db:"unit_price"`
	TotalPrice float64 `db:"total_price"`
}

type ListOrdersParams struct {
	Username string
	Status   string
	Limit    int32
	Offset   int32
}

type SellerTotalAmount struct {
	ProductId   string
	TotalAmount float64
}

//go:generate mockgen -source=order.go -destination=../repository/mock/order_repo_mock.go -package=mockrepo
type OrderRepository interface {
	Create(ctx context.Context, username string) (*Order, error)
	GetByID(ctx context.Context, id string) (*Order, error)
	GetByUID(ctx context.Context, uid string) (*Order, error)
	List(ctx context.Context, params ListOrdersParams) ([]Order, int64, error)
	UpdateStatus(ctx context.Context, id, status string) (*Order, error)
	// Checkout performs a transactional checkout: create order + items + deduct balance + increment sales
	Checkout(ctx context.Context, username string) (*Order, error)
}
