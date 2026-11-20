package grpc

import (
	"context"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/service"
	"github.com/nhassl3/servicehub-backend/internal/transport/grpc/interceptors"
	authv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthHandler implements authv1.AuthServiceServer.
//
// Implemented RPC methods:
//   - Register
//   - OAuthLogin
//   - Login
//   - Logout
//   - RefreshToken
//   - GetMe
type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) OAuthLogin(ctx context.Context, req *authv1.OAuthLoginRequest) (*authv1.AuthResponse, error) {
	user, tokens, err := h.svc.OAuthLogin(ctx, domain.OAuthLoginInput{
		Username: req.GetUsername(),
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &authv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         protoUserInfo(user),
	}, nil
}

func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.AuthResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, tokens, err := h.svc.Register(ctx, domain.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &authv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         protoUserInfo(user),
	}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, tokens, err := h.svc.Login(ctx, req.Username, req.Password)
	if err != nil {
		return nil, domainErr(err)
	}
	return &authv1.LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         protoUserInfo(user),
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, _ *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.FailedPrecondition, "missing token")
	}

	if err := h.svc.Logout(ctx, payload); err != nil {
		return nil, domainErr(err)
	}

	return &authv1.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	tokens, err := h.svc.RefreshToken(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, domainErr(err)
	}
	return &authv1.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (h *AuthHandler) GetMe(ctx context.Context, _ *authv1.GetMeRequest) (*authv1.GetMeResponse, error) {
	payload, ok := interceptors.PayloadFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing auth payload")
	}
	user, err := h.svc.GetMe(ctx, payload.Username)
	if err != nil {
		return nil, domainErr(err)
	}
	return &authv1.GetMeResponse{User: protoUserInfo(user)}, nil
}

// ── Shared proto mapper ───────────────────────────────────────────────────────

// protoUserInfo converts a domain.User to authv1.UserInfo.
// The same helper is used by the UserHandler via the package-level function.
func protoUserInfo(u *domain.User) *authv1.UserInfo {
	return &authv1.UserInfo{
		Username:  u.Username,
		Uid:       u.UID,
		Email:     u.Email,
		FullName:  u.FullName,
		AvatarUrl: u.AvatarURL,
		Role:      u.Role,
		CreatedAt: SafeTimestamp(u.CreatedAt),
		IsActive:  u.IsActive,
	}
}
