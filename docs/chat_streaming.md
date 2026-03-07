# Chat Streaming Architecture

This document analyzes the architecture for real-time chat streaming using WebSockets, an inline Gateway, a NATS message queue, and a Backend interacting with a Database.

## Sequence Diagram

This sequence diagram illustrates the step-by-step flow of sending a message from one user to another. It includes the path from the sender's frontend down to the database, and back up to the recipient's frontend.

```mermaid
sequenceDiagram
    autonumber
    
    actor F1 as Frontend (User A)
    participant GW as API Gateway
    participant NQ as NATS
    participant BE as Backend Service
    participant DB as Database
    actor F2 as Frontend (User B)

    note over F1, DB: Client Message Submission
    F1->>GW: Send Message (WebSocket frame)
    GW->>NQ: Publish message to subject (e.g. `chat.incoming`)
    
    note over NQ, DB: Message Processing & Persistence
    NQ->>BE: Deliver message to backend subscriber
    BE->>BE: Validate & Sanitize payload
    BE->>DB: Save to `messages` table
    DB-->>BE: Acknowledge save success
    
    note over BE, F2: Message Fan-out & Delivery
    BE->>NQ: Publish event to recipient's subject (e.g. `chat.user.<UserB_ID>`)
    NQ->>GW: Deliver to Gateway holding User B's connection
    GW->>F2: Push message (WebSocket frame)
    
    note over GW, DB: Delivery Ack back to User A
    GW->>NQ: Publish message to subject (e.g. `chat.sent.<UserA_ID>`)
    NQ->>BE: Deliver to backend subscriber
    BE->>BE: Validate & Sanitize payload
    BE->>DB: Save to `messages` table
    DB-->>BE: Acknowledge save success
    BE->>NQ: Publish ack to sender's subject (e.g. `chat.user.<UserA_ID>`)
    NQ->>GW: Deliver to Gateway holding User A's connection
    GW->>F1: Push 'Message Sent' confirmation
```
