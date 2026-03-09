# Chat Streaming Implementation Plan

This document outlines the implementation plan for the real-time chat and notification streaming architecture using WebSockets and NATS.

## Architecture Flow

The architecture follows a pub/sub model where the Gateway manages WebSocket connections and translates messages to and from the NATS message broker. The Backend handles business logic and database persistence.

### Notification Channel

**Direction:** One-way (Server to User)

1. **Connection & Subscription:**
   - Frontend connects to Gateway via WebSocket.
   - Gateway authenticates the user (e.g., `User1`) and automatically subscribes their connection to the NATS subject: `notifications.receive.user1`.
2. **Event Trigger:**
   - An event occurs (e.g., a friend request is created).
   - Backend publishes the notification payload to the NATS subject: `notifications.receive.user1`.
3. **Delivery:**
   - Gateway receives the message from NATS and pushes it down the WebSocket connection to User1.

### Message Channel

**Direction:** Two-way (User to User)

1. **Connection & Subscription:**
   - Uses the same WebSocket connection established above.
   - Gateway automatically subscribes the connection to the NATS subject for receiving messages: `messages.receive.user1`.
2. **Sending a Message (User1 to User2):**
   - User1 sends a WebSocket frame: `{"action": "send_message", "to": "user2", "content": "Hi!"}`.
   - Gateway receives the frame and **publishes** the payload to the NATS incoming queue: `messages.send`.
   - Backend consumes messages from `messages.send`, validates the payload, and persists it to the Database.
   - Backend **publishes** the saved message to the recipient's NATS subject: `messages.receive.user2`.
3. **Receiving a Message (User2 receives from User1):**
   - User2's Gateway connection (subscribed to `messages.receive.user2`) receives the message from NATS.
   - Gateway pushes the message payload down the WebSocket to User2.

## NATS Subject Naming Convention

| Purpose | Subject Pattern | Publisher | Subscriber |
| :--- | :--- | :--- | :--- |
| Incoming Messages | `messages.send` | Gateway | Backend |
| Outgoing Messages | `messages.receive.<user_id>` | Backend | Gateway |
| Outgoing Notifications | `notifications.receive.<user_id>` | Backend | Gateway |

## Implementation Steps

### 1. Gateway Setup
- Implement WebSocket connection handler with JWT authentication.
- Implement NATS connection established on Gateway startup.
- On successful WebSocket connection, create temporary NATS subscriptions for `messages.receive.<user_id>` and `notifications.receive.<user_id>`.
- Route incoming NATS messages to the corresponding WebSocket client.
- Route incoming WebSocket messages labeled as `send_message` to publish to the `messages.send` NATS subject.
- Ensure NATS subscriptions are properly unsubscribed / drained when the WebSocket connection closes.

### 2. Backend Setup
- Implement NATS connection on Backend startup.
- Create a durable/queue subscription to listen on `messages.send`.
- Implement message handler to validate, save to DB, and publish to `messages.receive.<recipient_id>`.
- Update existing notification logic (e.g., friend requests) to publish events to `notifications.receive.<user_id>`.

### 3. Frontend Setup
- Establish WebSocket connection to the Gateway on user login.
- Implement reconnection logic for WebSocket drops.
- Handle incoming WebSocket frames and route them to the appropriate state management (e.g., add message to chat UI, show notification toast).
