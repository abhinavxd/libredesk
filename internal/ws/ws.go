// Package ws handles WebSocket connections and broadcasting messages to clients.
package ws

import (
	"sync"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// Hub maintains the set of registered websockets clients.
type Hub struct {
	// Client ID to WS Client map, user can connect from multiple devices and each device will have a separate client.
	clients      map[int][]*Client
	clientsMutex sync.RWMutex

	userStore userStore
}

type userStore interface {
	UpdateLastActive(userID int) error
}

// NewHub creates a new websocket hub.
func NewHub(userStore userStore) *Hub {
	return &Hub{
		clients:      make(map[int][]*Client, 10000),
		clientsMutex: sync.RWMutex{},
		userStore:    userStore,
	}
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
