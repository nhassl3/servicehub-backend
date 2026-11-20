package mock

import (
	"time"

	"github.com/nhassl3/servicehub-backend/pkg/auth"
)

// ─── Mock TokenManager ────────────────────────────────────────────────────────

type MockTokenManager struct {
	CreateErr error
	VerifyErr error
}

func (m *MockTokenManager) CreateRefreshToken(_ *auth.CreateTokenParams) (string, *auth.Payload, error) {
	if m.CreateErr != nil {
		return "", nil, m.CreateErr
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

func (m *MockTokenManager) CreateToken(_ *auth.CreateTokenParams) (string, error) {
	if m.CreateErr != nil {
		return "", m.CreateErr
	}
	return "test-access-token", nil
}

func (m *MockTokenManager) VerifyToken(_ string) (*auth.Payload, error) {
	if m.VerifyErr != nil {
		return nil, m.VerifyErr
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
