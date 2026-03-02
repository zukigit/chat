-- name: MarkMessageAsRead :exec
-- Inserts a read receipt; silently ignored if already read (ON CONFLICT DO NOTHING).
INSERT INTO message_reads (message_id, user_username)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: GetUnreadMessageCount :one
-- Counts unread messages in a conversation for a given user.
SELECT COUNT(*) AS unread_count
FROM messages m
WHERE m.conversation_id = $1
  AND m.sender_username <> $2
  AND m.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM message_reads mr
      WHERE mr.message_id    = m.id
        AND mr.user_username = $2
  );

-- name: GetReadReceiptsForMessage :many
-- Returns who has read a given message and when.
SELECT mr.user_username, mr.read_at,
       u.display_name, u.avatar_url
FROM message_reads mr
JOIN users u ON u.user_name = mr.user_username
WHERE mr.message_id = $1;
