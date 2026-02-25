package service

import (
	"context"

	"github.com/nhassl3/servicehub/internal/domain"
)

type CategoryService struct {
	repo domain.CategoryRepository
}

func NewCategoryService(repo domain.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
}
