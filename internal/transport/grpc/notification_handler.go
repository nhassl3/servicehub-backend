package grpc

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/repository/redis"
	"github.com/nhassl3/servicehub-backend/internal/service"
	notificationv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/notification/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var redisKeys = map[notificationv1.StorageKey]string{
	notificationv1.StorageKey_RESET_PASSWORD: redis.ResetPasswordEnterKey,
	notificationv1.StorageKey_VERIFY_EMAIL:   redis.VerifyEmailEnterKey,
}

type NotificationHandler struct {
	svc *service.NotificationService
	notificationv1.UnimplementedNotificationServiceServer
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

func (n *NotificationHandler) ApproveCode(ctx context.Context, req *notificationv1.ApproveCodeRequest) (*notificationv1.ApproveCodeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	entryKey, ok := redisKeys[req.GetKey()]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "unknown entry key")
	}

	resetToken, err := n.svc.ApproveCode(ctx, entryKey, req.GetOperationId(), req.GetCode())
	if err != nil {
		return nil, domainErr(err)
	}

	return &notificationv1.ApproveCodeResponse{
		ResetToken: resetToken,
	}, nil
}
