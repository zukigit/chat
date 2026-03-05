# Chat Streaming Architecture

This document analyzes the architecture for real-time chat streaming using WebSockets, an inline Gateway, a NATS message queue, and a Backend interacting with a Database.

## Architecture Analysis

The proposed architecture introduces a robust, decoupled, and scalable approach to handling real-time chat:

1. **Frontend**: Establishes a persistent, full-duplex WebSocket connection to the Gateway. This allows for low-latency, real-time message pushing and receiving without the overhead of HTTP polling.
2. **Gateway**: Acts as the connection manager. It terminates the WebSocket connections from thousands of clients, handling the stateful connections. It translates incoming WebSocket frames into NATS messages and vice-versa, effectively offloading connection management from the core backend services.
3. **NATS Message Queue**: Serves as the high-performance central nervous system of the chat architecture. It decouples the Gateway and Backend. Gateways publish messages to subjects, and Backends subscribe to them. NATS supports publish-subscribe patterns which are perfect for routing messages to specific users or chat rooms.
4. **Backend**: Contains the core business logic. As a stateless service, it subscribes to NATS, processes incoming messages (e.g., validation, sanitization, saving to the database), and then publishes the necessary outgoing messages or events back to NATS for distribution to recipients.
5. **Database (DB)**: The persistent storage layer maintaining chat histories, user states, and conversational metadata. The database is accessed securely and exclusively by the Backend.

### Key Benefits
- **Scalability**: The Gateway layer can be scaled horizontally to support more concurrent WebSocket connections. The Backend layer can be scaled independently to support higher message processing throughput.
- **Decoupling**: If the Backend crashes or is redeployed, the Gateway can maintain the open WebSocket connections with the Frontend, preventing mass disconnects. 
- **Performance**: NATS provides extremely low-latency messaging between the Gateway and Backend components.

---

## Flow Diagram

This diagram visualizes the high-level components and the protocols used for communication between them.

```mermaid
graph TD
    classDef client fill:#d4edda,stroke:#28a745,stroke-width:2px;
    classDef gateway fill:#cce5ff,stroke:#007bff,stroke-width:2px;
    classDef queue fill:#fff3cd,stroke:#ffc107,stroke-width:2px;
    classDef backend fill:#e2e3e5,stroke:#6c757d,stroke-width:2px;
    classDef db fill:#f8d7da,stroke:#dc3545,stroke-width:2px;

    subgraph Client Layer
        F[Frontend Client]:::client
    end

    subgraph Edge / API Layer
        GW[API Gateway<br/>WebSocket Server]:::gateway
    end

    subgraph Messaging Layer
        NATS{{NATS Message Broker}}:::queue
    end

    subgraph Compute Layer
        BE[Chat Backend Service]:::backend
    end

    subgraph Storage Layer
        DB[(Database)]:::db
    end

    F <-->|WebSocket Connection<br/>ws:// / wss://| GW
    GW <-->|Publish / Subscribe| NATS
    NATS <-->|Publish / Subscribe| BE
    BE <-->|Read / Write| DB
```

---

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
    
    note right of BE: Optional Delivery Ack back to User A
    BE->>NQ: Publish ack to sender's subject (e.g. `chat.user.<UserA_ID>`)
    NQ->>GW: Deliver to Gateway holding User A's connection
    GW->>F1: Push 'Message Sent' confirmation
```
