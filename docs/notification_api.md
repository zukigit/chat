# Notification API

This document details the REST API endpoints provided by the `NotificationHandler` in the gateway service.

Base path: `http://<GATEWAY_ADDRESS>:8080` (or as configured by `GATEWAY_LISTEN_ADDRESS`)

All endpoints require a valid JWT passed in the `Authorization` header.

---

## Authorization

- **Header:** `Authorization: Bearer <JWT_STRING>`

If the header is missing or malformed:

```json
{
  "success": false,
  "message": "missing or malformed Authorization header"
}
```

---

## 1. Mark Notification as Read

Marks a notification as read for the authenticated user.

- **URL path:** `/notifications/read`
- **Method:** `POST`
- **Content-Type:** `application/json`

### Request Body

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

| Field | Type     | Required | Description              |
|-------|----------|----------|--------------------------|
| `id`  | `string` | Yes      | UUID of the notification |

### Responses

#### 200 OK (Success)

```json
{
  "success": true,
  "message": "notification status updated"
}
```

#### 400 Bad Request
Returned when the request body is malformed or the `id` field is invalid.
```json
{
  "success": false,
  "message": "invalid request body: <detail>"
}
```

#### 401 Unauthorized
```json
{
  "success": false,
  "message": "missing or malformed Authorization header"
}
```

#### 404 Not Found
Returned when no notification with the specified ID exists (or it does not belong to the caller).
```json
{
  "success": false,
  "message": "<error detail from backend>"
}
```

#### 500 Internal Server Error
```json
{
  "success": false,
  "message": "<error detail>"
}
```
