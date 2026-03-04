# Authentication API

This document details the REST API endpoints provided by the `AuthHandler` in the gateway service, which delegate to the underlying gRPC authentication service.

Base path: `http://<GATEWAY_ADDRESS>:8080` (or as configured by `GATEWAY_LISTEN_ADDRESS`)

---

## 1. Login

Authenticates a user and returns a JSON Web Token (JWT).

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
Returned when the request body is malformed or invalid JSON.
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

#### 409 Conflict
Returned when attempting to log in while already having an active/valid session (if enforced by the backend).
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
Returned when the request body is malformed or invalid JSON.
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
