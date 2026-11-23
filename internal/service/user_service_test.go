package service

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	repoRedis "github.com/nhassl3/servicehub-backend/internal/repository/redis"
	"github.com/nhassl3/servicehub-backend/internal/service/mock"
	"github.com/nhassl3/servicehub-backend/pkg/hash"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const emailResetWindow = 72 * time.Hour

func newUserService(
	repo domain.UserRepository,
	storage domain.PhotoStorage,
	redis domain.UserRedis,
	notifier *mock.MockNotifier,
) *UserService {
	tm := &mock.MockTokenManager{}
	log := zap.NewNop()
	publishService := NewPublishService(mock.NoopPublisher{}, log)
	tokenService := NewTokenService(tm, tm, redis, repo, log)
	return NewUserService(repo, storage, redis, publishService, emailResetWindow, notifier, tokenService, log)
}

func newUserServiceWithRedis(repo domain.UserRepository, redis domain.UserRedis) *UserService {
	return newUserService(repo, &mock.MockPhotoStorage{}, redis, &mock.MockNotifier{})
}

func TestUserService_RequestResetEmail_OK(t *testing.T) {
	var (
		stored        *domain.CodeState
		storedKey     string
		notifiedCode  string
		notifiedEmail string
	)

	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, params *domain.GetUserParams) (*domain.User, error) {
			if params != nil && params.Username != nil {
				return &domain.User{
					Username:  *params.Username,
					Email:     "old@example.com",
					IsActive:  false,
					CreatedAt: time.Now().Add(-time.Hour),
				}, nil
			}
			return nil, domain.ErrNotFound
		},
	}
	redis := &mock.MockUserRedis{
		SetCodeFunc: func(_ context.Context, entryKey, _ string, state *domain.CodeState) error {
			storedKey = entryKey
			stored = state
			return nil
		},
	}
	notifier := &mock.MockNotifier{
		NotifyEmailConfirmationFunc: func(_ context.Context, code, email string) error {
			notifiedCode, notifiedEmail = code, email
			return nil
		},
	}

	operationID, err := newUserService(repo, &mock.MockPhotoStorage{}, redis, notifier).
		RequestResetEmail(context.Background(), "alice", "new@example.com")

	require.NoError(t, err)
	require.NotEmpty(t, operationID)
	require.Equal(t, repoRedis.ResetEmailEnterKey, storedKey)
	require.NotNil(t, stored)
	require.Equal(t, "new@example.com", stored.Email)
	require.Equal(t, "alice", stored.Username)
	require.Len(t, stored.Code, 6)
	require.Equal(t, stored.Code, notifiedCode)
	require.Equal(t, "new@example.com", notifiedEmail)
}

func TestUserService_RequestResetEmail_WindowPassed(t *testing.T) {
	resetAt := time.Now().Add(-73 * time.Hour)
	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, params *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{
				Username:     *params.Username,
				EmailResetAt: resetAt,
			}, nil
		},
	}

	operationID, err := newUserServiceWithRedis(repo, &mock.MockUserRedis{}).
		RequestResetEmail(context.Background(), "alice", "new@example.com")

	require.NoError(t, err)
	require.NotEmpty(t, operationID)
}

func TestUserService_RequestResetEmail_AlreadyReset(t *testing.T) {
	resetAt := time.Now().Add(-time.Hour)
	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, params *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{
				Username:     *params.Username,
				CreatedAt:    time.Now().Add(-24 * time.Hour),
				EmailResetAt: resetAt,
			}, nil
		},
	}

	operationID, err := newUserServiceWithRedis(repo, &mock.MockUserRedis{}).
		RequestResetEmail(context.Background(), "alice", "new@example.com")

	require.ErrorIs(t, err, domain.EmailResetManyAttempts)
	require.Empty(t, operationID)
}

func TestUserService_RequestResetEmail_EmailTaken(t *testing.T) {
	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, params *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{Username: *params.Username, CreatedAt: time.Now().Add(-time.Hour)}, nil
		},
		UserExistsFunc: func(_ context.Context, _ *domain.UserExistsParams) (bool, error) {
			return true, nil
		},
	}

	_, err := newUserServiceWithRedis(repo, &mock.MockUserRedis{}).
		RequestResetEmail(context.Background(), "alice", "taken@example.com")

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestUserService_RequestResetEmail_CodeAlreadyExists(t *testing.T) {
	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, params *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{Username: *params.Username, CreatedAt: time.Now().Add(-time.Hour)}, nil
		},
	}
	redis := &mock.MockUserRedis{
		CodeExistsFunc: func(_ context.Context, _ string, _ string) bool {
			return true
		},
	}

	_, err := newUserServiceWithRedis(repo, redis).
		RequestResetEmail(context.Background(), "alice", "new@example.com")

	require.ErrorIs(t, err, domain.ErrCodeAlreadyExists)
}

func TestUserService_RequestVerifyEmail_NotifiesUser(t *testing.T) {
	var (
		stored        *domain.CodeState
		storedKey     string
		notifiedCode  string
		notifiedEmail string
	)

	redis := &mock.MockUserRedis{
		SetCodeFunc: func(_ context.Context, entryKey, _ string, state *domain.CodeState) error {
			storedKey = entryKey
			stored = state
			return nil
		},
	}
	notifier := &mock.MockNotifier{
		NotifyEmailConfirmationFunc: func(_ context.Context, code, email string) error {
			notifiedCode, notifiedEmail = code, email
			return nil
		},
	}

	operationID, err := newUserService(&mock.MockUserRepo{}, &mock.MockPhotoStorage{}, redis, notifier).
		RequestVerifyEmail(context.Background(), "alice@example.com")

	require.NoError(t, err)
	require.NotEmpty(t, operationID)
	require.Equal(t, repoRedis.VerifyEmailEnterKey, storedKey)
	require.NotNil(t, stored)
	require.Equal(t, "alice@example.com", stored.Email)
	require.Equal(t, stored.Code, notifiedCode)
	require.Equal(t, "alice@example.com", notifiedEmail)
}

func TestUserService_RequestResetPassword_ByUsername(t *testing.T) {
	var (
		stored       *domain.CodeState
		storedKey    string
		notifiedCode string
	)

	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, _ *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{Username: "alice", Email: "alice@example.com"}, nil
		},
	}
	redis := &mock.MockUserRedis{
		SetCodeFunc: func(_ context.Context, entryKey, _ string, state *domain.CodeState) error {
			storedKey = entryKey
			stored = state
			return nil
		},
	}
	notifier := &mock.MockNotifier{
		NotifyResetPasswordFunc: func(_ context.Context, code, _ string) error {
			notifiedCode = code
			return nil
		},
	}
	username := "alice"

	operationID, err := newUserService(repo, &mock.MockPhotoStorage{}, redis, notifier).
		RequestResetPassword(context.Background(), domain.RequestResetPasswordParams{Username: &username})

	require.NoError(t, err)
	require.NotEmpty(t, operationID)
	require.Equal(t, repoRedis.ResetPasswordEnterKey, storedKey)
	require.NotNil(t, stored)
	require.Equal(t, "alice@example.com", stored.Email)
	require.Equal(t, stored.Code, notifiedCode)
}

func TestUserService_RequestResetPassword_ByEmail(t *testing.T) {
	repo := &mock.MockUserRepo{
		UserExistsFunc: func(_ context.Context, params *domain.UserExistsParams) (bool, error) {
			return params.Email != nil && *params.Email == "alice@example.com", nil
		},
	}

	operationID, err := newUserServiceWithRedis(repo, &mock.MockUserRedis{}).
		RequestResetPassword(context.Background(), domain.RequestResetPasswordParams{Email: ptr("alice@example.com")})

	require.NoError(t, err)
	require.NotEmpty(t, operationID)
}

func TestUserService_RequestResetPassword_EmailNotFound(t *testing.T) {
	_, err := newUserServiceWithRedis(&mock.MockUserRepo{}, &mock.MockUserRedis{}).
		RequestResetPassword(context.Background(), domain.RequestResetPasswordParams{Email: ptr("nobody@example.com")})

	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_RequestResetPassword_InvalidParams(t *testing.T) {
	_, err := newUserServiceWithRedis(&mock.MockUserRepo{}, &mock.MockUserRedis{}).
		RequestResetPassword(context.Background(), domain.RequestResetPasswordParams{})

	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestUserService_RequestResetPassword_NotifierError(t *testing.T) {
	repo := &mock.MockUserRepo{
		UserExistsFunc: func(_ context.Context, _ *domain.UserExistsParams) (bool, error) {
			return true, nil
		},
	}
	notifier := &mock.MockNotifier{
		NotifyResetPasswordFunc: func(_ context.Context, _, _ string) error {
			return domain.ErrNotFound
		},
	}

	_, err := newUserService(repo, &mock.MockPhotoStorage{}, &mock.MockUserRedis{}, notifier).
		RequestResetPassword(context.Background(), domain.RequestResetPasswordParams{Email: ptr("alice@example.com")})

	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserService_VerifyEmailAccount_ResetsEmail(t *testing.T) {
	var got *domain.ResetEmailParams

	repo := &mock.MockUserRepo{
		ConfirmEmailResetFunc: func(_ context.Context, params *domain.ResetEmailParams) (*domain.User, error) {
			got = params
			return &domain.User{
				Username: params.Username,
				Email:    params.NewEmail,
				UID:      "uid-123",
				Role:     "buyer",
				IsActive: true,
			}, nil
		},
	}
	redis := &mock.MockUserRedis{
		VerifiedFunc: func(_ context.Context, entryKey, token string) (*domain.VerifiedState, error) {
			require.Equal(t, repoRedis.VerifyEmailEnterKey, entryKey)
			require.Equal(t, "verify-token", token)
			return &domain.VerifiedState{Email: "new@example.com", Username: "alice"}, nil
		},
	}

	tokens, err := newUserServiceWithRedis(repo, redis).VerifyEmailAccount(context.Background(), "verify-token")

	require.NoError(t, err)
	require.NotNil(t, tokens)
	require.NotEmpty(t, tokens.AccessToken)
	require.Equal(t, &domain.ResetEmailParams{Username: "alice", NewEmail: "new@example.com"}, got)
}

func TestUserService_VerifyEmailAccount_VerifiesEmail(t *testing.T) {
	var gotEmail string

	repo := &mock.MockUserRepo{
		VerifyEmailFunc: func(_ context.Context, params *domain.VerifyEmailAccount) (*domain.User, error) {
			require.NotNil(t, params.Email)
			gotEmail = *params.Email
			return &domain.User{
				Username: "alice",
				Email:    gotEmail,
				UID:      "uid-123",
				Role:     "buyer",
				IsActive: true,
			}, nil
		},
	}
	redis := &mock.MockUserRedis{
		VerifiedFunc: func(_ context.Context, _, _ string) (*domain.VerifiedState, error) {
			return &domain.VerifiedState{Email: "alice@example.com"}, nil
		},
	}

	tokens, err := newUserServiceWithRedis(repo, redis).VerifyEmailAccount(context.Background(), "verify-token")

	require.NoError(t, err)
	require.NotNil(t, tokens)
	require.Equal(t, "alice@example.com", gotEmail)
}

func TestUserService_VerifyEmailAccount_InvalidToken(t *testing.T) {
	_, err := newUserServiceWithRedis(&mock.MockUserRepo{}, &mock.MockUserRedis{}).
		VerifyEmailAccount(context.Background(), "expired-token")

	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestResetEmail_RequestApproveAndReset(t *testing.T) {
	ctx := context.Background()
	var got *domain.ResetEmailParams
	type redisKey struct{ namespace, id string }
	codes := make(map[redisKey]*domain.CodeState)
	verified := map[redisKey]*domain.VerifiedState{
		{repoRedis.VerifyEmailEnterKey, "verification-token"}: {Email: "old@example.com"},
	}
	var deliveredCode string

	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, params *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{Username: *params.Username}, nil
		},
		ConfirmEmailResetFunc: func(_ context.Context, params *domain.ResetEmailParams) (*domain.User, error) {
			got = params
			return &domain.User{Username: params.Username, Email: params.NewEmail}, nil
		},
	}
	redis := &mock.MockUserRedis{
		SetCodeFunc: func(_ context.Context, namespace, id string, state *domain.CodeState) error {
			codes[redisKey{namespace, id}] = state
			return nil
		},
		CodeFunc: func(_ context.Context, namespace, id string) (*domain.CodeState, error) {
			if state := codes[redisKey{namespace, id}]; state != nil {
				return state, nil
			}
			return nil, domain.ErrRedisNotFound
		},
		SetVerifiedFunc: func(_ context.Context, namespace, token string, state *domain.VerifiedState) error {
			verified[redisKey{namespace, token}] = state
			return nil
		},
		VerifiedFunc: func(_ context.Context, namespace, token string) (*domain.VerifiedState, error) {
			if state := verified[redisKey{namespace, token}]; state != nil {
				return state, nil
			}
			return nil, domain.ErrRedisNotFound
		},
		DelVerifiedFunc: func(_ context.Context, namespace, token string) error {
			delete(verified, redisKey{namespace, token})
			return nil
		},
	}
	notifier := &mock.MockNotifier{
		NotifyEmailConfirmationFunc: func(_ context.Context, code, email string) error {
			require.Equal(t, "new@example.com", email)
			deliveredCode = code
			return nil
		},
	}
	users := newUserService(repo, &mock.MockPhotoStorage{}, redis, notifier)
	notifications := NewNotificationService(redis, repo, nil, zap.NewNop())

	ok, err := users.ResetEmail(ctx, "verification-token")
	require.ErrorIs(t, err, domain.ErrInvalidToken)
	require.False(t, ok)
	require.Nil(t, got)

	operationID, err := users.RequestResetEmail(ctx, "alice", "new@example.com")
	require.NoError(t, err)
	token, err := notifications.ApproveCode(ctx, repoRedis.VerifyEmailEnterKey, operationID, deliveredCode)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	ok, err = users.ResetEmail(ctx, token)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, &domain.ResetEmailParams{Username: "alice", NewEmail: "new@example.com"}, got)

	ok, err = users.ResetEmail(ctx, token)
	require.ErrorIs(t, err, domain.ErrInvalidToken)
	require.False(t, ok)
}

func TestUserService_ResetEmail_InvalidToken(t *testing.T) {
	ok, err := newUserServiceWithRedis(&mock.MockUserRepo{}, &mock.MockUserRedis{}).
		ResetEmail(context.Background(), "expired-token")

	require.ErrorIs(t, err, domain.ErrInvalidToken)
	require.False(t, ok)
}

func TestUserService_UploadAvatar_OK(t *testing.T) {
	data := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00, 0x00}
	var (
		deletedKey      string
		uploadedName    string
		uploadedType    string
		uploadedSize    int64
		uploadedContent []byte
	)

	storage := &mock.MockPhotoStorage{
		DeleteFunc: func(_ context.Context, objectName string) error {
			deletedKey = objectName
			return nil
		},
		UploadFunc: func(_ context.Context, objectName, contentType string, reader io.Reader, size int64) (string, error) {
			uploadedName, uploadedType, uploadedSize = objectName, contentType, size
			var err error
			uploadedContent, err = io.ReadAll(reader)
			return "http://localhost:9000/servicehub-avatars/" + objectName, err
		},
	}
	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, _ *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{
				Username:  "alice",
				FullName:  "Alice",
				AvatarURL: "http://localhost:9000/servicehub-avatars/avatars/users/alice/old.png",
			}, nil
		},
		UpdateFunc: func(_ context.Context, params *domain.UpdateUserParams) (*domain.User, error) {
			return &domain.User{
				Username:  params.Username,
				FullName:  params.FullName,
				AvatarURL: params.AvatarURL,
			}, nil
		},
	}

	user, err := newUserService(repo, storage, &mock.MockUserRedis{}, &mock.MockNotifier{}).
		UploadAvatar(context.Background(), domain.UploadAvatarParams{Username: "alice", FileData: data})

	require.NoError(t, err)
	require.Equal(t, "avatars/users/alice/old.png", deletedKey)
	require.Contains(t, uploadedName, "avatars/users/alice/")
	require.Equal(t, "image/png", uploadedType)
	require.Equal(t, int64(len(data)), uploadedSize)
	require.Equal(t, data, uploadedContent)
	require.Equal(t, "http://localhost:9000/servicehub-avatars/"+uploadedName, user.AvatarURL)
}

func TestUserService_UploadAvatar_InvalidFileType(t *testing.T) {
	storage := &mock.MockPhotoStorage{
		UploadFunc: func(_ context.Context, _, _ string, _ io.Reader, _ int64) (string, error) {
			t.Fatal("upload must not be called for an invalid file")
			return "", nil
		},
	}

	_, err := newUserService(&mock.MockUserRepo{}, storage, &mock.MockUserRedis{}, &mock.MockNotifier{}).
		UploadAvatar(context.Background(), domain.UploadAvatarParams{
			Username: "alice",
			FileData: []byte("not an image"),
		})

	require.ErrorIs(t, err, domain.ErrInvalidFileType)
}

func TestUserService_UploadAvatar_FileTooLarge(t *testing.T) {
	data := make([]byte, maxAvatarSize+1)
	copy(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})

	_, err := newUserService(&mock.MockUserRepo{}, &mock.MockPhotoStorage{}, &mock.MockUserRedis{}, &mock.MockNotifier{}).
		UploadAvatar(context.Background(), domain.UploadAvatarParams{Username: "alice", FileData: data})

	require.ErrorIs(t, err, domain.ErrFileTooLarge)
}

func TestUserService_ResetPassword_OK(t *testing.T) {
	oldHash, err := hash.CreateHashPassword("oldpassword123")
	require.NoError(t, err)

	var updated *domain.UpdateUserPasswordParams
	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, _ *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{Username: "alice", Email: "alice@example.com", PasswordHash: oldHash}, nil
		},
		UpdatePasswordFunc: func(_ context.Context, params *domain.UpdateUserPasswordParams) (*domain.User, error) {
			updated = params
			return &domain.User{Username: params.Username}, nil
		},
	}
	redis := &mock.MockUserRedis{
		VerifiedFunc: func(_ context.Context, _, _ string) (*domain.VerifiedState, error) {
			return &domain.VerifiedState{Username: "alice", Email: "alice@example.com"}, nil
		},
	}

	ok, err := newUserServiceWithRedis(repo, redis).ResetPassword(context.Background(), "reset-token", "newpassword456")

	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, updated)
	require.Equal(t, "alice", updated.Username)
	ok, err = hash.VerifyPassword("newpassword456", updated.NewPassword)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestUserService_ResetPassword_TooSimilarPasswords(t *testing.T) {
	oldHash, err := hash.CreateHashPassword("samepassword123")
	require.NoError(t, err)

	repo := &mock.MockUserRepo{
		GetUserFunc: func(_ context.Context, _ *domain.GetUserParams) (*domain.User, error) {
			return &domain.User{Username: "alice", PasswordHash: oldHash}, nil
		},
	}
	redis := &mock.MockUserRedis{
		VerifiedFunc: func(_ context.Context, _, _ string) (*domain.VerifiedState, error) {
			return &domain.VerifiedState{Username: "alice"}, nil
		},
	}

	ok, err := newUserServiceWithRedis(repo, redis).ResetPassword(context.Background(), "reset-token", "samepassword123")

	require.ErrorIs(t, err, domain.ErrTooSimilarPasswords)
	require.False(t, ok)
}

func TestUserService_ResetPassword_InvalidToken(t *testing.T) {
	ok, err := newUserServiceWithRedis(&mock.MockUserRepo{}, &mock.MockUserRedis{}).
		ResetPassword(context.Background(), "expired-token", "newpassword456")

	require.ErrorIs(t, err, domain.ErrNotFound)
	require.False(t, ok)
}

func TestUserService_UpdateProfile_OK(t *testing.T) {
	var got *domain.UpdateUserParams
	repo := &mock.MockUserRepo{
		UpdateFunc: func(_ context.Context, params *domain.UpdateUserParams) (*domain.User, error) {
			got = params
			return &domain.User{Username: params.Username, FullName: params.FullName}, nil
		},
	}

	user, err := newUserServiceWithRedis(repo, &mock.MockUserRedis{}).UpdateProfile(context.Background(), &domain.UpdateUserParams{
		Username: "alice",
		FullName: "Alice Smith",
	})

	require.NoError(t, err)
	require.Equal(t, &domain.UpdateUserParams{Username: "alice", FullName: "Alice Smith"}, got)
	require.Equal(t, "Alice Smith", user.FullName)
}

func ptr[T any](value T) *T {
	return &value
}
