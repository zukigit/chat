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
