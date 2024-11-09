
-- name: CreateModule :one
INSERT INTO modules (
    user_id,
    name,
    description,
    code,
    visibility
) VALUES (
    $1::uuid,
    $2::varchar(100),
    $3::text,
    $4::text,
    $5::visibility
) RETURNING *;

-- name: GetModulesByUserID :many
SELECT * FROM modules
WHERE user_id = $1::uuid
ORDER BY created_at DESC;

-- name: GetPublicModules :many
SELECT * FROM modules
WHERE visibility = 'public'
ORDER BY created_at DESC
LIMIT $1::integer OFFSET $2::integer;

-- name: UpdateModule :one
UPDATE modules
SET 
    name = COALESCE($2::varchar(100), name),
    description = COALESCE($3::text, description),
    code = COALESCE($4::text, code),
    visibility = COALESCE($5::visibility, visibility)
WHERE id = $1::bigint
RETURNING *;

-- name: DeleteModule :exec
DELETE FROM modules
WHERE id = $1::bigint;
