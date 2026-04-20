-- name: GetUserByID :one
SELECT user_id, user_name, hashed_passwd, signup_type, display_name, avatar_url, last_seen_at, created_at, updated_at
FROM users
WHERE user_id = $1
LIMIT 1;

-- name: GetUserByUsername :one
SELECT user_id, user_name, hashed_passwd, signup_type, display_name, avatar_url, last_seen_at, created_at, updated_at
FROM users
WHERE user_name = $1
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (user_name, hashed_passwd, signup_type)
VALUES ($1, $2, $3)
RETURNING user_id, user_name, hashed_passwd, signup_type, display_name, avatar_url, last_seen_at, created_at, updated_at;

-- name: UpdateUserProfile :one
UPDATE users
SET display_name = $2,
    avatar_url   = $3,
    updated_at   = NOW()
WHERE user_name = $1
RETURNING user_id, user_name, hashed_passwd, signup_type, display_name, avatar_url, last_seen_at, created_at, updated_at;

-- name: UpdateLastSeen :exec
UPDATE users
SET last_seen_at = NOW()
WHERE user_name = $1;

-- name: SearchUsers :many
SELECT user_id, user_name, display_name, avatar_url
FROM users
WHERE user_name ILIKE '%' || $1 || '%'
   OR display_name ILIKE '%' || $1 || '%'
ORDER BY user_name
LIMIT 50;
