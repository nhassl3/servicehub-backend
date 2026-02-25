package domain

import (
	"context"
	"time"
)

type Seller struct {
	ID          string    `db:"id"`
	Username    string    `db:"username"`
	DisplayName string    `db:"display_name"`
	Description string    `db:"description"`
	AvatarURL   string    `db:"avatar_url"`
	Rating      float64   `db:"rating"`
	TotalSales  int       `db:"total_sales"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type CreateSellerParams struct {
	Username    string
	DisplayName string
	Description string
}

type UpdateSellerParams struct {
	Username    string
	DisplayName string
	Description string
	AvatarURL   string
}

//go:generate mockgen -source=seller.go -destination=../repository/mock/seller_repo_mock.go -package=mockrepo
type SellerRepository interface {
	Create(ctx context.Context, params CreateSellerParams) (*Seller, error)
	GetByUsername(ctx context.Context, username string) (*Seller, error)
	Update(ctx context.Context, params UpdateSellerParams) (*Seller, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}
