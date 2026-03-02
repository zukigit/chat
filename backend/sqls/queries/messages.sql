-- name: SendMessage :one
INSERT INTO messages (sender_username, receiver_username, content)
VALUES ($1, $2, $3)
RETURNING id, sender_username, receiver_username, content, is_read, created_at;

-- name: GetConversation :many
-- Returns messages between two users, oldest first, with pagination.
SELECT id, sender_username, receiver_username, content, is_read, created_at
FROM messages
WHERE
    (sender_username = $1 AND receiver_username = $2)
    OR
    (sender_username = $2 AND receiver_username = $1)
ORDER BY created_at ASC
LIMIT $3 OFFSET $4;

-- name: GetInboxMessages :many
-- Returns messages received by a user, newest first, with pagination.
SELECT id, sender_username, receiver_username, content, is_read, created_at
FROM messages
WHERE receiver_username = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkMessageAsRead :one
UPDATE messages
SET is_read = TRUE
WHERE id = $1
RETURNING id, sender_username, receiver_username, content, is_read, created_at;
