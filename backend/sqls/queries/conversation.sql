-- name: CreateConversation :one
INSERT INTO conversations (is_group, name)
VALUES ($1, $2)
RETURNING id, is_group, name, created_at, updated_at;

-- name: GetConversation :one
SELECT id, is_group, name, created_at, updated_at
FROM conversations
WHERE id = $1
LIMIT 1;

-- name: GetConversationsByUser :many
-- Returns all conversations a user is a member of, most recently updated first.
SELECT c.id, c.is_group, c.name, c.created_at, c.updated_at
FROM conversations c
JOIN conversation_members cm ON cm.conversation_id = c.id
WHERE cm.user_id = $1
ORDER BY c.updated_at DESC;

-- name: AddMemberToConversation :one
INSERT INTO conversation_members (conversation_id, user_id)
VALUES ($1, $2)
RETURNING conversation_id, user_id, joined_at;

-- name: GetConversationMembers :many
SELECT cm.conversation_id, cm.user_id, cm.joined_at,
       u.user_id, u.user_name, u.display_name, u.avatar_url, u.last_seen_at
FROM conversation_members cm
JOIN users u ON u.user_id = cm.user_id
WHERE cm.conversation_id = $1;

-- name: RemoveMemberFromConversation :exec
DELETE FROM conversation_members
WHERE conversation_id = $1
  AND user_id         = $2;
