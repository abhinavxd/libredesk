package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
)

const (
	pongWait        = 60 * time.Second
	pingPeriod      = 25 * time.Second
	writeWait       = 10 * time.Second
	closeFrameWait  = 1 * time.Second
	maxMessageSize  = 64 << 10
	maxListSubUUIDs = 500
)

// Client is a single connected WS user.
type Client struct {
	// Client ID.
	ID int

	// Hub.
	Hub *Hub

	// WebSocket connection.
	Conn *websocket.Conn

	// Buffered channel of outbound ws messages.
	Send chan models.WSMessage

	// sendMu guards every send on Send and its close; a send racing the close panics.
	sendMu sync.Mutex
	closed bool
}

// Serve handles heartbeats and sending messages to the client.
func (c *Client) Serve() {
	var heartBeatTicker = time.NewTicker(pingPeriod)
	defer heartBeatTicker.Stop()
	defer c.Conn.Close()

	for {
		select {
		case <-heartBeatTicker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(msg.MessageType, msg.Data); err != nil {
				return
			}
		}
	}
}

// Listen is a block method that listens for incoming messages from the client.
func (c *Client) Listen() {
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType != websocket.TextMessage {
			break
		}
		c.processIncomingMessage(msg)
	}
	c.Hub.RemoveClient(c)
	c.close()
}

// processIncomingMessage processes incoming messages from the client.
func (c *Client) processIncomingMessage(data []byte) {
	if string(data) == "ping" {
		if _, err := c.Hub.userStore.UpdateLastActive(c.ID); err != nil {
			c.Hub.lo.Error("UpdateLastActive failed", "client_id", c.ID, "error", err)
		}
		c.SendMessage([]byte("pong"), websocket.TextMessage)
		return
	}

	// Try to parse as JSON message
	var msg models.IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.SendError("invalid message format")
		return
	}

	switch msg.Type {
	case models.MessageTypeConversationSubscribe:
		c.handleConversationSubscribe(msg.Data)
	case models.MessageTypeListSubscribeReplace:
		c.handleListSubscribe(msg.Data)
	case models.MessageTypeTyping:
		c.handleTyping(msg.Data)
	default:
		c.SendError("unknown message type")
	}
}

func (c *Client) handleListSubscribe(data json.RawMessage) {
	var payload struct {
		UUIDs []string `json:"uuids"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		c.SendError("invalid list_subscribe payload")
		return
	}
	if len(payload.UUIDs) > maxListSubUUIDs {
		payload.UUIDs = payload.UUIDs[:maxListSubUUIDs]
	}
	authorized, err := c.Hub.conversationStore.FilterAuthorizedListUUIDs(c.ID, payload.UUIDs)
	if err != nil {
		c.Hub.lo.Error("FilterAuthorizedListUUIDs failed", "client_id", c.ID, "error", err)
		return
	}
	c.Hub.SubscribeListReplace(c, authorized)
}

// handleConversationSubscribe registers the open-conversation sub; authz is enforced because content (not just typing) flows through it.
func (c *Client) handleConversationSubscribe(data json.RawMessage) {
	var subscribeMsg models.ConversationSubscribe
	if err := json.Unmarshal(data, &subscribeMsg); err != nil {
		c.SendError("invalid subscription format")
		return
	}

	if subscribeMsg.ConversationUUID == "" {
		c.SendError("conversation_uuid is required")
		return
	}

	// Authz: silently reject if the agent can't read this conversation.
	authorized, err := c.Hub.conversationStore.FilterAuthorizedListUUIDs(c.ID, []string{subscribeMsg.ConversationUUID})
	if err != nil || len(authorized) == 0 {
		return
	}

	c.Hub.SubscribeOpenConv(c, subscribeMsg.ConversationUUID)
}

// handleTyping handles typing indicator messages.
//
// Same trust assumption as handleConversationSubscribe: the sender is an
// authenticated agent. A hostile agent could broadcast fake typing to any
// conversation UUID (including widget clients), but typing is ephemeral and
// cosmetic; adding per-frame authz isn't worth the DB cost today.
func (c *Client) handleTyping(data json.RawMessage) {
	var typingMsg models.TypingMessage
	if err := json.Unmarshal(data, &typingMsg); err != nil {
		c.SendError("invalid typing format")
		return
	}

	if typingMsg.ConversationUUID == "" {
		c.SendError("conversation_uuid is required for typing")
		return
	}

	c.Hub.BroadcastTypingToConversation(typingMsg.ConversationUUID, typingMsg)
}

// close closes the Send channel; it is idempotent.
func (c *Client) close() {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.Send)
}

// trySend reports whether the message was queued; it drops when the client is closed or its buffer is full.
func (c *Client) trySend(msg models.WSMessage) bool {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if c.closed {
		return false
	}
	select {
	case c.Send <- msg:
		return true
	default:
		return false
	}
}

// SendError sends an error message to client.
func (c *Client) SendError(msg string) {
	out := models.Message{
		Type: models.MessageTypeError,
		Data: msg,
	}
	b, _ := json.Marshal(out)

	if !c.trySend(models.WSMessage{Data: b, MessageType: websocket.TextMessage}) {
		c.Hub.lo.Warn("could not queue error message to client, closing connection", "client_id", c.ID)
		c.Hub.RemoveClient(c)
		c.close()
	}
}

// SendMessage sends a message to client.
func (c *Client) SendMessage(b []byte, typ int) {
	c.trySend(models.WSMessage{Data: b, MessageType: typ})
}
