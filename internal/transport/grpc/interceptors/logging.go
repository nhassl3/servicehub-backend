package interceptors

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// LoggingInterceptor returns a gRPC unary interceptor that logs method calls.
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		st, _ := status.FromError(err)

		logger.Info("gRPC call",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", st.Code().String()),
			zap.Error(err),
		)

		return resp, err
	}
}
