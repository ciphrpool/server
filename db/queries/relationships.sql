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
