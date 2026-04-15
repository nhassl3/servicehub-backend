package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/redis/go-redis/v9"
)

const (
	adminKeyPrefix = "admin:"
	lockPrefix     = "moderation:lock:"
)

// releaseScript atomically deletes the lock IF the caller is still the owner.
// We compare on admin_id, parsing the JSON value via cjson which is shipped
// with every supported Redis build.
const releaseScript = `
local v = redis.call("GET", KEYS[1])
if not v then return 0 end
local ok, decoded = pcall(cjson.decode, v)
if not ok then return 0 end
if decoded.admin_username == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`

// refreshScript extends the TTL only if the caller is still the owner.
const refreshScript = `
local v = redis.call("GET", KEYS[1])
if not v then return 0 end
local ok, decoded = pcall(cjson.decode, v)
if not ok then return 0 end
if decoded.admin_username == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`

type AdminRedis struct {
	client     *redis.Client
	profileTTL time.Duration
	claimTTL   time.Duration
}

func NewAdminRedis(client *redis.Client, profileTTL, claimTTL time.Duration) *AdminRedis {
	return &AdminRedis{
		client:     client,
		profileTTL: profileTTL,
		claimTTL:   claimTTL,
	}
}

// ── Profile cache ────────────────────────────────────────────────────────────

func (a *AdminRedis) Profile(ctx context.Context, username string) (*domain.Admin, error) {
	var admin domain.Admin
	if err := a.client.Get(ctx, adminKeyPrefix+username).Scan(&admin); err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrRedisNotFound
		}
		return nil, err
	}
	return &admin, nil
}

func (a *AdminRedis) SetProfile(ctx context.Context, admin *domain.Admin) error {
	return a.client.Set(ctx, adminKeyPrefix+admin.Username, admin, a.profileTTL).Err()
}

// ── Moderation lock ──────────────────────────────────────────────────────────

// Acquire tries to take the lock. On conflict, it fetches the existing owner
// so the caller can show "locked by <username>" without an extra round trip.
func (a *AdminRedis) Acquire(ctx context.Context, productID, adminUsername string) (bool, *domain.ModerationLock, error) {
	lock := &domain.ModerationLock{
		AdminUsername: adminUsername,
		ClaimedAt:     time.Now().UTC(),
	}
	payload, err := json.Marshal(lock)
	if err != nil {
		return false, nil, err
	}

	if err := a.client.SetArgs(ctx, lockPrefix+productID, payload, redis.SetArgs{
		Mode: "NX",
		Get:  false,
		TTL:  a.claimTTL,
	}).Err(); err != nil {
		// Someone else holds the lock — try to surface its owner.
		current, getErr := a.Get(ctx, productID)
		if getErr != nil && !errors.Is(getErr, domain.ErrRedisNotFound) {
			return false, nil, getErr
		}
		return false, current, nil
	}

	return true, lock, nil
}

func (a *AdminRedis) Get(ctx context.Context, productID string) (*domain.ModerationLock, error) {
	raw, err := a.client.Get(ctx, lockPrefix+productID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrRedisNotFound
		}
		return nil, err
	}
	lock := &domain.ModerationLock{}
	if err := json.Unmarshal(raw, lock); err != nil {
		return nil, err
	}
	return lock, nil
}

// Release deletes the lock IF the caller is still its owner. Idempotent —
// missing lock or different owner are not errors.
func (a *AdminRedis) Release(ctx context.Context, productID, adminUsername string) error {
	return a.client.Eval(ctx, releaseScript, []string{lockPrefix + productID}, adminUsername).Err()
}

// Refresh extends the TTL while the caller still holds the lock.
func (a *AdminRedis) Refresh(ctx context.Context, productID, adminUsername string) error {
	ms := a.claimTTL.Milliseconds()
	return a.client.Eval(ctx, refreshScript, []string{lockPrefix + productID}, adminUsername, ms).Err()
}
