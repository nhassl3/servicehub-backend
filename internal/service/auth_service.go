package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	"github.com/nhassl3/servicehub-backend/pkg/hash"
	"go.uber.org/zap"
)

type AuthService struct {
	userRepo       domain.UserRepository
	userRedis      domain.UserRedis
	tokenManager   auth.TokenManager
	refreshManager auth.TokenManager
	blacklist      auth.TokenBlacklist
	eventService   *PublishService
	tokenService   *TokenService
	log            *zap.Logger
}

func NewAuthService(
	userRepo domain.UserRepository, userRedis domain.UserRedis,
	blacklist auth.TokenBlacklist,
	eventService *PublishService, tokenService *TokenService,
	log *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		userRedis:      userRedis,
		tokenManager:   tokenService.tokenManager,
		refreshManager: tokenService.refreshManager,
		blacklist:      blacklist,
		eventService:   eventService,
		tokenService:   tokenService,
		log:            log,
	}
}

type TokenPair struct {
	AccessToken         string
	RefreshToken        string
	RefreshTokenPayload *auth.Payload
}

func (s *AuthService) OAuthLogin(ctx context.Context, input domain.OAuthLoginInput) (*domain.User, *TokenPair, error) {
	clientIP, userAgent := GetMetadataFromContext(ctx)

	if err := s.checkBlockAuth(ctx, clientIP); err != nil {
		return nil, nil, err
	}

	var (
		user *domain.User
		err  error
	)

	if exists, err := s.userRepo.UserExists(ctx, new(domain.UserExistsParams{
		Username: &input.Username,
		Email:    &input.Email,
	})); err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register: failed to check user exists: %w", err)
	} else if exists {
		user, err = s.userRepo.GetUser(ctx, new(domain.GetUserParams{
			Email: &input.Email,
		}))
		if err != nil {
			return nil, nil, fmt.Errorf("auth_service.oAuthLogin: failed to get user by email: %w", err)
		}
		s.eventService.PublishNewUserHistory(ctx,
			user.Username, user.Username,
			domain.Login, domain.Login.LastLoginChanges(user.LastLogin, time.Now()),
			clientIP, userAgent,
		)
	} else {
		user, err = createUser(ctx, input, func(in domain.OAuthLoginInput) string { return in.Password },
			func(ctx context.Context, input domain.OAuthLoginInput, passwordHash string) (*domain.User, error) {
				return s.userRepo.OAuthLogin(ctx, new(domain.OAuthLoginParams{
					Username:     input.Username,
					Email:        input.Email,
					PasswordHash: passwordHash,
				}))
			}, "OAuthLogin")
		if err != nil {
			return nil, nil, fmt.Errorf("auth_service.oAuthLogin: failed to create new user: %w", err)
		}

		// Kafka events sending async
		s.eventService.PublishUserRegistered(ctx, user, clientIP, userAgent)
	}

	tokens, err := s.tokenService.CreateTokensAndSession(ctx, user, input.Username)
	if err != nil {
		return nil, nil, err
	}

	if err := s.userRedis.SetAuthBlock(ctx, clientIP); err != nil {
		s.log.Warn("auth_service.OAuthLogin: failed to set auth block",
			zap.Error(err),
		)
	}

	return user, tokens, nil
}

func (s *AuthService) Register(ctx context.Context, input domain.RegisterInput) (*domain.User, *TokenPair, error) {
	clientIP, userAgent := GetMetadataFromContext(ctx)

	if err := s.checkBlockAuth(ctx, clientIP); err != nil {
		return nil, nil, err
	}

	if exists, err := s.userRepo.UserExists(ctx, new(domain.UserExistsParams{
		Username: &input.Username,
		Email:    &input.Email,
	})); err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register: failed to check user exists: %w", err)
	} else if exists {
		return nil, nil, domain.ErrAlreadyExists
	}

	user, err := createUser(ctx, input, func(in domain.RegisterInput) string { return in.Password },
		func(ctx context.Context, in domain.RegisterInput, passwordHash string) (*domain.User, error) {
			return s.userRepo.Create(ctx, new(domain.CreateUserParams{
				Username:     in.Username,
				Email:        in.Email,
				PasswordHash: passwordHash,
				FullName:     in.FullName,
			}))
		}, "Register")
	if err != nil {
		return nil, nil, fmt.Errorf("auth_service.Register create user: %w", err)
	}

	s.eventService.PublishUserRegistered(ctx, user, clientIP, userAgent)

	tokens, err := s.tokenService.CreateTokensAndSession(ctx, user, input.Username)
	if err != nil {
		return nil, nil, err
	}

	if err := s.userRedis.SetAuthBlock(ctx, clientIP); err != nil {
		s.log.Warn("auth_service.OAuthLogin: failed to set auth block",
			zap.Error(err),
		)
	}

	return user, tokens, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.User, *TokenPair, error) {
	clientIP, userAgent := GetMetadataFromContext(ctx)

	if err := s.checkBlockAuth(ctx, clientIP); err != nil {
		return nil, nil, err
	}

	user, err := s.userRepo.GetUser(ctx, new(domain.GetUserParams{Username: &username}))
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

	tokens, err := s.tokenService.CreateTokensAndSession(ctx, user, user.Username)
	if err != nil {
		return nil, nil, err
	}

	s.eventService.PublishNewUserHistory(ctx,
		user.Username, user.Username,
		domain.Login, domain.Login.LastLoginChanges(user.LastLogin, time.Now()),
		clientIP, userAgent,
	)

	if err := s.userRedis.SetAuthBlock(ctx, clientIP); err != nil {
		s.log.Warn("auth_service.OAuthLogin: failed to set auth block",
			zap.Error(err),
		)
	}

	return user, tokens, nil
}

func (s *AuthService) Logout(ctx context.Context, accessPayload *auth.Payload) error {
	if accessPayload != nil && accessPayload.JTI != "" {
		if err := s.blacklist.Blacklist(ctx, accessPayload.JTI, accessPayload.ExpiredAt); err != nil {
			return fmt.Errorf("auth_service.Logout blacklist access token: %w", err)
		}
		if err := s.userRedis.DelProfile(ctx, accessPayload.Username); err != nil {
			return fmt.Errorf("auth_service.Logout delete redis user profile: %w", err)
		}
		if err := s.userRedis.DelSession(ctx, accessPayload.Username); err != nil {
			return fmt.Errorf("auth_service.Logout delete redis user session: %w", err)
		}
		return s.userRepo.DeleteSession(ctx, accessPayload.Username)
	}
	return domain.ErrInvalidToken
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	payload, err := s.refreshManager.VerifyToken(refreshToken)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	session, err := s.userRedis.Session(ctx, payload.Username)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			session, err = s.userRepo.GetSession(ctx, refreshToken)
			if err != nil {
				return nil, fmt.Errorf("auth_service.RefreshToken get session: %w", err)
			}
			if err := s.userRedis.SetSession(ctx, session); err != nil {
				return nil, fmt.Errorf("auth_service.RefreshToken set session (redis): %w", err)
			}
		} else {
			return nil, fmt.Errorf("auth_service.RefreshToken get session (redis): %w", err)
		}
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, domain.ErrExpiredToken
	} else if session.IsBlocked {
		return nil, domain.ErrSessionIsBlocked
	}

	accessToken, err := s.tokenService.CreateAccessToken(
		session.Username, payload.UID, payload.Role, payload.Email, payload.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("auth_service: create access token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: session.RefreshToken,
	}, nil
}

func (s *AuthService) GetMe(ctx context.Context, username string) (*domain.User, error) {
	user, err := s.userRedis.Profile(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			user, err = s.userRepo.GetUser(ctx, new(domain.GetUserParams{Username: &username}))
			if err != nil {
				return nil, fmt.Errorf("auth_service.GetMe get from database: %w", err)
			}
			if err := s.userRedis.SetProfile(ctx, user); err != nil {
				return nil, fmt.Errorf("auth_service.GetMe set profile (for redis): %w", err)
			}
			return user, nil
		}
		return nil, fmt.Errorf("auth_service:GetMe get from redis: %w", err)
	}
	return user, nil
}

// checkBlockAuth checks auth blocking in redis storage cache
func (s *AuthService) checkBlockAuth(ctx context.Context, clientIP string) error {
	block, ttl, err := s.userRedis.AuthBlock(ctx, clientIP)
	if err != nil {
		return fmt.Errorf("auth_service.OAuthLogin: check auth block: %w", err)
	}
	if block {
		return fmt.Errorf(
			"%w: %.0f seconds remaining",
			domain.ErrAuthBlock,
			ttl,
		)
	}
	return nil
}

// createUser creates user by any types
func createUser[T any](
	ctx context.Context,
	input T,
	getPassword func(T) string,
	create func(context.Context, T, string) (*domain.User, error),
	operation string,
) (*domain.User, error) {
	passwordHash, err := hash.CreateHashPassword(getPassword(input))
	if err != nil {
		return nil, fmt.Errorf("auth_service.%s: failed to hash password: %w", operation, err)
	}
	user, err := create(ctx, input, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("auth_service.%s: failed to create user: %w", operation, err)
	}
	return user, nil
}
