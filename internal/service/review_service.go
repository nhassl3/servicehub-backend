package service

import (
	"context"

	"github.com/nhassl3/servicehub/internal/domain"
)

type ReviewService struct {
	reviewRepo domain.ReviewRepository
}

func NewReviewService(reviewRepo domain.ReviewRepository) *ReviewService {
	return &ReviewService{reviewRepo: reviewRepo}
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
	return s.reviewRepo.Create(ctx, params)
}

func (s *ReviewService) DeleteReview(ctx context.Context, username string, id int64) error {
	return s.reviewRepo.Delete(ctx, id, username)
}
