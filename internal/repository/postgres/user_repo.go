package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type UserRepo struct {
	store *db.Store
}

func NewUserRepo(store *db.Store) *UserRepo {
	return &UserRepo{store: store}
}

func (r *UserRepo) Create(ctx context.Context, params domain.CreateUserParams) (*domain.User, error) {
	row, err := r.store.CreateUser(ctx, db.CreateUserParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
		FullName:     params.FullName,
	})
	if err != nil {
		return nil, fmt.Errorf("user_repo.Create: %w", err)
	}
	return mapUser(db.User{
		Username:     row.Username,
		Uid:          row.Uid,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		FullName:     row.FullName,
		AvatarUrl:    row.AvatarUrl,
		Role:         row.Role,
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}), nil
}

func (r *UserRepo) CreateSession(ctx context.Context, params domain.CreateSessionParams) error {
	err := r.store.CreateSession(ctx, db.CreateSessionParams{
		Username:     params.Username,
		RefreshToken: params.RefreshToken,
		UserAgent:    params.UserAgent,
		ClientIp:     params.ClientIp,
		ExpiresAt:    pgtype.Timestamptz{Time: params.ExpiresAt, Valid: true},
		IsBlocked:    params.IsBlocked,
	})
	if err != nil {
		return fmt.Errorf("user_repo.CreateSession: %w", err)
	}
	return nil
}

func (r *UserRepo) GetSession(ctx context.Context, username string) (*domain.Session, error) {
	row, err := r.store.GetSession(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetSession: %w", err)
	}
	return mapSession(row), nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	row, err := r.store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetByUsername: %w", err)
	}
	return mapUser(row), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetByEmail: %w", err)
	}
	return mapUser(row), nil
}

func (r *UserRepo) GetByUID(ctx context.Context, uid string) (*domain.User, error) {
	u, err := parseUUID(uid)
	if err != nil {
		return nil, fmt.Errorf("user_repo.GetByUID: invalid uid: %w", err)
	}
	row, err := r.store.GetUserByUID(ctx, u)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetByUID: %w", err)
	}
	return mapUser(row), nil
}

func (r *UserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	exists, err := r.store.UserExistsByUsername(ctx, username)
	if err != nil {
		return false, fmt.Errorf("user_repo.ExistsByUsername: %w", err)
	}
	return exists, nil
}

func (r *UserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	exists, err := r.store.UserExistsByEmail(ctx, email)
	if err != nil {
		return false, fmt.Errorf("user_repo.ExistsByEmail: %w", err)
	}
	return exists, nil
}

func (r *UserRepo) Update(ctx context.Context, params domain.UpdateUserParams) (*domain.User, error) {
	row, err := r.store.UpdateUser(ctx, db.UpdateUserParams{
		Username:  params.Username,
		FullName:  params.FullName,
		AvatarUrl: params.AvatarURL,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.Update: %w", err)
	}
	return mapUser(db.User{
		Username:     row.Username,
		Uid:          row.Uid,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		FullName:     row.FullName,
		AvatarUrl:    row.AvatarUrl,
		Role:         row.Role,
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}), nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, params domain.UpdateUserPasswordParams) (*domain.User, error) {
	row, err := r.store.UpdatePassword(ctx, db.UpdatePasswordParams{
		Username:    params.Username,
		NewPassword: params.NewPassword,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.UpdatePassword: %w", err)
	}
	return mapUser(db.User{
		Username:     row.Username,
		Uid:          row.Uid,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		FullName:     row.FullName,
		AvatarUrl:    row.AvatarUrl,
		Role:         row.Role,
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}), nil
}

// ── Mapping ──────────────────────────────────────────────────────────────────

func mapUser(u db.User) *domain.User {
	return &domain.User{
		Username:     u.Username,
		UID:          u.Uid.String(),
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName,
		AvatarURL:    u.AvatarUrl,
		Role:         u.Role,
		IsActive:     u.IsActive,
		CreatedAt:    pgTimeTZ(u.CreatedAt, time.UTC),
		UpdatedAt:    pgTimeTZ(u.UpdatedAt, time.UTC),
	}
}

func mapSession(s db.Session) *domain.Session {
	return &domain.Session{
		ID:           s.ID.String(),
		Username:     s.Username,
		RefreshToken: s.RefreshToken,
		UserAgent:    s.UserAgent,
		ClientIP:     s.ClientIp,
		ExpiresAt:    pgTimeTZ(s.ExpiresAt, time.UTC),
		IsBlocked:    s.IsBlocked,
		CreatedAt:    pgTimeTZ(s.CreatedAt, time.UTC),
	}
}
