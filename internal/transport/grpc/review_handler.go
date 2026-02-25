package grpc

import (
	"context"

	reviewv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/review/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
)

// ReviewHandler implements reviewv1.ReviewServiceServer.
//
// Implemented RPC methods:
//   - ListReviews
//   - CreateReview
//   - DeleteReview
type ReviewHandler struct {
	reviewv1.UnimplementedReviewServiceServer
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) ListReviews(ctx context.Context, req *reviewv1.ListReviewsRequest) (*reviewv1.ListReviewsResponse, error) {
	reviews, total, err := h.svc.ListReviews(ctx, domain.ListReviewsParams{
		ProductID: req.ProductId,
		Limit:     req.Limit,
		Offset:    req.Offset,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	proto := make([]*reviewv1.Review, len(reviews))
	for i, r := range reviews {
		proto[i] = protoReview(&r)
	}
	return &reviewv1.ListReviewsResponse{Reviews: proto, Total: total}, nil
}

func (h *ReviewHandler) CreateReview(ctx context.Context, req *reviewv1.CreateReviewRequest) (*reviewv1.CreateReviewResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	review, err := h.svc.CreateReview(ctx, domain.CreateReviewParams{
		ProductID: req.ProductId,
		Username:  username,
		Rating:    int(req.Rating),
		Comment:   req.Comment,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &reviewv1.CreateReviewResponse{Review: protoReview(review)}, nil
}

func (h *ReviewHandler) DeleteReview(ctx context.Context, req *reviewv1.DeleteReviewRequest) (*reviewv1.DeleteReviewResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.svc.DeleteReview(ctx, username, req.Id); err != nil {
		return nil, domainErr(err)
	}
	return &reviewv1.DeleteReviewResponse{Success: true}, nil
}

// ── Proto mapper ─────────────────────────────────────────────────────────────

func protoReview(r *domain.Review) *reviewv1.Review {
	return &reviewv1.Review{
		Id:        r.ID,
		ProductId: r.ProductID,
		Username:  r.Username,
		Rating:    int32(r.Rating),
		Comment:   r.Comment,
		CreatedAt: safeTimestamp(r.CreatedAt),
	}
}
