package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Notification struct {
	ID,
	Message,
	Username,
	GroupSlug string
	CreatedAt time.Time
}

func (n *Notification) MarshalBinary() ([]byte, error) {
	return json.Marshal(n)
}

func (n *Notification) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, n)
}

type CreateNotificationParams struct {
	Username,
	NotificationMessage,
	GroupSlug string
}

type ListNotificationParams struct {
	Offset int32
	ToUserId,
	ToUserUsername *string
}

type NotificationRepository interface {
	CreateNotification(ctx context.Context, params CreateNotificationParams) (*string, error)
	GetNotification(ctx context.Context, id string) (*Notification, error)
	ListNotification(ctx context.Context, params ListNotificationParams) ([]*Notification, error)
}

type NotificationRedis interface {
	Save(ctx context.Context, notification *Notification) error
	Get(ctx context.Context, id string) (*Notification, error)
}
