package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/pkg/auth"
	"github.com/nhassl3/servicehub/pkg/hash"
	"google.golang.org/grpc/metadata"
)

type AuthService struct {
	userRepo       domain.UserRepository
	tokenManager   auth.TokenManager
	refreshManager auth.TokenManager
}

func NewAuthService(userRepo domain.UserRepository, tokenManager, refreshManager auth.TokenManager) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		tokenManager:   tokenManager,
		refreshManager: refreshManager,
	}
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
	FullName string
}

type TokenPair struct {
	AccessToken         string
	RefreshToken        string
	RefreshTokenPayload *auth.Payload
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*domain.User, *TokenPair, error) {
	existsUsername, err := s.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register check username: %w", err)
	}
	if existsUsername {
		return nil, nil, domain.ErrAlreadyExists
	}

	existsEmail, err := s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register check email: %w", err)
	}
	if existsEmail {
		return nil, nil, domain.ErrAlreadyExists
	}

	passwordHash, err := hash.CreateHashPassword(input.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register hash: %w", err)
	}

	user, err := s.userRepo.Create(ctx, domain.CreateUserParams{
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: passwordHash,
		FullName:     input.FullName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register create: %w", err)
	}

	tokens, err := s.createTokenPair(user.Username, user.UID, user.Role)
	if err != nil {
		return nil, nil, err
	}

	if err := s.createSession(ctx, input.Username, tokens.RefreshToken, tokens.RefreshTokenPayload.ExpiredAt); err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.User, *TokenPair, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("auth_service.Login get user: %w", err)
	}

	ok, err := hash.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Login verify: %w", err)
	}
	if !ok {
		return nil, nil, domain.ErrInvalidCredentials
	}

	tokens, err := s.createTokenPair(user.Username, user.UID, user.Role)
	if err != nil {
		return nil, nil, err
	}

	if err := s.createSession(ctx, username, tokens.RefreshToken, tokens.RefreshTokenPayload.ExpiredAt); err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, username string) (*TokenPair, error) {
	session, err := s.userRepo.GetSession(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("auth_service.RefreshToken get session: %w", err)
	}

	if session.RefreshToken == "" {
		return nil, domain.ErrInvalidToken
	} else if time.Now().After(session.ExpiresAt) {
		return nil, domain.ErrExpiredToken
	} else if session.IsBlocked {
		return nil, domain.ErrSessionIsBlocked
	}

	payload, err := s.refreshManager.VerifyToken(session.RefreshToken)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	accessToken, err := s.tokenManager.CreateToken(username, payload.UID, payload.Role)
	if err != nil {
		return nil, fmt.Errorf("auth_service: create access token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: session.RefreshToken,
	}, nil
}

func (s *AuthService) GetMe(ctx context.Context, username string) (*domain.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}

func (s *AuthService) createTokenPair(username, uid, role string) (*TokenPair, error) {
	accessToken, err := s.tokenManager.CreateToken(username, uid, role)
	if err != nil {
		return nil, fmt.Errorf("auth_service: create access token: %w", err)
	}

	refreshToken, payload, err := s.refreshManager.CreateRefreshToken(username, uid, role)
	if err != nil {
		return nil, fmt.Errorf("auth_service: create refresh token: %w", err)
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, RefreshTokenPayload: payload}, nil
}

func (s *AuthService) createSession(ctx context.Context, username, refreshToken string, expiredAt time.Time) error {
	var clientIp, userAgent string
	if headers, ok := metadata.FromIncomingContext(ctx); ok {
		xForwardFor := headers.Get("x-forwarded-for")
		if len(xForwardFor) > 0 && xForwardFor[0] != "" {
			ips := strings.Split(xForwardFor[0], ",")
			if len(ips) > 0 {
				clientIp = ips[0]
			}
		}
		usrAgent := headers.Get("user-agent")
		if len(usrAgent) >= 1 && usrAgent[0] != "" {
			userAgent = usrAgent[0]
		}
	}

	// Creating session record about user session
	if err := s.userRepo.CreateSession(ctx, domain.CreateSessionParams{
		Username:     username,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIp,
		IsBlocked:    false,
		ExpiresAt:    expiredAt,
	}); err != nil {
		return fmt.Errorf("auth_service.Register create session: %w", err)
	}

	return nil
}
