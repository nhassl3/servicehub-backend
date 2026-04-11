package domain

import (
	"context"
	"encoding/json"
	"time"
)

// Moderation is a domain representation of a row in the `moderation` table.
// It links a product currently under review to the admin who claimed it.
type Moderation struct {
	ID,
	ProductID,
	AdminID string
	Active bool
	CreatedAt,
	UpdatedAt time.Time
}

// QueueRow is the unified row type returned by both Queue() and MyReviews().
// Moderation is nil when the product is in the queue but no admin has
// claimed it yet (left join produced NULLs on the m.* columns).
type QueueRow struct {
	Product       Product
	Moderation    *Moderation
	AdminUsername string
}

// ModerationStats aggregates dashboard counters surfaced via /moderation/stats.
type ModerationStats struct {
	TotalPending  int32 // moderation.status='draft', moderation.active=true, moderation.admin_id=ID
	TotalApproved int32 // moderation.status='active', moderation.active=false, moderation.admin_id=ID
	TotalRejected int32 // moderation.status='inactive', moderation.active=false, moderation.amdin_id=ID
}

// ModerationLock is the value persisted in Redis under
// `moderation:lock:<product_id>` while a moderator is reviewing a product.
type ModerationLock struct {
	AdminUsername string    `json:"admin_username"`
	ClaimedAt     time.Time `json:"claimed_at"`
}

// MarshalBinary lets go-redis serialize the lock value as JSON.
func (m *ModerationLock) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

// UnmarshalBinary lets go-redis deserialize the lock value from JSON.
func (m *ModerationLock) UnmarshalBinary(data []byte) error {
	if m == nil {
		return ErrRedisNotFound
	}
	return json.Unmarshal(data, m)
}

type CreateModerationParams struct {
	ProductID,
	AdminID string
	Active bool
}

type GetModerationParams struct {
	ID string
}

type ListModerationParams struct {
	AdminID string
	Limit   int32
	Offset  int32
}

type ListParams struct {
	Limit,
	Offset int32
}

type ClaimProductParams struct {
	ProductID string
	AdminID   string
}

type ReleaseProductParams struct {
	ProductID string
	Status    string
}

//go:generate mockgen -source=moderation.go -destination=../repository/mock/moderation_repo_mock.go -package=mockrepo
type ModerationRepository interface {
	CreateModeration(ctx context.Context, params CreateModerationParams) (*Moderation, error)
	GetModeration(ctx context.Context, params GetModerationParams) (*Moderation, error)
	MyReviews(ctx context.Context, params ListModerationParams) ([]*QueueRow, int64, error)
	Queue(ctx context.Context, params ListParams) ([]*QueueRow, int64, error)
	ClaimProduct(ctx context.Context, params ClaimProductParams) (*Moderation, error)
	ReleaseProduct(ctx context.Context, params ReleaseProductParams) error
	Stats(ctx context.Context, adminID string) (*ModerationStats, error)
}

// ModerationLockRedis is the fast-path source of truth for "who is reviewing
// product X right now". The TTL guarantees that crashed/forgotten claims are
// released automatically. Postgres still owns the durable state.
//
// Key: `moderation:lock:<product_id>`.
type ModerationLockRedis interface {
	// Acquire tries to take the lock atomically (SET NX). Returns acquired=true
	// and the freshly stored lock on success. On conflict, acquired=false and
	// the existing lock value (if readable) so callers can show the owner.
	Acquire(ctx context.Context, productID, adminUsername string) (acquired bool, lock *ModerationLock, err error)

	// Get returns the current lock for product, or ErrRedisNotFound if absent.
	Get(ctx context.Context, productID string) (*ModerationLock, error)

	// Release deletes the lock IF the caller is the current owner. Idempotent
	// — releasing a missing lock is not an error.
	Release(ctx context.Context, productID, adminUsername string) error

	// Refresh extends the TTL while the moderator is still reviewing.
	// Only the current owner can refresh.
	Refresh(ctx context.Context, productID, adminUsername string) error
}
