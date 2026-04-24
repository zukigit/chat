# Version API

Returns the current chat protocol version number. Clients can use this to confirm compatibility before connecting.

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
    "version": 1
  }
}
```

The `version` field reflects `ChatProtocolVersion` in the backend (`lib/chat_envelope.go`). It is incremented on breaking changes to the WebSocket message envelope format.
