-- name: ListCategories :many
SELECT id, slug, name, description, icon_url
FROM categories
ORDER BY id;

-- name: GetCategoryBySlug :one
SELECT id, slug, name, description, icon_url
FROM categories
WHERE slug = $1;
