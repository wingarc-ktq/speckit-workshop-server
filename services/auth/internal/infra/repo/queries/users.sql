-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, name)
VALUES ($1, $2, $3, $4)
RETURNING id, email, password_hash, name, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, password_hash, name, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, name, created_at, updated_at
FROM users
WHERE LOWER(email) = LOWER($1);
