package conversation

import (
	"encoding/json"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/ws"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
)

// BroadcastNewConversation signals all agents that a new conversation exists.
func (m *Manager) BroadcastNewConversation() {
	m.broadcastToUsers([]int{}, wsmodels.Message{
		Type: wsmodels.MessageTypeNewConversation,
	})
}

// BroadcastNewMessage broadcasts a new message to subscribers.
func (m *Manager) BroadcastNewMessage(message *cmodels.Message, preview string) {
	data := map[string]any{
		"conversation_uuid": message.ConversationUUID,
		"uuid":              message.UUID,
		"preview":           preview,
		"created_at":        message.CreatedAt.Format(time.RFC3339),
		"sender_type":       message.SenderType,
	}

	// Include echo_id from meta so clients can match WS events to pending messages.
	var meta map[string]any
	if len(message.Meta) > 0 {
		if err := json.Unmarshal(message.Meta, &meta); err == nil {
			if echoID, ok := meta["echo_id"].(string); ok && echoID != "" {
				data["echo_id"] = echoID
			}
		}
	}

	m.broadcastToConversationListSubs(message.ConversationUUID, wsmodels.Message{
		Type: wsmodels.MessageTypeNewMessage,
		Data: data,
	})
}

// BroadcastMessageUpdate broadcasts a partial message update to list subscribers.
func (m *Manager) BroadcastMessageUpdate(conversationUUID, messageUUID string, data map[string]any) {
	data["conversation_uuid"] = conversationUUID
	data["uuid"] = messageUUID
	m.broadcastToConversationListSubs(conversationUUID, wsmodels.Message{
		Type: wsmodels.MessageTypeMessageUpdate,
		Data: data,
	})
}

// BroadcastConversationUpdate broadcasts a partial conversation update to list subscribers.
func (m *Manager) BroadcastConversationUpdate(conversationUUID string, data map[string]any) {
	data["uuid"] = conversationUUID
	m.broadcastToConversationListSubs(conversationUUID, wsmodels.Message{
		Type: wsmodels.MessageTypeConversationUpdate,
		Data: data,
	})
}

func (m *Manager) BroadcastContactUpdate(contactID int, data map[string]any) {
	data["contact_id"] = contactID
	var uuids []string
	if err := m.q.GetConversationUUIDsByContact.Select(&uuids, contactID); err != nil {
		m.lo.Error("error fetching contact's conversations for broadcast", "contact_id", contactID, "error", err)
		return
	}
	if len(uuids) == 0 {
		return
	}
	seen := map[*ws.Client]struct{}{}
	for _, uuid := range uuids {
		for _, c := range m.wsHub.ListSubscribers(uuid) {
			seen[c] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return
	}
	clients := make([]*ws.Client, 0, len(seen))
	for c := range seen {
		clients = append(clients, c)
	}
	messageBytes, err := json.Marshal(wsmodels.Message{
		Type: "contact_update",
		Data: data,
	})
	if err != nil {
		m.lo.Error("error marshalling contact_update WS message", "error", err)
		return
	}
	m.wsHub.PushToClients(clients, messageBytes)
}

// BroadcastTypingToConversation broadcasts typing status to all subscribers of a conversation.
// Set broadcastToWidgets to false when the typing event originates from a widget client to avoid echo.
func (m *Manager) BroadcastTypingToConversation(conversationUUID string, isTyping bool, broadcastToWidgets bool) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeTyping,
		Data: map[string]any{
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

// broadcastToConversationListSubs pushes a message to the conversation's list and open subscribers.
func (m *Manager) broadcastToConversationListSubs(conversationUUID string, message wsmodels.Message) {
	clients := m.wsHub.ListSubscribers(conversationUUID)
	if len(clients) == 0 {
		return
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		m.lo.Error("error marshalling WS message", "error", err)
		return
	}
	m.wsHub.PushToClients(clients, messageBytes)
}

// broadcastTypingToWidgetClients broadcasts typing status to widget clients (customers) for a conversation.
func (m *Manager) broadcastTypingToWidgetClients(conversationUUID string, isTyping bool) {
	conversation, err := m.GetConversation(0, conversationUUID, "")
	if err != nil {
		m.lo.Error("error getting conversation for widget typing broadcast", "error", err, "conversation_uuid", conversationUUID)
		return
	}

	inboxInstance, err := m.inboxStore.Get(conversation.InboxID)
	if err != nil {
		m.lo.Error("error getting inbox for widget typing broadcast", "error", err, "inbox_id", conversation.InboxID)
		return
	}

	if liveChatInbox, ok := inboxInstance.(*livechat.LiveChat); ok {
		liveChatInbox.BroadcastTypingToClients(conversationUUID, conversation.ContactID, isTyping)
	}
}

// BroadcastAgentStatusToWidget sends a lightweight assignee availability update
// to widget clients for all active livechat conversations assigned to the given agent.
func (m *Manager) BroadcastAgentStatusToWidget(agentID int, status string) {
	var conversations []struct {
		UUID      string `db:"uuid"`
		ContactID int    `db:"contact_id"`
		InboxID   int    `db:"inbox_id"`
	}
	if err := m.q.GetActiveLivechatConversationsByAgent.Select(&conversations, agentID); err != nil {
		m.lo.Error("error fetching active livechat conversations for agent", "error", err, "agent_id", agentID)
		return
	}
	for _, conv := range conversations {
		m.BroadcastConversationToWidget(conv.UUID, conv.ContactID, conv.InboxID, map[string]any{
			"assignee": map[string]any{"availability_status": status},
		})
	}
}

// BroadcastConversationToWidget broadcasts a partial conversation update to widget clients.
func (m *Manager) BroadcastConversationToWidget(conversationUUID string, contactID, inboxID int, data map[string]any) {
	inboxInstance, err := m.inboxStore.Get(inboxID)
	if err != nil {
		if err == inbox.ErrInboxNotFound {
			return
		}
		m.lo.Error("error getting inbox for widget conversation broadcast", "error", err, "inbox_id", inboxID)
		return
	}

	if liveChatInbox, ok := inboxInstance.(*livechat.LiveChat); ok {
		data["uuid"] = conversationUUID
		liveChatInbox.BroadcastConversationToClients(conversationUUID, contactID, data)
	}
}
