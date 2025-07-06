package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	// TODO: Can have a global route that serves media files with a signature and expiry.
	// Or use the same existing `/uploads`
	chatWidgetMediaURL = "/api/v1/widget/media/%s?signature=%s&expires=%d"
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
	Conversation models.ChatConversation `json:"conversation"`
	Messages     []models.ChatMessage    `json:"messages"`
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

	var contactID int
	var conversationUUID string
	var isVisitor bool

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
		// Clean up conversation if message insert fails.
		if err := app.conversation.DeleteConversation(conversationUUID); err != nil {
			app.lo.Error("error deleting conversation after message insert failure", "conversation_uuid", conversationUUID, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
		}
		app.lo.Error("error inserting initial message", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching created conversation", "conversation_uuid", conversationUUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil, envelope.GeneralError)
	}

	// Build response with conversation and messages.
	resp, err := buildConversationResponse(app, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]any{
		"conversation": resp.Conversation,
		"messages":     resp.Messages,
		"jwt":          req.JWT,
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
	resp, err := buildConversationResponse(app, conversation)
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
		senderType       = models.SenderTypeContact
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

	// Insert message.
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(map[string]bool{"success": true})
}

// handleWidgetMediaUpload handles media uploads for the widget.
func handleWidgetMediaUpload(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		cleanUp  = false
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

	// Verify conversation exists and user has access
	conversation, err := app.conversation.GetConversation(0, conversationUUID)
	if err != nil {
		app.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Make sure the conversation belongs to the sender
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

	// Delete files on any error.
	var uuid = uuid.New()
	thumbName := thumbPrefix + uuid.String()
	defer func() {
		if cleanUp {
			app.media.Delete(uuid.String())
			app.media.Delete(thumbName)
		}
	}()

	// Generate and upload thumbnail and store image dimensions in the media meta.
	var meta = []byte("{}")
	if slices.Contains(image.Exts, srcExt) {
		file.Seek(0, 0)
		thumbFile, err := image.CreateThumb(image.DefThumbSize, file)
		if err != nil {
			app.lo.Error("error creating thumb image", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.thumbnail}"), nil, envelope.GeneralError)
		}
		thumbName, err = app.media.Upload(thumbName, srcContentType, thumbFile)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		// Store image dimensions in media meta, storing dimensions for image previews in future.
		file.Seek(0, 0)
		width, height, err := image.GetDimensions(file)
		if err != nil {
			cleanUp = true
			app.lo.Error("error getting image dimensions", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorUploading", "name", "{globals.terms.media}"), nil, envelope.GeneralError)
		}
		meta, _ = json.Marshal(map[string]any{
			"width":  width,
			"height": height,
		})
	}

	file.Seek(0, 0)
	_, err = app.media.Upload(uuid.String(), srcContentType, file)
	if err != nil {
		cleanUp = true
		app.lo.Error("error uploading file", "error", err)
		return sendErrorEnvelope(r, err)
	}

	// Insert message with empty content and after insert link the media to the message.
	message := models.Message{
		ConversationUUID: conversationUUID,
		SenderID:         senderID,
		Type:             models.MessageIncoming,
		SenderType:       models.SenderTypeContact,
		Status:           models.MessageStatusReceived,
		Content:          "",
		ContentType:      models.ContentTypeText,
		Private:          false,
	}
	if err := app.conversation.InsertMessage(&message); err != nil {
		cleanUp = true
		app.lo.Error("error inserting message", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil, envelope.GeneralError)
	}

	// Insert media linked to the just inserted message.
	media, err := app.media.Insert(null.StringFrom(attachment.DispositionAttachment), srcFileName, srcContentType, "" /**content_id**/, null.NewString("messages", true), uuid.String(), null.NewInt(message.ID, true), int(srcFileSize), meta)
	if err != nil {
		cleanUp = true
		app.lo.Error("error inserting metadata into database", "error", err)
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(media)
}

// handleWidgetServeMedia serves media files for the widget
func handleWidgetServeMedia(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		uuid       = r.RequestCtx.UserValue("uuid").(string)
		signature  = string(r.RequestCtx.QueryArgs().Peek("signature"))
		expiresStr = string(r.RequestCtx.QueryArgs().Peek("expires"))
	)

	if uuid == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "uuid"), nil, envelope.InputError)
	}

	if signature == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Signature missing", nil, envelope.InputError)
	}

	if expiresStr == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Expiry missing", nil, envelope.InputError)
	}

	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid expiry", nil, envelope.InputError)
	}

	// Verify signature and expiration.
	expiresAt := time.Unix(expires, 0)
	if !VerifySignedURL(uuid, signature, expiresAt, getJWTSecret()) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	// Get media DB record.
	_, err = app.media.Get(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	consts := app.consts.Load().(*constants)
	switch consts.UploadProvider {
	case "fs":
		fasthttp.ServeFile(r.RequestCtx, filepath.Join(ko.String("upload.fs.upload_path"), uuid))
	case "s3":
		r.RequestCtx.Redirect(app.media.GetURL(uuid), http.StatusFound)
	}
	return nil
}

// buildConversationResponse builds the response for a conversation including its messages
func buildConversationResponse(app *App, conversation models.Conversation) (conversationResp, error) {
	var resp = conversationResp{}

	// Fetch last 1000 messages, this should suffice as chats shouldn't have too many messages.
	private := false
	messages, _, err := app.conversation.GetConversationMessages(conversation.UUID, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing}, &private, 1, 1000)
	if err != nil {
		app.lo.Error("error fetching conversation messages", "conversation_uuid", conversation.UUID, "error", err)
		return resp, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.message}"), nil)
	}

	// Convert to chat message format, Generate signed widget URL for all attachments - expires in 1 hour.
	chatMessages := make([]models.ChatMessage, len(messages))
	for i, msg := range messages {
		attachments := msg.Attachments
		for j := range attachments {
			expiresAt := time.Now().Add(1 * time.Hour)
			signature := GenerateSignedURL(attachments[j].UUID, expiresAt, getJWTSecret())
			attachments[j].URL = fmt.Sprintf(chatWidgetMediaURL, attachments[j].UUID, signature, expiresAt.Unix())
		}
		chatMessages[i] = models.ChatMessage{
			UUID:           msg.UUID,
			Content:        msg.Content,
			CreatedAt:      msg.CreatedAt,
			SenderType:     msg.SenderType,
			ConversationID: msg.ConversationUUID,
			Attachments:    attachments,
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
		// TODO: Instead of this hardcoded URL, make it a central handler.
		if assignee.AvatarURL.Valid && assignee.AvatarURL.String != "" {
			avatarPath := assignee.AvatarURL.String
			if strings.HasPrefix(avatarPath, "/uploads/") {
				avatarUUID := strings.TrimPrefix(avatarPath, "/uploads/")
				// Generate signed URL for avatar with 1 hour expiry
				expiresAt := time.Now().Add(1 * time.Hour)
				signature := GenerateSignedURL(avatarUUID, expiresAt, getJWTSecret())
				assignee.AvatarURL = null.StringFrom(fmt.Sprintf(chatWidgetMediaURL, avatarUUID, signature, expiresAt.Unix()))
			}
		}
	}

	resp = conversationResp{
		Conversation: models.ChatConversation{
			UUID:     conversation.UUID,
			Status:   conversation.Status.String,
			Assignee: assignee,
		},
		Messages: chatMessages,
	}

	return resp, nil
}

// GenerateSignedURL generates a signature for media access with expiration
func GenerateSignedURL(uuid string, expiresAt time.Time, secret []byte) string {
	exp := expiresAt.Unix()
	payload := fmt.Sprintf("%s:%d", uuid, exp)
	sig := hmacSha256(payload, secret)
	return sig
}

// VerifySignedURL verifies a signed URL with expiration
func VerifySignedURL(uuid, signature string, expiresAt time.Time, secret []byte) bool {
	// Check if expired
	if time.Now().After(expiresAt) {
		return false
	}

	// Generate expected signature
	expectedSignature := GenerateSignedURL(uuid, expiresAt, secret)

	// Compare signatures using constant time comparison to prevent timing attacks
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// hmacSha256 generates HMAC-SHA256 hash
func hmacSha256(data string, secret []byte) string {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
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
