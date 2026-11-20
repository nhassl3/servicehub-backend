package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	"go.uber.org/zap"
)

type PublishService struct {
	eventPublisher domain.EventPublisher
	log            *zap.Logger
}

func NewPublishService(eventPublisher domain.EventPublisher, log *zap.Logger) *PublishService {
	return &PublishService{
		eventPublisher: eventPublisher,
		log:            log,
	}
}

func (s *PublishService) PublishUserRegistered(ctx context.Context, user *domain.User, clientIP, userAgent string) {
	if err := s.eventPublisher.PublishUserRegistered(ctx, domain.UserRegisteredPayload{
		UID:       user.UID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}); err != nil {
		s.log.Warn("(Kafka) analytics: failed to publish user.registered", zap.Error(err))
	}

	s.PublishNewUserHistory(ctx,
		user.Username, user.Username,
		domain.Created, domain.Created.AccountCreatedChanges(user.CreatedAt),
		clientIP, userAgent,
	)
}

func (s *PublishService) PublishNewUserHistory(ctx context.Context,
	username, actorUsername string,
	action domain.Action,
	changes map[string]interface{},
	clientIP,
	userAgent string) {
	if err := s.eventPublisher.PublishCreateUserHistory(ctx, domain.CreateUserHistoryParams{
		Username:      username,
		ActorUsername: actorUsername,
		Action:        action,
		Changes:       changes,
		IpAddress:     clientIP,
		UserAgent:     userAgent,
		RequestId:     uuid.NewString(),
	}); err != nil {
		s.log.Warn("(Kafka) history: failed to publish user_history.create_user_history", zap.Error(err))
	}
}
