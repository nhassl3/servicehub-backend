package servers

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nhassl3/servicehub-backend/internal/service"
	transport "github.com/nhassl3/servicehub-backend/internal/transport/grpc"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	balancev1 "github.com/nhassl3/servicehub-contracts/pkg/pb/balance/v1"
	cartv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/cart/v1"
	orderv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/order/v1"
	userv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/user/v1"
	wishlistv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/wishlist/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const UserServerName = "user"

var (
	// UserHandlerFn register user handlers for registration gRPC Gateway (HTTP mux)
	UserHandlerFn = []func(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error{
		userv1.RegisterUserServiceHandler,
		cartv1.RegisterCartServiceHandler,
		orderv1.RegisterOrderServiceHandler,
		wishlistv1.RegisterWishlistServiceHandler,
		balancev1.RegisterBalanceServiceHandler,
	}
)

type UserServices struct {
	User     *service.UserService
	Cart     *service.CartService
	Balance  *service.BalanceService
	Order    *service.OrderService
	Wishlist *service.WishlistService
}

type UserHandlers struct {
	User     *transport.UserHandler
	Cart     *transport.CartHandler
	Balance  *transport.BalanceHandler
	Order    *transport.OrderHandler
	Wishlist *transport.WishlistHandler
}

// NewUserServer creates new server with user, cart, balance, order and wishlist handlers
func NewUserServer(services *Services, tokenManager auth.TokenManager, log *zap.Logger) *Server {
	grpcUserServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(tokenManager),
		),
	)

	handlers := &Handlers{
		User:     transport.NewUserHandler(services.User),
		Cart:     transport.NewCartHandler(services.Cart),
		Balance:  transport.NewBalanceHandler(services.Balance),
		Order:    transport.NewOrderHandler(services.Order),
		Wishlist: transport.NewWishlistHandler(services.Wishlist),
	}

	// Register services for server
	userv1.RegisterUserServiceServer(grpcUserServer, handlers.User)
	cartv1.RegisterCartServiceServer(grpcUserServer, handlers.Cart)
	balancev1.RegisterBalanceServiceServer(grpcUserServer, handlers.Balance)
	orderv1.RegisterOrderServiceServer(grpcUserServer, handlers.Order)
	wishlistv1.RegisterWishlistServiceServer(grpcUserServer, handlers.Wishlist)
	reflection.Register(grpcUserServer) // enable gRPCurl / Evans in dev

	return &Server{
		grpcServer:  grpcUserServer,
		handlers:    handlers,
		logger:      log,
		serviceName: UserServerName,
		HandlerFn:   UserHandlerFn,
	}
}
