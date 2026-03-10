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

type SellerRepo struct {
	store *db.Store
}

func NewSellerRepo(store *db.Store) *SellerRepo {
	return &SellerRepo{store: store}
}

func (r *SellerRepo) Create(ctx context.Context, params domain.CreateSellerParams) (*domain.Seller, error) {
	var seller *domain.Seller

	err := r.store.ExecTx(ctx, func(q *db.Queries) error {
		row, err := q.CreateSeller(ctx, db.CreateSellerParams{
			Username:    params.Username,
			DisplayName: params.DisplayName,
			Description: params.Description,
		})
		if err != nil {
			return err
		}
		seller = mapSeller(row)

		_, err = q.SetUserRole(ctx, db.SetUserRoleParams{
			Username: params.Username,
			Role:     "seller",
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("seller_repo.Create: %w", err)
	}
	return seller, nil
}

func (r *SellerRepo) GetSeller(ctx context.Context, params domain.GetSellerProfileParams) (*domain.Seller, error) {
	sellerID := uuidPtrToNullable(params.SellerId)
	if params.SellerId != nil && !sellerID.Valid {
		return nil, domain.ErrInvalidInput
	}
	row, err := r.store.GetSeller(ctx, db.GetSellerParams{
		Username: usernamePtrToNullable(params.Username),
		SellerID: sellerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("seller_repo.GetSeller: %w", err)
	}
	return mapSeller(row), nil
}

func (r *SellerRepo) Update(ctx context.Context, params domain.UpdateSellerParams) (*domain.Seller, error) {
	row, err := r.store.UpdateSeller(ctx, db.UpdateSellerParams{
		Username:    params.Username,
		DisplayName: params.DisplayName,
		Description: params.Description,
		AvatarUrl:   params.AvatarURL,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("seller_repo.Update: %w", err)
	}
	return mapSeller(row), nil
}

func (r *SellerRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	exists, err := r.store.SellerExistsByUsername(ctx, username)
	if err != nil {
		return false, fmt.Errorf("seller_repo.ExistsByUsername: %w", err)
	}
	return exists, nil
}

// ── Mapping ──────────────────────────────────────────────────────────────────

func mapSeller(s db.Seller) *domain.Seller {
	return &domain.Seller{
		ID:          s.ID.String(),
		Username:    s.Username,
		DisplayName: s.DisplayName,
		Description: s.Description,
		AvatarURL:   s.AvatarUrl,
		Rating:      s.Rating,
		TotalSales:  int(s.TotalSales),
		CreatedAt:   pgTimeTZ(s.CreatedAt, time.UTC),
		UpdatedAt:   pgTimeTZ(s.UpdatedAt, time.UTC),
	}
}
