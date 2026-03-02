CREATE TYPE signup_type AS ENUM ('email', 'google', 'github');

CREATE TABLE IF NOT EXISTS users (
    user_name     VARCHAR(50)     PRIMARY KEY,
    hashed_passwd TEXT            NOT NULL,
    signup_type   signup_type     NOT NULL DEFAULT 'email',
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ── Friendships ────────────────────────────────────────────────────────────────
CREATE TYPE friendship_status AS ENUM ('pending', 'accepted', 'rejected');

CREATE TABLE IF NOT EXISTS friendships (
    id                  BIGSERIAL           PRIMARY KEY,
    requester_username  VARCHAR(50)         NOT NULL REFERENCES users(user_name) ON DELETE CASCADE,
    addressee_username  VARCHAR(50)         NOT NULL REFERENCES users(user_name) ON DELETE CASCADE,
    status              friendship_status   NOT NULL DEFAULT 'pending',
    created_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    UNIQUE (requester_username, addressee_username)
);

-- ── Messages ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS messages (
    id                  BIGSERIAL   PRIMARY KEY,
    sender_username     VARCHAR(50) NOT NULL REFERENCES users(user_name) ON DELETE CASCADE,
    receiver_username   VARCHAR(50) NOT NULL REFERENCES users(user_name) ON DELETE CASCADE,
    content             TEXT        NOT NULL,
    is_read             BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Notifications ──────────────────────────────────────────────────────────────
CREATE TYPE notification_type AS ENUM ('message', 'friend_request');

CREATE TABLE IF NOT EXISTS notifications (
    id              BIGSERIAL           PRIMARY KEY,
    user_username   VARCHAR(50)         NOT NULL REFERENCES users(user_name) ON DELETE CASCADE,
    type            notification_type   NOT NULL,
    message_id      BIGINT              REFERENCES messages(id) ON DELETE SET NULL,
    is_read         BOOLEAN             NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);
