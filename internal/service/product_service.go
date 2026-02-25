package service

import (
	"context"
	"fmt"

	"github.com/nhassl3/servicehub/internal/domain"
)

type ProductService struct {
	productRepo domain.ProductRepository
	sellerRepo  domain.SellerRepository
}

func NewProductService(productRepo domain.ProductRepository, sellerRepo domain.SellerRepository) *ProductService {
	return &ProductService{productRepo: productRepo, sellerRepo: sellerRepo}
}

func (s *ProductService) ListProducts(ctx context.Context, params domain.ListProductsParams) ([]domain.Product, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	return s.productRepo.List(ctx, params)
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	return s.productRepo.GetByID(ctx, id)
}

func (s *ProductService) SearchProducts(ctx context.Context, params domain.SearchProductsParams) ([]domain.Product, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	return s.productRepo.Search(ctx, params)
}

func (s *ProductService) CreateProduct(ctx context.Context, username string, params domain.CreateProductParams) (*domain.Product, error) {
	seller, err := s.sellerRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, domain.ErrForbidden
	}
	params.SellerID = seller.ID
	p, err := s.productRepo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("product_service.Create: %w", err)
	}
	return p, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, username string, params domain.UpdateProductParams) (*domain.Product, error) {
	existing, err := s.productRepo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	seller, err := s.sellerRepo.GetByUsername(ctx, username)
	if err != nil || seller.ID != existing.SellerID {
		return nil, domain.ErrForbidden
	}

	return s.productRepo.Update(ctx, params)
}

func (s *ProductService) DeleteProduct(ctx context.Context, username, id string) error {
	existing, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	seller, err := s.sellerRepo.GetByUsername(ctx, username)
	if err != nil || seller.ID != existing.SellerID {
		return domain.ErrForbidden
	}

	return s.productRepo.Delete(ctx, id)
}
