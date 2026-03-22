-- name: CreateSession :one
INSERT INTO sessions (
    user_userid,
    type,
    status,
    listen_path
) VALUES (
    $1, $2, $3, $4
)
RETURNING id, user_userid, type, status, listen_path, created_at, updated_at;

-- name: GetSession :many
SELECT id, user_userid, type, status, listen_path, created_at, updated_at
FROM sessions
WHERE user_userid = $1 AND type = $2 AND status = 'active';

-- name: UpdateSessionStatus :exec
UPDATE sessions
SET status = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateListenPath :exec
UPDATE sessions
SET listen_path = $2, updated_at = NOW()
WHERE id = $1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;
