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

## 2. Get Messages

Retrieves messages in a conversation, ordered from newest to oldest (cursor-based pagination).

- **URL path:** `/conversations/messages`
- **Method:** `GET`

### Query Parameters

| Parameter         | Type    | Required | Description                                                            |
|-------------------|---------|----------|------------------------------------------------------------------------|
| `conversation_id` | `int64` | Yes      | The ID of the conversation to fetch messages from                      |
| `limit`           | `int32` | No       | Maximum number of messages to return (default determined by backend)   |
| `cursor`          | `int64` | No       | Message ID to paginate from; returns messages older than this ID       |

### Example Request

```
GET /conversations/messages?conversation_id=42&limit=20&cursor=150
Authorization: Bearer <JWT_STRING>
```

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "data": {
    "messages": [
      {
        "id": 149,
        "conversation_id": 42,
        "sender_id": "550e8400-e29b-41d4-a716-446655440000",
        "reply_to_message_id": null,
        "content": "hello world",
        "message_type": "text",
        "media_url": null,
        "is_edited": false,
        "deleted_at": null,
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ],
    "next_cursor": 100
  }
}
```

#### 400 Bad Request
Returned when `conversation_id` is missing or non-numeric, or `limit`/`cursor` are non-numeric.
```json
{
  "success": false,
  "message": "conversation_id is required"
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
Returned when the caller is not a member of the specified conversation.
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
