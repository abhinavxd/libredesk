package main

import (
	"encoding/json"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/fasthttp/websocket"
	"github.com/zerodha/fastglue"
)

// Widget WebSocket message types
const (
	WidgetMsgTypeJoin    = "join"
	WidgetMsgTypeMessage = "message"
	WidgetMsgTypeTyping  = "typing"
	WidgetMsgTypePing    = "ping"
	WidgetMsgTypePong    = "pong"
	WidgetMsgTypeError   = "error"
	WidgetMsgTypeNewMsg  = "new_message"
	WidgetMsgTypeStatus  = "status"
	WidgetMsgTypeJoined  = "joined"
)

// WidgetMessage represents a message sent through the widget WebSocket
type WidgetMessage struct {
	Type string      `json:"type"`
	JWT  string      `json:"jwt,omitempty"`
	Data interface{} `json:"data"`
}

// WidgetJoinData represents data for joining a conversation
type WidgetJoinData struct {
	ConversationUUID string `json:"conversation_uuid"`
}

// WidgetMessageData represents a chat message through the widget
type WidgetMessageData struct {
	ConversationUUID string `json:"conversation_uuid"`
	Content          string `json:"content"`
	SenderName       string `json:"sender_name,omitempty"`
	SenderType       string `json:"sender_type"`
	Timestamp        int64  `json:"timestamp"`
}

// WidgetTypingData represents typing indicator data
type WidgetTypingData struct {
	ConversationUUID string `json:"conversation_uuid"`
	IsTyping         bool   `json:"is_typing"`
}

// handleWidgetWS handles the widget WebSocket connection for live chat.
func handleWidgetWS(r *fastglue.Request) error {
	var app = r.Context.(*App)

	if err := upgrader.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		// To store client and live chat references for cleanup.
		var client *livechat.Client
		var liveChat *livechat.LiveChat

		// Clean up client when connection closes.
		defer func() {
			conn.Close()
			if client != nil && liveChat != nil {
				liveChat.RemoveClient(client)
				close(client.Channel)
				app.lo.Debug("cleaned up client on websocket disconnect", "client_id", client.ID)
			}
		}()

		// Read messages from the WebSocket connection.
		for {
			var msg WidgetMessage
			if err := conn.ReadJSON(&msg); err != nil {
				app.lo.Debug("widget websocket connection closed", "error", err)
				break
			}

			var (
				claims Claims
				err    error
			)
			// Validate JWT if present, except for ping messages
			if msg.Type != WidgetMsgTypePing {
				claims, err = validateWidgetMessageJWT(msg.JWT)
				if err != nil {
					app.lo.Error("invalid JWT in widget message", "error", err)
					sendWidgetError(conn, "Invalid JWT token")
					continue
				}
			}

			switch msg.Type {
			// Join conversation request.
			case WidgetMsgTypeJoin:
				var joinedClient *livechat.Client
				var joinedLiveChat *livechat.LiveChat
				if joinedClient, joinedLiveChat, err = handleWidgetJoin(app, conn, &msg, claims); err != nil {
					app.lo.Error("error handling widget join", "error", err)
					sendWidgetError(conn, "Failed to join conversation")
					continue
				}
				// Store the client and livechat reference for cleanup.
				client = joinedClient
				liveChat = joinedLiveChat
			// Typing.
			case WidgetMsgTypeTyping:
				if err := handleWidgetTyping(app, &msg, claims); err != nil {
					app.lo.Error("error handling widget typing", "error", err)
					continue
				}
			// Ping.
			case WidgetMsgTypePing:
				if err := conn.WriteJSON(WidgetMessage{
					Type: WidgetMsgTypePong,
				}); err != nil {
					app.lo.Error("error writing pong to widget client", "error", err)
				}
			}
		}
	}); err != nil {
		app.lo.Error("error upgrading widget websocket connection", "error", err)
	}
	return nil
}

// handleWidgetJoin handles a client joining a conversation
func handleWidgetJoin(app *App, conn *websocket.Conn, msg *WidgetMessage, claims Claims) (*livechat.Client, *livechat.LiveChat, error) {
	userID := claims.UserID

	joinDataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid join data: %w", err)
	}

	var joinData WidgetJoinData
	if err := json.Unmarshal(joinDataBytes, &joinData); err != nil {
		return nil, nil, fmt.Errorf("invalid join data format: %w", err)
	}

	// Get conversation to find the inbox
	conversation, err := app.conversation.GetConversation(0, joinData.ConversationUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("conversation not found: %w", err)
	}

	// Make sure conversation belongs to the user.
	if conversation.ContactID != userID {
		return nil, nil, fmt.Errorf("conversation does not belong to the user")
	}

	// Make sure inbox is active.
	inbox, err := app.inbox.GetDBRecord(conversation.InboxID)
	if err != nil {
		return nil, nil, fmt.Errorf("inbox not found: %w", err)
	}

	if !inbox.Enabled {
		return nil, nil, fmt.Errorf("inbox is not enabled")
	}

	// Get live chat inbox
	lcInbox, err := app.inbox.Get(inbox.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("live chat inbox not found: %w", err)
	}

	// Assert type.
	liveChat, ok := lcInbox.(*livechat.LiveChat)
	if !ok {
		return nil, nil, fmt.Errorf("inbox is not a live chat inbox")
	}

	// Add client to live chat session
	userIDStr := fmt.Sprintf("%d", userID)
	client, err := liveChat.AddClient(userIDStr, joinData.ConversationUUID)
	if err != nil {
		app.lo.Error("error adding client to live chat", "error", err, "user_id", userIDStr, "conversation_uuid", joinData.ConversationUUID)
		return nil, nil, err
	}

	// Start listening for messages from the live chat channel.
	go func() {
		for msgData := range client.Channel {
			if err := conn.WriteMessage(websocket.TextMessage, msgData); err != nil {
				app.lo.Error("error forwarding message to widget client", "error", err)
				return
			}
		}
	}()

	// Send join confirmation
	joinResp := WidgetMessage{
		Type: WidgetMsgTypeJoined,
		Data: map[string]string{
			"message":           "Joined conversation successfully",
			"conversation_uuid": joinData.ConversationUUID,
		},
	}

	if err := conn.WriteJSON(joinResp); err != nil {
		return nil, nil, err
	}

	return client, liveChat, nil
}

// handleWidgetTyping handles typing indicators
func handleWidgetTyping(app *App, msg *WidgetMessage, claims Claims) error {
	userID := claims.UserID
	typingDataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		app.lo.Error("error marshalling typing data", "error", err)
		return fmt.Errorf("invalid typing data: %w", err)
	}

	var typingData WidgetTypingData
	if err := json.Unmarshal(typingDataBytes, &typingData); err != nil {
		app.lo.Error("error unmarshalling typing data", "error", err)
		return fmt.Errorf("invalid typing data format: %w", err)
	}
	// Broadcast this to agents.
	app.lo.Debug("Received typing data for user", "user_id", userID, "is_typing", typingData.IsTyping, "conversation_uuid", typingData.ConversationUUID)
	return nil
}

// validateWidgetMessageJWT validates the incoming widget message JWT and returns the claims
func validateWidgetMessageJWT(jwt string) (Claims, error) {
	// Verify JWT
	claims, err := verifyStandardJWT(jwt)
	if err != nil {
		return Claims{}, fmt.Errorf("invalid JWT token: %w", err)
	}

	// Return claims as a map
	return claims, nil
}

// sendWidgetError sends an error message to the widget client
func sendWidgetError(conn *websocket.Conn, message string) {
	errorMsg := WidgetMessage{
		Type: WidgetMsgTypeError,
		Data: map[string]string{
			"message": message,
		},
	}
	conn.WriteJSON(errorMsg)
}
