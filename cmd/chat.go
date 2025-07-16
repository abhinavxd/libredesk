package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/stringutil"
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
	UserID    int    `json:"user_id,omitempty"`
	IsVisitor bool   `json:"is_visitor,omitempty"`
	Username  string `json:"username,omitempty"`
	Email     string `json:"email,omitempty"`
	jwt.RegisteredClaims
}

// Chat widget initialization request
type chatInitReq struct {
	onlyJWT
	VisitorName  string `json:"visitor_name,omitempty"`
	VisitorEmail string `json:"visitor_email,omitempty"`
	Message      string `json:"message,omitempty"`
	InboxID      int    `json:"inbox_id"`
}

type conversationResp struct {
	Conversation cmodels.ChatConversation `json:"conversation"`
	Messages     []cmodels.ChatMessage    `json:"messages"`
}

type chatMessageReq struct {
	Message string `json:"message"`
	onlyJWT
}

type chatSettingsResponse struct {
	livechat.Config
	BusinessHours          []bhmodels.BusinessHours `json:"business_hours,omitempty"`
	DefaultBusinessHoursID int                      `json:"default_business_hours_id,omitempty"`
}

// conversationResponseWithBusinessHours includes business hours info for the widget
type conversationResponseWithBusinessHours struct {
	conversationResp
	BusinessHoursID       *int `json:"business_hours_id,omitempty"`
	WorkingHoursUTCOffset *int `json:"working_hours_utc_offset,omitempty"`
}

//	TODO: live chat widget can have a different language setting than the main app, handle this.
//
// handleGetChatLauncherSettings returns the live chat launcher settings for the widget
func handleGetChatLauncherSettings(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		inboxID = r.RequestCtx.QueryArgs().GetUintOrZero("inbox_id")
	)

	if inboxID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	// Get inbox configuration
	inbox, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		app.lo.Error("error fetching inbox", "inbox_id", inboxID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.NotFoundError)
	}

	if inbox.Channel != livechat.ChannelLiveChat {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]any{
		"launcher": config.Launcher,
		"colors":   config.Colors,
	})
}

// handleGetChatSettings returns the live chat settings for the widget
func handleGetChatSettings(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		inboxID = r.RequestCtx.QueryArgs().GetUintOrZero("inbox_id")
	)

	if inboxID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	// Get inbox configuration
	inbox, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		app.lo.Error("error fetching inbox", "inbox_id", inboxID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.NotFoundError)
	}

	if inbox.Channel != livechat.ChannelLiveChat {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Get business hours data if office hours feature is enabled.
	response := chatSettingsResponse{
		Config: config,
	}

	if config.ShowOfficeHoursInChat {
		// Get all business hours.
		businessHours, err := app.businessHours.GetAll()
		if err != nil {
			app.lo.Error("error fetching business hours", "error", err)
		} else {
			response.BusinessHours = businessHours
		}

		// Get default business hours ID from general settings which is the default / fallback.
		out, err := app.setting.GetByPrefix("app")
		if err != nil {
			app.lo.Error("error fetching general settings", "error", err)
		} else {
			var settings map[string]any
			if err := json.Unmarshal(out, &settings); err == nil {
				if bhID, ok := settings["app.business_hours_id"].(string); ok {
					response.DefaultBusinessHoursID, _ = strconv.Atoi(bhID)
				}
			}
		}
	}

	return r.SendEnvelope(response)
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.message}"), nil, envelope.InputError)
	}

	// Get inbox configuration
	inbox, err := app.inbox.GetDBRecord(req.InboxID)
	if err != nil {
		app.lo.Error("error fetching inbox", "inbox_id", req.InboxID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.NotFoundError)
	}

	// Make sure the inbox is enabled and of the correct type
	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}
	if inbox.Channel != livechat.ChannelLiveChat {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	// Parse inbox config
	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}
	var (
		contactID        int
		conversationUUID string
		isVisitor        bool
	)
	// Handle authenticated user
	if req.JWT != "" {
		claims, err := verifyStandardJWT(req.JWT)
		if err != nil {
			app.lo.Error("invalid JWT", "jwt", req.JWT, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.messages.sessionExpired"), nil, envelope.UnauthorizedError)
		}
		contactID = claims.UserID
		isVisitor = claims.IsVisitor
	} else {
		isVisitor = true
		visitor := umodels.User{
			Email:     null.NewString(req.VisitorEmail, req.VisitorEmail != ""),
			FirstName: req.VisitorName,
		}

		if err := app.user.CreateVisitor(&visitor); err != nil {
			app.lo.Error("error creating visitor contact", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
		}
		contactID = visitor.ID

		// Generate visitor JWT it has longer expiry as short lived jwts will create new visitors every time.
		req.JWT, err = generateUserJWT(contactID, isVisitor, time.Now().Add(87600*time.Hour)) // 10 years
		if err != nil {
			app.lo.Error("error generating visitor JWT", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorGenerating", "name", "{globals.terms.session}"), nil, envelope.GeneralError)
		}
	}

	// Check conversation permissions based on user type.
	userConfig := config.Visitors
	if !isVisitor {
		userConfig = config.Users
	}

	if !userConfig.AllowStartConversation {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("globals.messages.notAllowed}"), nil, envelope.PermissionError)
	}

	if userConfig.PreventMultipleConversations {
		conversations, err := app.conversation.GetContactChatConversations(contactID)
		if err != nil {
			userType := "visitor"
			if !isVisitor {
				userType = "user"
			}
			app.lo.Error("error fetching "+userType+" conversations", "contact_id", contactID, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil, envelope.GeneralError)
		}
		if len(conversations) > 0 {
			userType := "visitor"
			if !isVisitor {
				userType = "user"
			}
			app.lo.Info(userType+" attempted to start new conversation but already has one", "contact_id", contactID, "conversations_count", len(conversations))
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("globals.messages.notAllowed}"), nil, envelope.PermissionError)
		}
	}

	app.lo.Info("creating new live chat conversation for user", "user_id", contactID, "inbox_id", req.InboxID, "is_visitor", isVisitor)

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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	// Insert initial message.
	message := cmodels.Message{
		ConversationUUID: conversationUUID,
		SenderID:         contactID,
		Type:             cmodels.MessageIncoming,
		SenderType:       cmodels.SenderTypeContact,
		Status:           cmodels.MessageStatusReceived,
		Content:          req.Message,
		ContentType:      cmodels.ContentTypeText,
		Private:          false,
	}
	if err := app.conversation.InsertMessage(&message); err != nil {
		// Clean up conversation if message insert fails.
		if err := app.conversation.DeleteConversation(conversationUUID); err != nil {
			app.lo.Error("error deleting conversation after message insert failure", "conversation_uuid", conversationUUID, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
		}
		app.lo.Error("error inserting initial message", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	// Process post-message hooks for the new conversation and initial message.
	if err := app.conversation.ProcessIncomingMessageHooks(conversationUUID, true); err != nil {
		app.lo.Error("error processing incoming message hooks for initial message", "conversation_uuid", conversationUUID, "error", err)
	}

	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching created conversation", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil, envelope.GeneralError)
	}

	// Build response with conversation and messages.
	resp, err := buildConversationResponseWithBusinessHours(app, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]any{
		"conversation":             resp.Conversation,
		"messages":                 resp.Messages,
		"business_hours_id":        resp.BusinessHoursID,
		"working_hours_utc_offset": resp.WorkingHoursUTCOffset,
		"jwt":                      req.JWT,
	})
}

// handleChatUpdateLastSeen updates contact last seen timestamp for a conversation
func handleChatUpdateLastSeen(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
		req              = onlyJWT{}
	)

	if conversationUUID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.conversation}"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling chat update last seen request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	// Verify JWT.
	claims, err := verifyStandardJWT(req.JWT)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", req.JWT, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}
	contactID := claims.UserID
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	// Fetch conversation.
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Make sure the conversation belongs to the contact.
	if conversation.ContactID != contactID {
		app.lo.Error("unauthorized access to conversation", "conversation_uuid", conversationUUID, "contact_id", contactID, "conversation_contact_id", conversation.ContactID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
	}

	// Update last seen timestamp.
	if err := app.conversation.UpdateConversationContactLastSeen(conversation.UUID); err != nil {
		app.lo.Error("error updating contact last seen timestamp", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]bool{"success": true})
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
	claims, err := verifyStandardJWT(chatReq.JWT)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", chatReq.JWT, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}
	contactID := claims.UserID
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	// Fetch conversation.
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Make sure the conversation belongs to the contact.
	if conversation.ContactID != contactID {
		app.lo.Error("unauthorized access to conversation", "conversation_uuid", conversationUUID, "contact_id", contactID, "conversation_contact_id", conversation.ContactID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
	}

	// Build conversation response with messages and attachments.
	resp, err := buildConversationResponseWithBusinessHours(app, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(resp)
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

	// Verify JWT.
	claims, err := verifyStandardJWT(req.JWT)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", req.JWT, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	// Fetch conversations for the contact and convert to ChatConversation format.
	chatConversations, err := app.conversation.GetContactChatConversations(claims.UserID)
	if err != nil {
		app.lo.Error("error fetching conversations for contact", "contact_id", claims.UserID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(chatConversations)
}

// handleChatSendMessage sends a message in a chat conversation
func handleChatSendMessage(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
		req              = chatMessageReq{}
		senderType       = cmodels.SenderTypeContact
		senderID         = 0
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling chat message request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.message}"), nil, envelope.InputError)
	}

	claims, err := verifyStandardJWT(req.JWT)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", req.JWT, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}
	senderID = claims.UserID

	// Fetch sender.
	sender, err := app.user.Get(senderID, "", "")
	if err != nil {
		app.lo.Error("error fetching sender user", "sender_id", senderID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
	}

	// Fetch conversation to ensure it exists.
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Make sure the conversation belongs to the sender.
	if conversation.ContactID != senderID {
		app.lo.Error("access denied: user attempted to access conversation owned by different contact", "conversation_uuid", conversationUUID, "requesting_contact_id", senderID, "conversation_owner_id", conversation.ContactID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
	}

	// Make sure the inbox is enabled.
	inbox, err := app.inbox.GetDBRecord(conversation.InboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	// Insert incoming message and run post processing hooks.
	message := cmodels.Message{
		ConversationUUID: conversationUUID,
		SenderID:         senderID,
		Type:             cmodels.MessageIncoming,
		SenderType:       senderType,
		Status:           cmodels.MessageStatusReceived,
		Content:          req.Message,
		ContentType:      cmodels.ContentTypeText,
		Private:          false,
	}
	if message, err = app.conversation.ProcessIncomingMessage(cmodels.IncomingMessage{
		Channel: livechat.ChannelLiveChat,
		Message: message,
		Contact: sender,
		InboxID: inbox.ID,
	}); err != nil {
		app.lo.Error("error processing incoming message", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	// Fetch just inserted message to return.
	message, err = app.conversation.GetMessage(message.UUID)
	if err != nil {
		app.lo.Error("error fetching inserted message", "message_uuid", message.UUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(cmodels.ChatMessage{
		UUID:             message.UUID,
		CreatedAt:        message.CreatedAt,
		Content:          message.Content,
		TextContent:      message.TextContent,
		ConversationUUID: message.ConversationUUID,
		Status:           message.Status,
		Author: umodels.ChatUser{
			ID:                 sender.ID,
			FirstName:          sender.FirstName,
			LastName:           sender.LastName,
			AvatarURL:          sender.AvatarURL,
			AvailabilityStatus: sender.AvailabilityStatus,
			Type:               sender.Type,
		},
		Attachments: message.Attachments,
	})
}

// handleWidgetMediaUpload handles media uploads for the widget.
func handleWidgetMediaUpload(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		senderID = 0
	)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data.", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	// Get JWT token from form data
	jwtValues, jwtOk := form.Value["jwt"]
	if !jwtOk || len(jwtValues) == 0 || jwtValues[0] == "" {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}
	jwtToken := jwtValues[0]

	// Get conversation UUID from form data
	conversationValues, convOk := form.Value["conversation_uuid"]
	if !convOk || len(conversationValues) == 0 || conversationValues[0] == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.conversation}"), nil, envelope.InputError)
	}
	conversationUUID := conversationValues[0]

	// Verify JWT and get user information
	claims, err := verifyStandardJWT(jwtToken)
	if err != nil {
		app.lo.Error("invalid JWT", "jwt", jwtToken, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	// Set sender ID from JWT claims.
	senderID = claims.UserID

	// Make sure the conversation belongs to the sender
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}
	if conversation.ContactID != senderID {
		app.lo.Error("access denied: user attempted to access conversation owned by different contact", "conversation_uuid", conversationUUID, "requesting_contact_id", senderID, "conversation_owner_id", conversation.ContactID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
	}

	// Make sure the inbox is enabled.
	inbox, err := app.inbox.GetDBRecord(conversation.InboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	// Make sure file upload is enabled for the inbox.
	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	if !config.Features.FileUpload {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.fileUpload}"), nil, envelope.InputError)
	}

	files, ok := form.File["files"]
	if !ok || len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.file}"), nil, envelope.InputError)
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error reading uploaded file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorReading", "name", "{globals.terms.file}"), nil, envelope.GeneralError)
	}
	defer file.Close()

	// Sanitize filename.
	srcFileName := stringutil.SanitizeFilename(fileHeader.Filename)
	srcContentType := fileHeader.Header.Get("Content-Type")
	srcFileSize := fileHeader.Size
	srcExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

	// Check file size
	consts := app.consts.Load().(*constants)
	if bytesToMegabytes(srcFileSize) > float64(consts.MaxFileUploadSizeMB) {
		app.lo.Error("error: uploaded file size is larger than max allowed", "size", bytesToMegabytes(srcFileSize), "max_allowed", consts.MaxFileUploadSizeMB)
		return r.SendErrorEnvelope(
			fasthttp.StatusRequestEntityTooLarge,
			app.i18n.Ts("media.fileSizeTooLarge", "size", fmt.Sprintf("%dMB", consts.MaxFileUploadSizeMB)),
			nil,
			envelope.GeneralError,
		)
	}

	// Make sure the file extension is allowed.
	if !slices.Contains(consts.AllowedUploadFileExtensions, "*") && !slices.Contains(consts.AllowedUploadFileExtensions, srcExt) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("media.fileTypeNotAllowed"), nil, envelope.InputError)
	}

	// Read file content into byte slice
	file.Seek(0, 0)
	fileContent := make([]byte, srcFileSize)
	if _, err := file.Read(fileContent); err != nil {
		app.lo.Error("error reading file content", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorReading", "name", "{globals.terms.file}"), nil, envelope.GeneralError)
	}

	// Get sender user for ProcessIncomingMessage
	sender, err := app.user.Get(senderID, "", "")
	if err != nil {
		app.lo.Error("error fetching sender user", "sender_id", senderID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
	}

	// Create message with attachment using existing infrastructure
	message := cmodels.Message{
		ConversationUUID: conversationUUID,
		SenderID:         senderID,
		Type:             cmodels.MessageIncoming,
		SenderType:       cmodels.SenderTypeContact,
		Status:           cmodels.MessageStatusReceived,
		Content:          "",
		ContentType:      cmodels.ContentTypeText,
		Private:          false,
		Attachments: attachment.Attachments{
			{
				Name:        srcFileName,
				ContentType: srcContentType,
				Size:        int(srcFileSize),
				Content:     fileContent,
				Disposition: attachment.DispositionAttachment,
			},
		},
	}

	// Process the incoming message with attachment.
	if message, err = app.conversation.ProcessIncomingMessage(cmodels.IncomingMessage{
		Channel: livechat.ChannelLiveChat,
		Message: message,
		Contact: sender,
		InboxID: inbox.ID,
	}); err != nil {
		app.lo.Error("error processing incoming message with attachment", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorInserting", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	// Fetch the inserted message to get the media information.
	insertedMessage, err := app.conversation.GetMessage(message.UUID)
	if err != nil {
		app.lo.Error("error fetching inserted message", "message_uuid", message.UUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(insertedMessage)
}

// buildConversationResponse builds the response for a conversation including its messages
func buildConversationResponse(app *App, conversation cmodels.Conversation) (conversationResp, error) {
	var resp = conversationResp{}

	// Fetch last 1000 messages, this should suffice as chats shouldn't have too many messages.
	private := false
	messages, _, err := app.conversation.GetConversationMessages(conversation.UUID, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing}, &private, 1, 1000)
	if err != nil {
		app.lo.Error("error fetching conversation messages", "conversation_uuid", conversation.UUID, "error", err)
		return resp, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.message}"), nil)
	}

	app.conversation.ProcessCSATStatus(messages)

	// Convert to chat message format, Generate signed widget URL for all attachments - expires in 1 hour.
	chatMessages := make([]cmodels.ChatMessage, len(messages))
	userCache := make(map[int]umodels.User)
	for i, msg := range messages {
		attachments := msg.Attachments
		for j := range attachments {
			expiresAt := time.Now().Add(8 * time.Hour)
			attachments[j].URL = app.media.GetSignedURL(attachments[j].UUID, expiresAt)
		}

		// Check if sender is cached, if not fetch from user store.
		var user umodels.User
		if _, ok := userCache[msg.SenderID]; !ok {
			user, err = app.user.Get(msg.SenderID, "", "")
			if err != nil {
				app.lo.Error("error fetching message sender user", "sender_id", msg.SenderID, "conversation_uuid", conversation.UUID, "error", err)
			} else {
				userCache[msg.SenderID] = user
			}
		} else {
			user = userCache[msg.SenderID]
		}

		chatMessages[i] = cmodels.ChatMessage{
			UUID:             msg.UUID,
			Status:           msg.Status,
			CreatedAt:        msg.CreatedAt,
			Content:          msg.Content,
			TextContent:      msg.TextContent,
			ConversationUUID: msg.ConversationUUID,
			Meta:             msg.Meta,
			Author: umodels.ChatUser{
				ID:                 user.ID,
				FirstName:          user.FirstName,
				LastName:           user.LastName,
				AvatarURL:          user.AvatarURL,
				AvailabilityStatus: user.AvailabilityStatus,
				Type:               user.Type,
			},
			Attachments: attachments,
		}
	}

	var (
		assignee umodels.User
	)
	if conversation.AssignedUserID.Int > 0 {
		assignee, err = app.user.GetAgent(conversation.AssignedUserID.Int, "")
		if err != nil {
			app.lo.Error("error fetching conversation assignee", "conversation_uuid", conversation.UUID, "error", err)
		}

		// Convert assignee avatar URL to widget format if set.
		if assignee.AvatarURL.Valid && assignee.AvatarURL.String != "" {
			avatarPath := assignee.AvatarURL.String
			if strings.HasPrefix(avatarPath, "/uploads/") {
				avatarUUID := strings.TrimPrefix(avatarPath, "/uploads/")
				// Generate signed URL for avatar with 1 hour expiry
				expiresAt := time.Now().Add(1 * time.Hour)
				assignee.AvatarURL = null.StringFrom(app.media.GetSignedURL(avatarUUID, expiresAt))
			}
		}
	}

	resp = conversationResp{
		Conversation: cmodels.ChatConversation{
			CreatedAt:          assignee.CreatedAt,
			UUID:               conversation.UUID,
			Status:             conversation.Status.String,
			UnreadMessageCount: conversation.UnreadMessageCount,
			Assignee: umodels.ChatUser{
				ID:                 assignee.ID,
				FirstName:          assignee.FirstName,
				LastName:           assignee.LastName,
				AvatarURL:          assignee.AvatarURL,
				AvailabilityStatus: assignee.AvailabilityStatus,
				Type:               assignee.Type,
				ActiveAt:           assignee.LastActiveAt,
			},
		},
		Messages: chatMessages,
	}

	return resp, nil
}

// buildConversationResponseWithBusinessHours builds conversation response with business hours info
func buildConversationResponseWithBusinessHours(app *App, conversation cmodels.Conversation) (conversationResponseWithBusinessHours, error) {
	baseResp, err := buildConversationResponse(app, conversation)
	if err != nil {
		return conversationResponseWithBusinessHours{}, err
	}

	resp := conversationResponseWithBusinessHours{
		conversationResp: baseResp,
	}

	// Calculate business hours info if assigned to team or use default
	var businessHoursID *int
	var timezone string

	// Check if conversation is assigned to a team with business hours
	if conversation.AssignedTeamID.Valid {
		team, err := app.team.Get(conversation.AssignedTeamID.Int)
		if err == nil && team.BusinessHoursID.Valid {
			businessHoursID = &team.BusinessHoursID.Int
			timezone = team.Timezone
		}
	}

	// Fallback to general settings if no team business hours
	if businessHoursID == nil {
		out, err := app.setting.GetByPrefix("app")
		if err == nil {
			var settings map[string]interface{}
			if err := json.Unmarshal(out, &settings); err == nil {
				if bhIDStr, ok := settings["app.business_hours_id"].(string); ok && bhIDStr != "" {
					// Parse the business hours ID
					if bhID, err := strconv.Atoi(bhIDStr); err == nil {
						businessHoursID = &bhID
					}
				}
				if tz, ok := settings["app.timezone"].(string); ok {
					timezone = tz
				}
			}
		}
	}

	// Set business hours info in response
	if businessHoursID != nil {
		resp.BusinessHoursID = businessHoursID

		// Calculate UTC offset for the timezone
		if timezone != "" {
			if loc, err := time.LoadLocation(timezone); err == nil {
				_, offset := time.Now().In(loc).Zone()
				offsetMinutes := offset / 60 // Convert seconds to minutes
				resp.WorkingHoursUTCOffset = &offsetMinutes
			}
		}
	}

	return resp, nil
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
func generateUserJWT(userID int, isVisitor bool, expirationTime time.Time) (string, error) {
	claims := &Claims{
		UserID:    userID,
		IsVisitor: isVisitor,
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
