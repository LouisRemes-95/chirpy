-- name: UpgradeUserByID :one
UPDATE users
SET updated_at = NOW(),
    is_chirpy_red = true
WHERE id = $1
RETURNING *;