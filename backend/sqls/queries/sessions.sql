-- name: CreateSession :one
INSERT INTO sessions (
    user_userid,
    type,
    status
) VALUES (
    $1, $2, $3
)
RETURNING id, user_userid, type, status, created_at, updated_at;

-- name: GetSession :one
SELECT id, user_userid, type, status, listen_path, created_at, updated_at
FROM sessions
WHERE user_userid = $1 AND type = $2 LIMIT 1;

-- name: UpdateSessionStatus :exec
UPDATE sessions
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;
