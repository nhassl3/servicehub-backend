package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type ReviewRepo struct {
	store *db.Store
}

func NewReviewRepo(store *db.Store) *ReviewRepo {
	return &ReviewRepo{store: store}
}

func (r *ReviewRepo) Create(ctx context.Context, params domain.CreateReviewParams) (*domain.Review, error) {
	productUID, err := parseUUID(params.ProductID)
	if err != nil {
		return nil, fmt.Errorf("review_repo.Create: invalid product_id: %w", err)
	}

	var review *domain.Review
	err = r.store.ExecTx(ctx, func(q *db.Queries) error {
		row, err := q.CreateReview(ctx, db.CreateReviewParams{
			ProductID: productUID,
			Username:  params.Username,
			Rating:    int16(params.Rating),
			Comment:   params.Comment,
		})
		if err != nil {
			return err
		}
		review = mapReview(row)

		// Recalculate and update the product rating atomically.
		avg, err := q.GetAvgRatingByProduct(ctx, productUID)
		if err != nil {
			return err
		}
		if err := q.UpdateProductRating(ctx, db.UpdateProductRatingParams{
			ID:     productUID,
			Rating: avg,
		}); err != nil {
			return err
		}

		// Update seller rating: recalculate average across all their products.
		sellerID, err := q.GetProductSellerID(ctx, productUID)
		if err != nil {
			return err
		}
		if err := q.UpdateSellerRating(ctx, sellerID); err != nil {
			return err
		}

		// Increase review count for the product.
		return q.IncreaseReviewsCount(ctx, productUID)
	})
	if err != nil {
		return nil, fmt.Errorf("review_repo.Create: %w", err)
	}
	return review, nil
}

func (r *ReviewRepo) List(ctx context.Context, params domain.ListReviewsParams) ([]domain.Review, int64, error) {
	productUID, err := parseUUID(params.ProductID)
	if err != nil {
		return nil, 0, fmt.Errorf("review_repo.List: invalid product_id: %w", err)
	}

	total, err := r.store.CountReviewsByProduct(ctx, productUID)
	if err != nil {
		return nil, 0, fmt.Errorf("review_repo.List count: %w", err)
	}

	rows, err := r.store.GetReviewsByProduct(ctx, db.GetReviewsByProductParams{
		ProductID: productUID,
		Limit:     params.Limit,
		Offset:    params.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("review_repo.List: %w", err)
	}

	reviews := make([]domain.Review, len(rows))
	for i, row := range rows {
		reviews[i] = *mapReview(row)
	}
	return reviews, total, nil
}

func (r *ReviewRepo) Delete(ctx context.Context, id int64, username string) error {
	row, err := r.store.GetReviewByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("review_repo.Delete get: %w", err)
	}
	if row.Username != username {
		return domain.ErrForbidden
	}

	affected, err := r.store.DeleteReview(ctx, id)
	if err != nil {
		return fmt.Errorf("review_repo.Delete: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}

	// Recalculate product rating after deletion.
	avg, err := r.store.GetAvgRatingByProduct(ctx, row.ProductID)
	if err == nil {
		_ = r.store.UpdateProductRating(ctx, db.UpdateProductRatingParams{
			ID:     row.ProductID,
			Rating: avg,
		})
	}
	return nil
}

func (r *ReviewRepo) ExistsByProductAndUser(ctx context.Context, productID, username string) (bool, error) {
	productUID, err := parseUUID(productID)
	if err != nil {
		return false, nil
	}
	exists, err := r.store.ReviewExistsByProductAndUser(ctx, db.ReviewExistsByProductAndUserParams{
		ProductID: productUID,
		Username:  username,
	})
	if err != nil {
		return false, fmt.Errorf("review_repo.ExistsByProductAndUser: %w", err)
	}
	return exists, nil
}

// ── Mapping ──────────────────────────────────────────────────────────────────

func mapReview(r db.Review) *domain.Review {
	return &domain.Review{
		ID:        r.ID,
		ProductID: r.ProductID.String(),
		Username:  r.Username,
		Rating:    int(r.Rating),
		Comment:   r.Comment,
		CreatedAt: pgTimeTZ(r.CreatedAt, time.UTC),
	}
}
