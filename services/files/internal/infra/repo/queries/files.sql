-- name: CreateFile :one
INSERT INTO files (id, owner_user_id, name, size, mime_type, description, uploaded_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
RETURNING id, owner_user_id, name, size, mime_type, description, uploaded_at;

-- name: ListFiles :many
SELECT id, owner_user_id, name, size, mime_type, description, uploaded_at
FROM files
WHERE $1::uuid IS NOT NULL
    AND ($2::text = '' OR name ILIKE '%' || $2 || '%')
ORDER BY uploaded_at DESC
LIMIT $3 OFFSET $4;

-- name: CountFiles :one
SELECT COUNT(*)::int
FROM files
WHERE $1::uuid IS NOT NULL
    AND ($2::text = '' OR name ILIKE '%' || $2 || '%');

-- name: GetFileByID :one
SELECT id, owner_user_id, name, size, mime_type, description, uploaded_at
FROM files
WHERE id = $1 AND $2::uuid IS NOT NULL;

-- name: UpdateFileMetadata :one
UPDATE files
SET name = $3,
    description = $4,
    uploaded_at = NOW()
WHERE id = $1 AND $2::uuid IS NOT NULL
RETURNING id, owner_user_id, name, size, mime_type, description, uploaded_at;

-- name: DeleteFile :exec
DELETE FROM files
WHERE id = $1 AND $2::uuid IS NOT NULL;

-- name: DeleteFilesByIDs :exec
DELETE FROM files
WHERE $1::uuid IS NOT NULL
    AND id = ANY($2::uuid[]);

-- name: ListFileTagIDs :many
SELECT tag_id
FROM file_tags
WHERE file_id = $1
ORDER BY tag_id;

-- name: DeleteFileTags :exec
DELETE FROM file_tags
WHERE file_id = $1;

-- name: AddFileTag :exec
INSERT INTO file_tags (file_id, tag_id)
VALUES ($1, $2);
