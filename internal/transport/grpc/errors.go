package grpc

import (
	"errors"
	"fmt"

	"github.com/nhassl3/servicehub/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// domainErr maps a domain error to the appropriate gRPC status error.
// Unknown errors are returned as Internal.
func domainErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInsufficientFunds):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrEmptyCart):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrOutOfStock):
		return status.Error(codes.ResourceExhausted, err.Error())
	case errors.Is(err, domain.ErrTooSimilarPasswords):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, domain.ErrPasswordDontMatch):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrExpiredToken):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, domain.ErrSessionIsBlocked):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("internal server error: %s", err.Error()))
	}
}
