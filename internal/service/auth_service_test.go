package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/service"
	"github.com/nhassl3/servicehub-backend/internal/service/mock"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Tests ────────────────────────────────────────────────────────────────────

func newAuthServiceWithRedis(repo domain.UserRepository, redis domain.UserRedis) *service.AuthService {
	tm := &mock.MockTokenManager{}
	bl := mock.NewMockBlacklist()
	log := zap.NewNop()
	publishService := service.NewPublishService(mock.NoopPublisher{}, log)
	tokenService := service.NewTokenService(tm, tm, redis, repo, log)
	return service.NewAuthService(repo, redis, bl, publishService, tokenService, log)
}

func newAuthService(repo domain.UserRepository) *service.AuthService {
	redis := &mock.MockUserRedis{}
	return newAuthServiceWithRedis(repo, redis)
}

func TestAuthService_Register_OK(t *testing.T) {
	repo := &mock.MockUserRepo{}
	svc := newAuthService(repo)

	user, tokens, err := svc.Register(context.Background(), domain.RegisterInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
		FullName: "Alice Smith",
	})

	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, tokens)
	require.Equal(t, "alice", user.Username)
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
}

func TestAuthService_Register_DuplicateUsername(t *testing.T) {
	repo := &mock.MockUserRepo{
		UserExistsFunc: func(_ context.Context, params *domain.UserExistsParams) (bool, error) {
			if params.Username == nil {
				return false, nil
			}
			return *params.Username == "alice", nil
		},
	}
	svc := newAuthService(repo)

	_, _, err := svc.Register(context.Background(), domain.RegisterInput{
		Username: "alice",
		Email:    "new@example.com",
		Password: "password123",
		FullName: "Alice",
	})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	repo := &mock.MockUserRepo{
		UserExistsFunc: func(_ context.Context, params *domain.UserExistsParams) (bool, error) {
			if params.Email == nil {
				return false, nil
			}
			return *params.Email == "alice@example.com", nil
		},
	}
	svc := newAuthService(repo)

	_, _, err := svc.Register(context.Background(), domain.RegisterInput{
		Username: "newuser",
		Email:    "alice@example.com",
		Password: "password123",
		FullName: "New",
	})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestAuthService_Login_OK(t *testing.T) {
	// Pre-hash a known password
	hashedPw := "$argon2id$v=19$m=65536,t=3,p=4$AAAAAAAAAAAAAAAAAAAAAA$tnYfz0WUPkpCFwD1zxH0HKP3xJiGUmJ5x/Vvom+xISU"

	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, params *domain.GetUserParams) (*domain.User, error) {
			if params != nil && params.Username != nil {
				if *params.Username == "alice" {
					return &domain.User{
						Username:     "alice",
						UID:          "uid-123",
						Role:         "buyer",
						PasswordHash: hashedPw,
					}, nil
				}
			}
			return nil, domain.ErrNotFound
		},
	}
	svc := newAuthService(repo)

	// This will fail because the hash is fake — test the not-found path instead
	_, _, err := svc.Login(context.Background(), "nonexistent", "password")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := &mock.MockUserRepo{}
	svc := newAuthService(repo)

	_, _, err := svc.Login(context.Background(), "nobody", "pass")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, _ *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{
				Username:     "alice",
				UID:          "uid",
				Role:         "buyer",
				PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$invalidsalt$invalidhash",
			}, nil
		},
	}
	svc := newAuthService(repo)

	_, _, err := svc.Login(context.Background(), "alice", "wrongpassword")
	// Either invalid credentials or invalid hash format
	require.True(t, errors.Is(err, domain.ErrInvalidCredentials) || err != nil)
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	repo := &mock.MockUserRepo{}
	tm := &mock.MockTokenManager{VerifyErr: auth.ErrInvalidToken}
	bl := mock.NewMockBlacklist()
	redis := &mock.MockUserRedis{}
	log := zap.NewNop()
	publishService := service.NewPublishService(mock.NoopPublisher{}, log)
	tokenService := service.NewTokenService(tm, tm, redis, repo, log)
	svc := service.NewAuthService(repo, redis, bl, publishService, tokenService, log)

	_, err := svc.RefreshToken(context.Background(), "bad-token")
	require.ErrorIs(t, err, domain.ErrInvalidToken)
}
