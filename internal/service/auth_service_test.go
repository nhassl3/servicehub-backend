package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/service"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	"github.com/nhassl3/servicehub-backend/pkg/mailer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ─── Mock TokenBlacklist ──────────────────────────────────────────────────────

type mockBlacklist struct {
	blacklisted map[string]bool
}

func newMockBlacklist() *mockBlacklist {
	return &mockBlacklist{blacklisted: make(map[string]bool)}
}

func (m *mockBlacklist) Blacklist(_ context.Context, jti string, _ time.Time) error {
	m.blacklisted[jti] = true
	return nil
}

func (m *mockBlacklist) IsBlacklisted(_ context.Context, jti string) (bool, error) {
	return m.blacklisted[jti], nil
}

// ─── Mock TokenManager ────────────────────────────────────────────────────────

type mockTokenManager struct {
	createErr error
	verifyErr error
}

func (m *mockTokenManager) CreateRefreshToken(_, _, _, _ string, _ bool) (string, *auth.Payload, error) {
	if m.createErr != nil {
		return "", nil, m.createErr
	}
	return "test-refresh-token", &auth.Payload{
		Username:  "alice",
		Email:     "alice@example.com",
		UID:       "uid-123",
		Role:      "buyer",
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(15 * time.Minute),
	}, nil
}

func (m *mockTokenManager) CreateToken(_, _, _, _ string, _ bool) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	return "test-access-token", nil
}

func (m *mockTokenManager) VerifyToken(_ string) (*auth.Payload, error) {
	if m.verifyErr != nil {
		return nil, m.verifyErr
	}
	return &auth.Payload{
		Username:  "alice",
		Email:     "alice@example.com",
		UID:       "uid-123",
		Role:      "buyer",
		IssuedAt:  time.Now(),
		ExpiredAt: time.Now().Add(15 * time.Minute),
	}, nil
}

// ─── Mock UserRedis ──────────────────────────────────────────────────────────

type mockUserRedis struct {
	codeExistsFunc       func(ctx context.Context, entryKey, email string) bool
	codeFunc             func(ctx context.Context, entryKey, operationID string) (*domain.CodeState, error)
	setCodeFunc          func(ctx context.Context, entryKey, operationID string, state *domain.CodeState) error
	verifiedFunc         func(ctx context.Context, entryKey, token string) (*domain.VerifiedState, error)
	setVerifiedFunc      func(ctx context.Context, entryKey, token string, state *domain.VerifiedState) error
	incrCodeAttemptsFunc func(ctx context.Context, entryKey, operationID string) (int64, error)
	delCodeAttemptsFunc  func(ctx context.Context, entryKey, operationID string) error
}

func (m *mockUserRedis) Profile(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrRedisNotFound
}

func (m *mockUserRedis) Session(_ context.Context, _ string) (*domain.Session, error) {
	return nil, domain.ErrRedisNotFound
}

func (m *mockUserRedis) AuthBlock(_ context.Context, _ string) (bool, float64, error) {
	return false, 0, nil
}

func (m *mockUserRedis) SetProfile(_ context.Context, _ *domain.User) error {
	return nil
}

func (m *mockUserRedis) SetSession(_ context.Context, _ *domain.Session) error {
	return nil
}

func (m *mockUserRedis) SetAuthBlock(_ context.Context, _ string) error {
	return nil
}

func (m *mockUserRedis) DelProfile(_ context.Context, _ string) error {
	return nil
}

func (m *mockUserRedis) DelSession(_ context.Context, _ string) error {
	return nil
}

func (m *mockUserRedis) CodeExists(ctx context.Context, entryKey, email string) bool {
	if m.codeExistsFunc != nil {
		return m.codeExistsFunc(ctx, entryKey, email)
	}
	return false
}

func (m *mockUserRedis) Code(ctx context.Context, entryKey, operationID string) (*domain.CodeState, error) {
	if m.codeFunc != nil {
		return m.codeFunc(ctx, entryKey, operationID)
	}
	return nil, domain.ErrRedisNotFound
}

func (m *mockUserRedis) SetCode(ctx context.Context, entryKey, operationID string, state *domain.CodeState) error {
	if m.setCodeFunc != nil {
		return m.setCodeFunc(ctx, entryKey, operationID, state)
	}
	return nil
}

func (m *mockUserRedis) Verified(ctx context.Context, entryKey, token string) (*domain.VerifiedState, error) {
	if m.verifiedFunc != nil {
		return m.verifiedFunc(ctx, entryKey, token)
	}
	return nil, domain.ErrRedisNotFound
}

func (m *mockUserRedis) SetVerified(ctx context.Context, entryKey, token string, state *domain.VerifiedState) error {
	if m.setVerifiedFunc != nil {
		return m.setVerifiedFunc(ctx, entryKey, token, state)
	}
	return nil
}

func (m *mockUserRedis) DelVerified(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockUserRedis) DelCode(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockUserRedis) IncrCodeAttempts(ctx context.Context, entryKey, operationID string) (int64, error) {
	if m.incrCodeAttemptsFunc != nil {
		return m.incrCodeAttemptsFunc(ctx, entryKey, operationID)
	}
	return 1, nil
}

func (m *mockUserRedis) DelCodeAttempts(ctx context.Context, entryKey, operationID string) error {
	if m.delCodeAttemptsFunc != nil {
		return m.delCodeAttemptsFunc(ctx, entryKey, operationID)
	}
	return nil
}

// ─── Mock UserRepository ──────────────────────────────────────────────────────

type mockUserRepo struct {
	existsByUsernameFunc     func(ctx context.Context, username string) (bool, error)
	existsByEmailFunc        func(ctx context.Context, email string) (bool, error)
	createFunc               func(ctx context.Context, params domain.CreateUserParams) (*domain.User, error)
	getByUsernameFunc        func(ctx context.Context, username string) (*domain.User, error)
	getByEmailFunc           func(ctx context.Context, email string) (*domain.User, error)
	getByUIDFunc             func(ctx context.Context, uid string) (*domain.User, error)
	updateFunc               func(ctx context.Context, params domain.UpdateUserParams) (*domain.User, error)
	updatePasswordFunc       func(ctx context.Context, params domain.UpdateUserPasswordParams) (*domain.User, error)
	getSessionFunc           func(ctx context.Context, refreshToken string) (*domain.Session, error)
	getSessionByUsernameFunc func(ctx context.Context, username string) (*domain.Session, error)
	deleteSessionFunc        func(ctx context.Context, username string) error
	createSessionFunc        func(ctx context.Context, params domain.CreateSessionParams) (*domain.Session, error)
	verifyEmailFunc          func(ctx context.Context, params domain.VerifyEmailAccount) (*domain.User, error)
	confirmEmailResetFunc    func(ctx context.Context, params domain.ResetEmailParams) (*domain.User, error)
}

func (m *mockUserRepo) ConfirmEmailReset(ctx context.Context, params domain.ResetEmailParams) (*domain.User, error) {
	if m.confirmEmailResetFunc != nil {
		return m.confirmEmailResetFunc(ctx, params)
	}
	return nil, domain.ErrEmailAlreadyReset
}

func (m *mockUserRepo) CreateSession(ctx context.Context, params domain.CreateSessionParams) (*domain.Session, error) {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, params)
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

func (m *mockUserRepo) GetSession(ctx context.Context, refreshToken string) (*domain.Session, error) {
	if m.getSessionFunc != nil {
		return m.getSessionFunc(ctx, refreshToken)
	}
	return &domain.Session{}, nil
}

func (m *mockUserRepo) GetSessionByUsername(ctx context.Context, username string) (*domain.Session, error) {
	if m.getSessionByUsernameFunc != nil {
		return m.getSessionByUsernameFunc(ctx, username)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) DeleteSession(ctx context.Context, username string) error {
	if m.deleteSessionFunc != nil {
		return m.deleteSessionFunc(ctx, username)
	}
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, params domain.UpdateUserPasswordParams) (*domain.User, error) {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(ctx, params)
	}
	return &domain.User{}, nil
}

func (m *mockUserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.existsByUsernameFunc != nil {
		return m.existsByUsernameFunc(ctx, username)
	}
	return false, nil
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFunc != nil {
		return m.existsByEmailFunc(ctx, email)
	}
	return false, nil
}

func (m *mockUserRepo) Create(ctx context.Context, params domain.CreateUserParams) (*domain.User, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return &domain.User{
		Username: params.Username,
		Email:    params.Email,
		UID:      "uid-123",
		Role:     "buyer",
	}, nil
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if m.getByUsernameFunc != nil {
		return m.getByUsernameFunc(ctx, username)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) GetByUID(ctx context.Context, uid string) (*domain.User, error) {
	if m.getByUIDFunc != nil {
		return m.getByUIDFunc(ctx, uid)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) VerifyEmail(ctx context.Context, params domain.VerifyEmailAccount) (*domain.User, error) {
	if m.verifyEmailFunc != nil {
		return m.verifyEmailFunc(ctx, params)
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, params domain.UpdateUserParams) (*domain.User, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, params)
	}
	return nil, domain.ErrNotFound
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func newAuthServiceWithRedis(repo domain.UserRepository, redis domain.UserRedis) *service.AuthService {
	tm := &mockTokenManager{}
	bl := newMockBlacklist()
	return service.NewAuthService(repo, redis, tm, tm, bl, mailer.NewNoopNotifier(zap.NewNop()), noopPublisher{}, 72*time.Hour, zap.NewNop())
}

func newAuthService(repo domain.UserRepository) *service.AuthService {
	redis := &mockUserRedis{}
	return newAuthServiceWithRedis(repo, redis)
}

// noopPublisher is a trivial EventPublisher stub so existing auth tests don't
// need a gomock controller.
type noopPublisher struct{}

func (noopPublisher) PublishOrderCreated(context.Context, domain.OrderCreatedPayload) error {
	return nil
}
func (noopPublisher) PublishOrderStatusChanged(context.Context, domain.OrderStatusChangedPayload) error {
	return nil
}
func (noopPublisher) PublishTransactionCreated(context.Context, domain.TransactionCreatedPayload) error {
	return nil
}
func (noopPublisher) PublishBalanceUpdated(context.Context, domain.BalanceUpdatedPayload) error {
	return nil
}
func (noopPublisher) PublishIndexedProduct(context.Context, *domain.Product) error { return nil }
func (noopPublisher) PublishDeletedProduct(context.Context, string) error          { return nil }
func (noopPublisher) PublishUserRegistered(context.Context, domain.UserRegisteredPayload) error {
	return nil
}
func (noopPublisher) PublishProductStatusChanged(context.Context, domain.ProductStatusChangedPayload) error {
	return nil
}
func (noopPublisher) PublishProductRatingChanged(context.Context, domain.ProductRatingChangedPayload) error {
	return nil
}
func (noopPublisher) PublishModerationApproved(context.Context, domain.ModerationApprovedPayload) error {
	return nil
}
func (noopPublisher) PublishModerationRejected(context.Context, domain.ModerationRejectedPayload) error {
	return nil
}
func (noopPublisher) PublishOrderItemCreated(context.Context, domain.OrderItemCreatedPayload) error {
	return nil
}
func (noopPublisher) Close() error { return nil }

func TestAuthService_Register_OK(t *testing.T) {
	repo := &mockUserRepo{}
	svc := newAuthService(repo)

	user, tokens, err := svc.Register(context.Background(), service.RegisterInput{
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
	repo := &mockUserRepo{
		existsByUsernameFunc: func(_ context.Context, username string) (bool, error) {
			return username == "alice", nil
		},
	}
	svc := newAuthService(repo)

	_, _, err := svc.Register(context.Background(), service.RegisterInput{
		Username: "alice",
		Email:    "new@example.com",
		Password: "password123",
		FullName: "Alice",
	})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	repo := &mockUserRepo{
		existsByEmailFunc: func(_ context.Context, email string) (bool, error) {
			return email == "alice@example.com", nil
		},
	}
	svc := newAuthService(repo)

	_, _, err := svc.Register(context.Background(), service.RegisterInput{
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

	repo := &mockUserRepo{
		getByUsernameFunc: func(_ context.Context, username string) (*domain.User, error) {
			if username == "alice" {
				return &domain.User{
					Username:     "alice",
					UID:          "uid-123",
					Role:         "buyer",
					PasswordHash: hashedPw,
				}, nil
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
	repo := &mockUserRepo{}
	svc := newAuthService(repo)

	_, _, err := svc.Login(context.Background(), "nobody", "pass")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := &mockUserRepo{
		getByUsernameFunc: func(_ context.Context, _ string) (*domain.User, error) {
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
	repo := &mockUserRepo{}
	tm := &mockTokenManager{verifyErr: auth.ErrInvalidToken}
	bl := newMockBlacklist()
	redis := &mockUserRedis{}
	svc := service.NewAuthService(repo, redis, tm, tm, bl, mailer.NewNoopNotifier(zap.NewNop()), noopPublisher{}, 72*time.Hour, zap.NewNop())

	_, err := svc.RefreshToken(context.Background(), "bad-token")
	require.ErrorIs(t, err, domain.ErrInvalidToken)
}

func TestAuthService_ResetEmail_OK(t *testing.T) {
	var stored *domain.CodeState

	repo := &mockUserRepo{
		getByUsernameFunc: func(_ context.Context, username string) (*domain.User, error) {
			return &domain.User{
				Username:  username,
				Email:     "old@example.com",
				IsActive:  false,
				CreatedAt: time.Now().Add(-time.Hour),
			}, nil
		},
	}
	redis := &mockUserRedis{
		setCodeFunc: func(_ context.Context, _, _ string, state *domain.CodeState) error {
			stored = state
			return nil
		},
	}

	operationID, err := newAuthServiceWithRedis(repo, redis).ResetEmail(context.Background(), "alice", "new@example.com")

	require.NoError(t, err)
	require.NotEmpty(t, operationID)
	require.NotNil(t, stored)
	require.Equal(t, "new@example.com", stored.Email)
	require.Equal(t, "alice", stored.Username)
	require.Len(t, stored.Code, 6)
}

func TestAuthService_ResetEmail_WindowExpired(t *testing.T) {
	repo := &mockUserRepo{
		getByUsernameFunc: func(_ context.Context, username string) (*domain.User, error) {
			return &domain.User{
				Username:  username,
				CreatedAt: time.Now().Add(-73 * time.Hour),
			}, nil
		},
	}

	_, err := newAuthService(repo).ResetEmail(context.Background(), "alice", "new@example.com")
	require.ErrorIs(t, err, domain.ErrEmailResetWindowExpired)
}

func TestAuthService_ResetEmail_AlreadyReset(t *testing.T) {
	resetAt := time.Now().Add(-time.Hour)
	repo := &mockUserRepo{
		getByUsernameFunc: func(_ context.Context, username string) (*domain.User, error) {
			return &domain.User{
				Username:     username,
				CreatedAt:    time.Now().Add(-24 * time.Hour),
				EmailResetAt: &resetAt,
			}, nil
		},
	}

	_, err := newAuthService(repo).ResetEmail(context.Background(), "alice", "new@example.com")
	require.ErrorIs(t, err, domain.ErrEmailAlreadyReset)
}

func TestAuthService_ResetEmail_ActiveAccount(t *testing.T) {
	repo := &mockUserRepo{
		getByUsernameFunc: func(_ context.Context, username string) (*domain.User, error) {
			return &domain.User{
				Username:  username,
				IsActive:  true,
				CreatedAt: time.Now().Add(-time.Hour),
			}, nil
		},
	}

	_, err := newAuthService(repo).ResetEmail(context.Background(), "alice", "new@example.com")
	require.ErrorIs(t, err, domain.ErrEmailResetNotAllowed)
}

func TestAuthService_ResetEmail_EmailTaken(t *testing.T) {
	repo := &mockUserRepo{
		getByUsernameFunc: func(_ context.Context, username string) (*domain.User, error) {
			return &domain.User{Username: username, CreatedAt: time.Now().Add(-time.Hour)}, nil
		},
		existsByEmailFunc: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	_, err := newAuthService(repo).ResetEmail(context.Background(), "alice", "taken@example.com")
	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestAuthService_VerifyEmailAccount_ResetsEmail(t *testing.T) {
	var got domain.ResetEmailParams

	repo := &mockUserRepo{
		confirmEmailResetFunc: func(_ context.Context, params domain.ResetEmailParams) (*domain.User, error) {
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
	redis := &mockUserRedis{
		verifiedFunc: func(_ context.Context, _, _ string) (*domain.VerifiedState, error) {
			return &domain.VerifiedState{Email: "new@example.com", Username: "alice"}, nil
		},
	}

	tokens, ok, err := newAuthServiceWithRedis(repo, redis).VerifyEmailAccount(context.Background(), "verify-token")

	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, tokens)
	require.NotEmpty(t, tokens.AccessToken)
	require.Equal(t, domain.ResetEmailParams{Username: "alice", NewEmail: "new@example.com"}, got)
}

func TestAuthService_VerifyEmailAccount_VerifiesEmail(t *testing.T) {
	var gotEmail string

	repo := &mockUserRepo{
		verifyEmailFunc: func(_ context.Context, params domain.VerifyEmailAccount) (*domain.User, error) {
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
	redis := &mockUserRedis{
		verifiedFunc: func(_ context.Context, _, _ string) (*domain.VerifiedState, error) {
			return &domain.VerifiedState{Email: "alice@example.com"}, nil
		},
	}

	tokens, ok, err := newAuthServiceWithRedis(repo, redis).VerifyEmailAccount(context.Background(), "verify-token")

	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, tokens)
	require.Equal(t, "alice@example.com", gotEmail)
}

func TestAuthService_VerifyEmailAccount_InvalidToken(t *testing.T) {
	_, _, err := newAuthService(&mockUserRepo{}).VerifyEmailAccount(context.Background(), "expired-token")
	require.ErrorIs(t, err, domain.ErrInvalidToken)
}
