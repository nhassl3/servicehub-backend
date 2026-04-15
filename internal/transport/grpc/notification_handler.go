package grpc

import (
	"github.com/nhassl3/servicehub-backend/internal/domain"
	notificationv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/notification/v1"
)

type NotificationHandler struct {
	svc *domain.Notification
	*notificationv1.UnimplementedNotificationServiceServer
}

func NewNotificationHandler(svc *domain.Notification) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}
