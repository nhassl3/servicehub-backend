package domain

import (
	"context"
	"encoding/json"
)

type Category struct {
	ID          int    `db:"id"`
	Slug        string `db:"slug"`
	Name        string `db:"name"`
	Description string `db:"description"`
	IconURL     string `db:"icon_url"`
}

// ListCategories this type needed for redis marshalling and unmarshalling
// it implements slice of categories ([]Category) - Category
type ListCategories []Category

// MarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// marshalling ListCategories type to the []byte code (JSON)
func (c *ListCategories) MarshalBinary() ([]byte, error) {
	if c == nil || len(*c) == 0 {
		return nil, ErrRedisNotFound
	}
	return json.Marshal(c)
}

// UnmarshalBinary this method needed for correct work of Redis
// because Redis only work with JSON, but not with a structures
// unmarshalling source data and convert to ListCategories type
func (c *ListCategories) UnmarshalBinary(data []byte) error {
	if err := json.Unmarshal(data, c); err != nil {
		return err
	}
	if c == nil || len(*c) == 0 {
		return ErrRedisNotFound
	}
	return nil
}

type UpdateCategoryParams struct {
	Slug        string
	Name        string
	Description string
	IconURL     string
}

type UploadCategoryIconParams struct {
	Slug        string
	FileData    []byte
	ContentType string
}

//go:generate mockgen -source=category.go -destination=../repository/mock/category_repo_mock.go -package=mockrepo
type CategoryRepository interface {
	List(ctx context.Context) (*ListCategories, error)
	GetBySlug(ctx context.Context, slug string) (*Category, error)
	Update(ctx context.Context, params UpdateCategoryParams) (*Category, error)
}

type CategoriesRedis interface {
	Categories(ctx context.Context) (*ListCategories, error)
	SetCategories(ctx context.Context, categories *ListCategories) error
}
