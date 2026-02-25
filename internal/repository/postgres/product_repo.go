package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type ProductRepo struct {
	store *db.Store
}

func NewProductRepo(store *db.Store) *ProductRepo {
	return &ProductRepo{store: store}
}

func (r *ProductRepo) Create(ctx context.Context, params domain.CreateProductParams) (*domain.Product, error) {
	sellerUID, err := parseUUID(params.SellerID)
	if err != nil {
		return nil, fmt.Errorf("product_repo.Create: invalid seller_id: %w", err)
	}
	row, err := r.store.CreateProduct(ctx, db.CreateProductParams{
		SellerID:    sellerUID,
		CategoryID:  int32(params.CategoryID),
		Title:       params.Title,
		Description: params.Description,
		Price:       params.Price,
		Tags:        params.Tags,
	})
	if err != nil {
		return nil, fmt.Errorf("product_repo.Create: %w", err)
	}
	return mapProduct(row.ID, row.SellerID, row.CategoryID, row.Title, row.Description,
		row.Price, row.Tags, row.Status, row.SalesCount, row.Rating, row.ReviewsCount,
		row.CreatedAt, row.UpdatedAt), nil
}

func (r *ProductRepo) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	row, err := r.store.GetProductByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("product_repo.GetByID: %w", err)
	}
	return mapProduct(row.ID, row.SellerID, row.CategoryID, row.Title, row.Description,
		row.Price, row.Tags, row.Status, row.SalesCount, row.Rating, row.ReviewsCount,
		row.CreatedAt, row.UpdatedAt), nil
}

func (r *ProductRepo) List(ctx context.Context, params domain.ListProductsParams) ([]domain.Product, int64, error) {
	status := params.Status
	if status == "" {
		status = "active"
	}

	countParams := db.CountListProductsParams{
		SellerID:   uuidPtrToNullable(params.SellerID),
		CategoryID: intPtrToNullable(params.CategoryID),
		MinPrice:   float64PtrToNullable(params.MinPrice),
		MaxPrice:   float64PtrToNullable(params.MaxPrice),
		Status:     status,
	}
	total, err := r.store.CountListProducts(ctx, countParams)
	if err != nil {
		return nil, 0, fmt.Errorf("product_repo.List count: %w", err)
	}

	rows, err := r.store.ListProducts(ctx, db.ListProductsParams{
		SellerID:   countParams.SellerID,
		CategoryID: countParams.CategoryID,
		MinPrice:   countParams.MinPrice,
		MaxPrice:   countParams.MaxPrice,
		Status:     status,
		Limit:      params.Limit,
		Offset:     params.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("product_repo.List: %w", err)
	}

	products := make([]domain.Product, len(rows))
	for i, row := range rows {
		products[i] = *mapProduct(row.ID, row.SellerID, row.CategoryID, row.Title, row.Description,
			row.Price, row.Tags, row.Status, row.SalesCount, row.Rating, row.ReviewsCount,
			row.CreatedAt, row.UpdatedAt)
	}
	return products, total, nil
}

func (r *ProductRepo) Search(ctx context.Context, params domain.SearchProductsParams) ([]domain.Product, int64, error) {
	total, err := r.store.CountSearchProducts(ctx, params.Query)
	if err != nil {
		return nil, 0, fmt.Errorf("product_repo.Search count: %w", err)
	}

	rows, err := r.store.SearchProducts(ctx, db.SearchProductsParams{
		Query:  params.Query,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("product_repo.Search: %w", err)
	}

	products := make([]domain.Product, len(rows))
	for i, row := range rows {
		products[i] = *mapProduct(row.ID, row.SellerID, row.CategoryID, row.Title, row.Description,
			row.Price, row.Tags, row.Status, row.SalesCount, row.Rating, row.ReviewsCount,
			row.CreatedAt, row.UpdatedAt)
	}
	return products, total, nil
}

func (r *ProductRepo) Update(ctx context.Context, params domain.UpdateProductParams) (*domain.Product, error) {
	uid, err := parseUUID(params.ID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	row, err := r.store.UpdateProduct(ctx, db.UpdateProductParams{
		ID:          uid,
		Title:       params.Title,
		Description: params.Description,
		Price:       params.Price,
		Tags:        params.Tags,
		Status:      params.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("product_repo.Update: %w", err)
	}
	return mapProduct(row.ID, row.SellerID, row.CategoryID, row.Title, row.Description,
		row.Price, row.Tags, row.Status, row.SalesCount, row.Rating, row.ReviewsCount,
		row.CreatedAt, row.UpdatedAt), nil
}

func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	uid, err := parseUUID(id)
	if err != nil {
		return domain.ErrNotFound
	}
	if err := r.store.DeleteProduct(ctx, uid); err != nil {
		return fmt.Errorf("product_repo.Delete: %w", err)
	}
	return nil
}

func (r *ProductRepo) IncrementSalesCount(ctx context.Context, id string, qty int) error {
	uid, err := parseUUID(id)
	if err != nil {
		return fmt.Errorf("product_repo.IncrementSalesCount: invalid id: %w", err)
	}
	return r.store.IncrementProductSales(ctx, db.IncrementProductSalesParams{
		ID:         uid,
		SalesCount: int32(qty),
	})
}

func (r *ProductRepo) UpdateRating(ctx context.Context, id string, newRating float64) error {
	uid, err := parseUUID(id)
	if err != nil {
		return fmt.Errorf("product_repo.UpdateRating: invalid id: %w", err)
	}
	return r.store.UpdateProductRating(ctx, db.UpdateProductRatingParams{
		ID:     uid,
		Rating: newRating,
	})
}

// ── Mapping ──────────────────────────────────────────────────────────────────

// mapProduct maps raw product fields (common across all SQLC product row types)
// to a domain.Product value.
func mapProduct(
	id, sellerID uuid.UUID,
	categoryID int32,
	title, description string,
	price float64,
	tags []string,
	status string,
	salesCount int32,
	rating float64,
	reviewsCount int32,
	createdAt, updatedAt pgtype.Timestamptz,
) *domain.Product {
	return &domain.Product{
		ID:           id.String(),
		SellerID:     sellerID.String(),
		CategoryID:   int(categoryID),
		Title:        title,
		Description:  description,
		Price:        price,
		Tags:         tags,
		Status:       status,
		SalesCount:   int(salesCount),
		Rating:       rating,
		ReviewsCount: int(reviewsCount),
		CreatedAt:    pgTimeTZ(createdAt, time.UTC),
		UpdatedAt:    pgTimeTZ(updatedAt, time.UTC),
	}
}
