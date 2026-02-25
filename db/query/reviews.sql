-- name: CreateReview :one
INSERT INTO reviews (product_id, username, rating, comment)
VALUES ($1, $2, $3, $4)
RETURNING id, product_id, username, rating, comment, created_at;

-- name: GetReviewsByProduct :many
SELECT id, product_id, username, rating, comment, created_at
FROM reviews
WHERE product_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountReviewsByProduct :one
SELECT COUNT(*) FROM reviews WHERE product_id = $1;

-- name: DeleteReview :execrows
DELETE FROM reviews WHERE id = $1;

-- name: ReviewExistsByProductAndUser :one
SELECT EXISTS(SELECT 1 FROM reviews WHERE product_id = $1 AND username = $2);

-- name: GetAvgRatingByProduct :one
SELECT COALESCE(AVG(rating::float), 0)::float AS avg_rating
FROM reviews
WHERE product_id = $1;

-- name: GetReviewByID :one
SELECT id, product_id, username, rating, comment, created_at
FROM reviews
WHERE id = $1;
