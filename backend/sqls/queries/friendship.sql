-- name: SendFriendRequest :one
-- Note: always pass user IDs in lexicographic order (smaller UUID first) to satisfy
-- the CHECK (requester_userid < addressee_userid) constraint.
-- initiator_userid is always the actual caller who triggered the request (before ordering).
INSERT INTO friendships (requester_userid, addressee_userid, initiator_userid)
VALUES ($1, $2, $3)
RETURNING requester_userid, addressee_userid, initiator_userid, status, created_at, updated_at;

-- name: UpdateFriendshipStatus :one
UPDATE friendships
SET    status     = $3,
       updated_at = NOW()
WHERE  requester_userid = $1
  AND  addressee_userid = $2
RETURNING requester_userid, addressee_userid, initiator_userid, status, created_at, updated_at;

-- name: ResetFriendRequest :one
-- Re-activates a previously rejected request as pending, updating the initiator.
UPDATE friendships
SET    status           = 'pending',
       initiator_userid = $3,
       updated_at       = NOW()
WHERE  requester_userid = $1
  AND  addressee_userid = $2
RETURNING requester_userid, addressee_userid, initiator_userid, status, created_at, updated_at;

-- name: GetFriendship :one
-- Always query with the lexicographically smaller UUID as $1.
SELECT requester_userid, addressee_userid, initiator_userid, status, created_at, updated_at
FROM friendships
WHERE requester_userid = $1
  AND addressee_userid = $2
LIMIT 1;

-- name: GetFriends :many
-- Returns all accepted friends for a user, including their user_id and username.
SELECT
    CASE
        WHEN f.requester_userid = $1 THEN f.addressee_userid
        ELSE f.requester_userid
    END AS friend_userid,
    u.user_name AS friend_username,
    f.status,
    f.created_at,
    f.updated_at
FROM friendships f
JOIN users u ON u.user_id = (
    CASE
        WHEN f.requester_userid = $1 THEN f.addressee_userid
        ELSE f.requester_userid
    END
)
WHERE (f.requester_userid = $1 OR f.addressee_userid = $1)
  AND f.status = 'accepted';

-- name: GetPendingRequests :many
-- Returns incoming friend requests pending for a user.
SELECT requester_userid, addressee_userid, initiator_userid, status, created_at, updated_at
FROM friendships
WHERE addressee_userid = $1
  AND status = 'pending'
ORDER BY created_at DESC;
