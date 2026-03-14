CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE signup_type AS ENUM ('email', 'google', 'github');

CREATE TABLE IF NOT EXISTS users (
    user_id       UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_name     VARCHAR(50)  NOT NULL UNIQUE,
    hashed_passwd TEXT         NOT NULL,
    signup_type   signup_type  NOT NULL DEFAULT 'email',
    display_name  VARCHAR(100),
    avatar_url    TEXT,
    last_seen_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Friendships ────────────────────────────────────────────────────────────────
CREATE TYPE friendship_status AS ENUM ('pending', 'accepted', 'rejected');

CREATE TABLE IF NOT EXISTS friendships (
    user1_userid      UUID              NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    user2_userid      UUID              NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    initiator_userid  UUID              NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    status              friendship_status NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user1_userid, user2_userid),
    -- Enforce canonical ordering so (A,B) and (B,A) cannot both exist
    CHECK (user1_userid < user2_userid)
);

-- ── Conversations ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS conversations (
    id          BIGSERIAL   PRIMARY KEY,
    is_group    BOOLEAN     NOT NULL DEFAULT FALSE,
    name        TEXT,                         -- NULL for DMs, required for groups
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversation_members (
    conversation_id BIGINT      NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users(user_id)  ON DELETE CASCADE,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (conversation_id, user_id)
);

-- ── Messages ───────────────────────────────────────────────────────────────────
CREATE TYPE message_type AS ENUM ('text', 'image', 'file', 'audio');

CREATE TABLE IF NOT EXISTS messages (
    id              BIGSERIAL    PRIMARY KEY,
    conversation_id BIGINT       NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID         NOT NULL REFERENCES users(user_id)  ON DELETE CASCADE,
    content         TEXT         NOT NULL,
    message_type    message_type NOT NULL DEFAULT 'text',
    media_url       TEXT,                     -- S3/CDN URL for non-text messages
    is_edited       BOOLEAN      NOT NULL DEFAULT FALSE,
    deleted_at      TIMESTAMPTZ,              -- NULL = not deleted (soft delete)
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Message reads (per-user read receipts) ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS message_reads (
    message_id    BIGINT      NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    read_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

-- ── Notifications ──────────────────────────────────────────────────────────────
CREATE TYPE notification_type AS ENUM ('message', 'friend_request');

CREATE TABLE IF NOT EXISTS notifications (
    id               BIGSERIAL         PRIMARY KEY,
    user_id          UUID              NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    sender_id        UUID              NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    type             notification_type NOT NULL,
    message          TEXT              NOT NULL,
    is_read          BOOLEAN           NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);
-- ── Sessions ───────────────────────────────────────────────────────────────────
CREATE TYPE session_type AS ENUM ('notification', 'chat');
CREATE TYPE session_status AS ENUM ('active', 'idel', 'terminate');

CREATE TABLE IF NOT EXISTS sessions (
    id           UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    user_userid  UUID           NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    type         session_type   NOT NULL,
    status       session_status NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

-- ── Indexes ────────────────────────────────────────────────────────────────────

-- users: look up by user_id (PK index created automatically; this covers FK joins)
CREATE INDEX IF NOT EXISTS idx_users_user_id
    ON users (user_id);

-- conversation_members: look up all conversations a user belongs to
CREATE INDEX IF NOT EXISTS idx_conv_members_user
    ON conversation_members (user_id);

-- messages: primary access pattern — messages in a conversation ordered by time
CREATE INDEX IF NOT EXISTS idx_messages_conv_time
    ON messages (conversation_id, created_at ASC);

-- messages: exclude soft-deleted rows efficiently
CREATE INDEX IF NOT EXISTS idx_messages_conv_not_deleted
    ON messages (conversation_id, created_at ASC)
    WHERE deleted_at IS NULL;

-- message_reads: check which messages a user has read
CREATE INDEX IF NOT EXISTS idx_message_reads_user
    ON message_reads (user_id, message_id);

-- notifications: user inbox sorted by time, filterable by is_read
CREATE INDEX IF NOT EXISTS idx_notifications_user_read_time
    ON notifications (user_id, is_read, created_at DESC);

-- friendships: accepted friends lookup per user (both sides)
CREATE INDEX IF NOT EXISTS idx_friendships_user1_status
    ON friendships (user1_userid, status);

CREATE INDEX IF NOT EXISTS idx_friendships_user2_status
    ON friendships (user2_userid, status);

-- friendships: pending incoming requests
CREATE INDEX IF NOT EXISTS idx_friendships_user2_status_time
    ON friendships (user2_userid, status, created_at DESC);

-- sessions: lookup by user and type
CREATE INDEX IF NOT EXISTS idx_sessions_user_type
    ON sessions (user_userid, type);
