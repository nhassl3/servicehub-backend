package redis

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	blacklistPrefix = "token_blacklist:"
	revokedValue    = "revoked"
)

// TokenBlacklist implements auth.TokenBlacklist using Redis SET with TTL.
type TokenBlacklist struct {
	client *redis.Client
}

// NewTokenBlacklist creates a Redis-backed token blacklist.
func NewTokenBlacklist(client *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{client: client}
}

// Blacklist stores the JTI in Redis with a TTL matching the token's remaining lifetime.
func (b *TokenBlacklist) Blacklist(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // token already expired, no need to blacklist
	}

	return b.client.Set(ctx, blacklistPrefix+jti, revokedValue, ttl).Err()
}

// IsBlacklisted checks whether the given JTI exists in the Redis blacklist.
func (b *TokenBlacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	res, err := b.client.Get(ctx, blacklistPrefix+jti).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return res == revokedValue, nil
}
