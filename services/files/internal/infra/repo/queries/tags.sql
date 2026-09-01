-- name: CreateTag :one
INSERT INTO tags (id, name, color, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
RETURNING id, name, color, created_at, updated_at;

-- name: ListTags :many
SELECT id, name, color, created_at, updated_at
FROM tags
ORDER BY created_at DESC;

-- name: GetTagByID :one
SELECT id, name, color, created_at, updated_at
FROM tags
WHERE id = $1;

-- name: UpdateTag :one
UPDATE tags
SET name = $2,
    color = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, name, color, created_at, updated_at;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE id = $1;
