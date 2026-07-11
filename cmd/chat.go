package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	realip "github.com/ferluci/fast-realip"
	"github.com/golang-jwt/jwt/v5"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	maxChatConversationsPerContact  = 50
	chatConversationRateLimitWindow = 24 * time.Hour
	widgetSessionPrefix             = "widget_session:"
	defaultSessionTTL               = 180 * 24 * time.Hour
	minSessionTTL                   = 1 * time.Hour
	maxChatMessageLength            = 10000
	maxEmailLength                  = 254
	maxNameLength                   = 128
	maxExternalUserIDLength         = 128
	maxPhoneNumberLength            = 20
	maxPhoneCountryCodeLength       = 10
	fieldTypePhone                  = "phone"
	phoneCountryCodeSuffix          = "_country_code"
)

// WidgetSession holds session data stored in Redis.
type WidgetSession struct {
	UserID         int
	InboxID        int
	IsVisitor      bool
	ExternalUserID string
}

// Claims holds JWT claims for a JWT user.
type Claims struct {
	UserID                  int            `json:"user_id,omitempty"`
	ExternalUserID          string         `json:"external_user_id,omitempty"`
	IsVisitor               bool           `json:"is_visitor,omitempty"`
	Email                   string         `json:"email,omitempty"`
	FirstName               string         `json:"first_name,omitempty"`
	LastName                string         `json:"last_name,omitempty"`
	PhoneNumber             string         `json:"phone_number,omitempty"`
	PhoneNumberCountryCode  string         `json:"phone_number_country_code,omitempty"`
	ContactCustomAttributes map[string]any `json:"contact_custom_attributes,omitempty"`
	jwt.RegisteredClaims
}

type conversationResp struct {
	Conversation cmodels.ChatConversation `json:"conversation"`
	Messages     []cmodels.ChatMessage    `json:"messages"`
}

type customAttributeWidget struct {
	ID       int      `json:"id"`
	Values   []string `json:"values"`
	Name     string   `json:"-"`
	DataType string   `json:"-"`
}

type chatInitReq struct {
	Message  string         `json:"message"`
	FormData map[string]any `json:"form_data"`
}

type chatSettingsResponse struct {
	livechat.Config
	// Hide server-side fields from the public widget response.
	TrustedDomains         *struct{}                     `json:"trusted_domains,omitempty"`
	BlockedIPs             *struct{}                     `json:"blocked_ips,omitempty"`
	Continuity             *struct{}                     `json:"continuity,omitempty"`
	SessionDuration        *struct{}                     `json:"session_duration,omitempty"`
	BusinessHours          []bhmodels.BusinessHours      `json:"business_hours,omitempty"`
	DefaultBusinessHoursID int                           `json:"default_business_hours_id,omitempty"`
	WorkingHoursUTCOffset  *int                          `json:"working_hours_utc_offset,omitempty"`
	CustomAttributes       map[int]customAttributeWidget `json:"custom_attributes,omitempty"`
}

// conversationResponseWithBusinessHours includes business hours info for the widget
type conversationResponseWithBusinessHours struct {
	conversationResp
	BusinessHoursID       *int `json:"business_hours_id,omitempty"`
	WorkingHoursUTCOffset *int `json:"working_hours_utc_offset,omitempty"`
}

// handleGetChatLauncherSettings returns the live chat launcher settings for the widget.
func handleGetChatLauncherSettings(r *fastglue.Request) error {
	r.RequestCtx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	config, err := getWidgetConfig(r)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, err.Error(), nil))
	}

	return r.SendEnvelope(map[string]any{
		"launcher": config.Launcher,
		"colors":   config.Colors,
	})
}

// handleGetChatSettings returns the live chat settings for the widget
func handleGetChatSettings(r *fastglue.Request) error {
	app := r.Context.(*App)

	config, err := getWidgetConfig(r)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, err.Error(), nil))
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

		// Get default business hours ID and UTC offset from general settings.
		out, err := app.setting.GetByPrefix("app")
		if err != nil {
			app.lo.Error("error fetching general settings", "error", err)
		} else {
			var settings map[string]any
			if err := json.Unmarshal(out, &settings); err == nil {
				if bhID, ok := settings["app.business_hours_id"].(string); ok {
					response.DefaultBusinessHoursID, _ = strconv.Atoi(bhID)
				}
				if tz, ok := settings["app.timezone"].(string); ok && tz != "" {
					if loc, err := time.LoadLocation(tz); err == nil {
						_, offset := time.Now().In(loc).Zone()
						offsetMinutes := offset / 60
						response.WorkingHoursUTCOffset = &offsetMinutes
					}
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
		newSessionToken   string
		conversationAttrs map[string]any
		visitor           umodels.User
	)

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling chat init request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	if req.Message == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.message}"), nil, envelope.InputError)
	}
	if len(req.Message) > maxChatMessageLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.maxLength", "max", strconv.Itoa(maxChatMessageLength)), nil, envelope.InputError)
	}

	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	config, err := getWidgetConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Check if user is already authenticated (has session token).
	contactID, _ = getWidgetContactID(r)
	if contactID > 0 {
		// Returning user (visitor or contact) with session token.
		// Custom attributes from JWT were already saved during /auth/exchange.
		// Only process form-level attributes here.
		isVisitor = getWidgetIsVisitor(r)
		conversationAttrs = saveContactAttrsAndCollectConvoAttrs(app, contactID, nil, req.FormData, config)
	} else {
		// New visitor - create visitor and session token.
		isVisitor = true
		visitor, newSessionToken, conversationAttrs, err = createVisitorContact(app, req.FormData, config, inbox)
		if err != nil {
			app.lo.Error("error creating visitor contact", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
		}
		contactID = visitor.ID
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

	// Add session token and user metadata when a new visitor is created.
	if newSessionToken != "" {
		response["session_token"] = newSessionToken
		response["user"] = map[string]any{
			"user_id":    contactID,
			"is_visitor": isVisitor,
			"first_name": visitor.FirstName,
			"last_name":  visitor.LastName,
		}
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

	_, conversation, err := getContactConversation(r, conversationUUID)
	if err != nil {
		return err
	}

	if err := app.conversation.UpdateConversationContactLastSeen(conversation.UUID); err != nil {
		app.lo.Error("error updating contact last seen timestamp", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleAuthExchange exchanges a JWT for a session token.
// Used by the setUser() flow for verified contacts.
func handleAuthExchange(r *fastglue.Request) error {
	app := r.Context.(*App)

	inbox, err := getWidgetInbox(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	config, err := getWidgetConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	var req struct {
		JWT string `json:"jwt"`
	}
	if err := r.Decode(&req, "json"); err != nil || req.JWT == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "jwt"), nil, envelope.InputError)
	}

	// Verify the customer-generated JWT.
	claims, err := verifyStandardJWT(req.JWT, inbox.Secret.String)
	if err != nil {
		app.lo.Error("invalid JWT in auth exchange", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	if claims.ExternalUserID == "" || len(claims.ExternalUserID) > maxExternalUserIDLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "external_user_id"), nil, envelope.InputError)
	}
	if claims.Email == "" || len(claims.Email) > maxEmailLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "email"), nil, envelope.InputError)
	}
	if claims.FirstName == "" || len(claims.FirstName) > maxNameLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "first_name"), nil, envelope.InputError)
	}
	if len(claims.LastName) > maxNameLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.maxLength", "max", strconv.Itoa(maxNameLength)), nil, envelope.InputError)
	}
	if len(claims.PhoneNumber) > maxPhoneNumberLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.maxLength", "max", strconv.Itoa(maxPhoneNumberLength)), nil, envelope.InputError)
	}
	// Country code is cosmetic - drop an invalid one instead of failing the whole exchange.
	if len(claims.PhoneNumberCountryCode) > maxPhoneCountryCodeLength {
		claims.PhoneNumberCountryCode = ""
	}

	// Resolve or create the contact.
	contactID, err := resolveOrCreateExternalContact(app, claims)
	if err != nil {
		app.lo.Error("error resolving contact during auth exchange", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Save custom attributes from JWT.
	if len(claims.ContactCustomAttributes) > 0 {
		if err := app.user.SaveCustomAttributes(contactID, claims.ContactCustomAttributes, false); err != nil {
			app.lo.Error("error saving custom attributes during auth exchange", "contact_id", contactID, "error", err)
		}
	}

	ctx := context.Background()
	reverseKey := fmt.Sprintf("widget_user:%d:%d", inbox.ID, contactID)
	sessionTTL := getSessionDuration(config)

	sendSession := func(token string) error {
		app.redis.Set(ctx, reverseKey, token, sessionTTL)
		return r.SendEnvelope(map[string]any{
			"session_token": token,
			"user": map[string]any{
				"user_id":    contactID,
				"is_visitor": false,
				"first_name": claims.FirstName,
				"last_name":  claims.LastName,
			},
		})
	}

	// Reuse existing valid session and refresh its TTL.
	if oldToken, err := app.redis.Get(ctx, reverseKey).Result(); err == nil && oldToken != "" {
		if _, err := loadSession(app, oldToken, config); err == nil {
			return sendSession(oldToken)
		}
	}

	// Generate new session token.
	token, err := generateSessionToken(app, contactID, inbox.ID, false, claims.ExternalUserID, sessionTTL)
	if err != nil {
		app.lo.Error("error generating session token", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	return sendSession(token)
}

// handleWidgetAuthMe returns the current authenticated user's metadata.
func handleWidgetAuthMe(r *fastglue.Request) error {
	app := r.Context.(*App)

	contactID, err := getWidgetContactID(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	u, err := app.user.Get(contactID, "", []string{umodels.UserTypeContact, umodels.UserTypeVisitor})
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]any{
		"user_id":    u.ID,
		"is_visitor": u.Type == umodels.UserTypeVisitor,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
	})
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
	app := r.Context.(*App)

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
	if len(req.Message) > maxChatMessageLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.maxLength", "max", strconv.Itoa(maxChatMessageLength)), nil, envelope.InputError)
	}

	senderID, conversation, err := getContactConversation(r, conversationUUID)
	if err != nil {
		return err
	}

	if err := canReply(r, conversation); err != nil {
		return sendErrorEnvelope(r, err)
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

	return sendChatMessageResponse(app, r, message.UUID)
}

// handleWidgetMediaUpload handles media uploads for the widget.
func handleWidgetMediaUpload(r *fastglue.Request) error {
	app := r.Context.(*App)

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

	if err := canReply(r, conversation); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Make sure file upload is enabled for the inbox.
	config, err := getWidgetConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
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

	if srcFileSize <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("media.fileEmpty"), nil, envelope.InputError)
	}

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

	fileContent, err := io.ReadAll(file)
	if err != nil {
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

	return sendChatMessageResponse(app, r, message.UUID)
}

// sendChatMessageResponse fetches an inserted message by UUID, signs attachment and
// avatar URLs, and sends the formatted ChatMessage response.
func sendChatMessageResponse(app *App, r *fastglue.Request, messageUUID string) error {
	message, err := app.conversation.GetMessage(messageUUID)
	if err != nil {
		app.lo.Error("error fetching inserted message", "message_uuid", messageUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	for i := range message.Attachments {
		message.Attachments[i].URL = app.media.GetSignedURL(message.Attachments[i].UUID)
	}
	app.conversation.SignAvatarURL(&message.Author.AvatarURL)

	// Strip agent email from widget responses.
	author := message.Author
	author.Email = null.String{}

	return r.SendEnvelope(cmodels.ChatMessage{
		UUID:             message.UUID,
		CreatedAt:        message.CreatedAt,
		Content:          message.Content,
		TextContent:      message.TextContent,
		ConversationUUID: message.ConversationUUID,
		Status:           message.Status,
		Author:           author,
		Attachments:      message.Attachments,
	})
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

	inbox, err := getWidgetInbox(r)
	if err != nil {
		app.lo.Error("error getting inbox from middleware context", "error", err)
		return 0, cmodels.Conversation{}, r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return 0, cmodels.Conversation{}, sendErrorEnvelope(r, err)
	}

	if conversation.ContactID != contactID || conversation.InboxID != inbox.ID {
		app.lo.Error("unauthorized access to conversation", "conversation_uuid", conversationUUID, "contact_id", contactID, "conversation_contact_id", conversation.ContactID, "session_inbox_id", inbox.ID, "conversation_inbox_id", conversation.InboxID)
		return 0, cmodels.Conversation{}, r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}

	return contactID, conversation, nil
}

func userTypeLabel(isVisitor bool) string {
	if isVisitor {
		return "visitor"
	}
	return "user"
}

// saveContactAttrsAndCollectConvoAttrs validates and saves contact custom attributes from JWT and form data.
// Returns conversation custom attributes from the pre-chat form for the caller to apply after conversation creation.
func saveContactAttrsAndCollectConvoAttrs(app *App, contactID int, claims *Claims, formData map[string]any, config livechat.Config) map[string]any {
	formContactAttrs, formConvoAttrs := validateCustomAttributes(formData, config, app)

	// Merge JWT contact attributes with form contact attributes (JWT takes precedence).
	var jwtContactAttrs map[string]any
	if claims != nil {
		jwtContactAttrs = claims.ContactCustomAttributes
	}
	mergedContactAttrs := mergeCustomAttributes(jwtContactAttrs, formContactAttrs)
	if len(mergedContactAttrs) > 0 {
		if err := app.user.SaveCustomAttributes(contactID, mergedContactAttrs, false); err != nil {
			app.lo.Error("error updating contact custom attributes", "contact_id", contactID, "error", err)
		}
	}

	return formConvoAttrs
}

// resolveOrCreateExternalContact finds a contact by external_user_id (syncing changed JWT fields) or creates one.
func resolveOrCreateExternalContact(app *App, claims Claims) (int, error) {
	user, err := resolveUserFromClaims(app, claims)
	if err != nil {
		if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
			return 0, err
		}
	}

	// Sync name/email/phone from JWT only if changed.
	if user.ID > 0 && claims.ExternalUserID != "" {
		if user.FirstName != claims.FirstName || user.LastName != claims.LastName || user.Email.String != claims.Email ||
			user.PhoneNumber.String != claims.PhoneNumber || user.PhoneNumberCountryCode.String != claims.PhoneNumberCountryCode {
			if err := app.user.UpdateContactBasicInfo(user.ID, claims.FirstName, claims.LastName, claims.Email, claims.PhoneNumber, claims.PhoneNumberCountryCode); err != nil {
				app.lo.Error("error updating contact basic info", "contact_id", user.ID, "error", err)
			}
		}
		return user.ID, nil
	}

	// Create contact if not found.
	if claims.ExternalUserID != "" {
		user := umodels.User{
			FirstName:              claims.FirstName,
			LastName:               claims.LastName,
			Email:                  null.NewString(claims.Email, claims.Email != ""),
			PhoneNumber:            null.NewString(claims.PhoneNumber, claims.PhoneNumber != ""),
			PhoneNumberCountryCode: null.NewString(claims.PhoneNumberCountryCode, claims.PhoneNumberCountryCode != ""),
			ExternalUserID:         null.NewString(claims.ExternalUserID, true),
			CustomAttributes:       marshalCustomAttributes(claims.ContactCustomAttributes, app),
		}
		if err := app.user.CreateContact(&user); err != nil {
			return 0, err
		}
		return user.ID, nil
	}

	return user.ID, nil
}

// createVisitorContact creates a new visitor contact from form data.
func createVisitorContact(app *App, formData map[string]any, config livechat.Config, inbox imodels.Inbox) (umodels.User, string, map[string]any, error) {
	// Validate form data and get final name/email/phone for new visitor.
	finalName, finalEmail, finalPhone, finalPhoneCountryCode, err := validateFormData(formData, config, nil)
	if err != nil {
		return umodels.User{}, "", nil, err
	}

	// Process custom attributes from form data, split by applies_to.
	formContactAttrs, formConvoAttrs := validateCustomAttributes(formData, config, app)

	visitor := umodels.User{
		Email:                  null.NewString(finalEmail, finalEmail != ""),
		FirstName:              finalName,
		PhoneNumber:            null.NewString(finalPhone, finalPhone != ""),
		PhoneNumberCountryCode: null.NewString(finalPhoneCountryCode, finalPhoneCountryCode != ""),
		CustomAttributes:       marshalCustomAttributes(formContactAttrs, app),
	}

	if err := app.user.CreateVisitor(&visitor); err != nil {
		app.lo.Error("error creating visitor contact", "error", err)
		return umodels.User{}, "", nil, err
	}

	token, err := generateSessionToken(app, visitor.ID, inbox.ID, true, "", defaultSessionTTL)
	if err != nil {
		app.lo.Error("error generating session token for visitor", "error", err)
		return umodels.User{}, "", nil, err
	}

	return visitor, token, formConvoAttrs, nil
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

// resolveUserFromClaims resolves the actual user from JWT claims,
// handling both regular user_id and external_user_id cases.
func resolveUserFromClaims(app *App, claims Claims) (umodels.User, error) {
	var (
		user umodels.User
		err  error
	)

	switch {
	case claims.UserID > 0:
		user, err = app.user.Get(claims.UserID, "", []string{umodels.UserTypeContact, umodels.UserTypeVisitor})
	case claims.ExternalUserID != "":
		user, err = app.user.GetByExternalID(claims.ExternalUserID)
	default:
		return umodels.User{}, errors.New("error fetching user")
	}

	if err != nil {
		app.lo.Error("error fetching user", "user_id", claims.UserID, "external_user_id", claims.ExternalUserID, "error", err)
		return umodels.User{}, err
	}
	if !user.Enabled {
		return umodels.User{}, envelope.NewError(envelope.PermissionError, "User is disabled", nil)
	}
	return user, nil
}

// verifyJWT verifies and validates a JWT token with proper signature verification.
func verifyJWT(tokenString string, secretKey []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secretKey, nil
	}, jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}

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

// generateSessionToken creates a random session token and stores it in Redis.
func generateSessionToken(app *App, userID, inboxID int, isVisitor bool, externalUserID string, ttl time.Duration) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	key := widgetSessionPrefix + token

	ctx := context.Background()
	fields := map[string]any{
		"user_id":          strconv.Itoa(userID),
		"inbox_id":         strconv.Itoa(inboxID),
		"is_visitor":       strconv.FormatBool(isVisitor),
		"external_user_id": externalUserID,
	}
	pipe := app.redis.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("storing session in Redis: %w", err)
	}
	return token, nil
}

// loadSession retrieves a session from Redis and refreshes its TTL using the
// default duration for visitors and the current config duration for contacts.
func loadSession(app *App, token string, config livechat.Config) (*WidgetSession, error) {
	ctx := context.Background()
	key := widgetSessionPrefix + token

	data, err := app.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("looking up session: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("session not found or expired")
	}

	userID, _ := strconv.Atoi(data["user_id"])
	inboxID, _ := strconv.Atoi(data["inbox_id"])
	isVisitor, _ := strconv.ParseBool(data["is_visitor"])

	ttl := getSessionDuration(config)
	if isVisitor {
		ttl = defaultSessionTTL
	}
	app.redis.Expire(ctx, key, ttl)

	return &WidgetSession{
		UserID:         userID,
		InboxID:        inboxID,
		IsVisitor:      isVisitor,
		ExternalUserID: data["external_user_id"],
	}, nil
}

// deleteSessionToken removes a session from Redis.
func deleteSessionToken(app *App, token string) {
	app.redis.Del(context.Background(), widgetSessionPrefix+token)
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
	phoneCompanionKeys := make(map[string]bool)
	for _, field := range config.PreChatForm.Fields {
		if field.Enabled {
			validFields[field.Key] = field
			if field.Type == fieldTypePhone {
				phoneCompanionKeys[field.Key+phoneCountryCodeSuffix] = true
			}
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

		if phoneCompanionKeys[key] {
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

// validateFormData returns the final name/email/phone/phone country code to persist from the pre-chat form.
func validateFormData(formData map[string]any, config livechat.Config, existingUser *umodels.User) (string, string, string, string, error) {
	var finalName, finalEmail, finalPhone, finalPhoneCountryCode string

	if !config.PreChatForm.Enabled {
		return finalName, finalEmail, finalPhone, finalPhoneCountryCode, nil
	}

	var exName, exEmail, exPhone, exPhoneCountryCode string
	if existingUser != nil {
		exName = existingUser.FirstName
		exEmail = existingUser.Email.String
		exPhone = existingUser.PhoneNumber.String
		exPhoneCountryCode = existingUser.PhoneNumberCountryCode.String
	}

	for _, field := range config.PreChatForm.Fields {
		if !field.Enabled || !field.IsDefault {
			continue
		}

		switch field.Key {
		case "name":
			finalName = resolveFormField(formData, field.Key, exName)
			if field.Required && finalName == "" {
				return "", "", "", "", fmt.Errorf("name is required")
			}
			if len(finalName) > maxNameLength {
				return "", "", "", "", fmt.Errorf("name too long")
			}

		case "email":
			finalEmail = resolveFormField(formData, field.Key, exEmail)
			if field.Required && finalEmail == "" {
				return "", "", "", "", fmt.Errorf("email is required")
			}
			if len(finalEmail) > maxEmailLength {
				return "", "", "", "", fmt.Errorf("email too long")
			}
			if finalEmail != "" && !stringutil.ValidEmail(finalEmail) {
				return "", "", "", "", fmt.Errorf("invalid email format")
			}

		case fieldTypePhone:
			finalPhone = resolveFormField(formData, field.Key, exPhone)
			finalPhoneCountryCode = resolveFormField(formData, field.Key+phoneCountryCodeSuffix, exPhoneCountryCode)
			if field.Required && (finalPhone == "" || finalPhoneCountryCode == "") {
				return "", "", "", "", fmt.Errorf("phone is required")
			}
			if len(finalPhone) > maxPhoneNumberLength {
				return "", "", "", "", fmt.Errorf("phone too long")
			}
			if len(finalPhoneCountryCode) > maxPhoneCountryCodeLength {
				return "", "", "", "", fmt.Errorf("phone country code too long")
			}
			if finalPhone == "" {
				finalPhoneCountryCode = ""
			}
		}
	}

	return finalName, finalEmail, finalPhone, finalPhoneCountryCode, nil
}

func resolveFormField(formData map[string]any, key, existing string) string {
	if existing != "" {
		return existing
	}
	if value, ok := formData[key].(string); ok {
		return value
	}
	return ""
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
			ID:       attr.ID,
			Values:   attr.Values,
			Name:     attr.Name,
			DataType: attr.DataType,
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
		if attr, exists := existingCustomAttrs[field.CustomAttributeID]; exists {
			// Sync label and type from the current custom attribute definition.
			field.Label = attr.Name
			field.Type = attr.DataType
			filteredFields = append(filteredFields, field)
		}
	}

	return filteredFields, existingCustomAttrs
}

// getSessionDuration returns the configured session TTL for authenticated users.
// Falls back to defaultSessionTTL if the config value is empty or invalid.
// Enforces a minimum of 1 hour.
func getSessionDuration(config livechat.Config) time.Duration {
	if config.SessionDuration == "" {
		return defaultSessionTTL
	}
	d, err := time.ParseDuration(config.SessionDuration)
	if err != nil || d < minSessionTTL {
		return defaultSessionTTL
	}
	return d
}

// canReply checks if the conversation is closed and if the sender is allowed to reply based on inbox settings.
func canReply(r *fastglue.Request, conversation cmodels.Conversation) error {
	if conversation.Status.String != cmodels.StatusClosed {
		return nil
	}
	app := r.Context.(*App)
	lcConfig, err := getWidgetConfig(r)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	preventReply := lcConfig.Visitors.PreventReplyToClosedConversation
	if !getWidgetIsVisitor(r) {
		preventReply = lcConfig.Users.PreventReplyToClosedConversation
	}
	if preventReply {
		return envelope.NewError(envelope.PermissionError, app.i18n.T("widget.conversationClosed"), nil)
	}
	return nil
}
