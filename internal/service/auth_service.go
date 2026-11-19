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
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type AuthService struct {
	userRepo         domain.UserRepository
	userRedis        domain.UserRedis
	tokenManager     auth.TokenManager
	refreshManager   auth.TokenManager
	blacklist        auth.TokenBlacklist
	smtpClient       mailer.Notifier
	eventPublisher   domain.EventPublisher
	emailResetWindow time.Duration
	log              *zap.Logger
}

func NewAuthService(
	userRepo domain.UserRepository, userRedis domain.UserRedis,
	tokenManager, refreshManager auth.TokenManager,
	blacklist auth.TokenBlacklist,
	smtpClient mailer.Notifier,
	eventPublisher domain.EventPublisher,
	emailResetWindow time.Duration,
	log *zap.Logger,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		userRedis:        userRedis,
		tokenManager:     tokenManager,
		refreshManager:   refreshManager,
		blacklist:        blacklist,
		smtpClient:       smtpClient,
		eventPublisher:   eventPublisher,
		emailResetWindow: emailResetWindow,
		log:              log,
	}
}

type TokenPair struct {
	AccessToken         string
	RefreshToken        string
	RefreshTokenPayload *auth.Payload
}

func (s *AuthService) OAuthLogin(ctx context.Context, input domain.OAuthLoginInput) (*domain.User, *TokenPair, error) {
	clientIP, userAgent := getMetadataFromContext(ctx)

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
		s.publishNewUserHistory(ctx,
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
		s.publishUserRegistered(ctx, user, clientIP, userAgent)
	}

	tokens, err := s.createTokensAndSession(ctx, user, input.Username)
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
	clientIP, userAgent := getMetadataFromContext(ctx)

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

	s.publishUserRegistered(ctx, user, clientIP, userAgent)

	tokens, err := s.createTokensAndSession(ctx, user, input.Username)
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
	clientIP, _ := getMetadataFromContext(ctx)

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

	tokens, err := s.createTokensAndSession(ctx, user, user.Username)
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

func (s *AuthService) RequestResetPassword(ctx context.Context, params domain.RequestResetPasswordParams) (string, error) {
	var email string
	if params.Username != nil && *params.Username != "" {
		user, err := s.userRedis.Profile(ctx, *params.Username)
		if err != nil {
			if errors.Is(err, domain.ErrRedisNotFound) {
				user, err = s.userRepo.GetUser(ctx, new(domain.GetUserParams{Username: params.Username}))
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
		ok, err := s.userRepo.UserExists(ctx, new(domain.UserExistsParams{Email: params.Email}))
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

	return s.requestToEntryKey(ctx, redis.ResetPasswordEnterKey, &domain.CodeState{Email: email})
}

func (s *AuthService) ResetPassword(ctx context.Context, resetToken, newPassword string) (bool, error) {
	state, err := s.userRedis.Verified(ctx, redis.ResetPasswordEnterKey, resetToken)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			return false, domain.ErrNotFound
		}
		return false, fmt.Errorf("auth_service.ResetPassword verify reset token: %w", err)
	}

	user, err := s.userRepo.GetUser(ctx, new(domain.GetUserParams{Email: &state.Email}))
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

	if _, err = s.userRepo.UpdatePassword(ctx, &params); err != nil {
		return false, fmt.Errorf("auth_service.UpdatePassword: %w", err)
	}

	_ = s.userRedis.DelVerified(ctx, redis.ResetPasswordEnterKey, resetToken) // optional operation

	return true, nil
}

func (s *AuthService) RequestVerifyEmail(ctx context.Context, email string) (string, error) {
	return s.requestToEntryKey(ctx, redis.VerifyEmailEnterKey, &domain.CodeState{Email: email})
}

// requestToEntryKey issues a six-digit code for an email-related flow, stores
// it in Redis together with the target email (and, for email resets, the
// account username) and sends it to the target address. It returns the
// operation id that is later used to confirm the code.
func (s *AuthService) requestToEntryKey(ctx context.Context, entryKey string, state *domain.CodeState) (string, error) {
	if ok := s.userRedis.CodeExists(ctx, entryKey, state.Email); ok {
		return "", domain.ErrCodeAlreadyExists
	}

	code := GenerateResetPasswordCode()
	if code == "" {
		return "", fmt.Errorf("auth_service.requestToEntryKey reset code is empty")
	}
	state.Code = code

	operationId := uuid.NewString()

	if err := s.userRedis.SetCode(ctx, entryKey, operationId, state); err != nil {
		return "", fmt.Errorf("auth_service.requestToEntryKey set code: %w", err)
	}

	var err error
	switch entryKey {
	case redis.VerifyEmailEnterKey:
		err = s.smtpClient.NotifyEmailConfirmation(ctx, code, state.Email)
	case redis.ResetPasswordEnterKey:
		err = s.smtpClient.NotifyResetPassword(ctx, code, state.Email)
	default:
		return "", fmt.Errorf("auth_service.requestToEntryKey unknown entry key")
	}
	if err != nil {
		return "", fmt.Errorf("auth_service.requestToEntryKey: failed to notify: %w", err)
	}

	return operationId, nil
}

func (s *AuthService) VerifyEmailAccount(ctx context.Context, verifyToken string) (*TokenPair, bool, error) {
	state, err := s.userRedis.Verified(ctx, redis.VerifyEmailEnterKey, verifyToken)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			return nil, false, domain.ErrInvalidToken
		}
		return nil, false, fmt.Errorf("auth_service.VerifyEmailAccount: failed to verify token: %w", err)
	}

	var user *domain.User
	if state.IsEmailReset() {
		// The email is switched only now, after the user has proved access to the
		// new address; ConfirmEmailReset also consumes the one-time reset flag.
		user, err = s.userRepo.ConfirmEmailReset(ctx, new(domain.ResetEmailParams{
			Username: state.Username,
			NewEmail: state.Email,
		}))
		if err != nil {
			return nil, false, err
		}
	} else {
		user, err = s.userRepo.VerifyEmail(ctx, new(domain.VerifyEmailAccount{Email: &state.Email}))
		if err != nil {
			return nil, false, fmt.Errorf("auth_service.VerifyEmailAccount: failed to verify email account: %w", err)
		}
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

// ResetEmail starts the one-time email change for an unverified account.
func (s *AuthService) ResetEmail(ctx context.Context, username, newEmail string) (string, error) {
	user, err := s.userRepo.GetUser(ctx, new(domain.GetUserParams{Username: &username}))
	if err != nil {
		return "", err
	}

	if user.IsActive {
		return "", domain.ErrEmailResetNotAllowed
	}
	if user.EmailResetAt != nil {
		return "", domain.ErrEmailAlreadyReset
	}
	if s.emailResetWindow > 0 && time.Since(user.CreatedAt) > s.emailResetWindow {
		return "", domain.ErrEmailResetWindowExpired
	}

	exists, err := s.userRepo.UserExists(ctx, new(domain.UserExistsParams{Email: &newEmail}))
	if err != nil {
		return "", fmt.Errorf("auth_service.ResetEmail check email exists: %w", err)
	}
	if exists {
		return "", domain.ErrAlreadyExists
	}

	return s.requestToEntryKey(ctx, redis.VerifyEmailEnterKey, &domain.CodeState{
		Email:    newEmail,
		Username: username,
	})
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

func (s *AuthService) createTokensAndSession(ctx context.Context, user *domain.User, sessionUsername string) (*TokenPair, error) {
	tokens, err := s.createTokenPair(user.Username, user.UID, user.Role, user.Email, user.IsActive)
	if err != nil {
		return nil, err
	}
	if err := s.createSession(ctx, sessionUsername, tokens.RefreshToken, tokens.RefreshTokenPayload.ExpiredAt); err != nil {
		return nil, err
	}
	if err = s.userRedis.SetProfile(ctx, user); err != nil {
		s.log.Warn("(Redis) failed to set profile for session creation", zap.Error(err))
	}
	return tokens, nil
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

func (s *AuthService) publishUserRegistered(ctx context.Context, user *domain.User, clientIP, userAgent string) {
	if err := s.eventPublisher.PublishUserRegistered(ctx, domain.UserRegisteredPayload{
		UID:       user.UID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}); err != nil {
		s.log.Warn("(Kafka) analytics: failed to publish user.registered", zap.Error(err))
	}

	s.publishNewUserHistory(ctx,
		user.Username, user.Username,
		domain.Created, domain.Created.AccountCreatedChanges(user.CreatedAt),
		clientIP, userAgent,
	)
}

func (s *AuthService) publishNewUserHistory(ctx context.Context,
	username, actorUsername string,
	action domain.Action,
	changes map[string]interface{},
	clientIP,
	userAgent string) {
	if err := s.eventPublisher.PublishCreateUserHistory(ctx, domain.CreateUserHistoryParams{
		Username:      username,
		ActorUsername: actorUsername,
		Action:        action,
		Changes:       changes,
		IpAddress:     clientIP,
		UserAgent:     userAgent,
		RequestId:     uuid.NewString(),
	}); err != nil {
		s.log.Warn("(Kafka) history: failed to publish user_history.create_user_history", zap.Error(err))
	}
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
