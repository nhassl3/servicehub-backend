package grpc

import (
	notificationv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/notification/v1"
	"github.com/nhassl3/servicehub/internal/domain"
)

type NotificationHandler struct {
	svc *domain.NOtification
	*notificationv1.UnimplementedNotificationServiceServer
}
