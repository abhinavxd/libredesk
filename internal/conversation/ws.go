package conversation

import (
	"encoding/json"
	"time"

	cmodels "github.com/jakedolan443/fernmail/internal/conversation/models"

	wsmodels "github.com/jakedolan443/fernmail/internal/ws/models"
)

// broadcastConv shadows the per-user fields (another agent's unread state) out of the shared payload.
type broadcastConv struct {
	*cmodels.ConversationListItem
	UnreadMessageCount   *struct{} `json:"unread_message_count,omitempty"`
	MentionedMessageUUID *struct{} `json:"mentioned_message_uuid,omitempty"`
}

func (m *Manager) BroadcastNewConversation(conv *cmodels.ConversationListItem) {
	m.broadcastConvToAuthorized(conv, nil)
}

func (m *Manager) BroadcastNewMessage(message *cmodels.Message, conv *cmodels.ConversationListItem, preview string) {
	if conv == nil {
		return
	}
	data := map[string]any{
		"conversation_uuid": message.ConversationUUID,
		"uuid":              message.UUID,
		"type":              message.Type,
		"preview":           preview,
		"created_at":        message.CreatedAt.Format(time.RFC3339),
		"sender_type":       message.SenderType,
		"private":           message.Private,
		"sender": map[string]string{
			"first_name": message.Author.FirstName,
			"last_name":  message.Author.LastName,
			"email":      message.Author.Email.String,
		},
		"conversation": convToBroadcast(conv),
	}

	var meta map[string]any
	if len(message.Meta) > 0 {
		if err := json.Unmarshal(message.Meta, &meta); err == nil {
			if echoID, ok := meta["echo_id"].(string); ok && echoID != "" {
				data["echo_id"] = echoID
			}
		}
	}

	userIDs := m.AuthorizedConnectedAgentIDs(conv.AssignedUserID, conv.AssignedTeamID, conv.AddressID, conv.InboxID)
	if len(userIDs) == 0 {
		return
	}
	m.broadcastToUsers(userIDs, wsmodels.Message{
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

func (m *Manager) broadcastConvToAuthorized(conv, oldConv *cmodels.ConversationListItem) {
	if conv == nil {
		return
	}
	userIDs := m.AuthorizedConnectedAgentIDs(conv.AssignedUserID, conv.AssignedTeamID, conv.AddressID, conv.InboxID)
	if oldConv != nil {
		seen := make(map[int]struct{}, len(userIDs))
		for _, id := range userIDs {
			seen[id] = struct{}{}
		}
		for _, id := range m.AuthorizedConnectedAgentIDs(oldConv.AssignedUserID, oldConv.AssignedTeamID, oldConv.AddressID, oldConv.InboxID) {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				userIDs = append(userIDs, id)
			}
		}
	}
	if len(userIDs) == 0 {
		return
	}
	m.broadcastToUsers(userIDs, wsmodels.Message{
		Type: wsmodels.MessageTypeNewConversation,
		Data: convToBroadcast(conv),
	})
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
	clients := m.wsHub.AuthorizedListSubscribers(conversationUUID)
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

func (m *Manager) BroadcastAgentAvailability(agentID int, status string) {
	m.broadcastToUsers([]int{}, wsmodels.Message{
		Type: wsmodels.MessageTypeAgentAvailability,
		Data: map[string]any{
			"agent_id":            agentID,
			"availability_status": status,
		},
	})

}

func convToBroadcast(conv *cmodels.ConversationListItem) *broadcastConv {
	if conv == nil {
		return nil
	}
	return &broadcastConv{ConversationListItem: conv}
}
