# Routing Guard Flow

This document details the authentication and routing guard flow for the front end.

## Flow Diagrams

### 1. Sequence Diagram
The following sequence diagram outlines the entire process from logging in to verifying access against a private route.

```mermaid
sequenceDiagram
    autonumber
    
    box rgba(0, 150, 255, 0.1) Authentication Setup
    participant F as Front End
    participant B as Backend
    end
    
    F->>B: Send credentials
    B-->>F: Return JWT Token
    F->>F: Save username as 'loggedin_username' in cookie
    F->>F: Save token in cookie

    box rgba(0, 200, 50, 0.1) Routing Guard Verification
    participant U as User
    participant R as Private Route (Frontend)
    end
    
    U->>R: Attempt to access private route
    R->>R: Get token from cookie
    R->>R: Extract username from JWT payload
    R->>R: Get 'loggedin_username' from cookie
    
    alt Usernames are identical
        R->>U: Allow access to the route
    else Usernames mismatch or missing
        R->>U: Deny access (Redirect to authentication/login)
    end
```

### 2. Flowchart
A step-by-step visual map based on the required guard logic:

```mermaid
flowchart TD
    %% Login Phase
    subgraph Login Flow
        F1[Front End] -->|Sends credentials| B[Backend]
        B -->|Returns token| F1
        F1 -->|Saves| C1[(Cookie: loggedin_username)]
        F1 -->|Saves| C2[(Cookie: token)]
    end
    
    %% Guard Phase
    subgraph Routing Guard
        U[User] -->|Attempts to visit| P[Private Route]
        P --> F2[Frontend Route Guard]
        F2 -->|1. Get token from cookie| T[JWT Payload]
        T -->|Extract| U1[Username from JWT]
        
        F2 -->|2. Get logged in username| U2[loggedin_username Cookie]
        
        U1 --> C{Are these two <br/>usernames identical?}
        U2 --> C
        
        C -->|Yes| Allow((Let user enter))
        C -->|No| Deny((Block / Redirect))
    end
```
