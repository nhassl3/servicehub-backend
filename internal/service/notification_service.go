package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

// maxCodeAttempts limits how many times one operation id may be guessed before
// the code is burnt. Six-digit codes are cheap to brute-force without it.
const maxCodeAttempts int64 = 5

type NotificationService struct {
	userRedis domain.UserRedis
	userRepo  domain.UserRepository
	repo      domain.NotificationRepository
}

func NewNotificationService(userRedis domain.UserRedis, userRepo domain.UserRepository, repo domain.NotificationRepository) *NotificationService {
	return &NotificationService{
		userRedis: userRedis,
		userRepo:  userRepo,
		repo:      repo,
	}
}

func (s *NotificationService) ApproveCode(ctx context.Context, enterKeyCode, operationId, code string) (string, error) {
	storeCode, err := s.userRedis.Code(ctx, enterKeyCode, operationId)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("notification_service.ApproveCode get code: %w", err)
	}

	attempts, err := s.userRedis.IncrCodeAttempts(ctx, enterKeyCode, operationId)
	if err != nil {
		return "", fmt.Errorf("notification_service.ApproveCode increment attempts: %w", err)
	}
	if attempts > maxCodeAttempts {
		_ = s.userRedis.DelCode(ctx, enterKeyCode, operationId) // optional operation
		return "", domain.ErrTooManyCodeAttempts
	}

	// Comparing codes with crypto package
	if subtle.ConstantTimeCompare([]byte(code), []byte(storeCode.Code)) != 1 {
		return "", domain.ErrInvalidCode
	}

	resetToken := uuid.NewString()
	if err = s.userRedis.SetVerified(ctx, enterKeyCode, resetToken, &domain.VerifiedState{
		Email:    storeCode.Email,
		Username: storeCode.Username, // for reset email address (can be empty)
	}); err != nil {
		return "", fmt.Errorf("notification_service.ApproveCode set verified reset token: %w", err)
	}

	_ = s.userRedis.DelCode(ctx, enterKeyCode, operationId)         // optional operation
	_ = s.userRedis.DelCodeAttempts(ctx, enterKeyCode, operationId) // optional operation

	return resetToken, nil
}

// GenerateResetPasswordCode crypto graphic safely method to create six-signs reset password code
// This will protect the code from predictability.
func GenerateResetPasswordCode() string {
	maxN := big.NewInt(1000000) // from 0 to 999999
	n, err := rand.Int(rand.Reader, maxN)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%06d", n)
}
