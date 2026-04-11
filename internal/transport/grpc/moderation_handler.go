package grpc

import (
	"context"

	moderationv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/moderation/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
)

const (
	draftStatus = "draft"
)

// ModerationHandler implements moderationv1.ModerationServiceServer.
//
// Implemented RPC methods:
//   - Queue
//   - MyReviews
//   - Claim
//   - Release
//   - Approve
//   - Reject
//   - Stats
//
// Legacy CRUD methods (CreateModeration / GetModeration / UpdateModeration)
// are intentionally not implemented — the embedded Unimplemented server
// returns Unimplemented for them. They are scheduled for removal from the
// proto contract once the frontend stops referencing them.
type ModerationHandler struct {
	moderationv1.UnimplementedModerationServiceServer
	svc *service.ModerationService
}

func NewModerationHandler(svc *service.ModerationService) *ModerationHandler {
	return &ModerationHandler{svc: svc}
}

func (h *ModerationHandler) Queue(ctx context.Context, req *moderationv1.QueueRequest) (*moderationv1.QueueResponse, error) {
	if _, err := mustUsername(ctx); err != nil {
		return nil, err
	}
	rows, total, err := h.svc.Queue(ctx, domain.ListParams{
		Limit:  req.GetLimit(),
		Offset: req.GetOffset(),
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &moderationv1.QueueResponse{
		Products: protoQueueProducts(rows),
		Total:    total,
	}, nil
}

func (h *ModerationHandler) MyReviews(ctx context.Context, req *moderationv1.MyReviewsRequest) (*moderationv1.MyReviewsResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	rows, total, err := h.svc.MyReviews(ctx, username, domain.ListParams{
		Limit:  req.GetLimit(),
		Offset: req.GetOffset(),
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &moderationv1.MyReviewsResponse{
		Products: protoQueueProducts(rows),
		Total:    total,
	}, nil
}

func (h *ModerationHandler) Claim(ctx context.Context, req *moderationv1.ClaimRequest) (*moderationv1.ClaimResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	moderation, err := h.svc.Claim(ctx, req.GetProductId(), username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &moderationv1.ClaimResponse{
		Moderation: protoModeration(moderation, username),
	}, nil
}

func (h *ModerationHandler) Release(ctx context.Context, req *moderationv1.ReleaseRequest) (*moderationv1.ReleaseResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Release(ctx, req.GetProductId(), draftStatus, username); err != nil {
		return nil, domainErr(err)
	}
	return &moderationv1.ReleaseResponse{}, nil
}

func (h *ModerationHandler) Approve(ctx context.Context, req *moderationv1.ApproveRequest) (*moderationv1.ApproveResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	product, err := h.svc.Approve(ctx, req.GetProductId(), username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &moderationv1.ApproveResponse{Product: ProtoProduct(product)}, nil
}

func (h *ModerationHandler) Reject(ctx context.Context, req *moderationv1.RejectRequest) (*moderationv1.RejectResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	product, err := h.svc.Reject(ctx, req.GetProductId(), username, req.GetReason())
	if err != nil {
		return nil, domainErr(err)
	}
	return &moderationv1.RejectResponse{Product: ProtoProduct(product)}, nil
}

func (h *ModerationHandler) Stats(ctx context.Context, _ *moderationv1.StatsRequest) (*moderationv1.StatsResponse, error) {
	username, err := mustUsername(ctx)
	if err != nil {
		return nil, err
	}
	stats, err := h.svc.Stats(ctx, username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &moderationv1.StatsResponse{
		TotalPending:  stats.TotalPending,
		TotalApproved: stats.TotalApproved,
		TotalRejected: stats.TotalRejected,
	}, nil
}

// ── Proto mappers ────────────────────────────────────────────────────────────

func protoModeration(m *domain.Moderation, fallbackUsername string) *moderationv1.Moderation {
	if m == nil {
		return nil
	}
	return &moderationv1.Moderation{
		Id:            m.ID,
		ProductId:     m.ProductID,
		AdminId:       m.AdminID,
		AdminUsername: fallbackUsername,
		Active:        m.Active,
		CreatedAt:     safeTimestamp(m.CreatedAt),
		UpdatedAt:     safeTimestamp(m.UpdatedAt),
	}
}

// protoQueueProducts maps the unified domain rows to QueueProduct messages.
// Moderation is left nil for unclaimed queue items so the UI can show them
// as "open" instead of "locked".
func protoQueueProducts(rows []*domain.QueueRow) []*moderationv1.QueueProduct {
	res := make([]*moderationv1.QueueProduct, 0, len(rows))
	for _, e := range rows {
		var moderation *moderationv1.Moderation
		if e.Moderation != nil {
			moderation = protoModeration(e.Moderation, e.AdminUsername)
		}
		res = append(res, &moderationv1.QueueProduct{
			Product:    ProtoProduct(&e.Product),
			Moderation: moderation,
		})
	}
	return res
}
