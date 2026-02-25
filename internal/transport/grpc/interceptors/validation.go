package interceptors

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type protoValidator interface {
	Validate(msg proto.Message) error
}

// ValidationInterceptor returns a gRPC unary interceptor that validates proto messages.
func ValidationInterceptor(validator protoValidator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if msg, ok := req.(proto.Message); ok {
			if err := validator.Validate(msg); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "validation error: %v", err)
			}
		}
		return handler(ctx, req)
	}
}
