package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-passwd/validator"
	"github.com/google/uuid"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/repository/redis"
	"github.com/nhassl3/servicehub-backend/pkg/auth"
	"github.com/nhassl3/servicehub-backend/pkg/hash"
	"github.com/nhassl3/servicehub-backend/pkg/mailer"
	"google.golang.org/grpc/metadata"
)

type AuthService struct {
	userRepo       domain.UserRepository
	userRedis      domain.UserRedis
	tokenManager   auth.TokenManager
	refreshManager auth.TokenManager
	blacklist      auth.TokenBlacklist
	smtpClient     mailer.Notifier
}

func NewAuthService(
	userRepo domain.UserRepository, userRedis domain.UserRedis,
	tokenManager, refreshManager auth.TokenManager,
	blacklist auth.TokenBlacklist,
	smtpClient mailer.Notifier,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		userRedis:      userRedis,
		tokenManager:   tokenManager,
		refreshManager: refreshManager,
		blacklist:      blacklist,
		smtpClient:     smtpClient,
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
	clientIP, _ := getMetadataFromContext(ctx)

	if block, ttl, err := s.userRedis.AuthBlock(ctx, clientIP); block || err != nil {
		return nil, nil, fmt.Errorf("%w: %.0f seconds remaining", domain.ErrAuthBlock, ttl)
	}

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

	tokens, err := s.createTokensAndSession(ctx, user, input.Username)
	if err != nil {
		return nil, nil, err
	}

	_ = s.userRedis.SetAuthBlock(ctx, clientIP)

	return user, tokens, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*domain.User, *TokenPair, error) {
	clientIP, _ := getMetadataFromContext(ctx)

	if block, ttl, err := s.userRedis.AuthBlock(ctx, clientIP); block || err != nil {
		return nil, nil, fmt.Errorf("%w: %.0f seconds remaining", domain.ErrAuthBlock, ttl)
	}

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

	tokens, err := s.createTokensAndSession(ctx, user, username)
	if err != nil {
		return nil, nil, err
	}

	_ = s.userRedis.SetAuthBlock(ctx, clientIP)

	return user, tokens, nil
}

func (s *AuthService) RequestResetPassword(ctx context.Context, params domain.RequestResetPasswordParams) (string, error) {
	var email string
	if params.Username != nil && *params.Username != "" {
		user, err := s.userRedis.Profile(ctx, *params.Username)
		if err != nil {
			if errors.Is(err, domain.ErrRedisNotFound) {
				user, err = s.userRepo.GetByUsername(ctx, *params.Username)
				if err != nil {
					return "", fmt.Errorf("auth_service.GetMe get from database: %w", err)
				}
				_ = s.userRedis.SetProfile(ctx, user)
			} else {
				return "", fmt.Errorf("auth_service:GetMe get from redis: %w", err)
			}
		}
		email = user.Email
	} else if params.Email != nil && *params.Email != "" && strings.Contains(*params.Email, "@") {
		ok, err := s.userRepo.ExistsByEmail(ctx, *params.Email)
		if err != nil {
			return "", fmt.Errorf("auth_serivce.ResetPassword check email: %w", err)
		}
		if !ok {
			return "", domain.ErrNotFound
		}
		email = *params.Email
	} else {
		return "", domain.ErrInvalidCredentials
	}

	return s.requestToEntryKey(ctx, redis.ResetPasswordEnterKey, email)
}

func (s *AuthService) ResetPassword(ctx context.Context, resetToken, newPassword string) (bool, error) {
	email, err := s.userRedis.Verified(ctx, redis.ResetPasswordEnterKey, resetToken)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			return false, domain.ErrNotFound
		}
		return false, fmt.Errorf("auth_service.ResetPassword verify reset token: %w", err)
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return false, err
	}

	ok, err := hash.VerifyPassword(newPassword, user.PasswordHash)
	if err != nil {
		return false, fmt.Errorf("user_service.UpdatePassword verify password: %w", err)
	}
	if ok {
		return false, domain.ErrTooSimilarPasswords
	}

	// similarity by default is 0.7 - DO NOT CHANGE THIS THRESHOLD
	passwordValidator := validator.New(
		validator.Similarity([]string{user.PasswordHash}, nil, domain.ErrTooSimilarPasswords),
	)
	if err := passwordValidator.Validate(newPassword); err != nil {
		return false, err
	}

	newPasswordHashed, err := hash.CreateHashPassword(newPassword)
	if err != nil {
		return false, fmt.Errorf("user_service.UpdatePassword.NewPassword: %w", err)
	}

	params := domain.UpdateUserPasswordParams{
		Username:    user.Username,
		NewPassword: newPasswordHashed,
	}

	if _, err = s.userRepo.UpdatePassword(ctx, params); err != nil {
		return false, fmt.Errorf("auth_service.UpdatePassword: %w", err)
	}

	_ = s.userRedis.DelVerified(ctx, redis.ResetPasswordEnterKey, resetToken) // optional operation

	return true, nil
}

func (s *AuthService) RequestVerifyEmail(ctx context.Context, email string) (string, error) {
	return s.requestToEntryKey(ctx, redis.VerifyEmailEnterKey, email)
}

func (s *AuthService) requestToEntryKey(ctx context.Context, entryKey, email string) (string, error) {
	if ok := s.userRedis.CodeExists(ctx, entryKey, email); ok {
		return "", domain.ErrCodeAlreadyExists
	}

	var code string
	for range 5 {
		code = GenerateResetPasswordCode()
		if code != "" {
			break
		}
	}
	if code == "" {
		return "", fmt.Errorf("auth_service.requestToEntryKey reset code is empty")
	}

	operationId := uuid.NewString()

	if err := s.userRedis.SetCode(ctx, entryKey, operationId, domain.NewCode(email, code)); err != nil {
		return "", fmt.Errorf("auth_service.requestToEntryKey set code: %w", err)
	}

	var err error
	switch entryKey {
	case redis.VerifyEmailEnterKey:
		err = s.smtpClient.NotifyEmailConfirmation(ctx, code, email)
	case redis.ResetPasswordEnterKey:
		err = s.smtpClient.NotifyResetPassword(ctx, code, email)
	default:
		return "", fmt.Errorf("auth_service.requestToEntryKey unknown entry key")
	}
	if err != nil {
		return "", fmt.Errorf("auth_service.requestToEntryKey: failed to notify: %w", err)
	}

	return operationId, nil
}

func (s *AuthService) VerifyEmailAccount(ctx context.Context, verifyToken string) (*TokenPair, bool, error) {
	email, err := s.userRedis.Verified(ctx, redis.VerifyEmailEnterKey, verifyToken)
	if err != nil {
		return nil, false, fmt.Errorf("auth_service.VerifyEmailAccount: failed to verify token: %w", err)
	}

	// slow request to database
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if !exists {
		if err != nil {
			return nil, false, fmt.Errorf("auth_service.VerifyEmailAccount verify email: %w", err)
		}
		return nil, false, domain.ErrNotFound
	}

	user, err := s.userRepo.VerifyEmail(ctx, domain.VerifyEmailAccount{
		Email: &email,
	})
	if err != nil {
		return nil, false, fmt.Errorf("auth_service.VerifyEmailAccount: failed to verify email account %w", err)
	}

	tokens, err := s.createTokensAndSession(ctx, user, user.Username)
	if err != nil {
		return nil, false, err
	}

	_ = s.userRedis.DelVerified(ctx, redis.VerifyEmailEnterKey, verifyToken)

	return tokens, true, nil
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

	accessToken, err := s.tokenManager.CreateToken(session.Username, payload.UID, payload.Role, payload.Email, payload.IsActive)
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
			user, err = s.userRepo.GetByUsername(ctx, username)
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

func (s *AuthService) createTokenPair(username, uid, role, email string, isActive bool) (*TokenPair, error) {
	accessToken, err := s.tokenManager.CreateToken(username, uid, role, email, isActive)
	if err != nil {
		return nil, fmt.Errorf("auth_service: create access token: %w", err)
	}

	refreshToken, payload, err := s.refreshManager.CreateRefreshToken(username, uid, role, email, isActive)
	if err != nil {
		return nil, fmt.Errorf("auth_service: create refresh token: %w", err)
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, RefreshTokenPayload: payload}, nil
}

func (s *AuthService) createSession(ctx context.Context, username, refreshToken string, expiredAt time.Time) error {
	clientIp, userAgent := getMetadataFromContext(ctx)

	// Select session by username because only username store old data for old refreshToken
	// new refresh token while not store in database
	// it's only will be created after check old session
	session, err := s.userRedis.Session(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			session, err = s.userRepo.GetSessionByUsername(ctx, username)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("auth_service.createSession get old session: %w", err)
			}
		} else {
			return fmt.Errorf("auth_service.createSession get old session (redis): %w", err)
		}
	}

	// TODO: implement IPs and useragent checker
	if session != nil && session.ClientIP != clientIp && session.UserAgent != userAgent {
		return domain.ErrDeviceMistake
	}

	// Creating session record about user session in main database
	newSession, err := s.userRepo.CreateSession(ctx, domain.CreateSessionParams{
		Username:     username,
		RefreshToken: refreshToken,
		UserAgent:    userAgent,
		ClientIp:     clientIp,
		IsBlocked:    false,
		ExpiresAt:    expiredAt,
	})
	if err != nil {
		return fmt.Errorf("auth_service.createSession create session: %w", err)
	}

	// Creating session record about user session in Redis
	if err := s.userRedis.SetSession(ctx, newSession); err != nil {
		return fmt.Errorf("auth_service.createSession create session (redis): %w", err)
	}

	return nil
}

func (s *AuthService) createTokensAndSession(ctx context.Context, user *domain.User, sessionUsername string) (*TokenPair, error) {
	tokens, err := s.createTokenPair(user.Username, user.UID, user.Role, user.Email, user.IsActive)
	if err != nil {
		return nil, err
	}

	if err := s.createSession(ctx, sessionUsername, tokens.RefreshToken, tokens.RefreshTokenPayload.ExpiredAt); err != nil {
		return nil, err
	}

	_ = s.userRedis.SetProfile(ctx, user)

	return tokens, nil
}

// getMetadataFromContext TODO: update function (wrong ip and user-agent on response)
func getMetadataFromContext(ctx context.Context) (clientIp string, userAgent string) {
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
	return
}
