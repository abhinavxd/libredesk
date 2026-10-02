package models

import "encoding/json"

// Action constants for WebSocket messages.
const (
	MessageTypeMessageUpdate         = "message_update"
	MessageTypeConversationUpdate    = "conversation_update"
	MessageTypeNewMessage            = "new_message"
	MessageTypeNewConversation       = "new_conversation"
	MessageTypeError                 = "error"
	MessageTypeConversationSubscribe = "conversation_subscribe"
	MessageTypeTyping                = "typing"
	MessageTypeListSubscribeReplace  = "list_subscribe_replace"
	MessageTypeAgentAvailability     = "agent_availability_update"
	MessageTypeSystemToast           = "system_toast"
	MessageTypeReviewCreated         = "review_created"
	MessageTypeReviewUpdated         = "review_updated"
)

// WSMessage represents a WS message.
type WSMessage struct {
	MessageType int
	Data        []byte
}

// Message represents a WebSocket message to be sent.
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// IncomingMessage is an inbound client frame with its payload kept raw for the per-type handlers.
type IncomingMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// BroadcastMessage represents a message to be pushed to users.
type BroadcastMessage struct {
	Data  []byte `json:"data"`
	Users []int  `json:"users"`
}

// ConversationSubscribe represents a conversation subscription message.
type ConversationSubscribe struct {
	ConversationUUID string `json:"conversation_uuid"`
}

// TypingMessage represents a typing indicator message.
type TypingMessage struct {
	UserID           int    `json:"user_id"`
	UserName         string `json:"user_name"`
	ConversationUUID string `json:"conversation_uuid"`
	IsTyping         bool   `json:"is_typing"`
	IsPrivateMessage bool   `json:"is_private_message"`
}
