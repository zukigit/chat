# Authentication Flow Implementation Plan

This document outlines the plan to complete the authentication system, following the specified flow:
`front_end` -> `REST` -> `gateway` -> `gRPC` -> `backend`

## Proposed Changes

### [Component] Gateway - Clients
Implement a gRPC client to facilitate communication between the Gateway and the Backend.

#### [MODIFY] [auth_client.go](file:///root/Documents/chat/backend/internal/clients/auth_client.go)
- Define `AuthClient` struct.
- Implement `NewAuthClient` constructor that establishes a gRPC connection to the backend.
- Implement `Login` and `Signup` methods that wrap the gRPC calls.

### [Component] Gateway - Handlers
Update the REST handlers to use the gRPC client instead of returning mock data.

#### [MODIFY] [auth_handler.go](file:///root/Documents/chat/backend/internal/handlers/auth_handler.go)
- Inject `AuthClient` into the handlers (or use a global instance if preferred by the architecture).
- In `LoginHandler`:
    - Call `AuthClient.Login`.
    - Handle errors and return appropriate REST responses.
- In `SignupHandler`:
    - Call `AuthClient.Signup`.
    - Handle errors and return appropriate REST responses.

## Verification Plan

### Manual Verification
1. **Signup Test**:
    - Trigger a POST request to `/signup` via a REST client (e.g., `curl`).
    - Verify that a success response is returned and a user is created in the backend database.
2. **Login Test**:
    - Trigger a POST request to `/login` via a REST client.
    - Verify that a valid JWT token is returned upon successful authentication.
    - Verify that incorrect credentials return an unauthorized error.

### Automated Tests
- Run existing tests in `backend/internal/services/auth_service_test.go` to ensure core auth logic is sound.
- Create a new integration test for the Gateway handlers if environment allows for local gRPC server execution.
