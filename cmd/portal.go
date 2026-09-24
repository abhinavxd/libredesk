package main

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	livechat "github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	portalExternalIDKey = "portal_external_user_id"
	portalContactKey    = "portal_contact"
	portalMaxFiles      = 10
	portalMessageMax    = 10000
	portalSubjectMax    = 255
)

type portalContactResponse struct {
	ID             int    `json:"id"`
	ExternalUserID string `json:"external_user_id"`
	Email          string `json:"email,omitempty"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name,omitempty"`
}

type portalConversationResponse struct {
	ID                 int       `json:"id"`
	UUID               string    `json:"uuid"`
	Subject            string    `json:"subject"`
	Status             string    `json:"status"`
	Priority           string    `json:"priority,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	LastActivityAt     time.Time `json:"last_activity_at"`
	LastMessagePreview string    `json:"last_message_preview"`
}

type portalMessageResponse struct {
	UUID        string                     `json:"uuid"`
	CreatedAt   time.Time                  `json:"created_at"`
	Type        string                     `json:"type"`
	SenderType  string                     `json:"sender_type"`
	Content     string                     `json:"content"`
	ContentType string                     `json:"content_type"`
	Attachments []portalAttachmentResponse `json:"attachments"`
}

type portalAttachmentResponse struct {
	Name         string `json:"name"`
	Size         int    `json:"size"`
	ContentType  string `json:"content_type"`
	Disposition  string `json:"disposition"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
}

type portalWriteRequest struct {
	InboxID int    `json:"inbox_id"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type portalContactSyncRequest struct {
	Email     string `json:"email,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

func portalAuth(next fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)
		if !ko.Bool("portal.enabled") || !ko.Exists("portal.api_key") {
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
		}
		secret := ko.String("portal.api_key")
		if len(secret) < 32 {
			app.lo.Error("portal API enabled with an invalid API key length")
			return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
		}
		header := string(r.RequestCtx.Request.Header.Peek("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") || !portalAPIKeyMatches(strings.TrimPrefix(header, "Bearer "), secret) {
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
		}
		externalUserID := string(r.RequestCtx.Request.Header.Peek("X-Portal-External-User-ID"))
		if externalUserID == "" || strings.TrimSpace(externalUserID) == "" || len(externalUserID) > maxExternalUserIDLength {
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
		}
		r.RequestCtx.SetUserValue(portalExternalIDKey, externalUserID)
		// Per-contact limits are independent of the existing IP-based public limit.
		method := string(r.RequestCtx.Method())
		write := method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete
		requestClass := "read"
		limit := int64(120)
		if write {
			requestClass = "write"
			limit = 30
		}
		key := fmt.Sprintf("portal_rate:%s:%s:%d", externalUserID, requestClass, time.Now().Unix()/60)
		count, err := app.redis.Incr(r.RequestCtx, key).Result()
		if err == nil {
			if count == 1 {
				app.redis.Expire(r.RequestCtx, key, 2*time.Minute)
			}
			if count > limit {
				return r.SendErrorEnvelope(fasthttp.StatusTooManyRequests, "Rate limit exceeded", nil, envelope.RateLimitError)
			}
		}
		if strings.HasSuffix(string(r.RequestCtx.Path()), "/contact") {
			return next(r)
		}
		contact, err := app.user.GetContactByExternalID(externalUserID)
		if err != nil || !contact.Enabled {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("globals.messages.notFound"), nil, envelope.NotFoundError)
		}
		r.RequestCtx.SetUserValue(portalContactKey, contact)
		return next(r)
	}
}

func portalAPIKeyMatches(provided, configured string) bool {
	if len(provided) != len(configured) || len(configured) < 32 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) == 1
}

func portalExternalUserID(r *fastglue.Request) string {
	return r.RequestCtx.UserValue(portalExternalIDKey).(string)
}
func portalContact(r *fastglue.Request) umodels.User {
	return r.RequestCtx.UserValue(portalContactKey).(umodels.User)
}

func handlePortalResolveContact(r *fastglue.Request) error {
	app := r.Context.(*App)
	externalUserID := portalExternalUserID(r)
	var profile portalContactSyncRequest
	decoder := json.NewDecoder(bytes.NewReader(r.RequestCtx.PostBody()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&profile); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return portalBadRequest(r, app, "Invalid contact profile")
	}
	if len(profile.Email) > maxEmailLength || profile.Email != "" && !stringutil.ValidEmail(profile.Email) || len(profile.FirstName) > maxNameLength || len(profile.LastName) > maxNameLength {
		return portalBadRequest(r, app, "Invalid contact profile")
	}
	var user umodels.User
	var err error
	user, err = app.user.GetContactByExternalID(externalUserID)
	if err != nil {
		if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
			return sendErrorEnvelope(r, err)
		}
		if profile.Email != "" {
			user, err = app.user.GetContactByEmailWithoutExtID(profile.Email)
			if err != nil {
				if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
					return sendErrorEnvelope(r, err)
				}
			}
			if err == nil {
				var assigned bool
				if assigned, err = app.user.SetExternalUserID(user.ID, externalUserID); err == nil && assigned {
					user.ExternalUserID = null.StringFrom(externalUserID)
				} else if err == nil {
					err = fmt.Errorf("contact identity conflict")
				}
			}
		}
		if err != nil {
			user = umodels.User{ExternalUserID: null.StringFrom(externalUserID), Email: null.NewString(profile.Email, profile.Email != ""), FirstName: profile.FirstName, LastName: profile.LastName, CustomAttributes: json.RawMessage(`{}`)}
			if err = app.user.ResolveContact(&user, umodels.ContactSync); err != nil {
				return sendErrorEnvelope(r, err)
			}
		}
	}
	if !user.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
	}
	first, last, email := user.FirstName, user.LastName, user.Email.String
	if profile.FirstName != "" {
		first = profile.FirstName
	}
	if profile.LastName != "" {
		last = profile.LastName
	}
	if profile.Email != "" {
		email = profile.Email
	}
	if first != user.FirstName || last != user.LastName || email != user.Email.String {
		if err := app.user.UpdateContactBasicInfo(user.ID, first, last, email, user.PhoneNumber.String, user.PhoneNumberCountryCode.String); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}
	app.lo.Info("portal contact resolved", "contact_id", user.ID)
	return r.SendEnvelope(portalContactResponse{ID: user.ID, ExternalUserID: externalUserID, Email: email, FirstName: first, LastName: last})
}

func handlePortalListConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	page, pageSize := getPagination(r)
	if raw := string(r.RequestCtx.QueryArgs().Peek("per_page")); raw != "" {
		if n, e := strconv.Atoi(raw); e == nil {
			pageSize = n
		}
	}
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}
	sort := string(r.RequestCtx.QueryArgs().Peek("sort"))
	if sort == "" {
		sort = "last_activity_at"
	}
	if sort != "last_activity_at" && sort != "created_at" {
		return portalBadRequest(r, app, "Invalid sort")
	}
	direction := strings.ToLower(string(r.RequestCtx.QueryArgs().Peek("direction")))
	if direction == "" {
		direction = "desc"
	}
	if direction != "asc" && direction != "desc" {
		return portalBadRequest(r, app, "Invalid direction")
	}
	items, total, err := app.conversation.GetPortalConversations(portalContact(r).ID, page, pageSize, string(r.RequestCtx.QueryArgs().Peek("status")), sort, direction)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(envelope.PageResults{Results: items, Total: total, Page: page, PerPage: pageSize, TotalPages: (total + pageSize - 1) / pageSize})
}

func handlePortalGetConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	conv, err := portalOwnedConversation(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	page, pageSize := getPagination(r)
	if raw := string(r.RequestCtx.QueryArgs().Peek("per_page")); raw != "" {
		if n, e := strconv.Atoi(raw); e == nil {
			pageSize = n
		}
	}
	if pageSize < 1 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}
	private := false
	messages, _, err := app.conversation.GetConversationMessages(conv.UUID, page, pageSize, &private, []string{cmodels.MessageIncoming, cmodels.MessageOutgoing})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result := make([]portalMessageResponse, 0, len(messages))
	rootURL, _ := app.setting.GetAppRootURL()
	for i := range messages {
		app.conversation.SignAttachmentURLs(messages[i].Attachments)
		resolveQuotedCIDs(app, &messages[i])
		resolveAttachmentCIDs(&messages[i], rootURL)
		result = append(result, portalMessageResponse{UUID: messages[i].UUID, CreatedAt: messages[i].CreatedAt, Type: messages[i].Type, SenderType: messages[i].SenderType, Content: messages[i].Content, ContentType: messages[i].ContentType, Attachments: portalAttachments(messages[i].Attachments)})
	}
	lastAt := conv.LastInteractionAt.Time
	if lastAt.IsZero() {
		lastAt = conv.CreatedAt
	}
	return r.SendEnvelope(map[string]any{"conversation": portalConversationResponse{ID: conv.ID, UUID: conv.UUID, Subject: conv.Subject.String, Status: conv.Status.String, Priority: conv.Priority.String, CreatedAt: conv.CreatedAt, LastActivityAt: lastAt, LastMessagePreview: conv.LastInteraction.String}, "messages": result, "page": page, "per_page": pageSize})
}

func portalOwnedConversation(r *fastglue.Request) (cmodels.Conversation, error) {
	app := r.Context.(*App)
	uuidValue, _ := r.RequestCtx.UserValue("uuid").(string)
	if _, err := uuid.Parse(uuidValue); err != nil {
		return cmodels.Conversation{}, envelope.NewError(envelope.NotFoundError, app.i18n.T("validation.notFoundConversation"), nil)
	}
	conv, err := app.conversation.GetConversation(0, uuidValue, "")
	if err != nil {
		return cmodels.Conversation{}, err
	}
	if conv.ContactID != portalContact(r).ID {
		app.lo.Warn("portal conversation access denied", "conversation_uuid", uuidValue, "contact_id", portalContact(r).ID)
		return cmodels.Conversation{}, envelope.NewError(envelope.NotFoundError, app.i18n.T("validation.notFoundConversation"), nil)
	}
	return conv, nil
}

func handlePortalCreateConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	req, media, err := decodePortalWrite(r, true, app)
	if err != nil {
		return err
	}
	if req.InboxID <= 0 || req.Subject == "" || len(req.Subject) > portalSubjectMax || strings.TrimSpace(req.Message) == "" || len(req.Message) > portalMessageMax {
		cleanupPortalMedia(app, media)
		return portalBadRequest(r, app, "Invalid conversation fields")
	}
	inbox, err := app.inbox.GetDBRecord(req.InboxID)
	if err != nil || !inbox.Enabled || (inbox.Channel != "email" && inbox.Channel != "livechat") {
		cleanupPortalMedia(app, media)
		return portalBadRequest(r, app, "Invalid inbox")
	}
	contact := portalContact(r)
	id, uuidValue, err := app.conversation.CreateConversation(contact.ID, inbox.ID, "", time.Now(), req.Subject, true, nil, nil, 0, 0)
	if err != nil {
		cleanupPortalMedia(app, media)
		return sendErrorEnvelope(r, err)
	}
	message, err := app.conversation.CreateContactMessage(media, contact.ID, uuidValue, req.Message, cmodels.ContentTypeText, true, "")
	if err != nil {
		_ = app.conversation.DeleteConversation(uuidValue)
		cleanupPortalMedia(app, media)
		return sendErrorEnvelope(r, err)
	}
	app.lo.Info("portal conversation created", "conversation_uuid", uuidValue, "contact_id", contact.ID, "inbox_id", inbox.ID)
	app.conversation.SignAttachmentURLs(message.Attachments)
	return r.SendEnvelope(map[string]any{"uuid": uuidValue, "id": id, "message": portalMessageResponse{UUID: message.UUID, CreatedAt: message.CreatedAt, Type: message.Type, SenderType: message.SenderType, Content: message.Content, ContentType: message.ContentType, Attachments: portalAttachments(message.Attachments)}})
}

func handlePortalReply(r *fastglue.Request) error {
	app := r.Context.(*App)
	conv, err := portalOwnedConversation(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	req, media, err := decodePortalWrite(r, false, app)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Message) == "" && len(media) == 0 || len(req.Message) > portalMessageMax {
		cleanupPortalMedia(app, media)
		return portalBadRequest(r, app, "Invalid message")
	}
	inbox, err := app.inbox.GetDBRecord(conv.InboxID)
	if err != nil || !inbox.Enabled {
		cleanupPortalMedia(app, media)
		return portalBadRequest(r, app, "Inbox is disabled")
	}
	if inbox.Channel == "livechat" && conv.Status.String == cmodels.StatusClosed {
		var cfg livechat.Config
		if json.Unmarshal(inbox.Config, &cfg) == nil && cfg.Users.PreventReplyToClosedConversation {
			cleanupPortalMedia(app, media)
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("widget.conversationClosed"), nil, envelope.PermissionError)
		}
	}
	contact := portalContact(r)
	message, err := app.conversation.CreateContactMessage(media, contact.ID, conv.UUID, req.Message, cmodels.ContentTypeText, false, "")
	if err != nil {
		cleanupPortalMedia(app, media)
		return sendErrorEnvelope(r, err)
	}
	app.lo.Info("portal message created", "conversation_uuid", conv.UUID, "message_uuid", message.UUID, "contact_id", contact.ID)
	app.conversation.SignAttachmentURLs(message.Attachments)
	return r.SendEnvelope(portalMessageResponse{UUID: message.UUID, CreatedAt: message.CreatedAt, Type: message.Type, SenderType: message.SenderType, Content: message.Content, ContentType: message.ContentType, Attachments: portalAttachments(message.Attachments)})
}

func decodePortalWrite(r *fastglue.Request, creating bool, app *App) (portalWriteRequest, []mmodels.Media, error) {
	var req portalWriteRequest
	ct, _, _ := mime.ParseMediaType(string(r.RequestCtx.Request.Header.ContentType()))
	if ct != "multipart/form-data" {
		dec := json.NewDecoder(bytes.NewReader(r.RequestCtx.PostBody()))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			return req, nil, portalBadRequest(r, app, "Invalid request body")
		}
		if dec.Decode(&struct{}{}) != io.EOF {
			return req, nil, portalBadRequest(r, app, "Invalid request body")
		}
		return req, nil, nil
	}
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return req, nil, portalBadRequest(r, app, "Invalid multipart body")
	}
	consts := app.consts.Load().(*constants)
	maxBody := int64(consts.MaxFileUploadSizeMB)*portalMaxFiles*1024*1024 + 1024*1024
	if int64(len(r.RequestCtx.PostBody())) > maxBody {
		return req, nil, portalBadRequest(r, app, "Request is too large")
	}
	for key := range form.File {
		if key != "files" {
			return req, nil, portalBadRequest(r, app, "Unknown file field")
		}
	}
	for key := range form.Value {
		if key != "inbox_id" && key != "subject" && key != "message" {
			return req, nil, portalBadRequest(r, app, "Unknown form field")
		}
	}
	for _, key := range []string{"inbox_id", "subject", "message"} {
		if len(form.Value[key]) > 1 {
			return req, nil, portalBadRequest(r, app, "Duplicate form field")
		}
	}
	if len(form.Value["inbox_id"]) > 0 {
		req.InboxID, _ = strconv.Atoi(form.Value["inbox_id"][0])
	}
	if len(form.Value["subject"]) > 0 {
		req.Subject = form.Value["subject"][0]
	}
	if len(form.Value["message"]) > 0 {
		req.Message = form.Value["message"][0]
	}
	if !creating && req.InboxID != 0 || !creating && req.Subject != "" {
		return req, nil, portalBadRequest(r, app, "Unexpected field")
	}
	files := form.File["files"]
	if len(files) > portalMaxFiles {
		return req, nil, portalBadRequest(r, app, "Too many files")
	}
	media := make([]mmodels.Media, 0, len(files))
	for _, fh := range files {
		name := stringutil.SanitizeFilename(fh.Filename)
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
		if fh.Size <= 0 || bytesToMegabytes(fh.Size) > float64(consts.MaxFileUploadSizeMB) || !slices.Contains(consts.AllowedUploadFileExtensions, "*") && !slices.Contains(consts.AllowedUploadFileExtensions, ext) {
			cleanupPortalMedia(app, media)
			return req, nil, portalBadRequest(r, app, "Invalid attachment")
		}
		f, e := fh.Open()
		if e != nil {
			cleanupPortalMedia(app, media)
			return req, nil, portalBadRequest(r, app, "Invalid attachment")
		}
		data, e := io.ReadAll(io.LimitReader(f, int64(consts.MaxFileUploadSizeMB)*1024*1024+1))
		f.Close()
		if e != nil || len(data) == 0 || bytesToMegabytes(int64(len(data))) > float64(consts.MaxFileUploadSizeMB) {
			cleanupPortalMedia(app, media)
			return req, nil, portalBadRequest(r, app, "Invalid attachment")
		}
		contentType := http.DetectContentType(data)
		m, e := app.media.UploadAndInsert(name, contentType, "", null.String{}, null.Int{}, bytes.NewReader(data), len(data), null.StringFrom(attachment.DispositionAttachment), []byte("{}"), true)
		if e != nil {
			cleanupPortalMedia(app, media)
			return req, nil, sendErrorEnvelope(r, e)
		}
		media = append(media, m)
	}
	return req, media, nil
}

func cleanupPortalMedia(app *App, media []mmodels.Media) {
	for _, m := range media {
		if m.UUID != "" {
			_ = app.media.Delete(m.UUID)
		}
	}
}

func portalAttachments(items attachment.Attachments) []portalAttachmentResponse {
	out := make([]portalAttachmentResponse, 0, len(items))
	for _, item := range items {
		out = append(out, portalAttachmentResponse{Name: item.Name, Size: item.Size, ContentType: item.ContentType, Disposition: item.Disposition, URL: item.URL, ThumbnailURL: item.ThumbnailURL})
	}
	return out
}
func portalBadRequest(r *fastglue.Request, app *App, msg string) error {
	return r.SendErrorEnvelope(fasthttp.StatusBadRequest, msg, nil, envelope.InputError)
}
