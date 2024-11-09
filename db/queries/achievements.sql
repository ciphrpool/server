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