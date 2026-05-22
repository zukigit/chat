-- name: GetPublicKeysByUser :many
SELECT id, user_id, key, created_at, updated_at
FROM public_keys
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: CreatePublicKey :one
INSERT INTO public_keys (user_id, key)
VALUES ($1, $2)
RETURNING id, user_id, key, created_at, updated_at;

-- name: DeletePublicKey :exec
DELETE FROM public_keys
WHERE id = $1 AND user_id = $2;

-- name: DeleteAllPublicKeysForUser :exec
DELETE FROM public_keys
WHERE user_id = $1;

-- name: GetPublicKeysByUserIDs :many
SELECT id, user_id, key, created_at, updated_at
FROM public_keys
WHERE user_id = ANY($1::uuid[])
ORDER BY created_at ASC;

-- name: GetLatestPublicKeyByUserID :one
SELECT id, user_id, key, created_at, updated_at
FROM public_keys
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;
