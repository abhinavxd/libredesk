package models

// Action constants for WebSocket messages.
const (
	MessageTypeMessageUpdate          = "message_update"
	MessageTypeConversationUpdate     = "conversation_update"
	MessageTypeNewMessage             = "new_message"
	MessageTypeNewConversation        = "new_conversation"
	MessageTypeNewNotification        = "new_notification"
	MessageTypeError                  = "error"
	MessageTypeConversationSubscribe  = "conversation_subscribe"
	MessageTypeTyping                 = "typing"
	MessageTypeListSubscribeReplace   = "list_subscribe_replace"
	MessageTypeAgentAvailability      = "agent_availability_update"
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
	ConversationUUID string `json:"conversation_uuid"`
	IsTyping         bool   `json:"is_typing"`
	IsPrivateMessage bool   `json:"is_private_message"`
}
