package domain

import (
	"context"
	"time"
)

type Product struct {
	ID           string    `db:"id"`
	SellerID     string    `db:"seller_id"`
	CategoryID   int       `db:"category_id"`
	Title        string    `db:"title"`
	Description  string    `db:"description"`
	Price        float64   `db:"price"`
	Tags         []string  `db:"tags"`
	Status       string    `db:"status"`
	SalesCount   int       `db:"sales_count"`
	Rating       float64   `db:"rating"`
	ReviewsCount int       `db:"reviews_count"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type ListProductsParams struct {
	CategoryID *int
	SellerID   *string
	MinPrice   *float64
	MaxPrice   *float64
	Status     string
	Limit      int32
	Offset     int32
}

type SearchProductsParams struct {
	Query  string
	Limit  int32
	Offset int32
}

type CreateProductParams struct {
	SellerID    string
	CategoryID  int
	Title       string
	Description string
	Price       float64
	Tags        []string
}

type UpdateProductParams struct {
	ID          string
	Title       string
	Description string
	Price       float64
	Tags        []string
	Status      string
}

//go:generate mockgen -source=product.go -destination=../repository/mock/product_repo_mock.go -package=mockrepo
type ProductRepository interface {
	Create(ctx context.Context, params CreateProductParams) (*Product, error)
	GetByID(ctx context.Context, id string) (*Product, error)
	List(ctx context.Context, params ListProductsParams) ([]Product, int64, error)
	Search(ctx context.Context, params SearchProductsParams) ([]Product, int64, error)
	Update(ctx context.Context, params UpdateProductParams) (*Product, error)
	Delete(ctx context.Context, id string) error
	IncrementSalesCount(ctx context.Context, id string, qty int) error
	UpdateRating(ctx context.Context, id string, newRating float64) error
}
