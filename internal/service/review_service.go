package service

import (
	"context"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"go.uber.org/zap"
)

type ReviewService struct {
	reviewRepo     domain.ReviewRepository
	productRepo    domain.ProductRepository
	eventPublisher domain.EventPublisher
	log            *zap.Logger
}

func NewReviewService(
	reviewRepo domain.ReviewRepository,
	productRepo domain.ProductRepository,
	eventPublisher domain.EventPublisher,
	log *zap.Logger,
) *ReviewService {
	return &ReviewService{
		reviewRepo:     reviewRepo,
		productRepo:    productRepo,
		eventPublisher: eventPublisher,
		log:            log,
	}
}

func (s *ReviewService) ListReviews(ctx context.Context, params domain.ListReviewsParams) ([]domain.Review, int64, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}
	return s.reviewRepo.List(ctx, params)
}

func (s *ReviewService) CreateReview(ctx context.Context, params domain.CreateReviewParams) (*domain.Review, error) {
	exists, err := s.reviewRepo.ExistsByProductAndUser(ctx, params.ProductID, params.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrAlreadyExists
	}
	review, err := s.reviewRepo.Create(ctx, params)
	if err != nil {
		return nil, err
	}
	s.publishRatingChanged(ctx, params.ProductID)
	return review, nil
}

func (s *ReviewService) DeleteReview(ctx context.Context, username string, id int64) error {
	return s.reviewRepo.Delete(ctx, id, username)
}

// publishRatingChanged emits a best-effort product_rating_changed fact after a
// review mutation so the analytics layer can track rating trends. Postgres is
// the source of truth — a failed publication only logs a warning.
func (s *ReviewService) publishRatingChanged(ctx context.Context, productID string) {
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		s.log.Warn("review_service: failed to load product for analytics event", zap.Error(err))
		return
	}
	if err := s.eventPublisher.PublishProductRatingChanged(ctx, domain.ProductRatingChangedPayload{
		ID:           product.ID,
		CategoryID:   product.CategoryID,
		Title:        product.Title,
		Rating:       product.Rating,
		ReviewsCount: product.ReviewsCount,
		OccurredAt:   time.Now(),
	}); err != nil {
		s.log.Warn("(Kafka) analytics: failed to publish product rating change", zap.Error(err))
	}
}
