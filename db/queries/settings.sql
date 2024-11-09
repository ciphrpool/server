-- db/queries/settings.sql
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