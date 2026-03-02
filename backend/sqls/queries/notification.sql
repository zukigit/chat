-- name: CreateNotification :one
INSERT INTO notifications (user_username, type, message_id)
VALUES ($1, $2, $3)
RETURNING id, user_username, type, message_id, is_read, created_at;

-- name: GetNotificationsForUser :many
-- Returns all notifications for a user, newest first.
SELECT id, user_username, type, message_id, is_read, created_at
FROM notifications
WHERE user_username = $1
ORDER BY created_at DESC;

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*) AS unread_count
FROM notifications
WHERE user_username = $1
  AND is_read = FALSE;

-- name: MarkNotificationAsRead :one
UPDATE notifications
SET is_read = TRUE
WHERE id = $1
RETURNING id, user_username, type, message_id, is_read, created_at;

-- name: MarkAllNotificationsAsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE user_username = $1
  AND is_read = FALSE;
