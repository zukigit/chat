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

    note over F1, F2: Immediate Message Delivery
    F1->>GW: Send Message (WebSocket frame)
    GW->>F2: Forward message to User B
    
    note over GW, DB: Asynchronous Persistence
    GW->>NQ: Publish message to subject (e.g. `chat.incoming`)
    NQ->>BE: Deliver message to backend subscriber
    BE->>DB: Save to `messages` table
    
    note over BE, F1: Delivery Confirmation
    BE->>NQ: Publish ack to sender's subject (e.g. `chat.user.<UserA_ID>`)
    NQ->>GW: Deliver ack to Gateway holding User A's connection
    GW->>F1: Push 'Message Sent' confirmation
```
