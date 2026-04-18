# Friendship API

This document details the REST API endpoints provided by the `FriendshipHandler` in the gateway service, which delegate to the underlying gRPC friendship service.

Base path: `http://<GATEWAY_ADDRESS>:8080` (or as configured by `GATEWAY_LISTEN_ADDRESS`)

All friendship endpoints require an active user session via a JSON Web Token.

---

## Authorization

All endpoints in this API require a valid JWT passed in the `Authorization` header.

- **Header:** `Authorization: Bearer <JWT_STRING>`

If the header is missing, malformed, or the token is expired/invalid, the API will return:

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "missing or malformed Authorization header" // or token expiration detail
}
```

---

## 1. Get Friends

Returns the list of accepted friends for the authenticated user.

- **URL path:** `/friends`
- **Method:** `GET`

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "data": [
    {
      "username": "alice",
      "display_name": "Alice Smith"
    }
  ]
}
```
*(Empty array if the user has no friends yet)*

#### 401 Unauthorized
Returned when the token is missing, malformed, or invalid.
```json
{
  "success": false,
  "message": "missing or malformed Authorization header"
}
```

#### 500 Internal Server Error
Returned on unexpected server errors.
```json
{
  "success": false,
  "message": "internal server error: <detail>"
}
```

---

## 2. Send Friend Request

Sends a friend request from the authenticated user to the specified target username.

- **URL path:** `/friends/request`
- **Method:** `POST`
- **Content-Type:** `application/json`

### Request Body

```json
{
  "username": "bob"
}
```
*(Where `username` is the target user to send the request to)*

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "message": "friend request sent"
}
```

#### 400 Bad Request
Returned when the request body is missing the username, or when attempting to send a request to yourself.
```json
{
  "success": false,
  "message": "invalid request body: username is required" // or "cannot send a friend request to yourself"
}
```

#### 404 Not Found
Returned when the target user does not exist.
```json
{
  "success": false,
  "message": "target bob not found: sql: no rows in result set"
}
```

#### 409 Conflict
Returned when a friend request (pending or accepted) already exists between the two users.
```json
{
  "success": false,
  "message": "friend request already exists with status: <status>"
}
```

#### 500 Internal Server Error
Returned on unexpected database or server errors.
```json
{
  "success": false,
  "message": "internal server error: <detail>"
}
```

---

## 3. Accept Friend Request

Accepts a pending friend request from the specified user. The authenticated caller must be the addressee (recipient) of the original request.

- **URL path:** `/friends/accept`
- **Method:** `POST`
- **Content-Type:** `application/json`

### Request Body

```json
{
  "username": "alice"
}
```
*(Where `username` is the user whose request you are accepting)*

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "message": "friend request accepted"
}
```

#### 400 Bad Request
Returned when the request body is missing the username.

#### 403 Forbidden
Returned when the caller attempts to accept a request they initiated themselves (only the recipient can respond).
```json
{
  "success": false,
  "message": "only the recipient can respond to a friend request"
}
```

#### 404 Not Found
Returned when no such friend request exists between the two users.
```json
{
  "success": false,
  "message": "friend request not found"
}
```

#### 409 Conflict
Returned when the friend request exists but is not in a `pending` state (e.g., already accepted or already rejected).
```json
{
  "success": false,
  "message": "friend request is not pending"
}
```

#### 500 Internal Server Error
Returned on unexpected server errors.

---

## 4. Reject Friend Request

Rejects a pending friend request from the specified user. The authenticated caller must be the addressee (recipient) of the original request.

- **URL path:** `/friends/reject`
- **Method:** `POST`
- **Content-Type:** `application/json`

### Request Body

```json
{
  "username": "alice"
}
```
*(Where `username` is the user whose request you are rejecting)*

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "message": "friend request rejected"
}
```

#### 400 Bad Request
Returned when the request body is missing the username.

#### 403 Forbidden
Returned when the caller attempts to reject a request they initiated themselves.
```json
{
  "success": false,
  "message": "only the recipient can respond to a friend request"
}
```

#### 404 Not Found
Returned when no such friend request exists between the two users.
```json
{
  "success": false,
  "message": "friend request not found"
}
```

#### 409 Conflict
Returned when the friend request exists but is not in a `pending` state (e.g., already accepted or already rejected).
```json
{
  "success": false,
  "message": "friend request is not pending"
}
```

#### 500 Internal Server Error
Returned on unexpected server errors.
