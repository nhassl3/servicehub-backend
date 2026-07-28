package domain

import (
	"context"
	"time"
)

type Product struct {
	ID           string    `json:"id"`
	SellerID     string    `json:"seller_id"`
	CategoryID   int       `json:"category_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Price        float64   `json:"price"`
	Tags         []string  `json:"tags"`
	Status       string    `json:"status"`
	SalesCount   int       `json:"sales_count"`
	Rating       float64   `json:"rating"`
	ReviewsCount int       `json:"reviews_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ListProductsParams struct {
	CategoryID *int
	SellerID   *string
	AdminId    *string
	MinPrice   *float64
	MaxPrice   *float64
	Status     string
	Limit      int32
	Offset     int32
}

type SearchProductsParams struct {
	Query      string
	CategoryID *int
	MinPrice   *float64
	MaxPrice   *float64
	Tags       []string
	SortBy     string // relevance | "price_asc" | "price_desc" | "rating" | "date" | "sales"
	Limit      int32
	Offset     int32
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
	Title       *string
	Description *string
	Price       *float64
	Tags        []string
	Status      *string
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

type ProductSearchRepository interface {
	IndexProduct(ctx context.Context, product *Product) error
	DeleteProductIndex(ctx context.Context, id string) error
	BulkIndexProducts(ctx context.Context, products []*Product) error
	Search(ctx context.Context, params SearchProductsParams) ([]Product, int64, error)
}
