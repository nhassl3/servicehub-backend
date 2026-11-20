package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	"go.uber.org/zap"
)

type TokenService struct {
	tokenManager   auth.TokenManager
	refreshManager auth.TokenManager
	userRedis      domain.UserRedis
	userRepo       domain.UserRepository
	log            *zap.Logger
}

func NewTokenService(tokenManager, refreshManager auth.TokenManager,
	userRedis domain.UserRedis, userRepo domain.UserRepository,
	log *zap.Logger,
) *TokenService {
	return &TokenService{
		tokenManager:   tokenManager,
		refreshManager: refreshManager,
		userRedis:      userRedis,
		userRepo:       userRepo,
		log:            log,
	}
}

func (s *TokenService) CreateTokenPair(username, uid, role, email string, isActive bool) (*TokenPair, error) {
	accessToken, err := s.CreateAccessToken(username, uid, role, email, isActive)

	refreshToken, payload, err := s.refreshManager.CreateRefreshToken(&auth.CreateTokenParams{
		Username: username,
		UID:      uid,
		Role:     role,
		Email:    email,
		IsActive: isActive,
	})
	if err != nil {
		return nil, fmt.Errorf("auth_service: create refresh token: %w", err)
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, RefreshTokenPayload: payload}, nil
}

func (s *TokenService) CreateAccessToken(username, uid, role, email string, isActive bool) (string, error) {
	accessToken, err := s.tokenManager.CreateToken(&auth.CreateTokenParams{
		Username: username,
		UID:      uid,
		Role:     role,
		Email:    email,
		IsActive: isActive,
	})
	if err != nil {
		return "", fmt.Errorf("token_service.createAccessToken: create access token: %w", err)
	}
	return accessToken, nil
}

func (s *TokenService) CreateSession(ctx context.Context, username, refreshToken string, expiredAt time.Time) error {
	clientIp, userAgent := GetMetadataFromContext(ctx)

	// Select session by username because only username store old data for old refreshToken
	// new refresh token while not store in database
	// it's only will be created after check old session
	session, err := s.userRedis.Session(ctx, username)
	if err != nil {
		s.log.Warn("(redis) failed to load session from cache", zap.Error(err))
		session, err = s.userRepo.GetSessionByUsername(ctx, username)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("auth_service.createSession get old session: %w", err)
		}
	}

	// TODO: IP-address can be change by new device. Deep check IP-address needed
	if session != nil && (session.ClientIP != clientIp || session.UserAgent != userAgent) {
		return domain.ErrDeviceMistake
	}

	// Creating session record about user session in main database
	newSession, err := s.userRepo.CreateSession(ctx, new(domain.CreateSessionParams{
		Username:     username,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIp,
		IsBlocked:    false,
		ExpiresAt:    expiredAt,
	}))
	if err != nil {
		return fmt.Errorf("auth_service.createSession create session: %w", err)
	}

	// Creating session record about user session in Redis
	if err = s.userRedis.SetSession(ctx, newSession); err != nil {
		s.log.Warn("(Redis) auth_service.createSession create session (redis)", zap.Error(err))
	}

	return nil
}

func (s *TokenService) CreateTokensAndSession(ctx context.Context, user *domain.User, sessionUsername string) (*TokenPair, error) {
	tokens, err := s.CreateTokenPair(user.Username, user.UID, user.Role, user.Email, user.IsActive)
	if err != nil {
		return nil, err
	}
	if err = s.CreateSession(ctx, sessionUsername, tokens.RefreshToken, tokens.RefreshTokenPayload.ExpiredAt); err != nil {
		return nil, err
	}
	if err = s.userRedis.SetProfile(ctx, user); err != nil {
		s.log.Warn("(Redis) failed to set profile for session creation", zap.Error(err))
	}
	return tokens, nil
}
