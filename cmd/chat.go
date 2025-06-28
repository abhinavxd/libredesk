package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

type onlyJWT struct {
	JWT string `json:"jwt"`
}

// Define JWT claims structure
type Claims struct {
	UserID   int    `json:"user_id,omitempty"`
	IsGuest  bool   `json:"is_guest,omitempty"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

// Chat widget initialization request
type chatInitReq struct {
	onlyJWT

	// For guest users
	GuestName  string `json:"guest_name,omitempty"`
	GuestEmail string `json:"guest_email,omitempty"`
	Message    string `json:"message,omitempty"`

	InboxID int `json:"inbox_id"`
}

type conversationResp struct {
	Conversation Conversation  `json:"conversation"`
	Messages     []chatMessage `json:"messages"`
}

type Conversation struct {
	UUID string `json:"uuid"`
}

type chatMessage struct {
	CreatedAt      time.Time `json:"created_at"`
	UUID           string    `json:"uuid"`
	Content        string    `json:"content"`
	SenderType     string    `json:"sender_type"`
	SenderName     string    `json:"sender_name"`
	ConversationID string    `json:"conversation_id"`
}

type chatMessageReq struct {
	Message string `json:"message"`
	onlyJWT
}

// handleGetChatSettings returns the live chat settings for the widget
func handleGetChatSettings(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		inboxID = r.RequestCtx.QueryArgs().GetUintOrZero("inbox_id")
	)

	if inboxID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Inbox ID is required", nil, envelope.InputError)
	}

	// Get inbox configuration
	inbox, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		app.lo.Error("error fetching inbox", "inbox_id", inboxID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.NotFoundError)
	}

	if inbox.Channel != livechat.ChannelLiveChat {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid inbox type for chat", nil, envelope.InputError)
	}

	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Inbox is disabled", nil, envelope.InputError)
	}

	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Invalid inbox configuration", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(config)
}

// handleChatInit initializes a new chat session.
func handleChatInit(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = chatInitReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling chat init request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message is required", nil, envelope.InputError)
	}

	// Get inbox configuration
	inbox, err := app.inbox.GetDBRecord(req.InboxID)
	if err != nil {
		app.lo.Error("error fetching inbox", "inbox_id", req.InboxID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.NotFoundError)
	}

	// Make sure the inbox is enabled and of the correct type
	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Inbox is disabled", nil, envelope.InputError)
	}

	if inbox.Channel != livechat.ChannelLiveChat {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid inbox type for chat", nil, envelope.InputError)
	}

	// Parse inbox config
	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Invalid inbox configuration", nil, envelope.GeneralError)
	}

	var contactID int
	var conversationUUID string
	var isGuest bool

	// Handle authenticated user
	if req.JWT != "" {
		claims, err := verifyStandardJWT(req.JWT)
		if err != nil {
			app.lo.Error("invalid JWT", "jwt", req.JWT, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid JWT", nil, envelope.InputError)
		}
		userID := claims.UserID
		isGuest = claims.IsGuest

		user := umodels.User{
			Email:     null.StringFrom(claims.Email),
			FirstName: claims.Username,
			LastName:  "",
		}

		// Get or Create contact / visitor user.
		if !isGuest {
			if err = app.user.CreateContact(&user); err != nil {
				app.lo.Error("error fetching authenticated user contact", "user_id", userID, "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusNotFound, "User not found", nil, envelope.NotFoundError)
			}
		} else {
			if err = app.user.CreateVisitor(&user); err != nil {
				app.lo.Error("error creating guest contact", "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error creating user, Please try again.", nil, envelope.GeneralError)
			}
		}
		contactID = user.ID
	} else {
		isGuest = true
		visitor := umodels.User{
			Email:     null.NewString(req.GuestEmail, req.GuestEmail != ""),
			FirstName: req.GuestName,
		}

		if err := app.user.CreateVisitor(&visitor); err != nil {
			app.lo.Error("error creating guest contact", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error creating user, Please try again.", nil, envelope.GeneralError)
		}
		contactID = visitor.ID

		// Generate guest JWT
		req.JWT, err = generateUserJWT(contactID, isGuest, time.Now().Add(24*time.Hour))
		if err != nil {
			app.lo.Error("error generating guest JWT", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to generate JWT, Please try again.", nil, envelope.GeneralError)
		}
	}

	app.lo.Info("creating new conversation for user", "user_id", contactID, "inbox_id", req.InboxID)

	// Create conversation.
	_, conversationUUID, err = app.conversation.CreateConversation(
		contactID,
		req.InboxID,
		"",
		time.Now(),
		"",
		false,
	)
	if err != nil {
		app.lo.Error("error creating conversation", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error creating conversation, Please try again.", nil, envelope.GeneralError)
	}

	// Send message to the just created conversation as user.
	message := models.Message{
		ConversationUUID: conversationUUID,
		SenderID:         contactID,
		Type:             models.MessageIncoming,
		SenderType:       models.SenderTypeContact,
		Status:           models.MessageStatusReceived,
		Content:          req.Message,
		ContentType:      models.ContentTypeText,
		Private:          false,
	}
	if err := app.conversation.InsertMessage(&message); err != nil {
		app.lo.Error("error inserting initial message", "conversation_uuid", conversationUUID, "error", err)
	}

	// Build response with conversation and messages.
	resp, err := buildConversationResponse(app, conversationUUID, contactID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error creating conversation, Please try again.", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]interface{}{
		"conversation": resp.Conversation,
		"messages":     resp.Messages,
		"jwt":          req.JWT,
	})
}

// buildConversationResponse builds the response for a conversation including its messages
func buildConversationResponse(app *App, conversationUUID string, contactID int) (*conversationResp, error) {
	// Fetch last 2000 messages, this should suffice as chats shouldn't have too many messages.
	private := false
	messages, _, err := app.conversation.GetConversationMessages(conversationUUID, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing}, &private, 1, 2000)
	if err != nil {
		app.lo.Error("error fetching conversation messages", "conversation_uuid", conversationUUID, "error", err)
		return nil, fmt.Errorf("failed to fetch conversation messages: %v", err)
	}

	// Convert to chat message format
	chatMessages := make([]chatMessage, len(messages))
	nameMap := make(map[int]string)
	for i, msg := range messages {
		// Get sender name
		senderName := nameMap[msg.SenderID]
		if msg.SenderType == models.SenderTypeContact {
			if senderName == "" {
				if contact, err := app.user.GetContact(contactID, ""); err == nil {
					senderName = contact.FullName()
					nameMap[msg.SenderID] = senderName
				}
			}
		}
		chatMessages[i] = chatMessage{
			UUID:           msg.UUID,
			Content:        msg.TextContent,
			CreatedAt:      msg.CreatedAt,
			SenderType:     msg.SenderType,
			SenderName:     senderName,
			ConversationID: msg.ConversationUUID,
		}
	}

	resp := &conversationResp{
		Conversation: Conversation{
			UUID: conversationUUID,
		},
		Messages: chatMessages,
	}

	return resp, nil
}

// handleChatGetConversation fetches a chat conversation by ID
func handleChatGetConversation(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
		chatReq          = chatInitReq{}
	)

	if conversationUUID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "conversation_id is required", nil, envelope.InputError)
	}

	// Decode chat request if present
	if err := r.Decode(&chatReq, "json"); err != nil {
		app.lo.Error("error unmarshalling chat request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Verify JWT.
	if chatReq.JWT == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "JWT is required", nil, envelope.InputError)
	}

	claims, err := verifyStandardJWT(chatReq.JWT)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", chatReq.JWT, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid JWT", nil,
			envelope.InputError)
	}

	contactID := claims.UserID
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid user ID in JWT", nil, envelope.InputError)
	}

	// Fetch conversation details
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Conversation not found", nil, envelope.NotFoundError)
	}

	// Make sure the conversation belongs to the contact
	if conversation.ContactID != contactID {
		app.lo.Error("unauthorized access to conversation", "conversation_uuid", conversationUUID, "contact_id", contactID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have access to this conversation", nil, envelope.PermissionError)
	}

	// Build conversation response with messages
	resp, err := buildConversationResponse(app, conversation.UUID, conversation.ContactID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch conversation messages", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(*resp)
}

// handleGetConversations fetches all chat conversations for a widget user
func handleGetConversations(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = onlyJWT{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling chat conversations request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if req.JWT == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "JWT is required", nil, envelope.InputError)
	}

	claims, err := verifyStandardJWT(req.JWT)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", req.JWT, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid JWT", nil, envelope.InputError)
	}

	contactID := claims.UserID
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid user ID in JWT", nil, envelope.InputError)
	}

	conversations, err := app.conversation.GetContactConversations(contactID)
	if err != nil {
		app.lo.Error("error fetching conversations for contact", "contact_id", contactID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to fetch conversations", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(conversations)
}

// handleChatSendMessage sends a message in a chat conversation
func handleChatSendMessage(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
		req              = chatMessageReq{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling chat message request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	var senderID int
	var senderType = models.SenderTypeContact
	if req.JWT == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "JWT is required", nil, envelope.InputError)
	}

	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Message content is required", nil, envelope.InputError)
	}

	claims, err := verifyStandardJWT(req.JWT)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", req.JWT, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid JWT", nil, envelope.InputError)
	}
	senderID = claims.UserID

	if senderID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid user ID in JWT", nil, envelope.InputError)
	}

	// Fetch conversation to ensure it exists
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Conversation not found", nil, envelope.NotFoundError)
	}

	// Make sure the conversation belongs to the sender
	if conversation.ContactID != senderID {
		app.lo.Error("unauthorized access to conversation", "conversation_uuid", conversationUUID, "contact_id", senderID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have access to this conversation", nil, envelope.PermissionError)
	}

	// Create and insert message
	message := models.Message{
		ConversationUUID: conversationUUID,
		SenderID:         senderID,
		Type:             models.MessageIncoming,
		SenderType:       senderType,
		Status:           models.MessageStatusReceived,
		Content:          req.Message,
		ContentType:      models.ContentTypeText,
		Private:          false,
	}

	if err := app.conversation.InsertMessage(&message); err != nil {
		app.lo.Error("error inserting chat message", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to send message", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]bool{"success": true})
}

// handleChatArchiveConversation archives a chat conversation
func handleChatCloseConversation(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
		onlyJWT          = onlyJWT{}
	)

	if conversationUUID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "conversation_id is required", nil, envelope.InputError)
	}

	// Decode
	if err := r.Decode(&onlyJWT, "json"); err != nil {
		app.lo.Error("error unmarshalling chat close request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Verify JWT
	if onlyJWT.JWT == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "JWT is required", nil, envelope.InputError)
	}

	claims, err := verifyStandardJWT(onlyJWT.JWT)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", onlyJWT.JWT, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, "Invalid JWT", nil, envelope.InputError)
	}
	contactID := claims.UserID
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid user ID in JWT", nil, envelope.InputError)
	}

	// Fetch conversation to ensure it exists
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching conversation for closing", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Conversation not found", nil, envelope.NotFoundError)
	}

	// Make sure the conversation belongs to the contact
	if conversation.ContactID != contactID {
		app.lo.Error("unauthorized access to conversation for closing", "conversation_uuid", conversationUUID, "contact_id", contactID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You do not have access to this conversation", nil, envelope.PermissionError)
	}

	contact, err := app.user.GetContact(contactID, "")
	if err != nil {
		app.lo.Error("error fetching contact for closing conversation", "contact_id", contactID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, "Contact not found", nil, envelope.NotFoundError)
	}

	err = app.conversation.UpdateConversationStatus(conversationUUID, 0, models.StatusClosed, "", contact)
	if err != nil {
		app.lo.Error("error archiving chat conversation", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Failed to archive conversation", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]bool{"success": true})
}

// verifyJWT verifies and validates a JWT token with proper signature verification
func verifyJWT(tokenString string, secretKey []byte) (*Claims, error) {
	// Parse and verify the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	// Extract claims if token is valid
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// verifyStandardJWT verifies a standard JWT token using proper JWT library
func verifyStandardJWT(jwtToken string) (Claims, error) {
	if jwtToken == "" {
		return Claims{}, fmt.Errorf("JWT token is empty")
	}

	claims, err := verifyJWT(jwtToken, getJWTSecret())
	if err != nil {
		return Claims{}, err
	}

	return *claims, nil
}

// getJWTSecret gets the JWT secret key from configuration or uses a default
func getJWTSecret() []byte {
	// TODO: Update this to pick from inbox config in db.
	return []byte("your-secret-key-change-this-in-production")
}

// generateUserJWT generates a JWT token for a user
func generateUserJWT(userID int, isGuest bool, expirationTime time.Time) (string, error) {
	claims := &Claims{
		UserID:  userID,
		IsGuest: isGuest,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(getJWTSecret())
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
