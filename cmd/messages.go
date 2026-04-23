package main

import (
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type messageReq struct {
	Attachments []int                  `json:"attachments"`
	Message     string                 `json:"message"`
	Private     bool                   `json:"private"`
	To          []string               `json:"to"`
	CC          []string               `json:"cc"`
	BCC         []string               `json:"bcc"`
	SenderType  string                 `json:"sender_type"`
	Mentions    []cmodels.MentionInput `json:"mentions"`
	EchoID      string                 `json:"echo_id"`
	// ForwardedTo, when non-empty, signals "forward this conversation to
	// the listed addresses". The original recipients (To) are ignored;
	// CC/BCC pass through unchanged so an agent can loop in teammates.
	ForwardedTo []string `json:"forwarded_to"`
}

// handleGetMessages returns messages for a conversation.
func handleGetMessages(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		uuid    = r.RequestCtx.UserValue("uuid").(string)
		auser   = r.RequestCtx.UserValue("user").(amodels.User)
		total   = 0
		private *bool
	)
	page, pageSize := getPagination(r)

	// Parse optional private filter (null = no filter)
	if r.RequestCtx.QueryArgs().Has("private") {
		p := r.RequestCtx.QueryArgs().GetBool("private")
		private = &p
	}

	// Parse repeated type params: ?type=incoming&type=outgoing
	var msgTypes []string
	for _, v := range r.RequestCtx.QueryArgs().PeekMulti("type") {
		msgTypes = append(msgTypes, string(v))
	}

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	messages, pageSize, err := app.conversation.GetConversationMessages(uuid, page, pageSize, private, msgTypes)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	rootURL, _ := app.setting.GetAppRootURL()
	for i := range messages {
		total = messages[i].Total
		// Populate attachment URLs
		for j := range messages[i].Attachments {
			att := messages[i].Attachments[j]
			messages[i].Attachments[j].URL = app.media.GetURL(att.UUID, att.ContentType, att.Name)
		}
		resolveContentCIDs(&messages[i], rootURL)
	}

	// Process CSAT status for all messages (will only affect CSAT messages)
	app.conversation.ProcessCSATStatus(messages)

	// Strip CSAT UUID from agent sessions to prevent self-rating.
	if r.RequestCtx.UserValue("auth_method") != "api_key" {
		for i := range messages {
			if messages[i].HasCSAT() {
				messages[i].StripCSATUUID()
			}
		}
	}

	return r.SendEnvelope(envelope.PageResults{
		Total:      total,
		Results:    messages,
		Page:       page,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	})
}

// handleGetMessage fetches a single from DB using the uuid.
func handleGetMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	message, err := app.conversation.GetMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Process CSAT status for the message (will only affect CSAT messages)
	messages := []cmodels.Message{message}
	app.conversation.ProcessCSATStatus(messages)
	message = messages[0]

	// Strip CSAT UUID from agent sessions to prevent self-rating.
	if r.RequestCtx.UserValue("auth_method") != "api_key" && message.HasCSAT() {
		message.StripCSATUUID()
	}

	rootURL, _ := app.setting.GetAppRootURL()
	for j := range message.Attachments {
		att := message.Attachments[j]
		message.Attachments[j].URL = app.media.GetURL(att.UUID, att.ContentType, att.Name)
	}
	resolveContentCIDs(&message, rootURL)

	return r.SendEnvelope(message)
}

// handleRetryMessage changes message status to `pending`, so it's enqueued for sending.
func handleRetryMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Only outgoing agent messages that have failed can be retried.
	msg, err := app.conversation.GetMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if msg.SenderType != cmodels.SenderTypeAgent || msg.Status != cmodels.MessageStatusFailed || msg.SenderID != user.ID || msg.ConversationUUID != cuuid {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	if err = app.conversation.MarkMessageAsPending(uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleSendMessage sends a message in a conversation.
func handleSendMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		req   = messageReq{}
	)

	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check access to conversation.
	conv, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling message request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	// Make sure the inbox is enabled.
	inbox, err := app.inbox.GetDBRecord(conv.InboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("status.disabledInbox"), nil, envelope.InputError)
	}

	if req.SenderType != umodels.UserTypeAgent && req.SenderType != umodels.UserTypeContact {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	// Contacts cannot send private messages
	if req.SenderType == umodels.UserTypeContact && req.Private {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Check if user has permission to send messages as contact
	if req.SenderType == umodels.UserTypeContact {
		parts := strings.Split(authzModels.PermMessagesWriteAsContact, ":")
		if len(parts) != 2 {
			app.lo.Error("error parsing permission string", "permission", authzModels.PermMessagesWriteAsContact)
			return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
		}
		ok, err := app.authz.Enforce(user, parts[0], parts[1])
		if err != nil {
			app.lo.Error("error checking permission", "error", err)
			return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
		}
		if !ok {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
		}
	}

	// Get media for all attachments, skip any already associated with a model.
	media, err := getUnassociatedMedia(app, req.Attachments)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Create contact message.
	if req.SenderType == umodels.UserTypeContact {
		message, err := app.conversation.CreateContactMessage(media, int(conv.ContactID), cuuid, req.Message, cmodels.ContentTypeHTML, false)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(message)
	}

	// Send private note.
	if req.Private {
		message, err := app.conversation.SendPrivateNote(media, user.ID, cuuid, req.Message, req.Mentions)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(message)
	}

	meta := map[string]any{}
	if req.EchoID != "" {
		meta["echo_id"] = req.EchoID
	}

	// Forward mode: route to the forwarded_to recipients instead of req.To.
	// CC/BCC pass through unchanged so an agent can loop teammates in.
	sendTo := req.To
	if len(req.ForwardedTo) > 0 {
		meta["forwarded"] = true
		meta["forwarded_to"] = req.ForwardedTo
		sendTo = req.ForwardedTo
	}

	message, err := app.conversation.QueueReply(media, conv.InboxID, user.ID, conv.ContactID, cuuid, req.Message, sendTo, req.CC, req.BCC, meta)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Log a forward activity note so the audit trail captures who forwarded
	// to whom. Best-effort: a logging failure shouldn't break the send.
	if len(req.ForwardedTo) > 0 {
		recipients := strings.Join(req.ForwardedTo, ", ")
		if err := app.conversation.InsertConversationActivity(cmodels.ActivityMessageForwarded, cuuid, recipients, user); err != nil {
			app.lo.Warn("failed to insert forward activity note", "error", err, "conversation_uuid", cuuid)
		}
	}

	return r.SendEnvelope(message)
}

// resolveContentCIDs replaces inline image cid: references in email message content
// with actual attachment URLs and resolves relative /uploads/ paths to absolute URLs.
func resolveContentCIDs(msg *cmodels.Message, rootURL string) {
	for _, att := range msg.Attachments {
		if att.ContentID != "" && att.URL != "" {
			msg.Content = strings.ReplaceAll(msg.Content, "cid:"+att.ContentID, att.URL)
		}
	}
	if rootURL != "" {
		msg.Content = strings.ReplaceAll(msg.Content, `src="/uploads/`, `src="`+rootURL+`/uploads/`)
		msg.Content = strings.ReplaceAll(msg.Content, `src='/uploads/`, `src='`+rootURL+`/uploads/`)
	}
}
