-- db/queries/users.sql
-- name: CreateUser :one
INSERT INTO users (
    username,
    profile_picture_url,
    bio,
    elo
) VALUES (
    $1::varchar(30),
    $2::text,
    $3::varchar(500),
    COALESCE($4::integer, 1000)
) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1::uuid;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1::varchar(30);

-- name: UpdateUser :one
UPDATE users
SET 
    username = COALESCE($2::varchar(30), username),
    profile_picture_url = COALESCE($3::text, profile_picture_url),
    bio = COALESCE($4::varchar(500), bio),
    elo = COALESCE($5::integer, elo)
WHERE id = $1::uuid
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1::uuid;