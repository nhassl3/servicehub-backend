package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

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
	if code != storeCode.Code {
		return "", nil
	}

	resetToken := uuid.NewString()
	if err = s.userRedis.SetVerified(ctx, enterKeyCode, resetToken, storeCode.Email); err != nil {
		return "", fmt.Errorf("notification_service.ApproveCode set verified reset token: %w", err)
	}

	_ = s.userRedis.DelCode(ctx, enterKeyCode, operationId) // optional operation

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
