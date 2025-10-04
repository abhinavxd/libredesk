// Package ws handles WebSocket connections and broadcasting messages to clients.
package ws

import (
	"encoding/json"
	"sync"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// Hub maintains the set of registered websockets clients.
type Hub struct {
	// Client ID to WS Client map, user can connect from multiple devices and each device will have a separate client.
	clients      map[int][]*Client
	clientsMutex sync.RWMutex

	// Conversation UUID to clients map for faster conversation broadcasting
	conversationClients      map[string][]*Client
	conversationClientsMutex sync.RWMutex

	userStore         userStore
	conversationStore conversationStore
}

type userStore interface {
	UpdateLastActive(userID int) error
}

type conversationStore interface {
	BroadcastTypingToWidgetClientsOnly(conversationUUID string, isTyping bool)
}

// NewHub creates a new websocket hub.
func NewHub(userStore userStore) *Hub {
	return &Hub{
		clients:                  make(map[int][]*Client, 10000),
		clientsMutex:             sync.RWMutex{},
		conversationClients:      make(map[string][]*Client),
		conversationClientsMutex: sync.RWMutex{},
		userStore:                userStore,
		// To be set later via conversationStore.
		conversationStore: nil,
	}
}

// SetConversationStore sets the conversation store for cross-broadcasting.
func (h *Hub) SetConversationStore(manager conversationStore) {
	h.conversationStore = manager
}

// AddClient adds a new client to the hub.
func (h *Hub) AddClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	h.clients[client.ID] = append(h.clients[client.ID], client)
}

// RemoveClient removes a client from the hub.
func (h *Hub) RemoveClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()

	// Remove from all conversation subscriptions
	h.conversationClientsMutex.Lock()
	h.removeClientFromAllConversations(client)
	h.conversationClientsMutex.Unlock()

	if clients, ok := h.clients[client.ID]; ok {
		for i, c := range clients {
			if c == client {
				h.clients[client.ID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
	}
}

// BroadcastMessage broadcasts a message to the specified users.
// If no users are specified, the message is broadcast to all users.
func (h *Hub) BroadcastMessage(msg models.BroadcastMessage) {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()

	// Broadcast to all users if no users are specified.
	if len(msg.Users) == 0 {
		for _, clients := range h.clients {
			for _, client := range clients {
				client.SendMessage(msg.Data, websocket.TextMessage)
			}
		}
		return
	}

	// Broadcast to specified users.
	for _, userID := range msg.Users {
		for _, client := range h.clients[userID] {
			client.SendMessage(msg.Data, websocket.TextMessage)
		}
	}
}

// SubscribeToConversation subscribes a client to a conversation.
func (h *Hub) SubscribeToConversation(client *Client, conversationUUID string) {
	h.conversationClientsMutex.Lock()
	defer h.conversationClientsMutex.Unlock()

	// Unsubscribe from previous conversation if any
	h.removeClientFromAllConversations(client)

	// Subscribe to new conversation
	h.conversationClients[conversationUUID] = append(h.conversationClients[conversationUUID], client)
}

// UnsubscribeFromConversation unsubscribes a client from a conversation.
func (h *Hub) UnsubscribeFromConversation(client *Client, conversationUUID string) {
	h.conversationClientsMutex.Lock()
	defer h.conversationClientsMutex.Unlock()
	h.unsubscribeFromConversationUnsafe(client, conversationUUID)
}

// unsubscribeFromConversationUnsafe removes a client from conversation subscription without locking.
// Must be called with conversationClientsMutex held.
func (h *Hub) unsubscribeFromConversationUnsafe(client *Client, conversationUUID string) {
	if clients, ok := h.conversationClients[conversationUUID]; ok {
		for i, c := range clients {
			if c == client {
				h.conversationClients[conversationUUID] = append(clients[:i], clients[i+1:]...)
				if len(h.conversationClients[conversationUUID]) == 0 {
					delete(h.conversationClients, conversationUUID)
				}
				break
			}
		}
	}
}

// removeClientFromAllConversations removes a client from all conversation subscriptions.
// Must be called with conversationClientsMutex held.
func (h *Hub) removeClientFromAllConversations(client *Client) {
	for conversationUUID, clients := range h.conversationClients {
		for i, c := range clients {
			if c == client {
				h.conversationClients[conversationUUID] = append(clients[:i], clients[i+1:]...)
				if len(h.conversationClients[conversationUUID]) == 0 {
					delete(h.conversationClients, conversationUUID)
				}
				break
			}
		}
	}
}

// BroadcastToConversation broadcasts a message to all clients subscribed to a specific conversation.
func (h *Hub) BroadcastToConversation(conversationUUID string, data []byte) {
	h.conversationClientsMutex.RLock()
	defer h.conversationClientsMutex.RUnlock()

	for _, client := range h.conversationClients[conversationUUID] {
		client.SendMessage(data, websocket.TextMessage)
	}
}

// BroadcastTypingToConversation broadcasts typing status to all clients subscribed to a conversation except the sender.
func (h *Hub) BroadcastTypingToConversation(conversationUUID string, typingMsg models.TypingMessage, sender *Client) {
	h.conversationClientsMutex.RLock()
	defer h.conversationClientsMutex.RUnlock()

	message := models.Message{
		Type: models.MessageTypeTyping,
		Data: typingMsg,
	}

	messageBytes, _ := json.Marshal(message)

	for _, client := range h.conversationClients[conversationUUID] {
		// Don't send typing indicator back to the sender.
		if client != sender {
			client.SendMessage(messageBytes, websocket.TextMessage)
		}
	}

	// Also broadcast to widget clients since this is an agent typing.
	if h.conversationStore != nil && !typingMsg.IsPrivateMessage {
		h.conversationStore.BroadcastTypingToWidgetClientsOnly(conversationUUID, typingMsg.IsTyping)
	}
}

// BroadcastTypingToAllConversationClients broadcasts typing status to all clients subscribed to a conversation.
func (h *Hub) BroadcastTypingToAllConversationClients(conversationUUID string, data []byte) {
	h.conversationClientsMutex.RLock()
	defer h.conversationClientsMutex.RUnlock()

	for _, client := range h.conversationClients[conversationUUID] {
		client.SendMessage(data, websocket.TextMessage)
	}
}
