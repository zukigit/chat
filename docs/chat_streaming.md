# Chat Streaming Architecture

This document describes the real-time message flow — from a sender's WebSocket frame through the backend to the recipient's WebSocket — using the actual components in this system.

## Components

| Component | Role |
|-----------|------|
| **Frontend** | React SPA; connects to two persistent WebSocket sessions (`/sessions/chat`, `/sessions/notification`) |
| **Gateway** | HTTP/WebSocket server; translates between REST/WS clients and gRPC backend, and owns the NATS JetStream consumers |
| **Backend** | gRPC server; handles all business logic, DB writes, and NATS publishes |
| **NATS JetStream** | Message bus with durable consumers; provides offline delivery guarantees |
| **Database** | PostgreSQL; persists messages, notifications, `last_delivered_message_id` |

## NATS Stream Configuration

The Gateway creates a single stream at startup:

| Property | Value |
|----------|-------|
| **Stream name** | `SESSIONS` |
| **Subjects** | `sessions.noti.>`, `sessions.chat.>` |
| **Storage** | File (persists to disk) |
| **Max age** | 24 hours |

Each connected device (identified by `login_id` from the JWT) creates or resumes two durable consumers:

| Consumer | Durable name | Filter subject |
|----------|-------------|----------------|
| Chat | `chat-{login_id}` | `sessions.chat.{user_id}` |
| Notification | `noti-{login_id}` | `sessions.noti.{user_id}` |

Because consumers are per-`login_id`, multiple devices logged in as the same user each receive every message independently.

---

## Message Send & Delivery Receipt Flow

```mermaid
sequenceDiagram
    autonumber

    actor A  as Frontend (User A — sender)
    participant GW as Gateway
    participant BE as Backend
    participant DB as Database
    participant NQ as NATS JetStream
    actor B  as Frontend (User B — recipient)

    note over A, GW: User A sends a message over the chat WebSocket
    A->>GW: WS frame — chatSendRequest JSON<br/>{conversation_id, content, message_type, ...}

    note over GW, DB: Gateway forwards to backend via gRPC
    GW->>BE: gRPC SendMessage(token, conversation_id, content, ...)
    BE->>DB: IsMember — verify caller is in conversation
    DB-->>BE: ✓ member

    BE->>DB: INSERT INTO messages
    DB-->>BE: saved message (id, sender_id, created_at, ...)

    BE->>DB: GetConversationMembers
    DB-->>BE: [User A, User B, ...]

    note over BE, NQ: For each member except the sender
    BE->>DB: INSERT INTO notifications (type=message, user_id=User B)
    BE->>NQ: Publish sessions.noti.{user_b_id}<br/>(raw notification JSON)
    BE->>NQ: Publish sessions.chat.{user_b_id}<br/>(ChatEnvelope {type:"message", data: message})

    BE-->>GW: SendMessageResponse {message_id}

    note over NQ, B: JetStream delivers to User B's chat consumer
    NQ->>GW: Deliver msg from consumer chat-{user_b_login_id}
    GW->>B: WS frame — ChatEnvelope {type:"message", data: {...}}
    GW->>B: WS frame — ChatEnvelope (via noti consumer)<br/>{notification JSON}

    note over GW, NQ: Gateway marks message delivered
    GW->>BE: gRPC UpdateLastDeliveredMessage(conversation_id, message_id, sender_id=User A)
    BE->>DB: UPDATE conversation_members<br/>SET last_delivered_message_id = message_id<br/>WHERE last_delivered_message_id < message_id
    DB-->>BE: 1 row updated
    BE->>NQ: Publish sessions.chat.{user_a_id}<br/>(ChatEnvelope {type:"delivered", data:{conversation_id, message_id}})

    note over NQ, A: JetStream delivers delivery receipt to User A
    NQ->>GW: Deliver msg from consumer chat-{user_a_login_id}
    GW->>A: WS frame — ChatEnvelope {type:"delivered", data:{conversation_id, message_id}}
```

---

## Offline Delivery

If User B is not connected when the message is published:

1. NATS JetStream **retains** the message in the `SESSIONS` stream (up to 24 hours).
2. The durable consumer `chat-{user_b_login_id}` remembers its position.
3. When User B reconnects and re-establishes `/sessions/chat`, the Gateway resumes the existing consumer — all unacknowledged messages are replayed immediately.
4. After each replayed message is written to the WebSocket, the Gateway calls `UpdateLastDeliveredMessage` — only the highest undelivered ID causes a DB update (the SQL guard `AND last_delivered_message_id < $new_id` prevents regressions).

---

## Notification Session Flow

The `/sessions/notification` WebSocket runs a separate durable consumer (`noti-{login_id}`) on `sessions.noti.{user_id}`. It delivers raw `Notification` JSON objects (persisted in the DB) for events such as incoming messages, friend requests, and friend request responses. The same offline-delivery guarantee applies.

```mermaid
sequenceDiagram
    actor Client as Frontend
    participant GW as Gateway
    participant NQ as NATS JetStream

    Client->>GW: GET /sessions/notification (WS upgrade, Bearer JWT)
    GW->>GW: ValidateSession (gRPC → Backend)
    GW->>NQ: CreateOrUpdateConsumer noti-{login_id}<br/>filter: sessions.noti.{user_id}
    loop While connected
        NQ->>GW: Deliver retained + new notifications
        GW->>Client: WS frame — raw Notification JSON
        GW->>NQ: Ack
    end
```

---

## Envelope Format Reference

All server-to-client chat WebSocket frames use this structure:

```json
{
  "version": 1,
  "type": "message | delivered",
  "data": { ... }
}
```

| `type` | `data` payload |
|--------|---------------|
| `"message"` | Full `Message` row from DB (`id`, `conversation_id`, `sender_id`, `content`, `message_type`, `created_at`, ...) |
| `"delivered"` | `{ "conversation_id": 42, "message_id": 149 }` |
