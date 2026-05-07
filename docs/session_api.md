# Session API (WebSocket)

This document details the two real-time WebSocket endpoints provided by the `SessionHandler` in the gateway service.

Base path: `ws://<GATEWAY_ADDRESS>:8080` (or `wss://` over TLS)

Both endpoints upgrade an HTTP GET request to a WebSocket connection. They require a valid JWT (passed in the `Authorization` header or via the `?token=` query parameter).

---

## Authentication

The JWT must contain `login_id` (UUID) and `user_id` (UUID). On connection:

1. The gateway parses the JWT claims (without signature verification — the gRPC backend enforces auth).
2. `GetListenPath` is called to validate the token and receive the NATS subject to subscribe to.
3. A durable JetStream consumer is created (or resumed) scoped to this specific login.

Token can be provided in two ways:

```
Authorization: Bearer <JWT_STRING>
```
or
```
GET /sessions/chat?token=<JWT_STRING>
```

If the token is missing, invalid, or the session has been invalidated (logout), the server responds with HTTP 401 before the WebSocket upgrade.

---

## Message Envelope Format

All messages pushed **from the server to the client** over both sessions use the `ChatResponseEnvelope` JSON structure:

```json
{
  "version": 1,
  "type": "<event_type>",
  "data": { ... }
}
```

| Field     | Type     | Description                                  |
|-----------|----------|----------------------------------------------|
| `version` | `int`    | Protocol version (currently `1`)             |
| `type`    | `string` | Event type: `"message"`, `"delivered"`, `"read"`, or `"error"` |
| `data`    | `object` | Event-specific payload (see types below)     |

### Event: `message`

A new chat message was delivered to the user.

```json
{
  "version": 1,
  "type": "message",
  "data": {
    "id": 149,
    "conversation_id": 42,
    "sender_id": "550e8400-e29b-41d4-a716-446655440000",
    "reply_to_message_id": null,
    "content": "hello!",
    "message_type": "text",
    "media_url": null,
    "is_edited": false,
    "deleted_at": null,
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

### Event: `delivered`

Notifies the sender that their message has been delivered to a recipient.

```json
{
  "version": 1,
  "type": "delivered",
  "data": {
    "conversation_id": 42,
    "message_id": 149
  }
}
```

### Event: `read`

Notifies the sender that their message has been read by the recipient.

```json
{
  "version": 1,
  "type": "read",
  "data": {
    "conversation_id": 42,
    "message_id": 149
  }
}
```

### Event: `error`

An error occurred during the WebSocket session. Sent over the WebSocket connection (after upgrade) when the server encounters a problem processing a request or subscribing to NATS.

```json
{
  "version": 1,
  "type": "error",
  "data": {
    "code": 500,
    "message": "failed to send message: connection refused"
  }
}
```

| Field     | Type     | Description                                  |
|-----------|----------|----------------------------------------------|
| `code`    | `int`    | HTTP-style error code (`400` for client errors, `500` for server errors) |
| `message` | `string` | Human-readable error description             |

---

## 1. Notification Session

Establishes a persistent WebSocket connection for receiving real-time notifications. Messages are delivered from the NATS JetStream consumer `noti-{login_id}` filtered to subject `sessions.noti.{user_id}`.

- **URL path:** `/sessions/notification`
- **Method:** `GET` (WebSocket upgrade)
- **Protocol:** WebSocket

### Behavior

- On connect, an `AckExplicit` durable consumer is created or resumed. Undelivered notifications from previous connections are replayed automatically.
- Each NATS message is pushed as a raw WebSocket text frame to the client.
- Messages are acknowledged after being written to the WebSocket.
- If the consumer is deleted server-side, the server sends a WebSocket close frame (`1001 Going Away`) and terminates the connection.
- The connection is closed when the client disconnects (read error on the WebSocket).
- The consumer persists for 24 hours after the last active connection, enabling offline delivery replay.

### Error Responses (HTTP, before upgrade)

| Status | Cause                                               |
|--------|-----------------------------------------------------|
| 401    | Missing/invalid token                               |
| 500    | Failed to create JetStream consumer or upgrade WS   |

### Error Responses (WebSocket, after upgrade)

If the JetStream subscription fails after the WebSocket upgrade, an `error` event is sent:

```json
{
  "version": 1,
  "type": "error",
  "data": {
    "code": 500,
    "message": "failed to subscribe to notifications: ..."
  }
}
```

---

## 2. Chat Session

Establishes a bidirectional WebSocket connection for sending and receiving chat messages in real time. Messages are delivered from the NATS JetStream consumer `chat-{login_id}` filtered to subject `sessions.chat.{user_id}`.

- **URL path:** `/sessions/chat`
- **Method:** `GET` (WebSocket upgrade)
- **Protocol:** WebSocket

### Behavior

**Receiving messages (server → client):**
- NATS messages are pushed as `ChatResponseEnvelope` JSON text frames to the client.
- For `"message"` events, the gateway automatically calls `UpdateLastDeliveredMessage` to track delivery and notify the sender with a `"delivered"` event.
- Messages are acknowledged after successful write.
- The consumer persists for 24 hours after last active connection.

**Sending messages (client → server):**

The client sends JSON frames wrapped in a `ChatRequestEnvelope`:

```json
{
  "version": 1,
  "type": "<request_type>",
  "data": { ... }
}
```

The `version` must match the current `chat_request_version` returned by `GET /version`.

### Request: `send`

Post a message to a conversation:

```json
{
  "version": 1,
  "type": "send",
  "data": {
    "conversation_id": 42,
    "content": "hello!",
    "message_type": "text",
    "reply_to_message_id": 0
  }
}
```

| Field                | Type     | Required | Description                                        |
|----------------------|----------|----------|----------------------------------------------------|
| `conversation_id`    | `int64`  | Yes      | Target conversation                                |
| `content`            | `string` | Yes      | Message body                                       |
| `message_type`       | `string` | No       | Message type (e.g. `"text"`, `"image"`); default `"text"` |
| `reply_to_message_id`| `int64`  | No       | ID of the message being replied to (`0` = none)    |

### Request: `read`

Mark messages in a conversation as read:

```json
{
  "version": 1,
  "type": "read",
  "data": {
    "conversation_id": 42,
    "message_id": 149,
    "sender_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

| Field                | Type     | Required | Description                                        |
|----------------------|----------|----------|----------------------------------------------------|
| `conversation_id`    | `int64`  | Yes      | Target conversation                                |
| `message_id`         | `int64`  | Yes      | ID of the message that was read                    |
| `sender_id`          | `string` | Yes      | UUID of the original message sender                |

Invalid or unparseable frames from the client result in an `error` event sent back over the WebSocket (the connection remains open).

### Error Responses (HTTP, before upgrade)

| Status | Cause                                               |
|--------|-----------------------------------------------------|
| 401    | Missing/invalid token                               |
| 500    | Failed to create JetStream consumer or upgrade WS   |

### Error Responses (WebSocket, after upgrade)

Errors that occur after the WebSocket upgrade are sent as `error` events:

| Cause | Code | Example Message |
|-------|------|-----------------|
| Invalid message format | 400 | `"invalid message format: ..."` |
| Invalid send request payload | 400 | `"invalid send request: ..."` |
| Invalid read request payload | 400 | `"invalid read request: ..."` |
| Unknown request type | 400 | `"unknown request type: \"foo\""` |
| Failed to send message (gRPC error) | 500 | `"failed to send message: ..."` |
| Failed to update read status (gRPC error) | 500 | `"failed to update read status: ..."` |
| Failed to subscribe to NATS | 500 | `"failed to subscribe to chat: ..."` |
| Invalid server message from NATS | 500 | `"invalid server message: ..."` |
| Invalid message data from NATS | 500 | `"invalid message data: ..."` |
| Failed to write to WebSocket | 500 | `"failed to deliver message: ..."` |
| Failed to update delivery status | 500 | `"failed to update delivery status: ..."` |
