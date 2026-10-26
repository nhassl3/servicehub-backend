package service_test

import (
	"context"
	"testing"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNotificationService_ApproveCode_OK(t *testing.T) {
	var verified *domain.VerifiedState

	redis := &mockUserRedis{
		codeFunc: func(_ context.Context, _, _ string) (*domain.CodeState, error) {
			return &domain.CodeState{Email: "new@example.com", Code: "123456", Username: "alice"}, nil
		},
		setVerifiedFunc: func(_ context.Context, _, _ string, state *domain.VerifiedState) error {
			verified = state
			return nil
		},
	}
	svc := service.NewNotificationService(redis, &mockUserRepo{}, nil)

	token, err := svc.ApproveCode(context.Background(), "verify_email:", "operation-id", "123456")

	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.NotNil(t, verified)
	require.Equal(t, "new@example.com", verified.Email)
	require.Equal(t, "alice", verified.Username)
	require.True(t, verified.IsEmailReset())
}

func TestNotificationService_ApproveCode_WrongCode(t *testing.T) {
	redis := &mockUserRedis{
		codeFunc: func(_ context.Context, _, _ string) (*domain.CodeState, error) {
			return &domain.CodeState{Email: "alice@example.com", Code: "123456"}, nil
		},
	}
	svc := service.NewNotificationService(redis, &mockUserRepo{}, nil)

	token, err := svc.ApproveCode(context.Background(), "verify_email:", "operation-id", "000000")

	require.ErrorIs(t, err, domain.ErrInvalidCode)
	require.Empty(t, token)
}

func TestNotificationService_ApproveCode_TooManyAttempts(t *testing.T) {
	redis := &mockUserRedis{
		codeFunc: func(_ context.Context, _, _ string) (*domain.CodeState, error) {
			return &domain.CodeState{Email: "alice@example.com", Code: "123456"}, nil
		},
		incrCodeAttemptsFunc: func(_ context.Context, _, _ string) (int64, error) {
			return 6, nil // maxCodeAttempts + 1
		},
	}
	svc := service.NewNotificationService(redis, &mockUserRepo{}, nil)

	token, err := svc.ApproveCode(context.Background(), "verify_email:", "operation-id", "123456")

	require.ErrorIs(t, err, domain.ErrTooManyCodeAttempts)
	require.Empty(t, token)
}

func TestNotificationService_ApproveCode_CodeNotFound(t *testing.T) {
	svc := service.NewNotificationService(&mockUserRedis{}, &mockUserRepo{}, nil)

	token, err := svc.ApproveCode(context.Background(), "verify_email:", "unknown", "123456")

	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Empty(t, token)
}
