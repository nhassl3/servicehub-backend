package servers

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nhassl3/servicehub-backend/internal/service"
	transport "github.com/nhassl3/servicehub-backend/internal/transport/grpc"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	adminv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/admin/v1"
	moderationv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/moderation/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const AdminServerName = "admin"

var (
	// AdminHandlersFn register admin handlers for registration gRPC Gateway (HTTP mux)
	AdminHandlersFn = []func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error{
		adminv1.RegisterAdminServiceHandler,
		moderationv1.RegisterModerationServiceHandler,
	}
)

type AdminServices struct {
	Admin     *service.AdminService
	Analytics *service.AnalyticsService
}

type AdminHandlers struct {
	Admin      *transport.AdminHandler
	Moderation *transport.ModerationHandler
}

// NewAdminServer creates new server with admin and moderation handlers
func NewAdminServer(services *Services, tokenManager auth.TokenManager, log *zap.Logger) *Server {
	grpcAdminServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(tokenManager),
		),
	)

	handlers := &Handlers{
		Admin:      transport.NewAdminHandler(services.Admin, services.Analytics),
		Moderation: transport.NewModerationHandler(services.Moderation),
	}

	// Register services for server
	adminv1.RegisterAdminServiceServer(grpcAdminServer, handlers.Admin)
	moderationv1.RegisterModerationServiceServer(grpcAdminServer, handlers.Moderation)
	reflection.Register(grpcAdminServer) // enable gRPCurl / Evans in dev

	return &Server{
		grpcServer:  grpcAdminServer,
		handlers:    handlers,
		logger:      log,
		serviceName: AdminServerName,
		HandlerFn:   AdminHandlersFn,
	}
}
