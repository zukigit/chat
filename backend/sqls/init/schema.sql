CREATE TYPE signup_type AS ENUM ('email', 'google', 'github');

CREATE TABLE IF NOT EXISTS users (
    user_name     VARCHAR(50)     PRIMARY KEY,
    hashed_passwd TEXT            NOT NULL,
    signup_type   signup_type     NOT NULL DEFAULT 'email',
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
