-- name: CreateSession :exec
INSERT INTO sessions (user_userid, login_id)
VALUES ($1, $2);

-- name: GetLoginIDsByUserID :many
SELECT login_id
FROM sessions
WHERE user_userid = $1;

-- name: ValidateSession :one
SELECT user_userid
FROM sessions
WHERE login_id = $1;

-- name: DeleteSessionByLoginID :exec
DELETE FROM sessions
WHERE login_id = $1;
