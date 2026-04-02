-- name: SendFriendRequest :one
-- Note: always pass user IDs in lexicographic order (smaller UUID first) to satisfy
-- the CHECK (user1_userid < user2_userid) constraint.
-- initiator_userid is always the actual caller who triggered the request (before ordering).
INSERT INTO friendships (user1_userid, user2_userid, initiator_userid)
VALUES ($1, $2, $3)
RETURNING user1_userid, user2_userid, initiator_userid, status, created_at, updated_at;

-- name: UpdateFriendshipStatus :one
UPDATE friendships
SET    status     = $3,
       updated_at = NOW()
WHERE  user1_userid = $1
  AND  user2_userid = $2
RETURNING user1_userid, user2_userid, initiator_userid, status, created_at, updated_at;

-- name: DeleteFriendship :exec
-- Deletes the friendship row (used when a request is rejected).
DELETE FROM friendships
WHERE user1_userid = $1
  AND user2_userid = $2;

-- name: GetFriendship :one
-- Always query with the lexicographically smaller UUID as $1.
SELECT user1_userid, user2_userid, initiator_userid, status, created_at, updated_at
FROM friendships
WHERE user1_userid = $1
  AND user2_userid = $2
LIMIT 1;

-- name: GetFriends :many
-- Returns all accepted friends for a user, including their user_id and username.
SELECT
    CASE
        WHEN f.user1_userid = $1 THEN f.user2_userid
        ELSE f.user1_userid
    END AS friend_userid,
    u.user_name AS friend_username,
    f.status,
    f.created_at,
    f.updated_at
FROM friendships f
JOIN users u ON u.user_id = (
    CASE
        WHEN f.user1_userid = $1 THEN f.user2_userid
        ELSE f.user1_userid
    END
)
WHERE (f.user1_userid = $1 OR f.user2_userid = $1)
  AND f.status = 'accepted';

-- name: GetPendingRequests :many
-- Returns incoming friend requests pending for a user.
SELECT user1_userid, user2_userid, initiator_userid, status, created_at, updated_at
FROM friendships
WHERE user2_userid = $1
  AND status = 'pending'
ORDER BY created_at DESC;
