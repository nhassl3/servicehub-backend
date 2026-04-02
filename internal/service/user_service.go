package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-passwd/validator"
	"github.com/google/uuid"
	"github.com/nhassl3/servicehub/internal/domain"
	"github.com/nhassl3/servicehub/pkg/hash"
)

const maxAvatarSize = 5 << 20 // 5 MB

var allowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type UserService struct {
	userRepo    domain.UserRepository
	fileStorage domain.PhotoStorage
	userRedis   domain.UserRedis
}

func NewUserService(userRepo domain.UserRepository, fileStorage domain.PhotoStorage, userRedis domain.UserRedis) *UserService {
	return &UserService{
		userRepo:    userRepo,
		fileStorage: fileStorage,
		userRedis:   userRedis,
	}
}

func (s *UserService) GetUser(ctx context.Context, username string) (*domain.User, error) {
	return s.userRepo.GetByUsername(ctx, username)
}

func (s *UserService) UpdateProfile(ctx context.Context, params domain.UpdateUserParams) (*domain.User, error) {
	return s.userRepo.Update(ctx, params)
}

func (s *UserService) UpdatePassword(ctx context.Context, params domain.UpdateUserPasswordParams) (*domain.User, error) {
	user, err := s.GetUser(ctx, params.Username)
	if err != nil {
		return nil, err
	}

	ok, err := hash.VerifyPassword(params.OldPassword, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("user_service.UpdatePassword verify password: %w", err)
	}
	if !ok {
		return nil, domain.ErrPasswordDontMatch
	}

	// similarity by default is 0.7 - DO NOT CHANGE THIS THRESHOLD
	passwordValidator := validator.New(
		validator.Similarity([]string{params.OldPassword}, nil, domain.ErrTooSimilarPasswords),
	)
	if err := passwordValidator.Validate(params.NewPassword); err != nil {
		return nil, err
	}

	newPasswordHashed, err := hash.CreateHashPassword(params.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("user_service.UpdatePassword.NewPassword: %w", err)
	}

	params = domain.UpdateUserPasswordParams{
		Username:    params.Username,
		NewPassword: newPasswordHashed,
	}

	return s.userRepo.UpdatePassword(ctx, params)
}

func (s *UserService) UploadAvatar(ctx context.Context, params domain.UploadAvatarParams) (*domain.User, error) {
	if err := ValidateFileUpload(params.FileData, params.ContentType); err != nil {
		return nil, err
	}

	user, err := s.userRedis.Profile(ctx, params.Username)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			user, err = s.userRepo.GetByUsername(ctx, params.Username)
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

	objectName := GenerateObjectName("avatars/users", params.Username, params.ContentType)

	url, err := s.fileStorage.Upload(ctx, objectName, params.ContentType, bytes.NewReader(params.FileData), int64(len(params.FileData)))
	if err != nil {
		return nil, fmt.Errorf("user_service.UploadAvatar upload: %w", err)
	}

	user, err = s.userRepo.Update(ctx, domain.UpdateUserParams{
		Username:  params.Username,
		FullName:  user.FullName,
		AvatarURL: url,
	})
	if err != nil {
		return nil, fmt.Errorf("user_service.UploadAvatar update user: %w", err)
	}

	_ = s.userRedis.SetProfile(ctx, user) // optional action

	return user, nil
}

// extractObjectName extracts the object path from a full MinIO URL.
// e.g. "http://localhost:9000/servicehub-avatars/avatars/users/john/abc.jpg"
// returns "avatars/users/john/abc.jpg"
func extractObjectName(url string) string {
	// Find the bucket name + "/" then take everything after it
	parts := strings.SplitN(url, "/", 5) // [http:, "", host, bucket, objectName]
	if len(parts) < 5 {
		return ""
	}
	return parts[4]
}

// ObjectNameFromURL is exported for use by other services.
func ObjectNameFromURL(url string) string {
	return extractObjectName(url)
}

// FileExtForContentType returns the file extension for a given content type.
func FileExtForContentType(contentType string) (string, bool) {
	ext, ok := allowedContentTypes[contentType]
	return ext, ok
}

// ValidateFileUpload checks file size and content type.
func ValidateFileUpload(data []byte, contentType string) error {
	if len(data) > maxAvatarSize {
		return domain.ErrFileTooLarge
	}
	if _, ok := allowedContentTypes[contentType]; !ok {
		return domain.ErrInvalidFileType
	}
	return nil
}

// GenerateObjectName creates a unique object path for file storage.
func GenerateObjectName(prefix, owner, contentType string) string {
	ext := allowedContentTypes[contentType]
	return filepath.Join(prefix, owner, uuid.NewString()+ext)
}
