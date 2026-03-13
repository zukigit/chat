-- name: SendMessage :one
INSERT INTO messages (conversation_id, sender_id, content, message_type, media_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, conversation_id, sender_id, content, message_type, media_url, is_edited, deleted_at, created_at, updated_at;

-- name: GetConversationMessages :many
-- Returns non-deleted messages in a conversation, oldest first, with pagination.
SELECT id, conversation_id, sender_id, content, message_type, media_url, is_edited, deleted_at, created_at, updated_at
FROM messages
WHERE conversation_id = $1
  AND deleted_at IS NULL
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

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
