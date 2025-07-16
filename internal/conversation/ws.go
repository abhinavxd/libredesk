package conversation

import (
	"encoding/json"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
)

// BroadcastNewMessage broadcasts a new message to all users.
func (m *Manager) BroadcastNewMessage(message *cmodels.Message) {
	m.broadcastToUsers([]int{}, wsmodels.Message{
		Type: wsmodels.MessageTypeNewMessage,
		Data: map[string]interface{}{
			"conversation_uuid": message.ConversationUUID,
			"content":           message.TextContent,
			"created_at":        message.CreatedAt.Format(time.RFC3339),
			"uuid":              message.UUID,
			"private":           message.Private,
			"type":              message.Type,
			"sender_type":       message.SenderType,
		},
	})
}

// BroadcastMessageUpdate broadcasts a message update to all users.
func (m *Manager) BroadcastMessageUpdate(conversationUUID, messageUUID, prop string, value any) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeMessagePropUpdate,
		Data: map[string]interface{}{
			"conversation_uuid": conversationUUID,
			"uuid":              messageUUID,
			"prop":              prop,
			"value":             value,
		},
	}
	m.broadcastToUsers([]int{}, message)
}

// BroadcastConversationUpdate broadcasts a conversation update to all users.
func (m *Manager) BroadcastConversationUpdate(conversationUUID, prop string, value any) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeConversationPropertyUpdate,
		Data: map[string]interface{}{
			"uuid":  conversationUUID,
			"prop":  prop,
			"value": value,
		},
	}
	m.broadcastToUsers([]int{}, message)
}

// BroadcastTypingToConversation broadcasts typing status to all subscribers of a conversation.
// Set broadcastToWidgets to false when the typing event originates from a widget client to avoid echo.
func (m *Manager) BroadcastTypingToConversation(conversationUUID string, isTyping bool, broadcastToWidgets bool) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeTyping,
		Data: map[string]interface{}{
			"conversation_uuid": conversationUUID,
			"is_typing":         isTyping,
		},
	}
	
	messageBytes, err := json.Marshal(message)
	if err != nil {
		m.lo.Error("error marshalling typing WS message", "error", err)
		return
	}
	
	// Always broadcast to agent clients (main app WebSocket clients)
	m.wsHub.BroadcastTypingToAllConversationClients(conversationUUID, messageBytes)
	
	// Broadcast to widget clients (customers) only if this typing event comes from agents
	if broadcastToWidgets {
		m.broadcastTypingToWidgetClients(conversationUUID, isTyping)
	}
}

// BroadcastTypingToWidgetClientsOnly broadcasts typing status only to widget clients.
func (m *Manager) BroadcastTypingToWidgetClientsOnly(conversationUUID string, isTyping bool) {
	m.broadcastTypingToWidgetClients(conversationUUID, isTyping)
}

// broadcastToUsers broadcasts a message to a list of users, if the list is empty it broadcasts to all users.
func (m *Manager) broadcastToUsers(userIDs []int, message wsmodels.Message) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		m.lo.Error("error marshalling WS message", "error", err)
		return
	}
	m.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{
		Data:  messageBytes,
		Users: userIDs,
	})
}

// broadcastTypingToWidgetClients broadcasts typing status to widget clients (customers) for a conversation.
func (m *Manager) broadcastTypingToWidgetClients(conversationUUID string, isTyping bool) {
	// Get the conversation to find its inbox ID
	conversation, err := m.GetConversation(0, conversationUUID)
	if err != nil {
		m.lo.Error("error getting conversation for widget typing broadcast", "error", err, "conversation_uuid", conversationUUID)
		return
	}
	
	// Get the inbox
	inboxInstance, err := m.inboxStore.Get(conversation.InboxID)
	if err != nil {
		m.lo.Error("error getting inbox for widget typing broadcast", "error", err, "inbox_id", conversation.InboxID)
		return
	}
	
	// Check if it's a livechat inbox and broadcast typing status
	if liveChatInbox, ok := inboxInstance.(*livechat.LiveChat); ok {
		liveChatInbox.BroadcastTypingToClients(conversationUUID, isTyping)
	}
}
