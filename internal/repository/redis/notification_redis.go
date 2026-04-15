package redis

import (
	"context"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/redis/go-redis/v9"
)

const notificationPrefix = "notification:"

type NotificationRedis struct {
	client          *redis.Client
	notificationTTL time.Duration
}

func NewNotificationRedis(client *redis.Client) *NotificationRedis {
	return &NotificationRedis{client: client}
}

func (r *NotificationRedis) Save(ctx context.Context, notification *domain.Notification) error {
	return r.client.Set(ctx, notificationPrefix+notification.ID, notification, r.notificationTTL).Err()
}

func (r *NotificationRedis) Get(ctx context.Context, id string) (*domain.Notification, error) {
	var notification *domain.Notification
	if err := r.client.Get(ctx, notificationPrefix+id).Scan(notification); err != nil {
		return nil, err
	}
	return notification, nil
}
