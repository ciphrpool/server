-- db/queries/auth.sql
-- name: CreateAuthAccount :one
INSERT INTO auth_accounts (
    user_id,
    email,
    auth_type,
    auth_id,
    auth_data,
    verified
) VALUES (
    $1::uuid,
    $2::email,
    $3::auth_type,
    $4::text,
    $5::jsonb,
    $6::boolean
) RETURNING *;

-- name: GetAuthAccountByEmail :one
SELECT * FROM auth_accounts
WHERE email = $1::email;

-- name: GetAuthAccountsByUserID :many
SELECT * FROM auth_accounts
WHERE user_id = $1::uuid;

-- name: UpdateAuthAccount :one
UPDATE auth_accounts
SET 
    email = COALESCE($2::email, email),
    auth_data = COALESCE($3::jsonb, auth_data),
    verified = COALESCE($4::boolean, verified),
    last_login_at = CURRENT_TIMESTAMP
WHERE id = $1::uuid
RETURNING *;