package grpc

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/service"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	userv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserHandler implements userv1.UserServiceServer.
//
// Implemented RPC methods:
//   - UpdateProfile
//   - UploadAvatar
//   - ResetPassword
//   - RequestResetPassword
//   - RequestVerifyEmail
//   - VerifyEmail
//   - RequestResetEmail
//   - ResetEmail
type UserHandler struct {
	userv1.UnimplementedUserServiceServer
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.UpdateProfileResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	user, err := h.svc.UpdateProfile(ctx, &domain.UpdateUserParams{
		Username:  payload.Username,
		FullName:  req.FullName,
		AvatarURL: req.AvatarUrl,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.UpdateProfileResponse{User: protoUserProfile(user)}, nil
}

func (h *UserHandler) UploadAvatar(ctx context.Context, req *userv1.UploadAvatarRequest) (*userv1.UploadAvatarResponse, error) {
	username, err := MustUsername(ctx)
	if err != nil {
		return nil, domainErr(err)
	}
	user, err := h.svc.UploadAvatar(ctx, domain.UploadAvatarParams{
		Username: username,
		FileData: req.GetFileData(),
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.UploadAvatarResponse{User: protoUserProfile(user)}, nil
}

func (h *UserHandler) ResetPassword(ctx context.Context, req *userv1.ResetPasswordRequest) (*userv1.ResetPasswordResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ok, err := h.svc.ResetPassword(ctx, req.GetResetToken(), req.GetPassword())
	if err != nil {
		return nil, domainErr(err)
	}

	return &userv1.ResetPasswordResponse{
		Success: ok,
	}, nil
}

func (h *UserHandler) RequestResetPassword(ctx context.Context, req *userv1.RequestResetPasswordRequest) (*userv1.RequestResetPasswordResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	operationId, err := h.svc.RequestResetPassword(ctx, domain.RequestResetPasswordParams{
		Username: req.Username,
		Email:    req.Email,
	})
	if err != nil {
		return nil, domainErr(err)
	}

	return &userv1.RequestResetPasswordResponse{
		OperationId: operationId,
	}, nil
}

func (h *UserHandler) RequestVerifyEmail(ctx context.Context, req *userv1.RequestVerifyEmailRequest) (*userv1.RequestVerifyEmailResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	operationId, err := h.svc.RequestVerifyEmail(ctx, req.GetEmail())
	if err != nil {
		return nil, domainErr(err)
	}

	return &userv1.RequestVerifyEmailResponse{
		OperationId: operationId,
	}, nil
}

func (h *UserHandler) VerifyEmail(ctx context.Context, req *userv1.VerifyEmailRequest) (*userv1.VerifyEmailResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	tokens, err := h.svc.VerifyEmailAccount(ctx, req.GetVerifyToken())
	if err != nil {
		return nil, domainErr(err)
	}
	if tokens == nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		return nil, domainErr(domain.ErrBuiltToken)
	}

	return &userv1.VerifyEmailResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (h *UserHandler) RequestResetEmail(ctx context.Context, req *userv1.RequestResetEmailRequest) (*userv1.RequestResetEmailResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var username string
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		username, ok = interceptors.GuestFromContext(ctx)
		if !ok {
			return nil, status.Error(codes.FailedPrecondition, "missing token")
		}
	} else {
		if payload == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "empty payload")
		}
		username = payload.Username
	}

	operationId, err := h.svc.RequestResetEmail(ctx, username, req.GetNewEmail())
	if err != nil {
		return nil, domainErr(err)
	}
	return &userv1.RequestResetEmailResponse{
		OperationId: operationId,
	}, nil
}

func (h *UserHandler) ResetEmail(ctx context.Context, req *userv1.ResetEmailRequest) (*userv1.ResetEmailResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	success, err := h.svc.ResetEmail(ctx, req.GetResetEmailToken())
	if err != nil {
		return nil, domainErr(err)
	}

	return &userv1.ResetEmailResponse{
		Success: success,
	}, nil
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
		CreatedAt: SafeTimestamp(u.CreatedAt),
		UpdatedAt: SafeTimestamp(u.UpdatedAt),
	}
}
