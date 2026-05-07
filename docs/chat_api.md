# Chat API

This document details the REST API endpoints provided by the `ChatHandler` in the gateway service.

Base path: `http://<GATEWAY_ADDRESS>:8080` (or as configured by `GATEWAY_LISTEN_ADDRESS`)

All endpoints require a valid JWT passed in the `Authorization` header.

---

## Authorization

- **Header:** `Authorization: Bearer <JWT_STRING>`

If the header is missing or malformed:

```json
{
  "success": false,
  "message": "missing or malformed Authorization header"
}
```

---

## 1. Create Conversation

Creates a new conversation (direct or group). The authenticated user is automatically added as a member.

- **URL path:** `/conversations`
- **Method:** `POST`
- **Content-Type:** `application/json`

### Request Body

```json
{
  "is_group": false,
  "name": "",
  "members_username": ["bob"]
}
```

| Field              | Type       | Required | Description                                             |
|--------------------|------------|----------|---------------------------------------------------------|
| `is_group`         | `bool`     | Yes      | `true` for a group conversation, `false` for direct DM |
| `name`             | `string`   | No       | Display name; required when `is_group` is `true`        |
| `members_username` | `[]string` | Yes      | Usernames to add (excluding the caller)                 |

### Responses

#### 201 Created (Success)

```json
{
  "success": true,
  "message": "conversation created",
  "data": {
    "conversation_id": 42
  }
}
```

#### 400 Bad Request
Returned when required fields are missing or invalid.
```json
{
  "success": false,
  "message": "<error detail from backend>"
}
```

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "missing or malformed Authorization header"
}
```

#### 403 Forbidden
Returned when the caller does not have permission to add one of the specified members (e.g. not friends).
```json
{
  "success": false,
  "message": "<error detail from backend>"
}
```

#### 500 Internal Server Error
```json
{
  "success": false,
  "message": "<error detail>"
}
```

---

## 2. Get Conversations

Retrieves all conversations that the authenticated user is a member of.

- **URL path:** `/conversations`
- **Method:** `GET`

### Example Request

```
GET /conversations
Authorization: Bearer <JWT_STRING>
```

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "data": {
    "conversations": [
      {
        "id": 42,
        "is_group": true,
        "name": "Project Team",
        "updated_at": "2024-01-15T10:30:00Z",
        "members": [
          {
            "user_id": "550e8400-e29b-41d4-a716-446655440000",
            "username": "alice",
            "display_name": "Alice",
            "avatar_url": ""
          }
        ]
      }
    ]
  }
}
```

| Field              | Type                    | Description                                    |
|--------------------|-------------------------|------------------------------------------------|
| `id`               | `int64`                 | Conversation ID                                |
| `is_group`         | `bool`                  | Whether this is a group conversation           |
| `name`             | `string`                | Group name (empty for DMs)                     |
| `updated_at`       | `string` (RFC3339)      | Last update time                               |
| `members`          | `ConversationMember[]`  | All members of the conversation                |
| `members.user_id`  | `string` (UUID)         | Member's user ID                               |
| `members.username` | `string`                | Member's username                              |
| `members.display_name` | `string`            | Member's display name                          |
| `members.avatar_url`   | `string`            | Member's avatar URL                            |

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "missing or malformed Authorization header"
}
```

#### 500 Internal Server Error
```json
{
  "success": false,
  "message": "<error detail>"
}
```

---

## 3. Search Conversations by Name

Searches conversations that the authenticated user is a member of.
- For **group** conversations: matches against the conversation `name`.
- For **DM** conversations: matches against the other member's `username`.

- **URL path:** `/conversations/search`
- **Method:** `GET`

### Query Parameters

| Parameter | Type     | Required | Description                                                        |
|-----------|----------|----------|--------------------------------------------------------------------|
| `name`    | `string` | Yes      | Search pattern (case-insensitive substring match)                  |

### Example Request

```
GET /conversations/search?name=alice
Authorization: Bearer <JWT_STRING>
```

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "data": {
    "conversations": [
      {
        "id": 42,
        "is_group": false,
        "name": "",
        "updated_at": "2024-01-15T10:30:00Z",
        "members": [
          {
            "user_id": "550e8400-e29b-41d4-a716-446655440000",
            "username": "alice",
            "display_name": "Alice",
            "avatar_url": ""
          }
        ]
      }
    ]
  }
}
```

#### 400 Bad Request
Returned when `name` query parameter is missing or empty.
```json
{
  "success": false,
  "message": "name query parameter is required"
}
```

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "missing or malformed Authorization header"
}
```

#### 500 Internal Server Error
```json
{
  "success": false,
  "message": "<error detail>"
}
```
