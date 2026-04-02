package auth

import (
	"context"
	"time"
)

// TokenBlacklist defines the interface for token revocation storage.
// Implementations store JTI (JWT ID) with TTL matching the token's expiration.
type TokenBlacklist interface {
	// Blacklist adds a token's JTI to the blacklist until expiresAt.
	Blacklist(ctx context.Context, jti string, expiresAt time.Time) error
	// IsBlacklisted checks whether a token's JTI has been revoked.
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}
