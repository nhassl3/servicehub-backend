package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nhassl3/servicehub-backend/internal/domain"
)

type SellerService struct {
	sellerRepo  domain.SellerRepository
	fileStorage domain.PhotoStorage
}

func NewSellerService(sellerRepo domain.SellerRepository, fileStorage domain.PhotoStorage) *SellerService {
	return &SellerService{
		sellerRepo:  sellerRepo,
		fileStorage: fileStorage,
	}
}

func (s *SellerService) CreateSeller(ctx context.Context, params domain.CreateSellerParams) (*domain.Seller, error) {
	exists, err := s.sellerRepo.ExistsByUsername(ctx, params.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrAlreadyExists
	}
	return s.sellerRepo.Create(ctx, params)
}

func (s *SellerService) GetSellerProfile(ctx context.Context, params domain.GetSellerProfileParams) (*domain.Seller, error) {
	if params.SellerId == nil && params.Username == nil {
		return nil, domain.ErrInvalidInput
	}
	if params.SellerId != nil {
		if _, err := uuid.Parse(*params.SellerId); err != nil {
			return nil, domain.ErrInvalidInput
		}
	}
	return s.sellerRepo.GetSeller(ctx, params)
}

func (s *SellerService) UpdateSeller(ctx context.Context, params domain.UpdateSellerParams) (*domain.Seller, error) {
	return s.sellerRepo.Update(ctx, params)
}

func (s *SellerService) UploadAvatar(ctx context.Context, params domain.UploadSellerAvatarParams) (*domain.Seller, error) {
	if err := ValidateFileUpload(params.FileData, params.ContentType); err != nil {
		return nil, err
	}

	seller, err := s.sellerRepo.GetSeller(ctx, domain.GetSellerProfileParams{Username: &params.Username})
	if err != nil {
		return nil, err
	}

	if seller.AvatarURL != "" {
		oldKey := ObjectNameFromURL(seller.AvatarURL)
		if oldKey != "" {
			_ = s.fileStorage.Delete(ctx, oldKey)
		}
	}

	objectName := GenerateObjectName("avatars/sellers", params.Username, params.ContentType)

	url, err := s.fileStorage.Upload(ctx, objectName, params.ContentType, bytes.NewReader(params.FileData), int64(len(params.FileData)))
	if err != nil {
		return nil, fmt.Errorf("seller_service.UploadAvatar upload: %w", err)
	}

	return s.sellerRepo.Update(ctx, domain.UpdateSellerParams{
		Username:    params.Username,
		DisplayName: seller.DisplayName,
		Description: seller.Description,
		AvatarURL:   url,
	})
}
