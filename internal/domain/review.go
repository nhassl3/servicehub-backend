package domain

import (
	"context"
	"time"
)

type Review struct {
	ID        int64     `db:"id"`
	ProductID string    `db:"product_id"`
	Username  string    `db:"username"`
	Rating    int       `db:"rating"`
	Comment   string    `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
}

type ListReviewsParams struct {
	ProductID string
	Limit     int32
	Offset    int32
}

type CreateReviewParams struct {
	ProductID string
	Username  string
	Rating    int
	Comment   string
}

//go:generate mockgen -source=review.go -destination=../repository/mock/review_repo_mock.go -package=mockrepo
type ReviewRepository interface {
	Create(ctx context.Context, params CreateReviewParams) (*Review, error)
	List(ctx context.Context, params ListReviewsParams) ([]Review, int64, error)
	Delete(ctx context.Context, id int64, username string) error
	ExistsByProductAndUser(ctx context.Context, productID, username string) (bool, error)
}
