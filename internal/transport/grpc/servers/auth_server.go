package servers

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nhassl3/servicehub-backend/internal/service"
	transport "github.com/nhassl3/servicehub-backend/internal/transport/grpc"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	authv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/auth/v1"
	moderationv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/moderation/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const AuthServerName = "auth"

var (
	// AuthHandlersFn register auth handlers for registration gRPC Gateway (HTTP mux)
	AuthHandlersFn = []func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error{
		authv1.RegisterAuthServiceHandler,
	}
)

type AuthService struct {
	*service.AuthService
}

type AuthHandlers struct {
	*transport.AuthHandler
}

// NewAuthServer creates new server with admin and moderation handlers
func NewAuthServer(services *Services, tokenManager auth.TokenManager, log *zap.Logger) *Server {
	grpcAuthServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(tokenManager),
		),
	)

	handlers := &Handlers{
		Auth:       transport.NewAuthHandler(services.Auth),
		Moderation: transport.NewModerationHandler(services.Moderation),
	}

	// Register services for server
	authv1.RegisterAuthServiceServer(grpcAuthServer, handlers.Auth)
	moderationv1.RegisterModerationServiceServer(grpcAuthServer, handlers.Moderation)
	reflection.Register(grpcAuthServer) // enable gRPCurl / Evans in dev

	return &Server{
		grpcServer:  grpcAuthServer,
		handlers:    handlers,
		logger:      log,
		serviceName: AuthServerName,
		HandlerFn:   AuthHandlersFn,
	}
}
