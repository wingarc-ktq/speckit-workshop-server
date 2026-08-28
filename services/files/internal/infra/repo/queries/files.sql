-- name: CreateFile :one
INSERT INTO files (id, name, size, mime_type, description, storage_key, tag_ids)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, size, mime_type, description, storage_key, tag_ids, uploaded_at;

-- name: GetFileByID :one
SELECT id, name, size, mime_type, description, storage_key, tag_ids, uploaded_at
FROM files
WHERE id = $1;

-- name: ListFiles :many
SELECT
    id, name, size, mime_type, description, storage_key, tag_ids, uploaded_at,
    COUNT(*) OVER() AS total_count
FROM files
WHERE
    (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search')::text || '%')
    AND (sqlc.narg('tag_ids')::uuid[] IS NULL OR tag_ids && sqlc.narg('tag_ids')::uuid[])
ORDER BY uploaded_at DESC
LIMIT sqlc.arg('lim')
OFFSET sqlc.arg('off');
