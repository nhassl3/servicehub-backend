package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-passwd/validator"
	"github.com/google/uuid"
	"github.com/nhassl3/servicehub-backend/internal/domain"
	"github.com/nhassl3/servicehub-backend/internal/repository/redis"
	"github.com/nhassl3/servicehub-backend/pkg/hash"
	"github.com/nhassl3/servicehub-backend/pkg/mailer"
	"go.uber.org/zap"
)

const maxAvatarSize = 5 << 20 // 5 MB

type UserService struct {
	userRepo         domain.UserRepository
	fileStorage      domain.PhotoStorage
	userRedis        domain.UserRedis
	eventService     *PublishService
	emailResetWindow time.Duration
	smtpClient       mailer.Notifier
	tokenService     *TokenService
	log              *zap.Logger
}

func NewUserService(
	userRepo domain.UserRepository,
	fileStorage domain.PhotoStorage,
	userRedis domain.UserRedis,
	publishService *PublishService,
	emailResetWindow time.Duration,
	smtpClient mailer.Notifier,
	tokenService *TokenService,
	log *zap.Logger,
) *UserService {
	return &UserService{
		userRepo:         userRepo,
		fileStorage:      fileStorage,
		userRedis:        userRedis,
		eventService:     publishService,
		emailResetWindow: emailResetWindow,
		smtpClient:       smtpClient,
		tokenService:     tokenService,
		log:              log,
	}
}

func (s *UserService) UpdateProfile(ctx context.Context, params *domain.UpdateUserParams) (*domain.User, error) {
	return s.userRepo.Update(ctx, params)
}

func (s *UserService) UploadAvatar(ctx context.Context, params domain.UploadAvatarParams) (*domain.User, error) {
	contentType := http.DetectContentType(params.FileData)

	if err := ValidateFileUpload(params.FileData, contentType); err != nil {
		return nil, err
	}

	user, err := s.userRedis.Profile(ctx, params.Username)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			user, err = s.userRepo.GetUser(ctx, new(domain.GetUserParams{
				Username: &params.Username,
			}))
			if err != nil {
				return nil, err
			}
			_ = s.userRedis.SetProfile(ctx, user) // optional action
		} else {
			return nil, err
		}
	}

	// Delete old avatar if exists
	if user.AvatarURL != "" {
		oldKey := extractObjectName(user.AvatarURL)
		if oldKey != "" {
			_ = s.fileStorage.Delete(ctx, oldKey) // optional action, but it's heavy for storage when many photos saved
		}
	}

	objectName := GenerateObjectName("avatars/users", params.Username, contentType)

	url, err := s.fileStorage.Upload(ctx, objectName, contentType, bytes.NewReader(params.FileData), int64(len(params.FileData)))
	if err != nil {
		return nil, fmt.Errorf("user_service.UploadAvatar upload: %w", err)
	}

	user, err = s.userRepo.Update(ctx, new(domain.UpdateUserParams{
		Username:  params.Username,
		FullName:  user.FullName,
		AvatarURL: url,
	}))
	if err != nil {
		return nil, fmt.Errorf("user_service.UploadAvatar update user: %w", err)
	}

	_ = s.userRedis.SetProfile(ctx, user) // optional action

	return user, nil
}

func (s *UserService) RequestResetPassword(ctx context.Context, params domain.RequestResetPasswordParams) (string, error) {
	var email string
	if params.Username != nil && *params.Username != "" {
		user, err := s.userRedis.Profile(ctx, *params.Username)
		if err != nil {
			if errors.Is(err, domain.ErrRedisNotFound) {
				user, err = s.userRepo.GetUser(ctx, new(domain.GetUserParams{Username: params.Username}))
				if err != nil {
					return "", fmt.Errorf("user_service.RequestResetPassword get from database: %w", err)
				}
				_ = s.userRedis.SetProfile(ctx, user)
			} else {
				return "", fmt.Errorf("user_service:RequestResetPassword get from redis: %w", err)
			}
		}
		email = user.Email
	} else if params.Email != nil && *params.Email != "" && strings.Contains(*params.Email, "@") {
		ok, err := s.userRepo.UserExists(ctx, new(domain.UserExistsParams{Email: params.Email}))
		if err != nil {
			return "", fmt.Errorf("auth_serivce.RequestResetPassword check email: %w", err)
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

func (s *UserService) ResetPassword(ctx context.Context, resetToken, newPassword string) (bool, error) {
	state, err := s.userRedis.Verified(ctx, redis.ResetPasswordEnterKey, resetToken)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			return false, domain.ErrNotFound
		}
		return false, fmt.Errorf("user_service.ResetPassword verify reset token: %w", err)
	}

	user, err := s.userRedis.Profile(ctx, state.Username)
	if err != nil {
		user, err = s.userRepo.GetUser(ctx, new(domain.GetUserParams{Username: &state.Username}))
		if err != nil {
			return false, fmt.Errorf("user_service.ResetPassword: failed to get user: %w", err)
		}
	}

	ok, err := hash.VerifyPassword(newPassword, user.PasswordHash)
	if err != nil {
		return false, fmt.Errorf("user_service.ResetPassword verify password: %w", err)
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
		return false, fmt.Errorf("user_service.ResetPassword.NewPassword: %w", err)
	}

	s.eventService.PublishNewUserHistory(ctx,
		user.Username, user.Username,
		domain.PasswordChanged, domain.PasswordChanged.PasswordChangedChanges(user.PasswordHash, newPasswordHashed),
		"", "")

	if _, err = s.userRepo.UpdatePassword(ctx, new(domain.UpdateUserPasswordParams{
		Username:    user.Username,
		NewPassword: newPasswordHashed,
	})); err != nil {
		return false, fmt.Errorf("user_service.ResetPassword: %w", err)
	}

	if err = s.userRedis.DelVerified(ctx, redis.ResetPasswordEnterKey, resetToken); err != nil {
		s.log.Warn("user_service.ResetPassword: failed to delete verified token in redis cache")
	}

	return true, nil
}

func (s *UserService) RequestVerifyEmail(ctx context.Context, email string) (string, error) {
	return s.requestToEntryKey(ctx, redis.VerifyEmailEnterKey, &domain.CodeState{Email: email})
}

// requestToEntryKey issues a six-digit code for an email-related flow, stores
// it in Redis together with the target email (and, for email resets, the
// account username) and sends it to the target address. It returns the
// operation id that is later used to confirm the code.
func (s *UserService) requestToEntryKey(ctx context.Context, entryKey string, state *domain.CodeState) (string, error) {
	if ok := s.userRedis.CodeExists(ctx, entryKey, state.Email); ok {
		return "", domain.ErrCodeAlreadyExists
	}

	code := GenerateCode()
	if code == "" {
		return "", fmt.Errorf("user_service.requestToEntryKey reset code is empty")
	}
	state.Code = code

	operationId := uuid.NewString()

	if err := s.userRedis.SetCode(ctx, entryKey, operationId, state); err != nil {
		return "", fmt.Errorf("user_service.requestToEntryKey set code: %w", err)
	}

	var err error
	switch entryKey {
	case redis.VerifyEmailEnterKey, redis.ResetEmailEnterKey:
		err = s.smtpClient.NotifyEmailConfirmation(ctx, code, state.Email)
	case redis.ResetPasswordEnterKey:
		err = s.smtpClient.NotifyResetPassword(ctx, code, state.Email)
	default:
		return "", fmt.Errorf("user_service.requestToEntryKey unknown entry key")
	}
	if err != nil {
		return "", fmt.Errorf("user_service.requestToEntryKey: failed to notify: %w", err)
	}

	return operationId, nil
}

func (s *UserService) VerifyEmailAccount(ctx context.Context, verifyToken string) (*TokenPair, error) {
	state, err := s.userRedis.Verified(ctx, redis.VerifyEmailEnterKey, verifyToken)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			return nil, domain.ErrInvalidToken
		}
		return nil, fmt.Errorf("user_service.VerifyEmailAccount: failed to verify token: %w", err)
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
			return nil, err
		}
	} else {
		user, err = s.userRepo.VerifyEmail(ctx, new(domain.VerifyEmailAccount{Email: &state.Email}))
		if err != nil {
			return nil, fmt.Errorf("user_service.VerifyEmailAccount: failed to verify email account: %w", err)
		}
	}

	tokens, err := s.tokenService.CreateTokensAndSession(ctx, user, user.Username)
	if err != nil {
		return nil, err
	}

	s.eventService.PublishNewUserHistory(ctx,
		user.Username, user.Username,
		domain.EmailChanged, domain.EmailChanged.EmailVerifiedChanges(),
		"", "")

	if err = s.userRedis.DelVerified(ctx, redis.VerifyEmailEnterKey, verifyToken); err != nil {
		s.log.Warn("user_service.VerifyEmailAccount: failed to delete verified token in redis cache")
	}

	return tokens, nil
}

// RequestResetEmail starts the many-time email change for an unverified and verified account.
func (s *UserService) RequestResetEmail(ctx context.Context, username, newEmail string) (string, error) {
	user, err := s.userRepo.GetUser(ctx, new(domain.GetUserParams{Username: &username}))
	if err != nil {
		return "", err
	}

	if s.emailResetWindow > 0 && time.Since(user.EmailResetAt) < s.emailResetWindow {
		return "", domain.EmailResetManyAttempts
	}

	exists, err := s.userRepo.UserExists(ctx, new(domain.UserExistsParams{Email: &newEmail}))
	if err != nil {
		return "", fmt.Errorf("user_service.RequestResetEmail check email exists: %w", err)
	}
	if exists {
		return "", domain.ErrAlreadyExists
	}

	return s.requestToEntryKey(ctx, redis.ResetEmailEnterKey, &domain.CodeState{
		Email:    newEmail,
		Username: username,
	})
}

func (s *UserService) ResetEmail(ctx context.Context, token string) (bool, error) {
	state, err := s.userRedis.Verified(ctx, redis.VerifyEmailEnterKey, token)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			return false, domain.ErrInvalidToken
		}
		return false, fmt.Errorf("user_service.ResetEmail: failed to verify token: %w", err)
	}

	user, err := s.userRepo.ConfirmEmailReset(ctx, new(domain.ResetEmailParams{
		Username: state.Username,
		NewEmail: state.Email,
	}))
	if err != nil {
		return false, fmt.Errorf("user_service.ResetEmail: failed to reset email: %w", err)
	}

	return user != nil && *user != domain.User{}, nil
}
