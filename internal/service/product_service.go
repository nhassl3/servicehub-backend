package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"go.uber.org/zap"
)

type ProductService struct {
	productRepo    domain.ProductRepository
	searchRepo     domain.ProductSearchRepository
	eventPublisher domain.EventPublisher
	sellerRepo     domain.SellerRepository
	log            *zap.Logger
}

func NewProductService(
	productRepo domain.ProductRepository,
	searchRepo domain.ProductSearchRepository,
	sellerRepo domain.SellerRepository,
	eventPublisher domain.EventPublisher,
	log *zap.Logger,
) *ProductService {
	return &ProductService{
		productRepo:    productRepo,
		searchRepo:     searchRepo,
		sellerRepo:     sellerRepo,
		eventPublisher: eventPublisher,
		log:            log,
	}
}

func (s *ProductService) ListProducts(ctx context.Context, params domain.ListProductsParams) ([]*domain.Product, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	} else if params.Offset < 0 {
		params.Offset = 0
	}
	return s.productRepo.List(ctx, params)
}

func (s *ProductService) GetProduct(ctx context.Context, id string) (*domain.Product, error) {
	return s.productRepo.GetByID(ctx, id)
}

func (s *ProductService) SearchProducts(ctx context.Context, params domain.SearchProductsParams) ([]*domain.Product, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.SortBy == "" {
		params.SortBy = "relevance"
	}

	products, total, err := s.searchRepo.Search(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("product_service.SearchProducts: failed to load products from Elasticsearch: %w", err)
	}
	if total != 0 {
		return products, total, nil
	}

	products, total, err = s.productRepo.Search(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("product_service.SearchProducts: failed to load products from PG: %w", err)
	}
	if total == 0 {
		return []*domain.Product{}, 0, nil
	}

	return products, total, nil
}

func (s *ProductService) CreateProduct(ctx context.Context, username string, params domain.CreateProductParams) (*domain.Product, error) {
	seller, err := s.sellerRepo.GetSeller(ctx, domain.GetSellerProfileParams{
		Username: &username,
	})
	if err != nil {
		return nil, domain.ErrForbidden
	}
	params.SellerID = seller.ID
	p, err := s.productRepo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("product_service.Create: %w", err)
	}

	{
		if err = s.eventPublisher.PublishIndexedProduct(ctx, p); err != nil {
			s.log.Warn("(Kafka) elasticsearch: failed to index product", zap.Error(err))
		}
	}

	return p, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, username string, params domain.UpdateProductParams) (*domain.Product, error) {
	existing, err := s.productRepo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	seller, err := s.sellerRepo.GetSeller(ctx, domain.GetSellerProfileParams{
		Username: &username,
	})
	if err != nil || seller.ID != existing.SellerID {
		return nil, domain.ErrForbidden
	}

	p, err := s.productRepo.Update(ctx, params)
	if err != nil {
		return nil, err
	}

	if existing.Status != p.Status {
		if err = s.eventPublisher.PublishProductStatusChanged(ctx, domain.ProductStatusChangedPayload{
			ID:           p.ID,
			SellerID:     p.SellerID,
			CategoryID:   p.CategoryID,
			Title:        p.Title,
			Status:       p.Status,
			Rating:       p.Rating,
			SalesCount:   p.SalesCount,
			ReviewsCount: p.ReviewsCount,
			OccurredAt:   time.Now(),
		}); err != nil {
			s.log.Warn("(Kafka) analytics: failed to publish product status change", zap.Error(err))
		}
	}

	{
		if err = s.eventPublisher.PublishIndexedProduct(ctx, p); err != nil {
			s.log.Warn("(Kafka) elasticsearch: failed to index product", zap.Error(err))
		}
	}

	return p, nil
}

func (s *ProductService) DeleteProduct(ctx context.Context, username, id string) error {
	existing, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	seller, err := s.sellerRepo.GetSeller(ctx, domain.GetSellerProfileParams{
		Username: &username,
	})
	if err != nil || seller.ID != existing.SellerID {
		return domain.ErrForbidden
	}

	if err = s.productRepo.Delete(ctx, id); err != nil {
		return err
	}

	{
		if err = s.eventPublisher.PublishDeletedProduct(ctx, id); err != nil {
			s.log.Warn("(Kafka) elasticsearch: failed to delete product index", zap.Error(err))
		}
	}

	return nil
}

func (s *ProductService) ReindexAllProducts(ctx context.Context) error {
	var offset int32
	const batchSize = 100

	for {
		products, _, err := s.productRepo.List(ctx, domain.ListProductsParams{
			Limit:  batchSize,
			Offset: offset,
		})
		if err != nil {
			return fmt.Errorf("product_service.ReindexAll: list products: %w", err)
		}

		if len(products) == 0 {
			break
		}

		if err := s.searchRepo.BulkIndexProducts(ctx, products); err != nil {
			return fmt.Errorf("product_service.ReindexAll: bulk index: %w", err)
		}

		s.log.Info("elasticsearch: reindex batch",
			zap.Int("count", len(products)),
			zap.Int32("offset", offset),
		)

		offset += batchSize
	}

	s.log.Info("elasticsearch: reindex complete")
	return nil
}
