# Database Schema — Table Relationships

## Entity Relationship Diagram

```mermaid
erDiagram
    users {
        uuid user_id PK
        varchar user_name UK
        text hashed_passwd
        signup_type signup_type
        varchar display_name
        text avatar_url
        timestamptz last_seen_at
        timestamptz created_at
        timestamptz updated_at
    }

    friendships {
        uuid user1_userid PK_FK
        uuid user2_userid PK_FK
        uuid initiator_userid FK
        friendship_status status
        timestamptz created_at
        timestamptz updated_at
    }

    conversations {
        bigserial id PK
        boolean is_group
        text name
        timestamptz created_at
        timestamptz updated_at
    }

    conversation_members {
        bigint conversation_id PK_FK
        uuid user_id PK_FK
        timestamptz joined_at
    }

    messages {
        bigserial id PK
        bigint conversation_id FK
        uuid sender_id FK
        text content
        message_type message_type
        text media_url
        boolean is_edited
        timestamptz deleted_at
        timestamptz created_at
        timestamptz updated_at
    }

    message_reads {
        bigint message_id PK_FK
        uuid user_id PK_FK
        timestamptz read_at
    }

    notifications {
        bigserial id PK
        uuid user_id FK
        uuid sender_id FK
        notification_type type
        text message
        boolean is_read
        timestamptz created_at
    }

    users ||--o{ friendships : "requests (user1_userid)"
    users ||--o{ friendships : "receives (user2_userid)"
    users ||--o{ conversation_members : "joins (user_id)"
    dm_peers {
        uuid user1_id PK_FK
        uuid user2_id PK_FK
        bigint conversation_id FK
    }

    conversations ||--o{ conversation_members : "has members"
    conversations ||--|| dm_peers : "dm lookup (conversation_id)"
    users ||--o{ dm_peers : "dm peer (user1_id)"
    users ||--o{ dm_peers : "dm peer (user2_id)"
    conversations ||--o{ messages : "contains (conversation_id)"
    users ||--o{ messages : "sends (sender_id)"
    messages ||--o{ message_reads : "read receipts"
    users ||--o{ message_reads : "reads (user_id)"
    users ||--o{ notifications : "notified (user_id)"
    users ||--o{ notifications : "sends (sender_id)"
```

---

## Tables

### `users`
Central table. Every other table references it.

| Column | Type | Notes |
|---|---|---|
| `user_id` | `UUID` | **PK** |
| `user_name` | `VARCHAR(50)` | **UNIQUE** |
| `hashed_passwd` | `TEXT` | |
| `signup_type` | `signup_type` | `email`, `google`, `github` |
| `display_name` | `VARCHAR(100)` | nullable |
| `avatar_url` | `TEXT` | nullable |
| `last_seen_at` | `TIMESTAMPTZ` | nullable |
| `created_at` | `TIMESTAMPTZ` | |
| `updated_at` | `TIMESTAMPTZ` | |

---

### `friendships`
Tracks friend relationships. The primary key is `(user1_userid, user2_userid)`. A `CHECK` constraint enforces canonical ordering (`user1_userid < user2_userid`) so `(A,B)` and `(B,A)` cannot both exist.

| Column | Type | Notes |
|---|---|---|
| `user1_userid` | `UUID` | **PK, FK** → `users.user_id` |
| `user2_userid` | `UUID` | **PK, FK** → `users.user_id` |
| `initiator_userid` | `UUID` | **FK** → `users.user_id` |
| `status` | `friendship_status` | `pending`, `accepted`, `rejected` |
| `created_at` | `TIMESTAMPTZ` | |
| `updated_at` | `TIMESTAMPTZ` | |

---

### `conversations`
A conversation is either a direct message (DM) or a group chat.

| Column | Type | Notes |
|---|---|---|
| `id` | `BIGSERIAL` | **PK** |
| `is_group` | `BOOLEAN` | `false` for DMs |
| `name` | `TEXT` | nullable — `NULL` for DMs, required for groups |
| `created_at` | `TIMESTAMPTZ` | |
| `updated_at` | `TIMESTAMPTZ` | |

---

### `dm_peers`
Fast lookup table for DM conversations. Maps a canonical user pair to a `conversation_id`. Only exists for DMs (`is_group = false`). The `CHECK (user1_id < user2_id)` enforces canonical ordering so each pair has exactly one row.

| Column | Type | Notes |
|---|---|---|
| `user1_id` | `UUID` | **PK, FK** → `users.user_id` (CASCADE) |
| `user2_id` | `UUID` | **PK, FK** → `users.user_id` (CASCADE) |
| `conversation_id` | `BIGINT` | **FK, UNIQUE** → `conversations.id` (CASCADE) |

---

### `conversation_members`
Join table linking users to the conversations they belong to.

| Column | Type | Notes |
|---|---|---|
| `conversation_id` | `BIGINT` | **PK, FK** → `conversations.id` (CASCADE) |
| `user_id` | `UUID` | **PK, FK** → `users.user_id` (CASCADE) |
| `joined_at` | `TIMESTAMPTZ` | |

---

### `messages`
Stores messages within a conversation. Supports soft-delete via `deleted_at`.

| Column | Type | Notes |
|---|---|---|
| `id` | `BIGSERIAL` | **PK** |
| `conversation_id` | `BIGINT` | **FK** → `conversations.id` (CASCADE) |
| `sender_id` | `UUID` | **FK** → `users.user_id` (CASCADE) |
| `content` | `TEXT` | |
| `message_type` | `message_type` | `text`, `image`, `file`, `audio` |
| `media_url` | `TEXT` | nullable — S3/CDN URL for non-text messages |
| `is_edited` | `BOOLEAN` | default `false` |
| `deleted_at` | `TIMESTAMPTZ` | nullable — `NULL` = not deleted (soft delete) |
| `created_at` | `TIMESTAMPTZ` | |
| `updated_at` | `TIMESTAMPTZ` | |

---

### `message_reads`
Per-user read receipts. One row per `(message, user)` pair.

| Column | Type | Notes |
|---|---|---|
| `message_id` | `BIGINT` | **PK, FK** → `messages.id` (CASCADE) |
| `user_id` | `UUID` | **PK, FK** → `users.user_id` (CASCADE) |
| `read_at` | `TIMESTAMPTZ` | |

---

### `notifications`
Notifies a user of an event.

| Column | Type | Notes |
|---|---|---|
| `id` | `BIGSERIAL` | **PK** |
| `user_id` | `UUID` | **FK** → `users.user_id` (CASCADE) |
| `sender_id` | `UUID` | **FK** → `users.user_id` (CASCADE) |
| `type` | `notification_type` | `message`, `friend_request` |
| `message` | `TEXT` | |
| `is_read` | `BOOLEAN` | default `false` |
| `created_at` | `TIMESTAMPTZ` | |

---

## Relationship Summary

| From | To | Via | Cardinality |
|---|---|---|---|
| `users` | `friendships` | `user1_userid` | one-to-many |
| `users` | `friendships` | `user2_userid` | one-to-many |
| `users` | `friendships` | `initiator_userid` | one-to-many |
| `users` | `conversation_members` | `user_id` | one-to-many |
| `conversations` | `conversation_members` | `conversation_id` | one-to-many |
| `conversations` | `messages` | `conversation_id` | one-to-many |
| `users` | `messages` | `sender_id` | one-to-many |
| `messages` | `message_reads` | `message_id` | one-to-many |
| `users` | `message_reads` | `user_id` | one-to-many |
| `users` | `notifications` | `user_id` | one-to-many |
| `users` | `notifications` | `sender_id` | one-to-many |

---

## Indexes

| Index | Table | Columns | Purpose |
|---|---|---|---|
| `idx_users_user_id` | `users` | `user_id` | Look up users by user_id |
| `idx_conv_members_user` | `conversation_members` | `user_id` | Look up all conversations a user belongs to |
| `idx_messages_conv_time` | `messages` | `(conversation_id, created_at ASC)` | Primary read pattern — messages in a conversation ordered by time |
| `idx_messages_conv_not_deleted` | `messages` | `(conversation_id, created_at ASC) WHERE deleted_at IS NULL` | Efficiently exclude soft-deleted rows |
| `idx_message_reads_user` | `message_reads` | `(user_id, message_id)` | Check which messages a user has read |
| `idx_notifications_user_read_time` | `notifications` | `(user_id, is_read, created_at DESC)` | User inbox sorted by time, filterable by read status |
| `idx_friendships_user1_status` | `friendships` | `(user1_userid, status)` | Accepted friends lookup per user |
| `idx_friendships_user2_status` | `friendships` | `(user2_userid, status)` | Accepted friends lookup per user (other side) |
| `idx_friendships_user2_status_time` | `friendships` | `(user2_userid, status, created_at DESC)` | Pending incoming friend requests |
