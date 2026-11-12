package grpc

import (
	"context"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SafeTimestamp converts a time.Time to a protobuf Timestamp.
// Zero times are returned as nil.
func SafeTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// MustUsername extracts the authenticated username from context or returns
// an Unauthenticated gRPC status error.
func MustUsername(ctx context.Context) (string, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing auth payload")
	}
	return payload.Username, nil
}
