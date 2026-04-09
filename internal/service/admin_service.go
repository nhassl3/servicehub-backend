package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/nhassl3/servicehub/internal/domain"
	repoRedis "github.com/nhassl3/servicehub/internal/repository/redis"
)

type AdminService struct {
	adminRepo   domain.AdminRepository
	adminRedis  domain.AdminRedis
	fileStorage domain.PhotoStorage
}

func NewAdminService(adminRepo domain.AdminRepository, fileStorage domain.PhotoStorage, adminRedis *repoRedis.AdminRedis) *AdminService {
	return &AdminService{
		adminRepo:   adminRepo,
		fileStorage: fileStorage,
		adminRedis:  adminRedis,
	}
}

func (a *AdminService) CreateAdmin(ctx context.Context, params domain.CreateAdminParams) (*domain.Admin, error) {
	exists, err := a.adminRepo.ExistsAdminByUsername(ctx, params.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrAlreadyExists
	}
	return a.adminRepo.CreateAdmin(ctx, params)
}

func (a *AdminService) GetAdminProfile(ctx context.Context, params domain.GetAdminProfileParams) (*domain.Admin, error) {
	return a.adminRepo.GetAdmin(ctx, params)
}

func (a *AdminService) UpdateAdminProfile(ctx context.Context, params domain.UpdateAdminsProfileParams) (*domain.Admin, error) {
	return a.adminRepo.UpdateAdminProfile(ctx, params)
}

func (a *AdminService) IncreaseTotalModerates(ctx context.Context, params domain.IncreaseTotalModeratesParams) error {
	return a.adminRepo.IncreaseTotalModerates(ctx, params)
}

func (a *AdminService) UploadAdminAvatar(ctx context.Context, params domain.UploadAdminAvatar) (*domain.Admin, error) {
	if err := ValidateFileUpload(params.FileData, params.ContentType); err != nil {
		return nil, err
	}

	admin, err := a.adminRedis.Profile(ctx, params.Username)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			admin, err = a.adminRepo.GetAdmin(ctx, domain.GetAdminProfileParams{
				Username: params.Username,
			})
			if err != nil {
				return nil, err
			}
			_ = a.adminRedis.SetProfile(ctx, admin) // optional action
		} else {
			return nil, err
		}
	}

	// Delete old avatar if exists
	if admin.AvatarURL != "" {
		oldKey := extractObjectName(admin.AvatarURL)
		if oldKey != "" {
			_ = a.fileStorage.Delete(ctx, oldKey) // optional action, but it's heavy for storage when many photos saved
		}
	}

	objectName := GenerateObjectName("avatars/admins", params.Username, params.ContentType)

	url, err := a.fileStorage.Upload(ctx, objectName, params.ContentType, bytes.NewReader(params.FileData), int64(len(params.FileData)))
	if err != nil {
		return nil, fmt.Errorf("admin_service.UploadAvatar upload: %w", err)
	}

	admin, err = a.adminRepo.UpdateAdminProfile(ctx, domain.UpdateAdminsProfileParams{
		Username:  params.Username,
		AvatarURL: &url,
	})
	if err != nil {
		return nil, fmt.Errorf("user_service.UploadAvatar update user: %w", err)
	}

	_ = a.adminRedis.SetProfile(ctx, admin) // optional action

	return admin, nil
}

func (a *AdminService) UploadCategoryAvatar(ctx context.Context, params domain.UploadCategoryAvatar) (*domain.Category, error) {
	return nil, nil
}
