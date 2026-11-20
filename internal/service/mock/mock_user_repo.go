package mock

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/domain"
)

// ─── Mock UserRepository ──────────────────────────────────────────────────────

type MockUserRepo struct {
	UserExistsFunc           func(ctx context.Context, params *domain.UserExistsParams) (bool, error)
	CreateFunc               func(ctx context.Context, params *domain.CreateUserParams) (*domain.User, error)
	GetUserFunc              func(ctx context.Context, params *domain.GetUserParams) (*domain.User, error)
	UpdateFunc               func(ctx context.Context, params *domain.UpdateUserParams) (*domain.User, error)
	UpdatePasswordFunc       func(ctx context.Context, params *domain.UpdateUserPasswordParams) (*domain.User, error)
	GetSessionFunc           func(ctx context.Context, refreshToken string) (*domain.Session, error)
	GetSessionByUsernameFunc func(ctx context.Context, username string) (*domain.Session, error)
	DeleteSessionFunc        func(ctx context.Context, username string) error
	CreateSessionFunc        func(ctx context.Context, params *domain.CreateSessionParams) (*domain.Session, error)
	VerifyEmailFunc          func(ctx context.Context, params *domain.VerifyEmailAccount) (*domain.User, error)
	ConfirmEmailResetFunc    func(ctx context.Context, params *domain.ResetEmailParams) (*domain.User, error)
	OAuthLoginFunc           func(ctx context.Context, params *domain.OAuthLoginParams) (*domain.User, error)
}

func (m *MockUserRepo) OAuthLogin(ctx context.Context, params *domain.OAuthLoginParams) (*domain.User, error) {
	if m.OAuthLoginFunc != nil {
		return m.OAuthLoginFunc(ctx, params)
	}
	return nil, domain.ErrAlreadyExists
}

func (m *MockUserRepo) ConfirmEmailReset(ctx context.Context, params *domain.ResetEmailParams) (*domain.User, error) {
	if m.ConfirmEmailResetFunc != nil {
		return m.ConfirmEmailResetFunc(ctx, params)
	}
	return nil, domain.ErrEmailAlreadyReset
}

func (m *MockUserRepo) CreateSession(ctx context.Context, params *domain.CreateSessionParams) (*domain.Session, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, params)
	}
	return &domain.Session{
		Username:     params.Username,
		RefreshToken: params.RefreshToken,
		UserAgent:    params.UserAgent,
		ClientIP:     params.ClientIp,
		IsBlocked:    params.IsBlocked,
		ExpiresAt:    params.ExpiresAt,
	}, nil
}

func (m *MockUserRepo) GetSession(ctx context.Context, refreshToken string) (*domain.Session, error) {
	if m.GetSessionFunc != nil {
		return m.GetSessionFunc(ctx, refreshToken)
	}
	return &domain.Session{}, nil
}

func (m *MockUserRepo) GetSessionByUsername(ctx context.Context, username string) (*domain.Session, error) {
	if m.GetSessionByUsernameFunc != nil {
		return m.GetSessionByUsernameFunc(ctx, username)
	}
	return nil, domain.ErrNotFound
}

func (m *MockUserRepo) DeleteSession(ctx context.Context, username string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(ctx, username)
	}
	return nil
}

func (m *MockUserRepo) UpdatePassword(ctx context.Context, params *domain.UpdateUserPasswordParams) (*domain.User, error) {
	if m.UpdatePasswordFunc != nil {
		return m.UpdatePasswordFunc(ctx, params)
	}
	return &domain.User{}, nil
}

func (m *MockUserRepo) UserExists(ctx context.Context, params *domain.UserExistsParams) (bool, error) {
	if m.UserExistsFunc != nil {
		return m.UserExistsFunc(ctx, params)
	}
	return false, nil
}

func (m *MockUserRepo) Create(ctx context.Context, params *domain.CreateUserParams) (*domain.User, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, params)
	}
	return &domain.User{
		Username: params.Username,
		Email:    params.Email,
		UID:      "uid-123",
		Role:     "buyer",
	}, nil
}

func (m *MockUserRepo) GetUser(ctx context.Context, params *domain.GetUserParams) (*domain.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, params)
	}
	return nil, domain.ErrNotFound
}

func (m *MockUserRepo) VerifyEmail(ctx context.Context, params *domain.VerifyEmailAccount) (*domain.User, error) {
	if m.VerifyEmailFunc != nil {
		return m.VerifyEmailFunc(ctx, params)
	}
	return nil, domain.ErrNotFound
}

func (m *MockUserRepo) Update(ctx context.Context, params *domain.UpdateUserParams) (*domain.User, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, params)
	}
	return nil, domain.ErrNotFound
}
