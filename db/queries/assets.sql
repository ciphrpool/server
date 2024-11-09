
-- name: CreateAsset :one
INSERT INTO assets (
    asset_type,
    name,
    description,
    metadata
) VALUES (
    $1::varchar(30),
    $2::text,
    $3::text,
    $4::jsonb
) RETURNING *;

-- name: GetAssetByID :one
SELECT * FROM assets
WHERE id = $1::uuid;

-- name: CreateUserAsset :one
INSERT INTO user_assets (
    user_id,
    asset_id,
    asset_type,
    acquisition_method,
    metadata
) VALUES (
    $1::uuid,
    $2::integer,
    $3::assets_type,
    $4::acquisition_method,
    $5::jsonb
) RETURNING *;

-- name: GetUserAssets :many
SELECT * FROM user_assets
WHERE user_id = $1::uuid;