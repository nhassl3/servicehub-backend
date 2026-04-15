package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/nhassl3/servicehub-backend/internal/domain"
)

// ModerationService implements the admin-panel moderation workflow:
// queue -> claim -> approve/reject. Concurrency between admins is mediated
// by a Redis lock; Postgres remains the source of truth.
type ModerationService struct {
	moderationRepo domain.ModerationRepository
	adminRepo      domain.AdminRepository
	productRepo    domain.ProductRepository
	lock           domain.ModerationLockRedis
	adminRedis     domain.AdminRedis
}

func NewModerationService(
	moderationRepo domain.ModerationRepository,
	adminRepo domain.AdminRepository,
	productRepo domain.ProductRepository,
	lock domain.ModerationLockRedis,
	adminRedis domain.AdminRedis,
) *ModerationService {
	return &ModerationService{
		moderationRepo: moderationRepo,
		adminRepo:      adminRepo,
		productRepo:    productRepo,
		lock:           lock,
		adminRedis:     adminRedis,
	}
}

// resolveAdminID converts the authenticated username (from token) into an
// admins.id UUID, which is what the moderation table actually references.
func (s *ModerationService) resolveAdminID(ctx context.Context, username string) (string, error) {
	admin, err := s.adminRedis.Profile(ctx, username)
	if err != nil {
		admin, err = s.adminRepo.GetAdmin(ctx, domain.GetAdminProfileParams{Username: username})
		if err != nil {
			return "", fmt.Errorf("moderation_service.resolveAdminID: %w", err)
		}
		if errors.Is(err, domain.ErrRedisNotFound) {
			_ = s.adminRedis.SetProfile(ctx, admin)
		}
	}
	return admin.ID, nil
}

// ── Read paths ───────────────────────────────────────────────────────────────

func (s *ModerationService) Queue(ctx context.Context, params domain.ListParams) ([]*domain.QueueRow, int32, error) {
	rows, total, err := s.moderationRepo.Queue(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("moderation_service.Queue: %w", err)
	}
	return rows, int32(total), nil
}

func (s *ModerationService) MyReviews(ctx context.Context, username string, list domain.ListParams) ([]*domain.QueueRow, int32, error) {
	adminID, err := s.resolveAdminID(ctx, username)
	if err != nil {
		return nil, 0, err
	}
	rows, total, err := s.moderationRepo.MyReviews(ctx, domain.ListModerationParams{
		AdminID: adminID,
		Limit:   list.Limit,
		Offset:  list.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("moderation_service.MyReviews: %w", err)
	}
	return rows, int32(total), nil
}

// ── Lock-mediated mutations ──────────────────────────────────────────────────

// Claim grabs the Redis lock first; only the winner inserts/updates the
// moderation row. If Postgres fails after a successful lock, the lock is
// released to keep the system consistent.
func (s *ModerationService) Claim(ctx context.Context, productID, username string) (*domain.Moderation, error) {
	adminID, err := s.resolveAdminID(ctx, username)
	if err != nil {
		return nil, err
	}

	acquired, current, err := s.lock.Acquire(ctx, productID, username)
	if err != nil {
		return nil, fmt.Errorf("moderation_service.Claim: redis: %w", err)
	}
	if !acquired {
		// Conflict — surface the existing owner so the UI can show "locked by".
		ownerUsername := ""
		if current != nil {
			ownerUsername = current.AdminUsername
		}
		return nil, fmt.Errorf("%w: locked by %s", domain.ErrAlreadyExists, ownerUsername)
	}

	moderation, err := s.moderationRepo.ClaimProduct(ctx, domain.ClaimProductParams{
		ProductID: productID,
		AdminID:   adminID,
	})
	if err != nil {
		// Roll the Redis lock back so the product is not stuck.
		_ = s.lock.Release(ctx, productID, adminID)
		return nil, fmt.Errorf("moderation_service.Claim: %w", err)
	}
	return moderation, nil
}

// Release is idempotent. Releasing a product not currently locked / not owned
// by the caller is silently a no-op.
func (s *ModerationService) Release(ctx context.Context, productID, status, username string) error {
	if err := s.moderationRepo.ReleaseProduct(ctx, domain.ReleaseProductParams{
		ProductID: productID,
		Status:    status,
	}); err != nil {
		return fmt.Errorf("moderation_service.Release: %w", err)
	}
	if err := s.lock.Release(ctx, productID, username); err != nil {
		return fmt.Errorf("moderation_service.Release: redis: %w", err)
	}
	return nil
}

// Approve transitions the product to `active`, releases the moderation row
// and lock, and bumps the admin's moderate counter.
func (s *ModerationService) Approve(ctx context.Context, productID, username string) (*domain.Product, error) {
	return s.finalize(ctx, productID, username, "active")
}

// Reject transitions the product to `inactive`. Reason is currently dropped
// (no audit-log table yet — see ADMIN_PANEL_BACKEND_TODO.md §5).
func (s *ModerationService) Reject(ctx context.Context, productID, username, _reason string) (*domain.Product, error) {
	return s.finalize(ctx, productID, username, "inactive")
}

func (s *ModerationService) finalize(ctx context.Context, productID, username, status string) (*domain.Product, error) {
	// Verify the caller owns the lock — prevents one admin from approving
	// another admin's claim.
	owner, lockErr := s.lock.Get(ctx, productID)
	switch {
	case lockErr == nil:
		if owner.AdminUsername != username {
			return nil, fmt.Errorf("%w: claim owned by %s", domain.ErrForbidden, owner.AdminUsername)
		}
	case errors.Is(lockErr, domain.ErrRedisNotFound):
		// No Redis lock — fall back to the durable Postgres state. This is the
		// path after a Redis restart / TTL expiry.
	default:
		return nil, fmt.Errorf("moderation_service.finalize: redis: %w", lockErr)
	}

	product, err := s.productRepo.Update(ctx, domain.UpdateProductParams{
		ID:     productID,
		Status: &status,
	})
	if err != nil {
		return nil, fmt.Errorf("moderation_service.finalize: update product: %w", err)
	}

	if err := s.Release(ctx, productID, status, username); err != nil {
		return nil, fmt.Errorf("moderation_service.finalize: release product: %w", err)
	}

	if err := s.adminRepo.IncreaseTotalModerates(ctx, domain.IncreaseTotalModeratesParams{
		Username:       username,
		TotalModerates: 1, // SQL is `total_moderation = total_moderation + $2`
	}); err != nil {
		// Counter is best-effort — don't roll back the moderation decision.
		_ = err
	}
	return product, nil
}

// Stats aggregates the dashboard counters for the calling admin.
func (s *ModerationService) Stats(ctx context.Context, username string) (*domain.ModerationStats, error) {
	adminID, err := s.resolveAdminID(ctx, username)
	if err != nil {
		return nil, err
	}
	stats, err := s.moderationRepo.Stats(ctx, adminID)
	if err != nil {
		return nil, fmt.Errorf("moderation_service.Stats: %w", err)
	}
	return stats, nil
}
