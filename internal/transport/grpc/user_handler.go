package grpc

import (
	"context"

	userv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/user/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
	"github.com/nhassl3/servicehub/internal/transport/grpc/interceptors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserHandler implements userv1.UserServiceServer.
//
// Implemented RPC methods:
//   - GetUser
//   - UpdateProfile
type UserHandler struct {
	userv1.UnimplementedUserServiceServer
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	user, err := h.svc.GetUser(ctx, req.Username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.GetUserResponse{User: protoUserProfile(user)}, nil
}

func (h *UserHandler) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UpdateProfileResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	user, err := h.svc.UpdateProfile(ctx, domain.UpdateUserParams{
		Username:  payload.Username,
		FullName:  req.FullName,
		AvatarURL: req.AvatarUrl,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.UpdateProfileResponse{User: protoUserProfile(user)}, nil
}

func (h *UserHandler) UpdatePassword(ctx context.Context, req *userv1.UpdatePasswordRequest) (*userv1.UpdatePasswordResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	if err := payload.Valid(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	user, err := h.svc.UpdatePassword(ctx, domain.UpdateUserPasswordParams{
		Username:    payload.Username,
		OldPassword: req.GetOldPassword(),
		NewPassword: req.GetNewPassword(),
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.UpdatePasswordResponse{User: protoUserProfile(user)}, nil
}

// ── Proto mapper ─────────────────────────────────────────────────────────────

func protoUserProfile(u *domain.User) *userv1.UserProfile {
	return &userv1.UserProfile{
		Username:  u.Username,
		Uid:       u.UID,
		Email:     u.Email,
		FullName:  u.FullName,
		AvatarUrl: u.AvatarURL,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: safeTimestamp(u.CreatedAt),
		UpdatedAt: safeTimestamp(u.UpdatedAt),
	}
}
