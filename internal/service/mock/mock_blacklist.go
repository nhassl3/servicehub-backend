package mock

import (
	"context"
	"time"
)

// ─── Mock TokenBlacklist ──────────────────────────────────────────────────────

type MockBlacklist struct {
	Blacklisted map[string]bool
}

func NewMockBlacklist() *MockBlacklist {
	return &MockBlacklist{Blacklisted: make(map[string]bool)}
}

func (m *MockBlacklist) Blacklist(_ context.Context, jti string, _ time.Time) error {
	m.Blacklisted[jti] = true
	return nil
}

func (m *MockBlacklist) IsBlacklisted(_ context.Context, jti string) (bool, error) {
	return m.Blacklisted[jti], nil
}
