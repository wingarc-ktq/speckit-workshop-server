-- name: UpsertTag :one
INSERT INTO tags (id, name, color, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  color = EXCLUDED.color,
  updated_at = NOW()
RETURNING id, name, color, created_at, updated_at;

-- name: GetTagByID :one
SELECT id, name, color, created_at, updated_at
FROM tags
WHERE id = $1;

-- name: InsertFileTag :exec
INSERT INTO file_tags (file_id, tag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: DeleteFileTags :exec
DELETE FROM file_tags
WHERE file_id = $1;
