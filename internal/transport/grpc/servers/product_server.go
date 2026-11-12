package servers

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nhassl3/servicehub-backend/internal/service"
	transport "github.com/nhassl3/servicehub-backend/internal/transport/grpc"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	categoryv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/category/v1"
	productv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/product/v1"
	reviewv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/review/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const ProductServerName = "ProductServer"

var (
	// ProductHandlerFn register product handlers for registration gRPC Gateway (HTTP mux)
	ProductHandlerFn = []func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error{
		productv1.RegisterProductServiceHandler,
		categoryv1.RegisterCategoryServiceHandler,
		reviewv1.RegisterReviewServiceHandler,
	}
)

type ProductService struct {
	Product  *service.ProductService
	Category *service.CategoryService
	Review   *service.ReviewService
}

type ProductHandler struct {
	Product  *transport.ProductHandler
	Category *transport.CategoryHandler
	Review   *transport.ReviewHandler
}

// NewProductServer creates new server with admin and moderation handlers
func NewProductServer(services *Services, tokenManager auth.TokenManager, log *zap.Logger) *Server {
	grpcProductServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(tokenManager),
		),
	)

	handlers := &Handlers{
		Product:  transport.NewProductHandler(services.Product),
		Category: transport.NewCategoryHandler(services.Category),
		Review:   transport.NewReviewHandler(services.Review),
	}

	// Register services for server
	productv1.RegisterProductServiceServer(grpcProductServer, handlers.Product)
	reviewv1.RegisterReviewServiceServer(grpcProductServer, handlers.Review)
	categoryv1.RegisterCategoryServiceServer(grpcProductServer, handlers.Category)
	reflection.Register(grpcProductServer) // enable gRPCurl / Evans in dev

	return &Server{
		grpcServer:  grpcProductServer,
		handlers:    handlers,
		logger:      log,
		serviceName: ProductServerName,
		HandlerFn:   ProductHandlerFn,
	}
}
