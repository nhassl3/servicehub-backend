package mock

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/domain"
)

// ─── Mock UserRedis ──────────────────────────────────────────────────────────

type MockUserRedis struct {
	ProfileFunc          func(ctx context.Context, username string) (*domain.User, error)
	SessionFunc          func(ctx context.Context, username string) (*domain.Session, error)
	AuthBlockFunc        func(ctx context.Context, clientIP string) (bool, float64, error)
	SetProfileFunc       func(ctx context.Context, user *domain.User) error
	SetSessionFunc       func(ctx context.Context, session *domain.Session) error
	SetAuthBlockFunc     func(ctx context.Context, clientIP string) error
	DelProfileFunc       func(ctx context.Context, username string) error
	DelSessionFunc       func(ctx context.Context, username string) error
	CodeExistsFunc       func(ctx context.Context, entryKey, email string) bool
	CodeFunc             func(ctx context.Context, entryKey, operationID string) (*domain.CodeState, error)
	SetCodeFunc          func(ctx context.Context, entryKey, operationID string, state *domain.CodeState) error
	VerifiedFunc         func(ctx context.Context, entryKey, token string) (*domain.VerifiedState, error)
	SetVerifiedFunc      func(ctx context.Context, entryKey, token string, state *domain.VerifiedState) error
	DelVerifiedFunc      func(ctx context.Context, entryKey, token string) error
	DelCodeFunc          func(ctx context.Context, entryKey, operationID string) error
	IncrCodeAttemptsFunc func(ctx context.Context, entryKey, operationID string) (int64, error)
	DelCodeAttemptsFunc  func(ctx context.Context, entryKey, operationID string) error
}

func (m *MockUserRedis) Profile(ctx context.Context, username string) (*domain.User, error) {
	if m.ProfileFunc != nil {
		return m.ProfileFunc(ctx, username)
	}
	return nil, domain.ErrRedisNotFound
}

func (m *MockUserRedis) Session(ctx context.Context, username string) (*domain.Session, error) {
	if m.SessionFunc != nil {
		return m.SessionFunc(ctx, username)
	}
	return nil, domain.ErrRedisNotFound
}

func (m *MockUserRedis) AuthBlock(ctx context.Context, clientIP string) (bool, float64, error) {
	if m.AuthBlockFunc != nil {
		return m.AuthBlockFunc(ctx, clientIP)
	}
	return false, 0, nil
}

func (m *MockUserRedis) SetProfile(ctx context.Context, user *domain.User) error {
	if m.SetProfileFunc != nil {
		return m.SetProfileFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRedis) SetSession(ctx context.Context, session *domain.Session) error {
	if m.SetSessionFunc != nil {
		return m.SetSessionFunc(ctx, session)
	}
	return nil
}

func (m *MockUserRedis) SetAuthBlock(ctx context.Context, clientIP string) error {
	if m.SetAuthBlockFunc != nil {
		return m.SetAuthBlockFunc(ctx, clientIP)
	}
	return nil
}

func (m *MockUserRedis) DelProfile(ctx context.Context, username string) error {
	if m.DelProfileFunc != nil {
		return m.DelProfileFunc(ctx, username)
	}
	return nil
}

func (m *MockUserRedis) DelSession(ctx context.Context, username string) error {
	if m.DelSessionFunc != nil {
		return m.DelSessionFunc(ctx, username)
	}
	return nil
}

func (m *MockUserRedis) CodeExists(ctx context.Context, entryKey, email string) bool {
	if m.CodeExistsFunc != nil {
		return m.CodeExistsFunc(ctx, entryKey, email)
	}
	return false
}

func (m *MockUserRedis) Code(ctx context.Context, entryKey, operationID string) (*domain.CodeState, error) {
	if m.CodeFunc != nil {
		return m.CodeFunc(ctx, entryKey, operationID)
	}
	return nil, domain.ErrRedisNotFound
}

func (m *MockUserRedis) SetCode(ctx context.Context, entryKey, operationID string, state *domain.CodeState) error {
	if m.SetCodeFunc != nil {
		return m.SetCodeFunc(ctx, entryKey, operationID, state)
	}
	return nil
}

func (m *MockUserRedis) Verified(ctx context.Context, entryKey, token string) (*domain.VerifiedState, error) {
	if m.VerifiedFunc != nil {
		return m.VerifiedFunc(ctx, entryKey, token)
	}
	return nil, domain.ErrRedisNotFound
}

func (m *MockUserRedis) SetVerified(ctx context.Context, entryKey, token string, state *domain.VerifiedState) error {
	if m.SetVerifiedFunc != nil {
		return m.SetVerifiedFunc(ctx, entryKey, token, state)
	}
	return nil
}

func (m *MockUserRedis) DelVerified(ctx context.Context, entryKey, token string) error {
	if m.DelVerifiedFunc != nil {
		return m.DelVerifiedFunc(ctx, entryKey, token)
	}
	return nil
}

func (m *MockUserRedis) DelCode(ctx context.Context, entryKey, operationID string) error {
	if m.DelCodeFunc != nil {
		return m.DelCodeFunc(ctx, entryKey, operationID)
	}
	return nil
}

func (m *MockUserRedis) IncrCodeAttempts(ctx context.Context, entryKey, operationID string) (int64, error) {
	if m.IncrCodeAttemptsFunc != nil {
		return m.IncrCodeAttemptsFunc(ctx, entryKey, operationID)
	}
	return 1, nil
}

func (m *MockUserRedis) DelCodeAttempts(ctx context.Context, entryKey, operationID string) error {
	if m.DelCodeAttemptsFunc != nil {
		return m.DelCodeAttemptsFunc(ctx, entryKey, operationID)
	}
	return nil
}
