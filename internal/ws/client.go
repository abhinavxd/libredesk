package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// SafeBool is a thread-safe boolean.
type SafeBool struct {
	flag bool
	mu   sync.RWMutex
}

// Set sets the value of the SafeBool.
func (b *SafeBool) Set(value bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flag = value
}

// Get returns the value of the SafeBool.
func (b *SafeBool) Get() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.flag
}

// Client is a single connected WS user.
type Client struct {
	// Client ID.
	ID int

	// Hub.
	Hub *Hub

	// WebSocket connection.
	Conn *websocket.Conn

	// To prevent pushes to the channel.
	Closed SafeBool

	// Buffered channel of outbound ws messages.
	Send chan models.WSMessage
}

// Serve handles heartbeats and sending messages to the client.
func (c *Client) Serve() {
	var heartBeatTicker = time.NewTicker(2 * time.Second)
	defer heartBeatTicker.Stop()

Loop:
	for {
		select {
		case <-heartBeatTicker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("error writing message", err)
				return
			}
		case msg, ok := <-c.Send:
			if !ok {
				break Loop
			}
			c.Conn.WriteMessage(msg.MessageType, msg.Data)
		}
	}
	c.Conn.Close()
}

// Listen is a block method that listens for incoming messages from the client.
func (c *Client) Listen() {
	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.TextMessage {
			c.processIncomingMessage(msg)
		} else {
			c.Hub.RemoveClient(c)
			c.close()
			return
		}
	}
	c.Hub.RemoveClient(c)
	c.close()
}

// processIncomingMessage processes incoming messages from the client.
func (c *Client) processIncomingMessage(data []byte) {
	// Handle ping messages, and update last active time for user.
	if string(data) == "ping" {
		c.Hub.userStore.UpdateLastActive(c.ID)
		c.SendMessage([]byte("pong"), websocket.TextMessage)
		return
	}

	// Try to parse as JSON message
	var msg models.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.SendError("invalid message format")
		return
	}

	switch msg.Type {
	case models.MessageTypeConversationSubscribe:
		c.handleConversationSubscribe(msg.Data)
	case models.MessageTypeTyping:
		c.handleTyping(msg.Data)
	default:
		c.SendError("unknown message type")
	}
}

// handleConversationSubscribe handles conversation subscription requests.
func (c *Client) handleConversationSubscribe(data interface{}) {
	// Convert the data to JSON and then unmarshal to ConversationSubscribe
	dataBytes, err := json.Marshal(data)
	if err != nil {
		c.SendError("invalid subscription data")
		return
	}

	var subscribeMsg models.ConversationSubscribe
	if err := json.Unmarshal(dataBytes, &subscribeMsg); err != nil {
		c.SendError("invalid subscription format")
		return
	}

	if subscribeMsg.ConversationUUID == "" {
		c.SendError("conversation_uuid is required")
		return
	}

	// Subscribe to the conversation using the Hub
	c.Hub.SubscribeToConversation(c, subscribeMsg.ConversationUUID)

	// Send confirmation back to client
	response := models.Message{
		Type: models.MessageTypeConversationSubscribed,
		Data: map[string]string{
			"conversation_uuid": subscribeMsg.ConversationUUID,
		},
	}

	responseBytes, _ := json.Marshal(response)
	c.SendMessage(responseBytes, websocket.TextMessage)
}

// handleTyping handles typing indicator messages.
func (c *Client) handleTyping(data interface{}) {
	// Convert the data to JSON and then unmarshal to TypingMessage
	dataBytes, err := json.Marshal(data)
	if err != nil {
		c.SendError("invalid typing data")
		return
	}

	var typingMsg models.TypingMessage
	if err := json.Unmarshal(dataBytes, &typingMsg); err != nil {
		c.SendError("invalid typing format")
		return
	}

	if typingMsg.ConversationUUID == "" {
		c.SendError("conversation_uuid is required for typing")
		return
	}

	// Set the user ID from the client
	typingMsg.UserID = c.ID

	// Broadcast typing status to all subscribers of this conversation (except sender)
	c.Hub.BroadcastTypingToConversation(typingMsg.ConversationUUID, typingMsg, c)
}

// close closes the client connection.
func (c *Client) close() {
	c.Closed.Set(true)
	close(c.Send)
}

// SendError sends an error message to client.
func (c *Client) SendError(msg string) {
	out := models.Message{
		Type: models.MessageTypeError,
		Data: msg,
	}
	b, _ := json.Marshal(out)

	select {
	case c.Send <- models.WSMessage{Data: b, MessageType: websocket.TextMessage}:
	default:
		log.Println("Client send channel is full. Could not send error message.")
		c.Hub.RemoveClient(c)
		c.close()
	}
}

// SendMessage sends a message to client.
func (c *Client) SendMessage(b []byte, typ byte) {
	if c.Closed.Get() {
		log.Println("Attempted to send message to closed client")
		return
	}
	select {
	case c.Send <- models.WSMessage{Data: b, MessageType: websocket.TextMessage}:
	default:
	}
}
