package interceptors

import (
	"context"
	"strings"

	"github.com/nhassl3/servicehub/pkg/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type contextKey string

const PayloadKey contextKey = "auth_payload"

// publicMethods lists gRPC methods that do not require authentication.
var publicMethods = map[string]struct{}{
	"/auth.v1.AuthService/Register":               {},
	"/auth.v1.AuthService/Login":                  {},
	"/auth.v1.AuthService/RefreshToken":            {},
	"/category.v1.CategoryService/ListCategories": {},
	"/product.v1.ProductService/ListProducts":      {},
	"/product.v1.ProductService/GetProduct":        {},
	"/product.v1.ProductService/SearchProducts":    {},
	"/seller.v1.SellerService/GetSellerProfile":    {},
	"/review.v1.ReviewService/ListReviews":         {},
}

// AuthInterceptor returns a gRPC unary interceptor for PASETO token verification.
func AuthInterceptor(tokenManager auth.TokenManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if _, ok := publicMethods[info.FullMethod]; ok {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		values := md.Get("authorization")
		if len(values) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		authHeader := values[0]
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		payload, err := tokenManager.VerifyToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
		}

		ctx = context.WithValue(ctx, PayloadKey, payload)
		return handler(ctx, req)
	}
}

// PayloadFromContext extracts the auth payload from context.
func PayloadFromContext(ctx context.Context) (*auth.Payload, bool) {
	payload, ok := ctx.Value(PayloadKey).(*auth.Payload)
	return payload, ok
}

// WithPayload stores the auth payload in context (for HTTP middleware use).
func WithPayload(ctx context.Context, payload *auth.Payload) context.Context {
	return context.WithValue(ctx, PayloadKey, payload)
}
