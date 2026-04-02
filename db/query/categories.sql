-- name: ListCategories :many
SELECT id, slug, name, description, icon_url
FROM categories
ORDER BY id;

-- name: GetCategoryBySlug :one
SELECT id, slug, name, description, icon_url
FROM categories
WHERE slug = $1;

-- name: UpdateCategory :one
UPDATE categories
SET name        = $2,
    description = $3,
    icon_url    = $4
WHERE slug = $1
RETURNING id, slug, name, description, icon_url;
