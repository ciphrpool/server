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

-- name: CreateUserSettings :one
INSERT INTO user_settings (
    user_id,
    preferred_language,
    notification_preferences,
    keyboard_shortcuts,
    accessibility_settings,
    game_settings
) VALUES (
    $1::uuid,
    $2::varchar(5),
    $3::jsonb,
    $4::jsonb,
    $5::jsonb,
    $6::jsonb
) RETURNING *;

-- name: GetUserSettings :one
SELECT * FROM user_settings
WHERE user_id = $1::uuid;

-- name: UpdateUserSettings :one
UPDATE user_settings
SET 
    preferred_language = COALESCE($2::varchar(5), preferred_language),
    notification_preferences = COALESCE($3::jsonb, notification_preferences),
    keyboard_shortcuts = COALESCE($4::jsonb, keyboard_shortcuts),
    accessibility_settings = COALESCE($5::jsonb, accessibility_settings),
    game_settings = COALESCE($6::jsonb, game_settings)
WHERE user_id = $1::uuid
RETURNING *;

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

-- name: CreateDuel :one
INSERT INTO duels (
    p1_id,
    p2_id,
    duel_type
) VALUES (
    $1::uuid,
    $2::uuid,
    $3::duel_type
) RETURNING *;

-- name: GetDuelByID :one
SELECT * FROM duels
WHERE id = $1::bigint;

-- name: GetUserDuels :many
SELECT * FROM duels
WHERE p1_id = $1::uuid OR p2_id = $1::uuid
ORDER BY date DESC
LIMIT $2::integer OFFSET $3::integer;

-- name: UpdateDuelOutcome :one
UPDATE duels
SET 
    winner_id = $2::uuid,
    loser_id = $3::uuid,
    p1_elo_delta = $4::integer,
    p2_elo_delta = $5::integer
WHERE id = $1::bigint
RETURNING *;

-- name: CreateUserRelationship :one
INSERT INTO user_relationships (
    user1_id,
    user2_id,
    relationship_type,
    relationship_status
) VALUES (
    $1::uuid,
    $2::uuid,
    $3::relationship,
    $4::relationship_status
) RETURNING *;

-- name: GetUserRelationships :many
SELECT * FROM user_relationships
WHERE user1_id = $1::uuid OR user2_id = $1::uuid;

-- name: UpdateUserRelationshipStatus :one
UPDATE user_relationships
SET relationship_status = $3::relationship_status
WHERE user1_id = $1::uuid AND user2_id = $2::uuid
RETURNING *;

-- name: CreateAchievement :one
INSERT INTO achievements (
    name,
    description,
    criteria,
    metadata
) VALUES (
    $1::text,
    $2::text,
    $3::jsonb,
    $4::jsonb
) RETURNING *;

-- name: GetAchievementByID :one
SELECT * FROM achievements
WHERE id = $1::bigint;

-- name: CreateUserAchievement :one
INSERT INTO user_achievements (
    user_id,
    achievement_id,
    progress
) VALUES (
    $1::uuid,
    $2::bigint,
    $3::jsonb
) RETURNING *;

-- name: GetUserAchievements :many
SELECT 
    ua.*,
    a.name,
    a.description,
    a.criteria,
    a.metadata
FROM user_achievements ua
JOIN achievements a ON ua.achievement_id = a.id
WHERE ua.user_id = $1::uuid;

-- name: UpdateUserAchievementProgress :one
UPDATE user_achievements
SET progress = $3::jsonb
WHERE user_id = $1::uuid AND achievement_id = $2::bigint
RETURNING *;