package lib

import "encoding/json"

// ChatResponseEnvelopeVersion is the current chat WebSocket protocol version.
// Increment this when making breaking changes to the message envelope format.
const ChatResponseEnvelopeVersion = 1

// ChatEventType identifies the kind of event inside a ChatResponseEnvelope.
type ChatEventType string

const (
	ChatEventMessage   ChatEventType = "message"
	ChatEventDelivered ChatEventType = "delivered"
	ChatEventRead      ChatEventType = "read"
)

// DeliveredEvent is the Data payload for ChatEventDelivered envelopes.
type DeliveredEvent struct {
	ConversationID int64 `json:"conversation_id"`
	MessageID      int64 `json:"message_id"`
}

// ReadEvent is the Data payload for ChatEventRead envelopes.
type ReadEvent struct {
	ConversationID int64 `json:"conversation_id"`
	MessageID      int64 `json:"message_id"`
}

// ChatResponseEnvelope is the typed wrapper for all NATS chat payloads
// pushed from server to client. Clients use the Type field to determine
// how to handle the Data payload.
type ChatResponseEnvelope struct {
	Version int             `json:"version"`
	Type    ChatEventType   `json:"type"`
	Data    json.RawMessage `json:"data"`
}

// NewChatResponseEnvelope marshals data into a ChatResponseEnvelope and returns the JSON bytes.
func NewChatResponseEnvelope(eventType ChatEventType, data any) ([]byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ChatResponseEnvelope{
		Version: ChatResponseEnvelopeVersion,
		Type:    eventType,
		Data:    raw,
	})
}

// ChatRequestEnvelopeVersion is the current client-to-server WebSocket protocol version.
// Increment this when making breaking changes to the request envelope format.
var ChatRequestEnvelopeVersion = 1

// ChatRequestType identifies the kind of request from client to server.
type ChatRequestType string

const (
	ChatRequestSend ChatRequestType = "send"
	ChatRequestRead ChatRequestType = "read"
)

// ChatRequestEnvelope is the typed wrapper for all WebSocket messages
// sent from client to server. The server uses the Type field to determine
// how to handle the Data payload.
type ChatRequestEnvelope struct {
	Version int             `json:"version"`
	Type    ChatRequestType `json:"type"`
	Data    json.RawMessage `json:"data"`
}
