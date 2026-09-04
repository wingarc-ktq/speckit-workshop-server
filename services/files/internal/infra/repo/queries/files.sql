-- name: GetFileByID :one
SELECT id, name, mime_type, description, storage_key, size, uploaded_at
FROM files
WHERE id = $1;

-- name: InsertFile :one
INSERT INTO files (id, name, mime_type, description, storage_key, size, uploaded_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
RETURNING id, name, mime_type, description, storage_key, size, uploaded_at;

-- name: ListFiles :many
SELECT id, name, mime_type, description, storage_key, size, uploaded_at
FROM files
WHERE ($1::text = '' OR name ILIKE '%' || $1 || '%')
ORDER BY uploaded_at DESC
LIMIT $2::bigint OFFSET $3::bigint;

-- name: CountFiles :one
SELECT COUNT(*)::bigint AS total
FROM files
WHERE ($1::text = '' OR name ILIKE '%' || $1 || '%');

-- name: GetFileTags :many
SELECT tag_id
FROM file_tags
WHERE file_id = $1;
