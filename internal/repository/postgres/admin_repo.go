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

type AdminRepo struct {
	store *db.Store
}

func NewAdminRepo(store *db.Store) *AdminRepo {
	return &AdminRepo{store: store}
}

func (r *AdminRepo) CreateAdmin(ctx context.Context, params domain.CreateAdminParams) (*domain.Admin, error) {
	var admin *domain.Admin

	if err := r.store.ExecTx(ctx, func(q *db.Queries) error {
		row, err := q.CreateAdmin(ctx, db.CreateAdminParams{
			Username:    params.Username,
			DisplayName: params.DisplayName,
			LevelRights: params.LevelRights,
		})
		if err != nil {
			return err
		}
		admin = mapAdmin(row)

		_, err = q.SetUserRole(ctx, db.SetUserRoleParams{
			Username: params.Username,
			Role:     "admin",
		})
		return err
	}); err != nil {
		return nil, fmt.Errorf("admin_repo.CreateAdmin: %w", err)
	}
	return admin, nil
}

func (r *AdminRepo) GetAdmin(ctx context.Context, params domain.GetAdminProfileParams) (*domain.Admin, error) {
	var getAdminParams db.GetAdminParams
	if params.AdminId != "" {
		getAdminParams = db.GetAdminParams{
			AdminID: uuidPtrToNullable(&params.AdminId),
		}
	} else if params.Username != "" {
		getAdminParams = db.GetAdminParams{
			Username: usernamePtrToNullable(&params.Username),
		}
	}
	row, err := r.store.GetAdmin(ctx, getAdminParams)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("admin_repo.GetAdmin: %w", err)
	}
	return mapAdmin(row), nil
}

func (r *AdminRepo) UpdateAdminProfile(ctx context.Context, params domain.UpdateAdminsProfileParams) (*domain.Admin, error) {
	var admin *domain.Admin

	if err := r.store.ExecTx(ctx, func(q *db.Queries) error {
		row, err := q.GetAdminForUpdate(ctx, db.GetAdminForUpdateParams{
			Username: params.Username,
		})
		if err != nil {
			return err
		}

		displayName := row.DisplayName
		avatarURL := row.AvatarUrl
		levelRights := row.LevelRights
		totalModerates := row.TotalModeration

		if params.DisplayName != nil {
			displayName = *params.DisplayName
		} else if params.AvatarURL != nil {
			avatarURL = *params.AvatarURL
		} else if params.LevelRights != nil {
			levelRights = *params.LevelRights
		} else if params.TotalModerates != nil {
			totalModerates = *params.TotalModerates
		}

		row, err = q.UpdateAdmin(ctx, db.UpdateAdminParams{
			DisplayName:     displayName,
			AvatarUrl:       avatarURL,
			LevelRights:     levelRights,
			TotalModeration: totalModerates,
		})
		if err != nil {
			return err
		}
		admin = mapAdmin(row)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("admin_repo.UpdateAdminProfile: %w", err)
	}
	return admin, nil
}

func (r *AdminRepo) UploadCategoryAvatar(ctx context.Context, params domain.UploadCategoryAvatar) (*domain.Category, error) {
	return nil, nil
}

func (r *AdminRepo) IncreaseTotalModerates(ctx context.Context, params domain.IncreaseTotalModeratesParams) error {
	return r.store.IncreaseTotalModerates(ctx, db.IncreaseTotalModeratesParams{
		Username:       params.Username,
		TotalModerates: params.TotalModerates,
	})
}

func (r *AdminRepo) ExistsAdminByUsername(ctx context.Context, username string) (bool, error) {
	exists, err := r.store.AdminExistsByUsername(ctx, username)
	if err != nil {
		return false, fmt.Errorf("admin_repo.ExistsAdminByUsername: %w", err)
	}
	return exists, nil
}

// ── Mapping ──────────────────────────────────────────────────────────────────

func mapAdmin(s db.Admin) *domain.Admin {
	return &domain.Admin{
		ID:             s.ID.String(),
		Username:       s.Username,
		DisplayName:    s.DisplayName,
		TotalModerates: s.TotalModeration,
		LevelRights:    s.LevelRights,
		AvatarURL:      s.AvatarUrl,
		CreatedAt:      pgTimeTZ(s.CreatedAt, time.UTC),
		UpdatedAt:      pgTimeTZ(s.UpdatedAt, time.UTC),
	}
}
