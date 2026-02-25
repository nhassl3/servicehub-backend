package domain

import "context"

type Category struct {
	ID          int    `db:"id"`
	Slug        string `db:"slug"`
	Name        string `db:"name"`
	Description string `db:"description"`
	IconURL     string `db:"icon_url"`
}

//go:generate mockgen -source=category.go -destination=../repository/mock/category_repo_mock.go -package=mockrepo
type CategoryRepository interface {
	List(ctx context.Context) ([]Category, error)
	GetBySlug(ctx context.Context, slug string) (*Category, error)
}
