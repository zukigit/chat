-- name: CreateNotification :one
INSERT INTO notifications (user_id, sender_id, type, message)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, sender_id, type, message, is_read, created_at;

-- name: GetNotificationsForUser :many
-- Returns all notifications for a user, newest first.
SELECT id, user_id, sender_id, type, message, is_read, created_at
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetUnreadNotificationCount :one
SELECT COUNT(*) AS unread_count
FROM notifications
WHERE user_id = $1
  AND is_read = FALSE;

-- name: MarkNotificationAsRead :one
UPDATE notifications
SET is_read = TRUE
WHERE id = $1
RETURNING id, user_id, sender_id, type, message, is_read, created_at;

-- name: MarkAllNotificationsAsRead :exec
UPDATE notifications
SET is_read = TRUE
WHERE user_id = $1
  AND is_read = FALSE;
