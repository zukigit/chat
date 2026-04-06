-- name: SendMessage :one
INSERT INTO messages (conversation_id, sender_id, reply_to_message_id, content, message_type, media_url)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, conversation_id, sender_id, reply_to_message_id, content, message_type, media_url, is_edited, deleted_at, created_at, updated_at;

-- name: GetConversationMessages :many
-- Cursor-based pagination: pass the last seen message id as cursor (0 for first page).
-- Returns non-deleted messages ordered oldest-first.
SELECT id, conversation_id, sender_id, reply_to_message_id, content, message_type, is_edited, created_at
FROM messages
WHERE conversation_id = $1
  AND deleted_at IS NULL
  AND id > $2
ORDER BY id ASC
LIMIT $3;

-- name: EditMessage :one
UPDATE messages
SET content    = $2,
    is_edited  = TRUE,
    updated_at = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING id, conversation_id, sender_id, content, message_type, media_url, is_edited, deleted_at, created_at, updated_at;

-- name: SoftDeleteMessage :one
UPDATE messages
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING id, conversation_id, sender_id, content, message_type, media_url, is_edited, deleted_at, created_at, updated_at;
