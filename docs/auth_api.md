# Authentication API

This document details the REST API endpoints provided by the `AuthHandler` in the gateway service, which delegate to the underlying gRPC authentication service.

Base path: `http://<GATEWAY_ADDRESS>:8080` (or as configured by `GATEWAY_LISTEN_ADDRESS`)

---

## 1. Login

Authenticates a user and returns a JSON Web Token (JWT). The JWT contains the user's `user_id` and a unique `login_id` (UUID generated per login), which identifies the session on the server.

- **URL path:** `/login`
- **Method:** `POST`
- **Content-Type:** `application/json`

### Request Body

```json
{
  "username": "alice",
  "password": "my_secret_password"
}
```

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "message": "login successful",
  "data": {
    "token": "<JWT_STRING>"
  }
}
```

#### 400 Bad Request
Returned when the request body is malformed, invalid JSON, or a required field is missing.
```json
{
  "success": false,
  "message": "invalid request body"
}
```

#### 401 Unauthorized
Returned when the username does not exist or the password is incorrect.
```json
{
  "success": false,
  "message": "<error detail from backend>"
}
```

#### 500 Internal Server Error
Returned on unexpected server errors or backend unavailability.
```json
{
  "success": false,
  "message": "<error detail>"
}
```

---

## 2. Signup

Registers a new user account.

- **URL path:** `/signup`
- **Method:** `POST`
- **Content-Type:** `application/json`

### Request Body

```json
{
  "username": "alice",
  "password": "my_secret_password"
}
```

### Responses

#### 201 Created (Success)

```json
{
  "success": true,
  "message": "user registered successfully"
}
```

#### 400 Bad Request
Returned when the request body is malformed, invalid JSON, or a required field is missing.
```json
{
  "success": false,
  "message": "invalid request body"
}
```

#### 409 Conflict
Returned when a user with the specified `username` already exists.
```json
{
  "success": false,
  "message": "<error detail from backend>"
}
```

#### 500 Internal Server Error
Returned on unexpected server errors or backend unavailability.
```json
{
  "success": false,
  "message": "<error detail>"
}
```

---

## 3. Logout

Invalidates the caller's current session by deleting it from the server. The `login_id` embedded in the JWT is used to identify which session to remove. Multiple devices (each with their own `login_id`) are unaffected.

- **URL path:** `/logout`
- **Method:** `POST`
- **Authorization:** `Bearer <JWT_STRING>`

### Request Body

None.

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "message": "logged out"
}
```

#### 401 Unauthorized
Returned when the `Authorization` header is missing, or the token is invalid/expired.
```json
{
  "success": false,
  "message": "Missing token"
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

## 4. Search Users

Searches for users by username or display name. Returns a list of matching users (up to 50 results).

- **URL path:** `/users/search`
- **Method:** `GET`
- **Authorization:** `Bearer <JWT_STRING>`

### Query Parameters

| Parameter | Type   | Required | Description                          |
|-----------|--------|----------|--------------------------------------|
| `q`       | string | Yes      | Search term (matches `user_name` or `display_name`, case-insensitive) |

### Example Request

```
GET /users/search?q=zuki
Authorization: Bearer <JWT_STRING>
```

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "message": "users found",
  "data": [
    {
      "user_id": "550e8400-e29b-41d4-a716-446655440000",
      "user_name": "zuki",
      "display_name": "Zuki",
      "avatar_url": "https://example.com/avatar1.png"
    },
    {
      "user_id": "660e8400-e29b-41d4-a716-446655440001",
      "user_name": "zuki_sama",
      "display_name": "Zuki Kazumi",
      "avatar_url": "https://example.com/avatar2.png"
    }
  ]
}
```

If no users match, returns an empty array:

```json
{
  "success": true,
  "message": "users found",
  "data": []
}
```

#### 400 Bad Request
Returned when the `q` query parameter is missing or empty.
```json
{
  "success": false,
  "message": "q query parameter is required"
}
```

#### 401 Unauthorized
Returned when the `Authorization` header is missing, or the token is invalid/expired.
```json
{
  "success": false,
  "message": "Missing token"
}
```

#### 500 Internal Server Error
Returned on unexpected server errors or backend unavailability.
```json
{
  "success": false,
  "message": "<error detail>"
}
```
