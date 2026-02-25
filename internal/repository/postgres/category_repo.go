package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/nhassl3/servicehub/internal/db"
	"github.com/nhassl3/servicehub/internal/domain"
)

type CategoryRepo struct {
	store *db.Store
}

func NewCategoryRepo(store *db.Store) *CategoryRepo {
	return &CategoryRepo{store: store}
}

func (r *CategoryRepo) List(ctx context.Context) ([]domain.Category, error) {
	rows, err := r.store.ListCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("category_repo.List: %w", err)
	}
	cats := make([]domain.Category, len(rows))
	for i, row := range rows {
		cats[i] = domain.Category{
			ID:          int(row.ID),
			Slug:        row.Slug,
			Name:        row.Name,
			Description: row.Description,
			IconURL:     row.IconUrl,
		}
	}
	return cats, nil
}

func (r *CategoryRepo) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	row, err := r.store.GetCategoryBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("category_repo.GetBySlug: %w", err)
	}
	return &domain.Category{
		ID:          int(row.ID),
		Slug:        row.Slug,
		Name:        row.Name,
		Description: row.Description,
		IconURL:     row.IconUrl,
	}, nil
}
