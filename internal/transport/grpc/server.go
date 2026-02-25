// Package grpc wires together the gRPC server, all service handlers, and
// the interceptor chain. Adding a new RPC service requires:
//
//  1. Generating the pb package with `make proto`.
//  2. Creating a handler file that embeds Unimplemented<Name>Server and
//     implements every method you need.
//  3. Adding the handler to the Handlers struct and registering it in
//     registerHandlers.
package grpc

import (
	"context"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	authv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/auth/v1"
	balancev1 "github.com/nhassl3/servicehub-contracts/pkg/pb/balance/v1"
	cartv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/cart/v1"
	categoryv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/category/v1"
	orderv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/order/v1"
	productv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/product/v1"
	reviewv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/review/v1"
	sellerv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/seller/v1"
	userv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/user/v1"
	wishlistv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/wishlist/v1"
	"github.com/nhassl3/servicehub/internal/service"
	"github.com/nhassl3/servicehub/internal/transport/grpc/interceptors"
	"github.com/nhassl3/servicehub/pkg/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

// Services aggregates all application services passed down to handlers.
type Services struct {
	Auth     *service.AuthService
	User     *service.UserService
	Category *service.CategoryService
	Product  *service.ProductService
	Cart     *service.CartService
	Order    *service.OrderService
	Review   *service.ReviewService
	Wishlist *service.WishlistService
	Seller   *service.SellerService
	Balance  *service.BalanceService
}

// Handlers holds all gRPC handler implementations.
// Each handler satisfies the generated Unimplemented<Name>Server contract.
//
// Implemented RPC methods per handler
// ─────────────────────────────────────────────────────────────────────────────
//
//	AuthHandler     : Register · Login · Logout · RefreshToken · GetMe
//	UserHandler     : GetUser · UpdateProfile
//	CategoryHandler : ListCategories
//	ProductHandler  : ListProducts · GetProduct · SearchProducts
//	                · CreateProduct · UpdateProduct · DeleteProduct
//	CartHandler     : GetCart · AddItem · RemoveItem · UpdateItemQty · ClearCart
//	OrderHandler    : CreateOrder · GetOrder · ListOrders
//	                · CancelOrder · UpdateOrderStatus
//	ReviewHandler   : ListReviews · CreateReview · DeleteReview
//	WishlistHandler : GetWishlist · AddItem · RemoveItem
//	SellerHandler   : CreateSeller · GetSellerProfile · UpdateSeller
//	BalanceHandler  : GetBalance · Deposit · GetTransactionHistory
type Handlers struct {
	Auth     *AuthHandler
	User     *UserHandler
	Category *CategoryHandler
	Product  *ProductHandler
	Cart     *CartHandler
	Order    *OrderHandler
	Review   *ReviewHandler
	Wishlist *WishlistHandler
	Seller   *SellerHandler
	Balance  *BalanceHandler
}

// Server wraps the gRPC server with its handler set.
type Server struct {
	grpcServer *grpc.Server
	handlers   *Handlers
	logger     *zap.Logger
}

// NewServer creates the gRPC server, wires interceptors, instantiates every
// handler, and registers them with the server.
func NewServer(services *Services, tokenManager auth.TokenManager, log *zap.Logger) *Server {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(log),
			interceptors.LoggingInterceptor(log),
			interceptors.AuthInterceptor(tokenManager),
		),
	)

	handlers := &Handlers{
		Auth:     NewAuthHandler(services.Auth, tokenManager),
		User:     NewUserHandler(services.User),
		Category: NewCategoryHandler(services.Category),
		Product:  NewProductHandler(services.Product),
		Cart:     NewCartHandler(services.Cart),
		Order:    NewOrderHandler(services.Order),
		Review:   NewReviewHandler(services.Review),
		Wishlist: NewWishlistHandler(services.Wishlist),
		Seller:   NewSellerHandler(services.Seller),
		Balance:  NewBalanceHandler(services.Balance),
	}

	registerHandlers(grpcServer, handlers)
	reflection.Register(grpcServer) // enable grpcurl / Evans in dev

	return &Server{grpcServer: grpcServer, handlers: handlers, logger: log}
}

// registerHandlers registers every service implementation with the gRPC server.
// To add a new service: implement its handler, add it to Handlers, and call
// the generated Register<Name>Server here.
func registerHandlers(srv *grpc.Server, h *Handlers) {
	authv1.RegisterAuthServiceServer(srv, h.Auth)
	userv1.RegisterUserServiceServer(srv, h.User)
	categoryv1.RegisterCategoryServiceServer(srv, h.Category)
	productv1.RegisterProductServiceServer(srv, h.Product)
	cartv1.RegisterCartServiceServer(srv, h.Cart)
	orderv1.RegisterOrderServiceServer(srv, h.Order)
	reviewv1.RegisterReviewServiceServer(srv, h.Review)
	wishlistv1.RegisterWishlistServiceServer(srv, h.Wishlist)
	sellerv1.RegisterSellerServiceServer(srv, h.Seller)
	balancev1.RegisterBalanceServiceServer(srv, h.Balance)
}

// Start begins accepting gRPC connections on addr (e.g. ":9090").
func (s *Server) Start(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.logger.Info("gRPC server listening", zap.String("addr", addr))
	return s.grpcServer.Serve(lis)
}

// StartGateway starts the HTTP/JSON REST gateway that proxies to the local
// gRPC server. The gateway is built with grpc-gateway and maps every
// google.api.http annotation in the proto files to an HTTP endpoint.
//
// grpcAddr must be the same address the gRPC server is listening on.
func (s *Server) StartGateway(ctx context.Context, grpcAddr, httpAddr string) error {
	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
				EmitUnpopulated: false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	// Register every service handler with the gateway mux.
	for _, fn := range []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error{
		authv1.RegisterAuthServiceHandler,
		userv1.RegisterUserServiceHandler,
		categoryv1.RegisterCategoryServiceHandler,
		productv1.RegisterProductServiceHandler,
		cartv1.RegisterCartServiceHandler,
		orderv1.RegisterOrderServiceHandler,
		reviewv1.RegisterReviewServiceHandler,
		wishlistv1.RegisterWishlistServiceHandler,
		sellerv1.RegisterSellerServiceHandler,
		balancev1.RegisterBalanceServiceHandler,
	} {
		if err := fn(ctx, mux, conn); err != nil {
			return err
		}
	}

	s.logger.Info("HTTP gateway listening", zap.String("addr", httpAddr))
	return http.ListenAndServe(httpAddr, corsMiddleware(mux))
}

// corsMiddleware adds CORS headers so that the React dev server at
// localhost:5173 (and any origin listed in allowedOrigins) can reach
// the HTTP gateway.
func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := map[string]struct{}{
		"http://localhost:5173": {},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowedOrigins[origin]; ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		}

		// Handle preflight requests.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Shutdown gracefully drains in-flight RPCs and stops the server.
func (s *Server) Shutdown(_ context.Context) error {
	s.grpcServer.GracefulStop()
	return nil
}
