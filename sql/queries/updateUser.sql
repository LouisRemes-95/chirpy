-- name: UpdateUser :one
UPDATE users
SET updated_at = NOW(),
    email = $2,
    hashed_password = $3
WHERE id = $1
RETURNING *;