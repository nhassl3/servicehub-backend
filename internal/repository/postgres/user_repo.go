package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nhassl3/servicehub-backend/internal/db"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

const uniqueViolationCode = "23505"

type UserRepo struct {
	store *db.Store
}

func NewUserRepo(store *db.Store) *UserRepo {
	return &UserRepo{store: store}
}

func (r *UserRepo) Create(ctx context.Context, params *domain.CreateUserParams) (*domain.User, error) {
	row, err := r.store.CreateUser(ctx, db.CreateUserParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
		FullName:     params.FullName,
	})
	if err != nil {
		return nil, fmt.Errorf("user_repo.Create: %w", err)
	}
	return mapUser(row), nil
}

func (r *UserRepo) CreateSession(ctx context.Context, params *domain.CreateSessionParams) (*domain.Session, error) {
	session, err := r.store.CreateSession(ctx, db.CreateSessionParams{
		Username:     params.Username,
		RefreshToken: params.RefreshToken,
		UserAgent:    params.UserAgent,
		ClientIp:     params.ClientIp,
		ExpiresAt:    pgtype.Timestamptz{Time: params.ExpiresAt, Valid: true},
		IsBlocked:    params.IsBlocked,
	})
	if err != nil {
		return nil, fmt.Errorf("user_repo.CreateSession: %w", err)
	}
	return mapSession(session), nil
}

func (r *UserRepo) GetSession(ctx context.Context, refreshToken string) (*domain.Session, error) {
	row, err := r.store.GetSession(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetSession: %w", err)
	}
	return mapSession(row), nil
}

func (r *UserRepo) GetSessionByUsername(ctx context.Context, username string) (*domain.Session, error) {
	row, err := r.store.GetSessionByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetSessionByUsername: %w", err)
	}
	return mapSession(row), nil
}

func (r *UserRepo) DeleteSession(ctx context.Context, username string) error {
	if err := r.store.DeleteSession(ctx, username); err != nil {
		return fmt.Errorf("user_repo.DeleteSession: %w", err)
	}
	return nil
}

func (r *UserRepo) GetUser(ctx context.Context, params *domain.GetUserParams) (*domain.User, error) {
	row, err := r.store.GetUser(ctx, db.GetUserParams{
		Username: stringPtrToNullable(params.Username),
		Email:    stringPtrToNullable(params.Email),
		Uid:      uuidPtrToNullable(params.UID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.GetUser: failed to get User: %w", err)
	}
	return mapUser(row), nil
}

func (r *UserRepo) UserExists(ctx context.Context, params *domain.UserExistsParams) (bool, error) {
	exists, err := r.store.UserExists(ctx, db.UserExistsParams{
		Username: stringPtrToNullable(params.Username),
		Email:    stringPtrToNullable(params.Email),
	})
	if err != nil {
		return false, fmt.Errorf("user_repo.UserExists: failed to check user: %w", err)
	}
	return exists, nil
}

// ConfirmEmailReset switches the account email after the new address has been
// confirmed. The SQL statement is the single source of truth for "once only":
// it updates, activates and consumes the flag atomically, and returns no rows
// when the reset is no longer allowed.
func (r *UserRepo) ConfirmEmailReset(ctx context.Context, params *domain.ResetEmailParams) (*domain.User, error) {
	row, err := r.store.ConfirmEmailReset(ctx, db.ConfirmEmailResetParams{
		Username: params.Username,
		NewEmail: params.NewEmail,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEmailAlreadyReset
		}

		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == uniqueViolationCode {
			return nil, domain.ErrAlreadyExists
		}

		return nil, fmt.Errorf("user_repo.ConfirmEmailReset: %w", err)
	}
	return mapUser(row), nil
}

func (r *UserRepo) Update(ctx context.Context, params *domain.UpdateUserParams) (*domain.User, error) {
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
	return mapUser(row), nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, params *domain.UpdateUserPasswordParams) (*domain.User, error) {
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
	return mapUser(row), nil
}

func (r *UserRepo) VerifyEmail(ctx context.Context, params *domain.VerifyEmailAccount) (*domain.User, error) {
	user, err := r.store.VerifyEmailAccount(ctx, db.VerifyEmailAccountParams{
		Email:    stringPtrToNullable(params.Email),
		Username: stringPtrToNullable(params.Username),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("user_repo.VerifyEmail: %w", err)
	}
	return mapUser(user), nil
}

func (r *UserRepo) OAuthLogin(ctx context.Context, params *domain.OAuthLoginParams) (*domain.User, error) {
	user, err := r.store.OAuthLogin(ctx, db.OAuthLoginParams{
		Username:     params.Username,
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
	})
	if err != nil {
		return nil, fmt.Errorf("user_repo.oAuthLogin: %w", err)
	}
	return mapUser(user), nil
}

// ── Mapping ──────────────────────────────────────────────────────────────────

func mapUser(u db.User) *domain.User {
	var emailResetAt *time.Time
	if u.EmailResetAt.Valid { // can be nil-pointer
		emailResetAt = new(u.EmailResetAt.Time)
	}

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
		EmailResetAt: emailResetAt,
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
