CREATE TYPE signup_type AS ENUM ('email', 'google', 'github');

CREATE TABLE IF NOT EXISTS users (
    id            SERIAL          PRIMARY KEY,
    user_name     VARCHAR(50)     NOT NULL UNIQUE,
    hashed_passwd TEXT            NOT NULL,
    signup_type   signup_type     NOT NULL DEFAULT 'email',
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
