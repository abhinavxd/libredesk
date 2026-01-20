package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
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
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

// Define JWT claims structure
type Claims struct {
	UserID           int            `json:"user_id,omitempty"`
	ExternalUserID   string         `json:"external_user_id,omitempty"`
	IsVisitor        bool           `json:"is_visitor,omitempty"`
	Email            string         `json:"email,omitempty"`
	FirstName        string         `json:"first_name,omitempty"`
	LastName         string         `json:"last_name,omitempty"`
	CustomAttributes map[string]any `json:"custom_attributes,omitempty"`
	jwt.RegisteredClaims
}

type conversationResp struct {
	Conversation cmodels.ChatConversation `json:"conversation"`
	Messages     []cmodels.ChatMessage    `json:"messages"`
}

type customAttributeWidget struct {
	ID     int      `json:"id"`
	Values []string `json:"values"`
}

type chatInitReq struct {
	Message  string         `json:"message"`
	FormData map[string]any `json:"form_data"`
}

type chatSettingsResponse struct {
	livechat.Config
	BusinessHours          []bhmodels.BusinessHours      `json:"business_hours,omitempty"`
	DefaultBusinessHoursID int                           `json:"default_business_hours_id,omitempty"`
	CustomAttributes       map[int]customAttributeWidget `json:"custom_attributes,omitempty"`
}

// conversationResponseWithBusinessHours includes business hours info for the widget
type conversationResponseWithBusinessHours struct {
	conversationResp
	BusinessHoursID       *int `json:"business_hours_id,omitempty"`
	WorkingHoursUTCOffset *int `json:"working_hours_utc_offset,omitempty"`
}

// validateLiveChatInbox validates inbox_id from query params and returns the inbox and parsed config.
// Used by public widget endpoints that don't require JWT authentication.
func validateLiveChatInbox(r *fastglue.Request) (imodels.Inbox, livechat.Config, error) {
	app := r.Context.(*App)
	inboxID := r.RequestCtx.QueryArgs().GetUintOrZero("inbox_id")

	if inboxID <= 0 {
		return imodels.Inbox{}, livechat.Config{}, r.SendErrorEnvelope(
			fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.required", "name", "{globals.terms.inbox}"),
			nil, envelope.InputError)
	}

	inbox, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		app.lo.Error("error fetching inbox", "inbox_id", inboxID, "error", err)
		return imodels.Inbox{}, livechat.Config{}, sendErrorEnvelope(r, err)
	}

	if inbox.Channel != livechat.ChannelLiveChat {
		return imodels.Inbox{}, livechat.Config{}, r.SendErrorEnvelope(
			fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"),
			nil, envelope.InputError)
	}

	if !inbox.Enabled {
		return imodels.Inbox{}, livechat.Config{}, r.SendErrorEnvelope(
			fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"),
			nil, envelope.InputError)
	}

	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return imodels.Inbox{}, livechat.Config{}, r.SendErrorEnvelope(
			fasthttp.StatusInternalServerError,
			app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.inbox}"),
			nil, envelope.GeneralError)
	}

	return inbox, config, nil
}

// handleGetChatLauncherSettings returns the live chat launcher settings for the widget
func handleGetChatLauncherSettings(r *fastglue.Request) error {
	_, config, err := validateLiveChatInbox(r)
	if err != nil {
		return err
	}

	return r.SendEnvelope(map[string]any{
		"launcher": config.Launcher,
		"colors":   config.Colors,
	})
}

// handleGetChatSettings returns the live chat settings for the widget
func handleGetChatSettings(r *fastglue.Request) error {
	app := r.Context.(*App)

	_, config, err := validateLiveChatInbox(r)
	if err != nil {
		return err
	}

	// Get business hours data if office hours feature is enabled.
	response := chatSettingsResponse{
		Config: config,
	}

	if config.ShowOfficeHoursInChat {
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

	// Filter out pre-chat form fields for which custom attributes don't exist anymore.
	if config.PreChatForm.Enabled && len(config.PreChatForm.Fields) > 0 {
		filteredFields, customAttributes := filterPreChatFormFields(config.PreChatForm.Fields, app)
		response.PreChatForm.Fields = filteredFields
		if len(customAttributes) > 0 {
			response.CustomAttributes = customAttributes
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

	// Get authenticated data from context (set by middleware), middleware always validates inbox, so we can safely use non-optional getters
	claims := getWidgetClaimsOptional(r)
	inboxID, err := getWidgetInboxID(r)
	if err != nil {
		app.lo.Error("error getting inbox ID from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}
	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	var (
		contactID        int
		conversationUUID string
		isVisitor        bool
		config           livechat.Config
		newJWT           string
	)

	// Parse inbox config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Handle authenticated user vs visitor
	if claims != nil {
		// Handle existing contacts with external user id - check if we need to create user
		if claims.ExternalUserID != "" {
			// Find or create user based on external_user_id.
			user, err := app.user.GetByExternalID(claims.ExternalUserID)
			if err != nil {
				envErr, ok := err.(envelope.Error)
				if ok && envErr.ErrorType != envelope.NotFoundError {
					app.lo.Error("error fetching user by external ID", "external_user_id", claims.ExternalUserID, "error", err)
					return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
				}

				// User doesn't exist, create new contact
				firstName := claims.FirstName
				lastName := claims.LastName
				email := claims.Email

				// Validate custom attribute
				formCustomAttributes := validateCustomAttributes(req.FormData, config, app)

				// Merge JWT and form custom attributes (form takes precedence)
				mergedAttributes := mergeCustomAttributes(claims.CustomAttributes, formCustomAttributes)

				// Marshal custom attributes
				customAttribJSON, err := json.Marshal(mergedAttributes)
				if err != nil {
					app.lo.Error("error marshalling custom attributes", "error", err)
					customAttribJSON = []byte("{}")
				}

				// Create new contact with external user ID.
				var user = umodels.User{
					FirstName:        firstName,
					LastName:         lastName,
					Email:            null.NewString(email, email != ""),
					ExternalUserID:   null.NewString(claims.ExternalUserID, claims.ExternalUserID != ""),
					CustomAttributes: customAttribJSON,
				}
				err = app.user.CreateContact(&user)
				if err != nil {
					app.lo.Error("error creating contact with external ID", "external_user_id", claims.ExternalUserID, "error", err)
					return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
				}
				contactID = user.ID
			} else {
				// User exists, update custom attributes from both JWT and form
				// Don't override existing name and email.

				// Validate custom attribute
				formCustomAttributes := validateCustomAttributes(req.FormData, config, app)

				// Merge JWT and form custom attributes (form takes precedence)
				mergedAttributes := mergeCustomAttributes(claims.CustomAttributes, formCustomAttributes)

				if len(mergedAttributes) > 0 {
					if err := app.user.SaveCustomAttributes(user.ID, mergedAttributes, false); err != nil {
						app.lo.Error("error updating contact custom attributes", "contact_id", user.ID, "error", err)
						// Don't fail the request for custom attributes update failure
					}
				}
				contactID = user.ID
			}
			isVisitor = false
		} else {
			// Authenticated visitor
			isVisitor = claims.IsVisitor
			contactID, err = getWidgetContactID(r)
			if err != nil {
				app.lo.Error("error getting contact ID from middleware context", "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
			}

			// Validate custom attribute
			formCustomAttributes := validateCustomAttributes(req.FormData, config, app)

			// Merge JWT and form custom attributes (form takes precedence)
			mergedAttributes := mergeCustomAttributes(claims.CustomAttributes, formCustomAttributes)

			// Update custom attributes from both JWT and form
			if len(mergedAttributes) > 0 {
				if err := app.user.SaveCustomAttributes(contactID, mergedAttributes, false); err != nil {
					app.lo.Error("error updating contact custom attributes", "contact_id", contactID, "error", err)
					// Don't fail the request for custom attributes update failure
				}
			}
		}
	} else {
		// Visitor user not authenticated, create a new visitor contact.
		isVisitor = true

		// Validate form data and get final name/email for new visitor
		finalName, finalEmail, err := validateFormData(req.FormData, config, nil)
		if err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, err.Error(), nil, envelope.InputError)
		}

		// Process custom attributes from form data
		formCustomAttributes := validateCustomAttributes(req.FormData, config, app)

		// Marshal custom attributes for storage
		var customAttribJSON []byte
		if len(formCustomAttributes) > 0 {
			customAttribJSON, err = json.Marshal(formCustomAttributes)
			if err != nil {
				app.lo.Error("error marshalling form custom attributes", "error", err)
				customAttribJSON = []byte("{}")
			}
		} else {
			customAttribJSON = []byte("{}")
		}

		visitor := umodels.User{
			Email:            null.NewString(finalEmail, finalEmail != ""),
			FirstName:        finalName,
			CustomAttributes: customAttribJSON,
		}

		if err := app.user.CreateVisitor(&visitor); err != nil {
			app.lo.Error("error creating visitor contact", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
		}
		contactID = visitor.ID
		secretToUse := []byte(inbox.Secret.String)
		newJWT, err = generateUserJWTWithSecret(contactID, isVisitor, time.Now().Add(87600*time.Hour), secretToUse) // 10 years
		if err != nil {
			app.lo.Error("error generating visitor JWT", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorGenerating", "name", "{globals.terms.session}"), nil, envelope.GeneralError)
		}
	}

	// Check conversation permissions based on user type.
	var allowStartConversation, preventMultipleConversations bool
	if isVisitor {
		allowStartConversation = config.Visitors.AllowStartConversation
		preventMultipleConversations = config.Visitors.PreventMultipleConversations
	} else {
		allowStartConversation = config.Users.AllowStartConversation
		preventMultipleConversations = config.Users.PreventMultipleConversations
	}

	if !allowStartConversation {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.notAllowed", "name", ""), nil, envelope.PermissionError)
	}

	if preventMultipleConversations {
		conversations, err := app.conversation.GetContactChatConversations(contactID, inboxID)
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
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.notAllowed", "name", ""), nil, envelope.PermissionError)
		}
	}

	app.lo.Info("creating new live chat conversation for user", "user_id", contactID, "inbox_id", inboxID, "is_visitor", isVisitor)

	// Create conversation.
	_, conversationUUID, err = app.conversation.CreateConversation(
		contactID,
		inboxID,
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

	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
	if err != nil {
		app.lo.Error("error fetching created conversation", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil, envelope.GeneralError)
	}

	// Build response with conversation and messages and add business hours info.
	resp, err := buildConversationResponseWithBusinessHours(app, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// For visitors, return the new JWT. For authenticated users, no JWT is needed in response.
	response := map[string]any{
		"conversation":             resp.Conversation,
		"messages":                 resp.Messages,
		"business_hours_id":        resp.BusinessHoursID,
		"working_hours_utc_offset": resp.WorkingHoursUTCOffset,
	}

	// Only add JWT for visitor creation
	if newJWT != "" {
		response["jwt"] = newJWT
	}

	return r.SendEnvelope(response)
}

// handleChatUpdateLastSeen updates contact last seen timestamp for a conversation
func handleChatUpdateLastSeen(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
	)

	if conversationUUID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.conversation}"), nil, envelope.InputError)
	}

	// Get authenticated data from middleware context
	contactID, err := getWidgetContactID(r)
	if err != nil {
		app.lo.Error("error getting contact ID from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
	}

	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
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

	// Also update custom attributes from JWT claims, if present.
	// This avoids a separate handler and ensures contact attributes stay in sync.
	// Since this endpoint is hit frequently during chat, it's a good place to keep them updated.
	claims := getWidgetClaimsOptional(r)
	if claims != nil && len(claims.CustomAttributes) > 0 {
		if err := app.user.SaveCustomAttributes(contactID, claims.CustomAttributes, false); err != nil {
			app.lo.Error("error updating contact custom attributes", "contact_id", contactID, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
		}
	}

	return r.SendEnvelope(true)
}

// handleChatGetConversation fetches a chat conversation by ID
func handleChatGetConversation(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
	)

	if conversationUUID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "conversation_id is required", nil, envelope.InputError)
	}

	// Get authenticated data from middleware context
	contactID, err := getWidgetContactID(r)
	if err != nil {
		app.lo.Error("error getting contact ID from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
	}

	// Fetch conversation
	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
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
	)

	// Get authenticated data from middleware context
	contactID, err := getWidgetContactID(r)
	if err != nil {
		app.lo.Error("error getting contact ID from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
	}

	inboxID, err := getWidgetInboxID(r)
	if err != nil {
		app.lo.Error("error getting inbox ID from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Fetch conversations for the contact and convert to ChatConversation format.
	chatConversations, err := app.conversation.GetContactChatConversations(contactID, inboxID)
	if err != nil {
		app.lo.Error("error fetching conversations for contact", "contact_id", contactID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(chatConversations)
}

// handleChatSendMessage sends a message in a chat conversation
func handleChatSendMessage(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		conversationUUID = r.RequestCtx.UserValue("uuid").(string)
		req              = struct {
			Message string `json:"message"`
		}{}
		senderType = cmodels.SenderTypeContact
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling chat message request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.message}"), nil, envelope.InputError)
	}

	// Get authenticated data from middleware context
	senderID, err := getWidgetContactID(r)
	if err != nil {
		app.lo.Error("error getting contact ID from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
	}

	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Fetch conversation to ensure it exists
	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Fetch sender.
	sender, err := app.user.Get(senderID, "", []string{})
	if err != nil {
		app.lo.Error("error fetching sender user", "sender_id", senderID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
	}

	// Make sure the conversation belongs to the sender.
	if conversation.ContactID != senderID {
		app.lo.Error("access denied: user attempted to access conversation owned by different contact", "conversation_uuid", conversationUUID, "requesting_contact_id", senderID, "conversation_owner_id", conversation.ContactID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
	}

	// Make sure the inbox is enabled.
	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
	}

	// Insert incoming message and run post processing hooks.
	message := cmodels.Message{
		ConversationUUID: conversationUUID,
		ConversationID:   conversation.ID,
		SenderID:         senderID,
		Type:             cmodels.MessageIncoming,
		SenderType:       senderType,
		Status:           cmodels.MessageStatusReceived,
		Content:          req.Message,
		ContentType:      cmodels.ContentTypeText,
		Private:          false,
	}
	if message, err = app.conversation.ProcessIncomingLiveChatMessage(message); err != nil {
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
		app = r.Context.(*App)
	)

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data.", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	// Get authenticated data from middleware context
	senderID, err := getWidgetContactID(r)
	if err != nil {
		app.lo.Error("error getting contact ID from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
	}

	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Get conversation UUID from form data
	conversationValues, convOk := form.Value["conversation_uuid"]
	if !convOk || len(conversationValues) == 0 || conversationValues[0] == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.conversation}"), nil, envelope.InputError)
	}
	conversationUUID := conversationValues[0]

	// Make sure the conversation belongs to the sender
	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	if conversation.ContactID != senderID {
		app.lo.Error("access denied: user attempted to access conversation owned by different contact", "conversation_uuid", conversationUUID, "requesting_contact_id", senderID, "conversation_owner_id", conversation.ContactID)
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
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

	message := cmodels.Message{
		ConversationUUID: conversationUUID,
		ConversationID:   conversation.ID,
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
	if message, err = app.conversation.ProcessIncomingLiveChatMessage(message); err != nil {
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

// buildConversationResponseWithBusinessHours builds conversation response with business hours info
func buildConversationResponseWithBusinessHours(app *App, conversation cmodels.Conversation) (conversationResponseWithBusinessHours, error) {
	widgetResp, err := app.conversation.BuildWidgetConversationResponse(conversation, true)
	if err != nil {
		return conversationResponseWithBusinessHours{}, err
	}

	resp := conversationResponseWithBusinessHours{
		conversationResp: conversationResp{
			Conversation: widgetResp.Conversation,
			Messages:     widgetResp.Messages,
		},
		BusinessHoursID:       widgetResp.BusinessHoursID,
		WorkingHoursUTCOffset: widgetResp.WorkingHoursUTCOffset,
	}

	return resp, nil
}

// resolveUserIDFromClaims resolves the actual user ID from JWT claims,
// handling both regular user_id and external_user_id cases
func resolveUserIDFromClaims(app *App, claims Claims) (int, error) {
	if claims.UserID > 0 {
		user, err := app.user.Get(claims.UserID, "", []string{})
		if err != nil {
			app.lo.Error("error fetching user by user ID", "user_id", claims.UserID, "error", err)
			return 0, errors.New("error fetching user")
		}
		if !user.Enabled {
			return 0, errors.New("user is disabled")
		}
		return user.ID, nil
	} else if claims.ExternalUserID != "" {
		user, err := app.user.GetByExternalID(claims.ExternalUserID)
		if err != nil {
			app.lo.Error("error fetching user by external ID", "external_user_id", claims.ExternalUserID, "error", err)
			return 0, errors.New("error fetching user")
		}
		if !user.Enabled {
			return 0, errors.New("user is disabled")
		}

		return user.ID, nil
	}

	return 0, errors.New("error fetching user")
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

// verifyStandardJWT verifies a JWT token using inbox secret
func verifyStandardJWT(jwtToken string, inboxSecret string) (Claims, error) {
	if jwtToken == "" {
		return Claims{}, fmt.Errorf("JWT token is empty")
	}

	if inboxSecret == "" {
		return Claims{}, fmt.Errorf("inbox `secret` is not configured for JWT verification")
	}

	claims, err := verifyJWT(jwtToken, []byte(inboxSecret))
	if err != nil {
		return Claims{}, err
	}

	return *claims, nil
}

// generateUserJWTWithSecret generates a JWT token for a user with a specific secret
func generateUserJWTWithSecret(userID int, isVisitor bool, expirationTime time.Time, secret []byte) (string, error) {
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
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// mergeCustomAttributes merges JWT and form custom attributes with form taking precedence
func mergeCustomAttributes(jwtAttributes, formAttributes map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Add JWT attributes first (as fallback)
	maps.Copy(merged, jwtAttributes)

	// Add form attributes second (takes precedence)
	maps.Copy(merged, formAttributes)

	return merged
}

// validateCustomAttributes validates and processes custom attributes from form data
func validateCustomAttributes(formData map[string]interface{}, config livechat.Config, app *App) map[string]interface{} {
	customAttributes := make(map[string]interface{})

	if !config.PreChatForm.Enabled || len(formData) == 0 {
		return customAttributes
	}

	// Validate total number of form fields
	const maxFormFields = 50
	if len(formData) > maxFormFields {
		app.lo.Warn("form data exceeds maximum allowed fields", "received", len(formData), "max", maxFormFields)
		return customAttributes
	}

	// Create a map of valid field keys for quick lookup
	validFields := make(map[string]livechat.PreChatFormField)
	for _, field := range config.PreChatForm.Fields {
		if field.Enabled {
			validFields[field.Key] = field
		}
	}

	// Process each form data field
	for key, value := range formData {
		// Validate field key length
		const maxKeyLength = 100
		if len(key) > maxKeyLength {
			app.lo.Warn("form field key exceeds maximum length", "key", key, "length", len(key), "max", maxKeyLength)
			continue
		}

		// Check if field is valid according to pre-chat form config
		field, exists := validFields[key]
		if !exists {
			app.lo.Warn("form field not found in pre-chat form configuration", "key", key)
			continue
		}

		// Skip default fields (name, email) - these are handled separately
		if field.IsDefault {
			continue
		}

		// Only process custom fields that have a custom_attribute_id
		if field.CustomAttributeID == 0 {
			continue
		}

		// Validate and process string values with length limits
		if strValue, ok := value.(string); ok {
			const maxValueLength = 1000
			if len(strValue) > maxValueLength {
				app.lo.Warn("form field value exceeds maximum length", "key", key, "length", len(strValue), "max", maxValueLength)
				// Truncate the value instead of rejecting it
				strValue = strValue[:maxValueLength]
			}
			customAttributes[field.Key] = strValue
		}

		// Numbers
		if numValue, ok := value.(float64); ok {
			if math.IsNaN(numValue) || math.IsInf(numValue, 0) {
				app.lo.Warn("form field contains invalid numeric value", "key", key, "value", numValue)
				continue
			}

			if numValue > 1e12 || numValue < -1e12 {
				app.lo.Warn("form field numeric value out of acceptable range", "key", key, "value", numValue)
				continue
			}

			customAttributes[field.Key] = numValue
		}

		// Set rest as is
		customAttributes[field.Key] = value
	}

	return customAttributes
}

// validateFormData validates form data against pre-chat form configuration
// Returns the final name/email to use and any validation errors
func validateFormData(formData map[string]interface{}, config livechat.Config, existingUser *umodels.User) (string, string, error) {
	var finalName, finalEmail string

	if !config.PreChatForm.Enabled {
		return finalName, finalEmail, nil
	}

	// Process each enabled field in the pre-chat form
	for _, field := range config.PreChatForm.Fields {
		if !field.Enabled {
			continue
		}

		switch field.Key {
		case "name":
			if value, exists := formData[field.Key]; exists {
				if nameStr, ok := value.(string); ok {
					// For existing users, ignore form name if they already have one
					if existingUser != nil && existingUser.FirstName != "" {
						finalName = existingUser.FirstName
					} else {
						finalName = nameStr
					}
				}
			}
			// Validate required field
			if field.Required && finalName == "" {
				return "", "", fmt.Errorf("name is required")
			}

		case "email":
			if value, exists := formData[field.Key]; exists {
				if emailStr, ok := value.(string); ok {
					// For existing users, ignore form email if they already have one
					if existingUser != nil && existingUser.Email.Valid && existingUser.Email.String != "" {
						finalEmail = existingUser.Email.String
					} else {
						finalEmail = emailStr
					}
				}
			}
			// Validate required field
			if field.Required && finalEmail == "" {
				return "", "", fmt.Errorf("email is required")
			}
			// Validate email format if provided
			if finalEmail != "" && !stringutil.ValidEmail(finalEmail) {
				return "", "", fmt.Errorf("invalid email format")
			}
		}
	}

	return finalName, finalEmail, nil
}

// filterPreChatFormFields filters out pre-chat form fields that reference non-existent custom attributes while retaining the default fields
func filterPreChatFormFields(fields []livechat.PreChatFormField, app *App) ([]livechat.PreChatFormField, map[int]customAttributeWidget) {
	if len(fields) == 0 {
		return fields, nil
	}

	// Collect custom attribute IDs and enabled fields
	customAttrIDs := make(map[int]bool)
	enabledFields := make([]livechat.PreChatFormField, 0, len(fields))

	for _, field := range fields {
		if field.Enabled {
			enabledFields = append(enabledFields, field)
			if field.CustomAttributeID > 0 {
				customAttrIDs[field.CustomAttributeID] = true
			}
		}
	}

	// Fetch existing custom attributes
	existingCustomAttrs := make(map[int]customAttributeWidget)
	for id := range customAttrIDs {
		attr, err := app.customAttribute.Get(id)
		if err != nil {
			app.lo.Warn("custom attribute referenced in pre-chat form no longer exists", "custom_attribute_id", id, "error", err)
			continue
		}
		existingCustomAttrs[id] = customAttributeWidget{
			ID:     attr.ID,
			Values: attr.Values,
		}
	}

	// Filter out fields with non-existent custom attributes
	filteredFields := make([]livechat.PreChatFormField, 0, len(enabledFields))
	for _, field := range enabledFields {
		// Keep default fields
		if field.IsDefault {
			filteredFields = append(filteredFields, field)
			continue
		}

		// Only keep custom fields if their custom attribute exists
		if _, exists := existingCustomAttrs[field.CustomAttributeID]; exists {
			filteredFields = append(filteredFields, field)
		}
	}

	return filteredFields, existingCustomAttrs
}
