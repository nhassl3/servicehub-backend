package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/nhassl3/servicehub/internal/domain"
)

type CategoryService struct {
	repo            domain.CategoryRepository
	categoriesRedis domain.CategoriesRedis
	fileStorage     domain.PhotoStorage
}

func NewCategoryService(repo domain.CategoryRepository, categoriesRedis domain.CategoriesRedis, fileStorage domain.PhotoStorage) *CategoryService {
	return &CategoryService{
		repo:            repo,
		categoriesRedis: categoriesRedis,
		fileStorage:     fileStorage,
	}
}

func (s *CategoryService) ListCategories(ctx context.Context) (*domain.ListCategories, error) {
	categories, err := s.categoriesRedis.Categories(ctx)
	if err != nil {
		if errors.Is(err, domain.ErrRedisNotFound) {
			categories, err = s.repo.List(ctx)
			if err != nil {
				return nil, fmt.Errorf("cateogory_service.ListCategories get from database: %w", err)
			}
			if err := s.categoriesRedis.SetCategories(ctx, categories); err != nil {
				return nil, fmt.Errorf("cateogory_service.ListCategories set categories (for redis): %w", err)
			}
			return categories, nil
		}
		return nil, fmt.Errorf("cateogory_service:ListCategories get from redis: %w", err)
	}
	return categories, nil
}

func (s *CategoryService) UpdateCategory(ctx context.Context, params domain.UpdateCategoryParams) (*domain.Category, error) {
	return s.repo.Update(ctx, params)
}

func (s *CategoryService) UploadCategoryIcon(ctx context.Context, params domain.UploadCategoryIconParams) (*domain.Category, error) {
	if err := ValidateFileUpload(params.FileData, params.ContentType); err != nil {
		return nil, err
	}

	categories, err := s.categoriesRedis.Categories(ctx)
	if err != nil {
		categories, err = s.repo.List(ctx)
		if errors.Is(err, domain.ErrRedisNotFound) {
			_ = s.categoriesRedis.SetCategories(ctx, categories) // optional action
		}
	}

	// IndexFunc better than BinarySearch because categories slice contain not many data
	categoryIdx := slices.IndexFunc(*categories, func(s domain.Category) bool {
		return strings.EqualFold(s.Slug, params.Slug)
	})
	if categoryIdx == -1 {
		return nil, domain.ErrCategoryNotFound
	}

	category := (*categories)[categoryIdx]
	if category.IconURL != "" {
		oldKey := ObjectNameFromURL(category.IconURL)
		if oldKey != "" {
			_ = s.fileStorage.Delete(ctx, oldKey) // optional action
		}
	}

	objectName := GenerateObjectName("icons/categories", params.Slug, params.ContentType)

	url, err := s.fileStorage.Upload(ctx, objectName, params.ContentType, bytes.NewReader(params.FileData), int64(len(params.FileData)))
	if err != nil {
		return nil, fmt.Errorf("category_service.UploadCategoryIcon upload: %w", err)
	}

	return s.repo.Update(ctx, domain.UpdateCategoryParams{
		Slug:        params.Slug,
		Name:        category.Name,
		Description: category.Description,
		IconURL:     url,
	})
}
