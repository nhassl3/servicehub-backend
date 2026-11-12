package servers

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nhassl3/servicehub-backend/internal/service"
	transport "github.com/nhassl3/servicehub-backend/internal/transport/grpc"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	notificationv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/notification/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const NotificationServerName = "notification"

var (
	// NotificationHandlerFn register notification handler for registration gRPC Gateway (HTTP mux)
	NotificationHandlerFn = []func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error{
		notificationv1.RegisterNotificationServiceHandler,
	}
)

type NotificationService struct {
	Notification *service.NotificationService
}

type NotificationHandler struct {
	Notification *transport.NotificationHandler
}

// NewNotificationServer creates new server with notification handler
func NewNotificationServer(services *Services, tokenManager auth.TokenManager, log *zap.Logger) *Server {
	grpcNotificationServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(tokenManager),
		),
	)

	handlers := &Handlers{
		Notification: transport.NewNotificationHandler(services.Notification),
	}

	// Register services for server
	notificationv1.RegisterNotificationServiceServer(grpcNotificationServer, handlers.Notification)
	reflection.Register(grpcNotificationServer) // enable gRPCurl / Evans in dev

	return &Server{
		grpcServer:  grpcNotificationServer,
		handlers:    handlers,
		logger:      log,
		serviceName: NotificationServerName,
		HandlerFn:   NotificationHandlerFn,
	}
}
