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
	Type string `json:"type"`
	JWT  string `json:"jwt,omitempty"`
	Data any    `json:"data"`
}

type WidgetInboxJoinRequest struct {
	InboxID int `json:"inbox_id"`
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
		var inboxID int

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

			switch msg.Type {
			// Inbox join request.
			case WidgetMsgTypeJoin:
				var joinedClient *livechat.Client
				var joinedLiveChat *livechat.LiveChat
				var joinedInboxID int
				var err error
				if joinedClient, joinedLiveChat, joinedInboxID, err = handleInboxJoin(app, conn, &msg); err != nil {
					app.lo.Error("error handling widget join", "error", err)
					sendWidgetError(conn, "Failed to join conversation")
					continue
				}
				// Store the client, livechat, and inbox ID for cleanup and future use.
				client = joinedClient
				liveChat = joinedLiveChat
				inboxID = joinedInboxID
			// Typing.
			case WidgetMsgTypeTyping:
				if err := handleWidgetTyping(app, &msg); err != nil {
					app.lo.Error("error handling widget typing", "error", err)
					continue
				}
			// Ping.
			case WidgetMsgTypePing:
				// Update user's last active timestamp if JWT is provided and client has joined
				if msg.JWT != "" && inboxID != 0 {
					if claims, err := validateWidgetMessageJWT(app, msg.JWT, inboxID); err == nil {
						if userID, err := resolveUserIDFromClaims(app, claims); err == nil {
							// Check if user was offline before updating
							wasOffline := app.user.IsOffline(userID)
							if err := app.user.UpdateLastActive(userID); err != nil {
								app.lo.Error("error updating user last active timestamp", "user_id", userID, "error", err)
							} else {
								app.lo.Debug("updated user last active timestamp", "user_id", userID)
								// Broadcast online status if user just came online
								if wasOffline {
									app.conversation.BroadcastContactStatus(userID, "online")
								}
							}
						}
					}
				}

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

// handleInboxJoin handles a websocket join request for a live chat inbox.
func handleInboxJoin(app *App, conn *websocket.Conn, msg *WidgetMessage) (*livechat.Client, *livechat.LiveChat, int, error) {
	joinDataBytes, err := json.Marshal(msg.Data)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("invalid join data: %w", err)
	}

	var joinData WidgetInboxJoinRequest
	if err := json.Unmarshal(joinDataBytes, &joinData); err != nil {
		return nil, nil, 0, fmt.Errorf("invalid join data format: %w", err)
	}

	// Validate JWT with inbox secret
	claims, err := validateWidgetMessageJWT(app, msg.JWT, joinData.InboxID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("JWT validation failed: %w", err)
	}

	// Resolve user ID.
	userID, err := resolveUserIDFromClaims(app, claims)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to resolve user ID from claims: %w", err)
	}

	// Make sure inbox is active.
	inbox, err := app.inbox.GetDBRecord(joinData.InboxID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("inbox not found: %w", err)
	}
	if !inbox.Enabled {
		return nil, nil, 0, fmt.Errorf("inbox is not enabled")
	}

	// Get live chat inbox
	lcInbox, err := app.inbox.Get(inbox.ID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("live chat inbox not found: %w", err)
	}

	// Assert type.
	liveChat, ok := lcInbox.(*livechat.LiveChat)
	if !ok {
		return nil, nil, 0, fmt.Errorf("inbox is not a live chat inbox")
	}

	// Add client to live chat session
	userIDStr := fmt.Sprintf("%d", userID)
	client, err := liveChat.AddClient(userIDStr)
	if err != nil {
		app.lo.Error("error adding client to live chat", "error", err, "user_id", userIDStr)
		return nil, nil, 0, err
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
			"message": "namaste!",
		},
	}

	if err := conn.WriteJSON(joinResp); err != nil {
		return nil, nil, 0, err
	}

	app.lo.Debug("widget client joined live chat", "user_id", userIDStr, "inbox_id", joinData.InboxID)

	return client, liveChat, joinData.InboxID, nil
}

// handleWidgetTyping handles typing indicators
func handleWidgetTyping(app *App, msg *WidgetMessage) error {
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

	// Get conversation to retrieve inbox ID for JWT validation
	if typingData.ConversationUUID == "" {
		return fmt.Errorf("conversation UUID is required for typing messages")
	}

	conversation, err := app.conversation.GetConversation(0, typingData.ConversationUUID, "")
	if err != nil {
		app.lo.Error("error fetching conversation for typing", "conversation_uuid", typingData.ConversationUUID, "error", err)
		return fmt.Errorf("conversation not found: %w", err)
	}

	// Validate JWT with inbox secret
	claims, err := validateWidgetMessageJWT(app, msg.JWT, conversation.InboxID)
	if err != nil {
		return fmt.Errorf("JWT validation failed: %w", err)
	}

	userID := claims.UserID

	// Broadcast typing status to agents via conversation manager
	// Set broadcastToWidgets=false to avoid echoing back to widget clients
	app.conversation.BroadcastTypingToConversation(typingData.ConversationUUID, typingData.IsTyping, false)

	app.lo.Debug("Broadcasted typing data from widget user to agents", "user_id", userID, "is_typing", typingData.IsTyping, "conversation_uuid", typingData.ConversationUUID)
	return nil
}

// validateWidgetMessageJWT validates the incoming widget message JWT using inbox secret.
func validateWidgetMessageJWT(app *App, jwtToken string, inboxID int) (Claims, error) {
	if inboxID <= 0 {
		return Claims{}, fmt.Errorf("inbox ID is required for JWT validation")
	}

	inbox, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		return Claims{}, fmt.Errorf("inbox not found: %w", err)
	}

	return verifyStandardJWT(jwtToken, inbox.Secret.String)
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
