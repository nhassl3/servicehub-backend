package servers

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nhassl3/servicehub-backend/internal/service"
	transport "github.com/nhassl3/servicehub-backend/internal/transport/grpc"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	sellerv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/seller/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const SellerServerName = "seller"

var (
	// SellerHandlerFn register seller handler for registration gRPC Gateway (HTTP mux)
	SellerHandlerFn = []func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error{
		sellerv1.RegisterSellerServiceHandler,
	}
)

type SellerService struct {
	Seller *service.SellerService
}

type SellerHandler struct {
	Seller *transport.SellerHandler
}

// NewSellerServer creates new server with seller handler
func NewSellerServer(services *Services, tokenManager auth.TokenManager, log *zap.Logger) *Server {
	grpcSellerServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(tokenManager),
		),
	)

	handlers := &Handlers{
		Seller: transport.NewSellerHandler(services.Seller),
	}

	// Register services for server
	sellerv1.RegisterSellerServiceServer(grpcSellerServer, handlers.Seller)
	reflection.Register(grpcSellerServer) // enable gRPCurl / Evans in dev

	return &Server{
		grpcServer:  grpcSellerServer,
		handlers:    handlers,
		logger:      log,
		serviceName: SellerServerName,
		HandlerFn:   SellerHandlerFn,
	}
}
