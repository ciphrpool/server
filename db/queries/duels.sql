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