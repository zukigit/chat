# Version API

Returns the current chat protocol version numbers. Clients should call this before establishing a WebSocket chat session to ensure they use the correct `ChatRequestEnvelope` version.

Base path: `http://<GATEWAY_ADDRESS>:8080` (or as configured by `GATEWAY_LISTEN_ADDRESS`)

---

## Get Version

- **URL path:** `/version`
- **Method:** `GET`
- **Authorization:** None required

### Responses

#### 200 OK

```json
{
  "success": true,
  "data": {
    "chat_request_version": 1
  }
}
```

| Field                  | Type  | Description                                                        |
|------------------------|-------|--------------------------------------------------------------------|
| `chat_request_version` | `int` | Version to use in `ChatRequestEnvelope.version` for client requests |

The server-to-client `ChatResponseEnvelope.version` is included in every response frame — clients do not need to fetch it separately.
