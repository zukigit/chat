-- name: GetUserByUsername :one
SELECT id, user_name, hashed_passwd, signup_type, created_at, updated_at
FROM users
WHERE user_name = $1
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (user_name, hashed_passwd, signup_type)
VALUES ($1, $2, $3)
RETURNING id, user_name, hashed_passwd, signup_type, created_at, updated_at;
