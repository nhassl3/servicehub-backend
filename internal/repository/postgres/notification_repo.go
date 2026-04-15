package postgres

import (
	"context"

	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type NotificationRepository struct {
	repo *db.Store
}

func NewNotificationRepository(repo *db.Store) *NotificationRepository {
	return &NotificationRepository{
		repo: repo,
	}
}

func (r *NotificationRepository) CreateNotification(ctx context.Context, params domain.CreateNotificationParams) (*string, error) {
	return nil, nil
}

func (r *NotificationRepository) GetNotification(ctx context.Context, id string) (*domain.Notification, error) {
	return nil, nil
}

func (r *NotificationRepository) ListNotification(ctx context.Context, params domain.ListNotificationParams) ([]*domain.Notification, error) {
	return nil, nil
}
