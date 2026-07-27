package auth

import "context"

// BlacklistedTokenManager is a decorator around TokenManager that checks
// token revocation status on every VerifyToken call via a TokenBlacklist.
type BlacklistedTokenManager struct {
	inner     TokenManager
	blacklist TokenBlacklist
}

// NewBlacklistedTokenManager wraps an existing TokenManager with blacklist verification.
func NewBlacklistedTokenManager(inner TokenManager, blacklist TokenBlacklist) *BlacklistedTokenManager {
	return &BlacklistedTokenManager{inner: inner, blacklist: blacklist}
}

func (m *BlacklistedTokenManager) CreateToken(username, uid, role, email string, isActive bool) (string, error) {
	return m.inner.CreateToken(username, uid, role, email, isActive)
}

func (m *BlacklistedTokenManager) CreateRefreshToken(username, uid, role, email string, isActive bool) (string, *Payload, error) {
	return m.inner.CreateRefreshToken(username, uid, role, email, isActive)
}

// VerifyToken delegates to the inner TokenManager and then checks the blacklist.
// Requires context — uses context.Background() as fallback since the interface is context-free.
func (m *BlacklistedTokenManager) VerifyToken(token string) (*Payload, error) {
	payload, err := m.inner.VerifyToken(token)
	if err != nil {
		return nil, err
	}

	revoked, err := m.blacklist.IsBlacklisted(context.Background(), payload.JTI)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, ErrTokenRevoked
	}

	return payload, nil
}
