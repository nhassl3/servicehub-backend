package grpc

import (
	"context"
	"time"

	authv1 "github.com/nhassl3/servicehub-contracts/pkg/pb/auth/v1"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/internal/service"
	"github.com/nhassl3/servicehub/internal/transport/grpc/interceptors"
	"github.com/nhassl3/servicehub/pkg/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthHandler implements authv1.AuthServiceServer.
//
// Implemented RPC methods:
//   - Register
//   - Login
//   - Logout
//   - RefreshToken
//   - GetMe
type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer
	svc *service.AuthService
	// refresh is kept here only to satisfy injection; token verification is
	// handled by the AuthInterceptor, not directly in handlers.
	_ auth.TokenManager
}

func NewAuthHandler(svc *service.AuthService, _ auth.TokenManager) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	user, tokens, err := h.svc.Register(ctx, service.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		return nil, domainErr(err)
	}
	return &authv1.RegisterResponse{
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

func (h *AuthHandler) Logout(_ context.Context, _ *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	// PASETO tokens are stateless; logout is client-side (discard the token).
	// Future: add a revocation store (Redis) for true server-side logout.
	return &authv1.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	tokens, err := h.svc.RefreshToken(ctx, req.Username)
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
		CreatedAt: safeTimestamp(u.CreatedAt),
		IsActive:  u.IsActive,
	}
}

// safeTimestamp converts a time.Time to a protobuf Timestamp.
// Zero times are returned as nil.
func safeTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
