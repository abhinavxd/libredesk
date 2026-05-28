// Package ws handles WebSocket connections and broadcasting messages to clients.
package ws

import (
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
	"github.com/zerodha/logf"
)

// Hub maintains the set of registered websockets clients.
type Hub struct {
	lo *logf.Logger

	clients      map[int][]*Client
	clientsMutex sync.RWMutex

	convSubsList   map[string]map[*Client]struct{}
	convSubsOpen   map[string]map[*Client]struct{}
	clientListSubs map[*Client]map[string]struct{}
	clientOpenSub  map[*Client]string
	subsMu         sync.RWMutex

	userStore         userStore
	conversationStore conversationStore
}

type userStore interface {
	UpdateLastActive(userID int) (bool, error)
}

type conversationStore interface {
	BroadcastTypingToWidgetClientsOnly(conversationUUID string, isTyping bool)
	FilterAuthorizedListUUIDs(agentID int, uuids []string) ([]string, error)
}

// NewHub creates a new websocket hub.
func NewHub(lo *logf.Logger, userStore userStore) *Hub {
	return &Hub{
		lo:                lo,
		clients:           make(map[int][]*Client, 64),
		clientsMutex:      sync.RWMutex{},
		convSubsList:      make(map[string]map[*Client]struct{}, 1024),
		convSubsOpen:      make(map[string]map[*Client]struct{}, 64),
		clientListSubs:    make(map[*Client]map[string]struct{}, 64),
		clientOpenSub:     make(map[*Client]string, 64),
		userStore:         userStore,
		conversationStore: nil,
	}
}

func (h *Hub) KickUser(userID int) {
	h.clientsMutex.RLock()
	clients := append([]*Client(nil), h.clients[userID]...)
	h.clientsMutex.RUnlock()
	if len(clients) == 0 {
		return
	}
	h.lo.Debug("kicking user ws connections", "user_id", userID, "connections", len(clients))
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "kicked")
	for _, c := range clients {
		_ = c.Conn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(time.Second))
		_ = c.Conn.Close()
	}
}

// SubscribeListReplace replaces list-source subs; open-source subs are untouched so deep links survive list refreshes.
func (h *Hub) SubscribeListReplace(client *Client, uuids []string) {
	h.subsMu.Lock()
	defer h.subsMu.Unlock()
	for uuid := range h.clientListSubs[client] {
		delete(h.convSubsList[uuid], client)
		if len(h.convSubsList[uuid]) == 0 {
			delete(h.convSubsList, uuid)
		}
	}
	h.clientListSubs[client] = make(map[string]struct{}, len(uuids))
	for _, uuid := range uuids {
		h.clientListSubs[client][uuid] = struct{}{}
		if h.convSubsList[uuid] == nil {
			h.convSubsList[uuid] = make(map[*Client]struct{})
		}
		h.convSubsList[uuid][client] = struct{}{}
	}
}

// SubscribeOpenConv sets the client's single open-conversation sub, replacing any previous one.
func (h *Hub) SubscribeOpenConv(client *Client, uuid string) {
	h.subsMu.Lock()
	defer h.subsMu.Unlock()
	if prev, ok := h.clientOpenSub[client]; ok && prev != uuid {
		delete(h.convSubsOpen[prev], client)
		if len(h.convSubsOpen[prev]) == 0 {
			delete(h.convSubsOpen, prev)
		}
	}
	h.clientOpenSub[client] = uuid
	if h.convSubsOpen[uuid] == nil {
		h.convSubsOpen[uuid] = make(map[*Client]struct{})
	}
	h.convSubsOpen[uuid][client] = struct{}{}
}

// ListSubscribers returns the union of list-source and open-source subscribers for a conversation.
func (h *Hub) ListSubscribers(uuid string) []*Client {
	h.subsMu.RLock()
	defer h.subsMu.RUnlock()
	listSet := h.convSubsList[uuid]
	openSet := h.convSubsOpen[uuid]
	if len(listSet) == 0 && len(openSet) == 0 {
		return nil
	}
	union := make(map[*Client]struct{}, len(listSet)+len(openSet))
	for c := range listSet {
		union[c] = struct{}{}
	}
	for c := range openSet {
		union[c] = struct{}{}
	}
	out := make([]*Client, 0, len(union))
	for c := range union {
		out = append(out, c)
	}
	return out
}

// ClearClientSubs drops all of a client's list and open subscriptions.
func (h *Hub) ClearClientSubs(client *Client) {
	h.subsMu.Lock()
	defer h.subsMu.Unlock()
	for uuid := range h.clientListSubs[client] {
		delete(h.convSubsList[uuid], client)
		if len(h.convSubsList[uuid]) == 0 {
			delete(h.convSubsList, uuid)
		}
	}
	delete(h.clientListSubs, client)
	if prev, ok := h.clientOpenSub[client]; ok {
		delete(h.convSubsOpen[prev], client)
		if len(h.convSubsOpen[prev]) == 0 {
			delete(h.convSubsOpen, prev)
		}
		delete(h.clientOpenSub, client)
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

	if clients, ok := h.clients[client.ID]; ok {
		for i, c := range clients {
			if c == client {
				h.clients[client.ID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
	}
	h.ClearClientSubs(client)
}

func (h *Hub) ConnectedUserIDs() []int {
	h.clientsMutex.RLock()
	defer h.clientsMutex.RUnlock()
	out := make([]int, 0, len(h.clients))
	for id, clients := range h.clients {
		if len(clients) > 0 {
			out = append(out, id)
		}
	}
	return out
}

// PushToClients sends a raw payload directly to the given client connections.
func (h *Hub) PushToClients(clients []*Client, data []byte) {
	for _, c := range clients {
		c.SendMessage(data, websocket.TextMessage)
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

func (h *Hub) BroadcastTypingToConversation(conversationUUID string, typingMsg models.TypingMessage) {
	if h.conversationStore != nil && !typingMsg.IsPrivateMessage {
		h.conversationStore.BroadcastTypingToWidgetClientsOnly(conversationUUID, typingMsg.IsTyping)
	}
}

func (h *Hub) BroadcastTypingToAllConversationClients(conversationUUID string, data []byte) {
	for _, c := range h.ListSubscribers(conversationUUID) {
		c.SendMessage(data, websocket.TextMessage)
	}
}
