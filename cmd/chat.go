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
	"github.com/abhinavxd/libredesk/internal/httputil"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	maxChatConversationsPerContact  = 50
	chatConversationRateLimitWindow = 24 * time.Hour
)

// Define JWT claims structure
type Claims struct {
	UserID                       int            `json:"user_id,omitempty"`
	ExternalUserID               string         `json:"external_user_id,omitempty"`
	IsVisitor                    bool           `json:"is_visitor,omitempty"`
	Email                        string         `json:"email,omitempty"`
	FirstName                    string         `json:"first_name,omitempty"`
	LastName                     string         `json:"last_name,omitempty"`
	ContactCustomAttributes      map[string]any `json:"contact_custom_attributes,omitempty"`
	ConversationCustomAttributes map[string]any `json:"conversation_custom_attributes,omitempty"`
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

// handleGetChatLauncherSettings returns the live chat launcher settings for the widget
func handleGetChatLauncherSettings(r *fastglue.Request) error {
	_, config, err := validateLiveChatInbox(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
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
		return sendErrorEnvelope(r, err)
	}

	response := chatSettingsResponse{
		Config: config,
	}

	// Get business hours data if office hours feature is enabled.
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
		app               = r.Context.(*App)
		req               = chatInitReq{}
		clientIP          = realip.FromRequest(r.RequestCtx)
		userAgent         = string(r.RequestCtx.Request.Header.Peek("User-Agent"))
		contactID         int
		isVisitor         bool
		newJWT            string
		conversationAttrs map[string]any
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling chat init request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.message}"), nil, envelope.InputError)
	}

	claims := getWidgetClaimsOptional(r)
	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	config, err := parseLiveChatConfig(inbox)
	if err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("validation.invalidInbox"), nil, envelope.GeneralError)
	}

	// Handle authenticated user vs visitor.
	if claims != nil {
		if claims.ExternalUserID != "" {
			if claims.Email == "" {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "email"), nil, envelope.InputError)
			}
			contactID, conversationAttrs, err = resolveOrCreateExternalContact(app, *claims, req.FormData, config)
			if err != nil {
				app.lo.Error("error resolving or creating contact for external_user_id", "external_user_id", claims.ExternalUserID, "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
			}
			isVisitor = false
		} else {
			// Non-visitor JWTs must include `external_user_id`.
			if !claims.IsVisitor {
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "external_user_id"), nil, envelope.InputError)
			}
			isVisitor = true
			contactID, err = getWidgetContactID(r)
			if err != nil {
				app.lo.Error("error getting contact ID from middleware context", "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
			}
			conversationAttrs = saveContactAttrsAndCollectConvoAttrs(app, contactID, claims, req.FormData, config)
		}
	} else {
		isVisitor = true
		var visitorErr error
		contactID, newJWT, conversationAttrs, visitorErr = createVisitorContact(app, req.FormData, config, inbox)
		if visitorErr != nil {
			app.lo.Error("error creating visitor contact", "error", visitorErr)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
		}
	}

	// Check conversation permissions based on user type.
	if err := checkConversationPermissions(app, config, isVisitor, contactID, inbox.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.lo.Info("creating new live chat conversation for user", "user_id", contactID, "inbox_id", inbox.ID, "is_visitor", isVisitor)

	// Create conversation and insert message.
	meta := map[string]any{
		"ip":         clientIP,
		"user_agent": userAgent,
	}
	_, conversationUUID, err := app.conversation.CreateConversation(
		contactID,
		inbox.ID,
		"",
		time.Now(),
		"",
		false,
		meta,
		conversationAttrs,
		maxChatConversationsPerContact,
		chatConversationRateLimitWindow,
	)
	if err != nil {
		if envErr, ok := err.(envelope.Error); ok && envErr.ErrorType == envelope.RateLimitError {
			return sendErrorEnvelope(r, err)
		}
		app.lo.Error("error creating conversation", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingMessage"), nil, envelope.GeneralError)
	}

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
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingMessage"), nil, envelope.GeneralError)
		}
		app.lo.Error("error inserting initial message", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingMessage"), nil, envelope.GeneralError)
	}

	// Process post-message hooks for the new conversation and initial message.
	if err := app.conversation.ProcessIncomingMessageHooks(conversationUUID, true); err != nil {
		app.lo.Error("error processing incoming message hooks for initial message", "conversation_uuid", conversationUUID, "error", err)
	}

	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
	if err != nil {
		app.lo.Error("error fetching created conversation", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Build response with conversation and messages and add business hours info.
	resp, err := buildConversationResponseWithBusinessHours(app, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	response := map[string]any{
		"conversation":             resp.Conversation,
		"messages":                 resp.Messages,
		"business_hours_id":        resp.BusinessHoursID,
		"working_hours_utc_offset": resp.WorkingHoursUTCOffset,
	}

	// Only add JWT when a new visitor is created.
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

	contactID, conversation, err := getContactConversation(r, conversationUUID)
	if err != nil {
		return err
	}

	// Update last seen timestamp.
	if err := app.conversation.UpdateConversationContactLastSeen(conversation.UUID); err != nil {
		app.lo.Error("error updating contact last seen timestamp", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Also update custom attributes from JWT claims, if present.
	// This avoids a separate handler and ensures contact attributes stay in sync.
	// Since this endpoint is hit frequently during chat, this should keep attribs in sync.
	claims := getWidgetClaimsOptional(r)
	if claims != nil && len(claims.ContactCustomAttributes) > 0 {
		if err := app.user.SaveCustomAttributes(contactID, claims.ContactCustomAttributes, false); err != nil {
			app.lo.Error("error updating contact custom attributes", "contact_id", contactID, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
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

	_, conversation, err := getContactConversation(r, conversationUUID)
	if err != nil {
		return err
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Fetch conversations for the contact and convert to ChatConversation format.
	chatConversations, err := app.conversation.GetContactChatConversations(contactID, inbox.ID)
	if err != nil {
		app.lo.Error("error fetching conversations for contact", "contact_id", contactID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.message}"), nil, envelope.InputError)
	}

	senderID, conversation, err := getContactConversation(r, conversationUUID)
	if err != nil {
		return err
	}

	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Fetch sender.
	sender, err := app.user.Get(senderID, "", []string{})
	if err != nil {
		app.lo.Error("error fetching sender user", "sender_id", senderID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Check if replies to closed conversations are allowed.
	if conversation.Status.String == cmodels.StatusClosed {
		lcConfig, err := parseLiveChatConfig(inbox)
		if err != nil {
			app.lo.Error("error parsing live chat config", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("validation.invalidInbox"), nil, envelope.GeneralError)
		}

		preventReply := lcConfig.Visitors.PreventReplyToClosedConversation
		if sender.Type != umodels.UserTypeVisitor {
			preventReply = lcConfig.Users.PreventReplyToClosedConversation
		}
		if preventReply {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("widget.conversationClosed"), nil, envelope.InputError)
		}
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingMessage"), nil, envelope.GeneralError)
	}

	// Fetch just inserted message to return.
	message, err = app.conversation.GetMessage(message.UUID)
	if err != nil {
		app.lo.Error("error fetching inserted message", "message_uuid", message.UUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	for i := range message.Attachments {
		message.Attachments[i].URL = app.media.GetSignedURL(message.Attachments[i].UUID)
	}

	return r.SendEnvelope(cmodels.ChatMessage{
		UUID:             message.UUID,
		CreatedAt:        message.CreatedAt,
		Content:          message.Content,
		TextContent:      message.TextContent,
		ConversationUUID: message.ConversationUUID,
		Status:           message.Status,
		Author:           message.Author,
		Attachments:      message.Attachments,
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("errors.parsingRequest"), nil, envelope.GeneralError)
	}

	// Get conversation UUID from form data
	conversationValues, convOk := form.Value["conversation_uuid"]
	if !convOk || len(conversationValues) == 0 || conversationValues[0] == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.conversation}"), nil, envelope.InputError)
	}
	conversationUUID := conversationValues[0]

	senderID, conversation, err := getContactConversation(r, conversationUUID)
	if err != nil {
		return err
	}

	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Make sure file upload is enabled for the inbox.
	config, err := parseLiveChatConfig(inbox)
	if err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("validation.invalidInbox"), nil, envelope.GeneralError)
	}

	if !config.Features.FileUpload {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("status.disabledFileUpload"), nil, envelope.InputError)
	}

	files, ok := form.File["files"]
	if !ok || len(files) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundFile"), nil, envelope.InputError)
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error reading uploaded file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingMessage"), nil, envelope.GeneralError)
	}

	// Fetch the inserted message to get the media information.
	insertedMessage, err := app.conversation.GetMessage(message.UUID)
	if err != nil {
		app.lo.Error("error fetching inserted message", "message_uuid", message.UUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	for i := range insertedMessage.Attachments {
		insertedMessage.Attachments[i].URL = app.media.GetSignedURL(insertedMessage.Attachments[i].UUID)
	}

	return r.SendEnvelope(insertedMessage)
}

// getContactConversation gets the contact ID from middleware, fetches the conversation,
// and verifies the conversation belongs to the contact.
func getContactConversation(r *fastglue.Request, conversationUUID string) (int, cmodels.Conversation, error) {
	app := r.Context.(*App)

	contactID, err := getWidgetContactID(r)
	if err != nil {
		app.lo.Error("error getting contact ID from middleware context", "error", err)
		return 0, cmodels.Conversation{}, r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return 0, cmodels.Conversation{}, sendErrorEnvelope(r, err)
	}

	if conversation.ContactID != contactID {
		app.lo.Error("unauthorized access to conversation", "conversation_uuid", conversationUUID, "contact_id", contactID, "conversation_contact_id", conversation.ContactID)
		return 0, cmodels.Conversation{}, r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}

	return contactID, conversation, nil
}

func parseLiveChatConfig(inbox imodels.Inbox) (livechat.Config, error) {
	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err != nil {
		return livechat.Config{}, fmt.Errorf("parsing live chat config: %w", err)
	}
	return config, nil
}

func userTypeLabel(isVisitor bool) string {
	if isVisitor {
		return "visitor"
	}
	return "user"
}

// validateLiveChatInbox validates inbox_id from query params and returns the inbox and parsed config.
// Used by public widget endpoints that don't require JWT authentication.
func validateLiveChatInbox(r *fastglue.Request) (imodels.Inbox, livechat.Config, error) {
	app := r.Context.(*App)
	inboxID := r.RequestCtx.QueryArgs().GetUintOrZero("inbox_id")

	if inboxID <= 0 {
		return imodels.Inbox{}, livechat.Config{}, envelope.NewError(envelope.InputError,
			app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	inbox, err := app.inbox.GetDBRecord(inboxID)
	if err != nil {
		app.lo.Error("error fetching inbox", "inbox_id", inboxID, "error", err)
		return imodels.Inbox{}, livechat.Config{}, envelope.NewError(envelope.GeneralError,
			app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if inbox.Channel != livechat.ChannelLiveChat {
		return imodels.Inbox{}, livechat.Config{}, envelope.NewError(envelope.InputError,
			app.i18n.T("validation.notFoundInbox"), nil)
	}

	if !inbox.Enabled {
		return imodels.Inbox{}, livechat.Config{}, envelope.NewError(envelope.InputError,
			app.i18n.T("status.disabledInbox"), nil)
	}

	config, err := parseLiveChatConfig(inbox)
	if err != nil {
		app.lo.Error("error parsing live chat config", "error", err)
		return imodels.Inbox{}, livechat.Config{}, envelope.NewError(envelope.GeneralError,
			app.i18n.T("validation.invalidInbox"), nil)
	}

	// Check if the client's IP is blocked.
	if len(config.BlockedIPs) > 0 {
		clientIP := realip.FromRequest(r.RequestCtx)
		if httputil.IsIPBlocked(clientIP, config.BlockedIPs) {
			app.lo.Info("client IP blocked for live chat inbox", "client_id", clientIP, "inbox_id", inboxID)
			return imodels.Inbox{}, livechat.Config{}, envelope.NewError(envelope.PermissionError,
				app.i18n.T("widget.ipBlocked"), nil)
		}
	}

	return inbox, config, nil
}

// saveContactAttrsAndCollectConvoAttrs validates and saves contact custom attributes from JWT and form data.
// Returns conversation custom attributes (merged from JWT + form) for the caller to apply after conversation creation.
func saveContactAttrsAndCollectConvoAttrs(app *App, contactID int, claims *Claims, formData map[string]any, config livechat.Config) map[string]any {
	var (
		jwtContactAttrs map[string]any
		jwtConvoAttrs   map[string]any
	)
	formContactAttrs, formConvoAttrs := validateCustomAttributes(formData, config, app)
	if claims != nil {
		jwtContactAttrs = claims.ContactCustomAttributes
		jwtConvoAttrs = claims.ConversationCustomAttributes
	}

	// Save contact custom attributes (JWT takes precedence).
	mergedContactAttrs := mergeCustomAttributes(jwtContactAttrs, formContactAttrs)
	if len(mergedContactAttrs) > 0 {
		if err := app.user.SaveCustomAttributes(contactID, mergedContactAttrs, false); err != nil {
			app.lo.Error("error updating contact custom attributes", "contact_id", contactID, "error", err)
		}
	}

	// Return merged conversation custom attributes (JWT takes precedence).
	return mergeCustomAttributes(jwtConvoAttrs, formConvoAttrs)
}

// resolveOrCreateExternalContact finds or creates a contact from JWT claims with external user ID.
// It tries:
//  1. lookup by external_user_id
//  2. lookup by email and set external_user_id if found
//  3. create new contact
func resolveOrCreateExternalContact(app *App, claims Claims, formData map[string]any, config livechat.Config) (int, map[string]any, error) {
	user, err := app.user.GetByExternalID(claims.ExternalUserID)
	if err == nil {
		convoAttrs := saveContactAttrsAndCollectConvoAttrs(app, user.ID, &claims, formData, config)
		return user.ID, convoAttrs, nil
	}

	envErr, ok := err.(envelope.Error)
	if ok && envErr.ErrorType != envelope.NotFoundError {
		app.lo.Error("error fetching user by external ID", "external_user_id", claims.ExternalUserID, "error", err)
		return 0, nil, err
	}

	// Not found by external_user_id, look up with email and enrich with ext user id.
	if claims.Email != "" {
		existingContact, emailErr := app.user.GetContactByEmail(strings.ToLower(claims.Email))
		if emailErr != nil {
			envErr, ok := emailErr.(envelope.Error)
			if !ok || envErr.ErrorType != envelope.NotFoundError {
				app.lo.Error("error fetching contact by email", "email", claims.Email, "error", emailErr)
				return 0, nil, emailErr
			}
		} else {
			if setErr := app.user.SetExternalUserID(existingContact.ID, claims.ExternalUserID); setErr != nil {
				app.lo.Error("error setting external user ID on existing contact", "contact_id", existingContact.ID, "error", setErr)
				return 0, nil, setErr
			}
			convoAttrs := saveContactAttrsAndCollectConvoAttrs(app, existingContact.ID, &claims, formData, config)
			return existingContact.ID, convoAttrs, nil
		}
	}

	return createExternalUser(app, claims, formData, config)
}

// createExternalUser creates a new contact from JWT claims with external user ID.
// Returns the new contact ID and conversation custom attributes.
func createExternalUser(app *App, claims Claims, formData map[string]any, config livechat.Config) (int, map[string]any, error) {
	formContactAttrs, formConvoAttrs := validateCustomAttributes(formData, config, app)
	mergedContactAttrs := mergeCustomAttributes(claims.ContactCustomAttributes, formContactAttrs)

	user := umodels.User{
		FirstName:        claims.FirstName,
		LastName:         claims.LastName,
		Email:            null.NewString(claims.Email, claims.Email != ""),
		ExternalUserID:   null.NewString(claims.ExternalUserID, claims.ExternalUserID != ""),
		CustomAttributes: marshalCustomAttributes(mergedContactAttrs, app),
	}
	if err := app.user.CreateContact(&user); err != nil {
		app.lo.Error("error creating contact with external ID", "external_user_id", claims.ExternalUserID, "error", err)
		return 0, nil, err
	}

	convoAttrs := mergeCustomAttributes(claims.ConversationCustomAttributes, formConvoAttrs)
	return user.ID, convoAttrs, nil
}

// createVisitorContact creates a new visitor contact from form data.
// Returns the contact ID, a new JWT for the visitor, and conversation custom attributes.
func createVisitorContact(app *App, formData map[string]any, config livechat.Config, inbox imodels.Inbox) (contactID int, jwt string, convoAttrs map[string]any, err error) {
	// Validate form data and get final name/email for new visitor
	finalName, finalEmail, err := validateFormData(formData, config, nil)
	if err != nil {
		return 0, "", nil, err
	}

	// Process custom attributes from form data, split by applies_to.
	formContactAttrs, formConvoAttrs := validateCustomAttributes(formData, config, app)
	convoAttrs = formConvoAttrs

	visitor := umodels.User{
		Email:            null.NewString(finalEmail, finalEmail != ""),
		FirstName:        finalName,
		CustomAttributes: marshalCustomAttributes(formContactAttrs, app),
	}

	if err := app.user.CreateVisitor(&visitor); err != nil {
		app.lo.Error("error creating visitor contact", "error", err)
		return 0, "", nil, err
	}

	newJWT, err := generateUserJWTWithSecret(visitor.ID, true, time.Now().Add(87600*time.Hour), []byte(inbox.Secret.String)) // 10 years
	if err != nil {
		app.lo.Error("error generating visitor JWT", "error", err)
		return 0, "", nil, err
	}

	return visitor.ID, newJWT, convoAttrs, nil
}

// checkConversationPermissions checks if the user is allowed to start a conversation based on inbox config.
// Returns an error if the user is not allowed to start a conversation.
func checkConversationPermissions(app *App, config livechat.Config, isVisitor bool, contactID, inboxID int) error {
	var allowStartConversation, preventMultipleConversations bool
	if isVisitor {
		allowStartConversation = config.Visitors.AllowStartConversation
		preventMultipleConversations = config.Visitors.PreventMultipleConversations
	} else {
		allowStartConversation = config.Users.AllowStartConversation
		preventMultipleConversations = config.Users.PreventMultipleConversations
	}

	if !allowStartConversation {
		return envelope.NewError(envelope.InputError, "Not allowed.", nil)
	}

	if preventMultipleConversations {
		conversations, err := app.conversation.GetContactChatConversations(contactID, inboxID)
		if err != nil {
			app.lo.Error("error fetching "+userTypeLabel(isVisitor)+" conversations", "contact_id", contactID, "error", err)
			return envelope.NewError(envelope.GeneralError, "Error checking existing conversations", nil)
		}
		if len(conversations) > 0 {
			app.lo.Info(userTypeLabel(isVisitor)+" attempted to start new conversation but already has one", "contact_id", contactID, "conversations_count", len(conversations))
			return envelope.NewError(envelope.PermissionError, "Multiple conversations are not allowed", nil)
		}
	}

	return nil
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
	var (
		user umodels.User
		err  error
	)

	switch {
	case claims.UserID > 0:
		user, err = app.user.Get(claims.UserID, "", []string{})
	case claims.ExternalUserID != "":
		user, err = app.user.GetByExternalID(claims.ExternalUserID)
	default:
		return 0, errors.New("error fetching user")
	}

	if err != nil {
		app.lo.Error("error fetching user", "user_id", claims.UserID, "external_user_id", claims.ExternalUserID, "error", err)
		return 0, errors.New("error fetching user")
	}
	if !user.Enabled {
		return 0, errors.New("user is disabled")
	}
	return user.ID, nil
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

// marshalCustomAttributes marshals custom attributes to JSON, returning "{}" on error or empty input.
func marshalCustomAttributes(attrs map[string]any, app *App) []byte {
	if len(attrs) == 0 {
		return []byte("{}")
	}
	b, err := json.Marshal(attrs)
	if err != nil {
		app.lo.Error("error marshalling custom attributes", "error", err)
		return []byte("{}")
	}
	return b
}

// mergeCustomAttributes merges JWT and form custom attributes.
// JWT attributes take precedence as they are server-signed and trusted.
func mergeCustomAttributes(jwtAttributes, formAttributes map[string]any) map[string]any {
	merged := make(map[string]any)
	maps.Copy(merged, formAttributes)
	maps.Copy(merged, jwtAttributes)
	return merged
}

// validateCustomAttributes validates pre chat form data and splits into contact and conversation attributes based on applies_to.
func validateCustomAttributes(formData map[string]any, config livechat.Config, app *App) (contactAttrs, conversationAttrs map[string]any) {
	contactAttrs = make(map[string]any)
	conversationAttrs = make(map[string]any)

	if !config.PreChatForm.Enabled || len(formData) == 0 {
		return contactAttrs, conversationAttrs
	}

	// Validate total number of form fields
	const maxFormFields = 100
	if len(formData) > maxFormFields {
		app.lo.Warn("form data exceeds maximum allowed fields", "received", len(formData), "max", maxFormFields)
		return contactAttrs, conversationAttrs
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

		// Validate value
		validated := validateAttributeValue(key, value, app)
		if validated == nil {
			continue
		}

		// Look up the custom attribute definition to determine applies_to
		attr, err := app.customAttribute.Get(field.CustomAttributeID)
		if err != nil {
			app.lo.Warn("custom attribute not found", "custom_attribute_id", field.CustomAttributeID, "error", err)
			continue
		}

		if attr.AppliesTo == "conversation" {
			conversationAttrs[field.Key] = validated
		} else {
			contactAttrs[field.Key] = validated
		}
	}

	return contactAttrs, conversationAttrs
}

// validateAttributeValue validates and sanitizes a single attribute value.
func validateAttributeValue(key string, value any, app *App) any {
	if strValue, ok := value.(string); ok {
		const maxValueLength = 1000
		if len(strValue) > maxValueLength {
			app.lo.Warn("form field value exceeds maximum length", "key", key, "length", len(strValue), "max", maxValueLength)
			return strValue[:maxValueLength]
		}
		return strValue
	}

	if numValue, ok := value.(float64); ok {
		if math.IsNaN(numValue) || math.IsInf(numValue, 0) {
			app.lo.Warn("form field contains invalid numeric value", "key", key, "value", numValue)
			return nil
		}
		if numValue > 1e12 || numValue < -1e12 {
			app.lo.Warn("form field numeric value out of acceptable range", "key", key, "value", numValue)
			return nil
		}
		return numValue
	}

	if boolValue, ok := value.(bool); ok {
		return boolValue
	}

	// Reject all other types (arrays, objects, etc.) to prevent arbitrary data in JSONB.
	app.lo.Warn("form field contains unsupported value type", "key", key)
	return nil
}

// validateFormData validates form data against pre-chat form configuration
// Returns the final name/email to use and any validation errors
func validateFormData(formData map[string]any, config livechat.Config, existingUser *umodels.User) (string, string, error) {
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
