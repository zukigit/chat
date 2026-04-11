package lib

import "encoding/json"

// ProtocolVersion is the current chat WebSocket protocol version.
// Increment this when making breaking changes to the message envelope format.
const ProtocolVersion = 1

// ChatEventType identifies the kind of event inside a ChatEnvelope.
type ChatEventType string

const (
	ChatEventMessage   ChatEventType = "message"
	ChatEventDelivered ChatEventType = "delivered"
)

// ChatEnvelope is the typed wrapper for all NATS chat payloads.
// Clients use the Type field to determine how to handle the Data payload.
type ChatEnvelope struct {
	Version int             `json:"version"`
	Type    ChatEventType   `json:"type"`
	Data    json.RawMessage `json:"data"`
}

// DeliveredEvent is the Data payload for ChatEventDelivered envelopes.
type DeliveredEvent struct {
	ConversationID int64 `json:"conversation_id"`
	MessageID      int64 `json:"message_id"`
}

// NewChatEnvelope marshals data into a ChatEnvelope and returns the JSON bytes.
func NewChatEnvelope(eventType ChatEventType, data any) ([]byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(ChatEnvelope{
		Version: ProtocolVersion,
		Type:    eventType,
		Data:    raw,
	})
}
