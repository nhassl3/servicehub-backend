package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/nhassl3/servicehub-backend/internal/db"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

type ModerationRepo struct {
	store *db.Store
}

func NewModerationRepo(store *db.Store) *ModerationRepo {
	return &ModerationRepo{
		store: store,
	}
}

func (repo *ModerationRepo) CreateModeration(ctx context.Context, params domain.CreateModerationParams) (*domain.Moderation, error) {
	adminID, err := parseUUID(params.AdminID)
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.CreateModeration: invalid admin id: %w", err)
	}
	productID, err := parseUUID(params.ProductID)
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.CreateModeration: invalid product id: %w", err)
	}
	moderation, err := repo.store.CreateModeration(ctx, db.CreateModerationParams{
		ProductID: productID,
		AdminID:   adminID,
	})
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.CreateModeration: %w", err)
	}
	return mapModeration(&moderation), nil
}

func (repo *ModerationRepo) GetModeration(ctx context.Context, params domain.GetModerationParams) (*domain.Moderation, error) {
	moderationID, err := parseUUID(params.ID)
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.GetModeration: invalid id: %w", err)
	}
	moderation, err := repo.store.GetModeration(ctx, moderationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("moderation_repo.GetModeration: %w", err)
	}
	return mapModeration(&moderation), nil
}

// MyReviews returns rows previously claimed by the given admin (active=true)
// together with the count of such rows. AdminID must reference admins.id, NOT
// the user's UID — the moderation table joins admins, not users.
func (repo *ModerationRepo) MyReviews(ctx context.Context, params domain.ListModerationParams) ([]*domain.QueueRow, int64, error) {
	adminID := uuidPtrToNullable(&params.AdminID)
	var (
		rows  []db.ListModerationItemsRow
		total int64
	)
	if err := repo.store.ExecTx(ctx, func(q *db.Queries) error {
		var fnErr error
		rows, fnErr = q.ListModerationItems(ctx, db.ListModerationItemsParams{
			AdminID: adminID,
			Offset:  params.Offset,
			Limit:   params.Limit,
		})
		if fnErr != nil {
			return fmt.Errorf("list: %w", fnErr)
		}
		total, fnErr = q.Total(ctx, db.TotalParams{
			AdminID: adminID,
			Active:  true,
		})
		if fnErr != nil {
			return fmt.Errorf("count: %w", fnErr)
		}
		return nil
	}); err != nil {
		return nil, 0, fmt.Errorf("moderation_repo.MyReviews: %w", err)
	}
	return mapQueueRows(rows), total, nil
}

// Queue returns ALL products in `draft` status, regardless of whether anyone
// has claimed them yet. The total counter reflects the size of the unfiltered
// draft set so the UI can paginate it.
func (repo *ModerationRepo) Queue(ctx context.Context, params domain.ListParams) ([]*domain.QueueRow, int64, error) {
	var (
		rows  []db.ListModerationItemsRow
		total int64
	)
	if err := repo.store.ExecTx(ctx, func(q *db.Queries) error {
		var fnErr error
		rows, fnErr = q.ListModerationItems(ctx, db.ListModerationItemsParams{
			Limit:  params.Limit,
			Offset: params.Offset,
		})
		if fnErr != nil {
			return fmt.Errorf("list: %w", fnErr)
		}
		total, fnErr = repo.store.CountListProducts(ctx, db.CountListProductsParams{
			Status: "draft",
		})
		if fnErr != nil {
			return fmt.Errorf("count: %w", fnErr)
		}
		return nil
	}); err != nil {
		return nil, 0, fmt.Errorf("moderation_repo.Queue: %w", err)
	}
	return mapQueueRows(rows), total, nil
}

// ClaimProduct upserts a moderation row marking the product as claimed by
// admin. The Postgres-side check is the authoritative tie-breaker; Redis lock
// is only the fast path. ON CONFLICT preserves the row across re-claims.
func (repo *ModerationRepo) ClaimProduct(ctx context.Context, params domain.ClaimProductParams) (*domain.Moderation, error) {
	pending, err := repo.store.Total(ctx, db.TotalParams{
		Active:  true,
		AdminID: uuidPtrToNullable(&params.AdminID),
		Status:  stringToNullable("draft"),
	})
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.Stats: pending: %w", err)
	}
	if pending >= 5 {
		return nil, domain.ErrMoreThan5Claimed
	}
	adminID, err := parseUUID(params.AdminID)
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.ClaimProduct: invalid admin id: %w", err)
	}
	productID, err := parseUUID(params.ProductID)
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.ClaimProduct: invalid product id: %w", err)
	}
	row, err := repo.store.CreateModeration(ctx, db.CreateModerationParams{
		ProductID: productID,
		AdminID:   adminID,
	})
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.ClaimProduct: %w", err)
	}
	return mapModeration(&row), nil
}

// ReleaseProduct deletes the moderation row for product. The product is left
// in its current status (draft); approve/reject change the status separately.
func (repo *ModerationRepo) ReleaseProduct(ctx context.Context, params domain.ReleaseProductParams) error {
	id, err := parseUUID(params.ProductID)
	if err != nil {
		return fmt.Errorf("moderation_repo.ReleaseProduct: invalid product id: %w", err)
	}
	if err := repo.store.Release(ctx, db.ReleaseParams{
		ProductID: id,
		Status:    params.Status,
	}); err != nil {
		return fmt.Errorf("moderation_repo.ReleaseProduct: %w", err)
	}
	return nil
}

// Stats returns the dashboard counters. Approved/Rejected are global because
// the schema doesn't keep an audit log per admin yet — see ADMIN_PANEL_BACKEND_TODO.md
// section "5. Опциональные улучшения".
func (repo *ModerationRepo) Stats(ctx context.Context, adminID string) (*domain.ModerationStats, error) {
	claimedAdminID := uuidPtrToNullable(&adminID)
	pending, err := repo.store.Total(ctx, db.TotalParams{
		Active:  true,
		AdminID: claimedAdminID,
		Status:  stringToNullable("draft"),
	})
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.Stats: pending: %w", err)
	}
	approved, err := repo.store.Total(ctx, db.TotalParams{
		Active:  false,
		AdminID: claimedAdminID,
		Status:  stringToNullable("active"),
	})
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.Stats: approved: %w", err)
	}
	rejected, err := repo.store.Total(ctx, db.TotalParams{
		Active:  false,
		AdminID: claimedAdminID,
		Status:  stringToNullable("inactive"),
	})
	if err != nil {
		return nil, fmt.Errorf("moderation_repo.Stats: rejected: %w", err)
	}

	return &domain.ModerationStats{
		TotalPending:  int32(pending),
		TotalApproved: int32(approved),
		TotalRejected: int32(rejected),
	}, nil
}

// ── Mapping ──────────────────────────────────────────────────────────────────

func mapModeration(moderation *db.Moderation) *domain.Moderation {
	return &domain.Moderation{
		ID:        moderation.ID.String(),
		ProductID: moderation.ProductID.String(),
		AdminID:   moderation.AdminID.String(),
		Active:    moderation.Active,
		CreatedAt: pgTimeTZ(moderation.CreatedAt, time.UTC),
		UpdatedAt: pgTimeTZ(moderation.UpdatedAt, time.UTC),
	}
}

// mapQueueRows converts sqlc rows into domain rows. Moderation is left nil
// when the LEFT JOIN produced NULLs (queue item not yet claimed).
func mapQueueRows(rows []db.ListModerationItemsRow) []*domain.QueueRow {
	res := make([]*domain.QueueRow, 0, len(rows))
	for _, e := range rows {
		var moderation *domain.Moderation
		if e.ModerationID.Valid {
			moderation = &domain.Moderation{
				ID:        uuidStringFromPg(e.ModerationID),
				AdminID:   uuidStringFromPg(e.ModerationAdminID),
				ProductID: e.ProductID.String(),
				Active:    e.ModerationActive.Bool,
				CreatedAt: pgTimeTZ(e.ModerationCreatedAt, time.UTC),
				UpdatedAt: pgTimeTZ(e.ModerationUpdatedAt, time.UTC),
			}
		}
		res = append(res, &domain.QueueRow{
			Product: domain.Product{
				ID:           e.ProductID.String(),
				SellerID:     e.SellerID.String(),
				CategoryID:   int(e.CategoryID),
				Title:        e.Title,
				Description:  e.Description,
				Price:        e.Price,
				Tags:         e.Tags,
				Status:       e.Status,
				SalesCount:   int(e.SalesCount),
				Rating:       e.Rating,
				ReviewsCount: int(e.ReviewsCount),
				CreatedAt:    pgTimeTZ(e.ProductCreatedAt, time.UTC),
				UpdatedAt:    pgTimeTZ(e.ProductUpdatedAt, time.UTC),
			},
			Moderation:    moderation,
			AdminUsername: e.AdminUsername.String,
		})
	}
	return res
}
