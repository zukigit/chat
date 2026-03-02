-- name: SendFriendRequest :one
-- Note: always pass usernames in lexicographic order (smaller first) to satisfy
-- the CHECK (requester_username < addressee_username) constraint.
INSERT INTO friendships (requester_username, addressee_username)
VALUES ($1, $2)
RETURNING requester_username, addressee_username, status, created_at, updated_at;

-- name: UpdateFriendshipStatus :one
UPDATE friendships
SET    status     = $3,
       updated_at = NOW()
WHERE  requester_username = $1
  AND  addressee_username = $2
RETURNING requester_username, addressee_username, status, created_at, updated_at;

-- name: GetFriendship :one
-- Always query with the lexicographically smaller username as $1.
SELECT requester_username, addressee_username, status, created_at, updated_at
FROM friendships
WHERE requester_username = $1
  AND addressee_username = $2
LIMIT 1;

-- name: GetFriends :many
-- Returns all accepted friends for a user.
SELECT
    CASE
        WHEN requester_username = $1 THEN addressee_username
        ELSE requester_username
    END AS friend_username,
    status,
    created_at,
    updated_at
FROM friendships
WHERE (requester_username = $1 OR addressee_username = $1)
  AND status = 'accepted';

-- name: GetPendingRequests :many
-- Returns incoming friend requests pending for a user.
SELECT requester_username, addressee_username, status, created_at, updated_at
FROM friendships
WHERE addressee_username = $1
  AND status = 'pending'
ORDER BY created_at DESC;
