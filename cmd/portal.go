package main

import (
	"strings"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type createPortalConversationRequest struct {
	InboxID int    `json:"inbox_id"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

type portalInbox struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func handleGetPortalMe(r *fastglue.Request) error {
	return r.SendEnvelope(r.RequestCtx.UserValue("user").(amodels.User))
}

func handleGetPortalInboxes(r *fastglue.Request) error {
	app := r.Context.(*App)
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result := make([]portalInbox, 0, len(inboxes))
	for _, candidate := range inboxes {
		if candidate.Enabled && candidate.Channel == inbox.ChannelEmail {
			result = append(result, portalInbox{ID: candidate.ID, Name: candidate.Name})
		}
	}
	return r.SendEnvelope(result)
}

func handleGetPortalConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	page, pageSize := getPagination(r)
	items, total, err := app.conversation.GetPortalConversations(user.ID, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(envelope.PageResults{
		Total: total, Results: items, Page: page, PerPage: pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	})
}

func handleCreatePortalConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	req := createPortalConversationRequest{}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	req.Subject = strings.TrimSpace(req.Subject)
	req.Content = strings.TrimSpace(req.Content)
	if req.Subject == "" || req.Content == "" || len(req.Subject) > 200 || len(req.Content) > 10000 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	selectedInbox, err := app.inbox.GetDBRecord(req.InboxID)
	if err != nil || !selectedInbox.Enabled || selectedInbox.Channel != inbox.ChannelEmail {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Support is not configured to receive new tickets.", nil, envelope.GeneralError)
	}

	conversationID, conversationUUID, err := app.conversation.CreateConversation(
		user.ID, selectedInbox.ID, "", time.Now(), req.Subject, true, nil, nil, 0, 0,
	)
	if err != nil {
		app.lo.Error("creating portal conversation", "contact_id", user.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	if _, err := app.conversation.CreateContactMessage(nil, user.ID, conversationUUID, req.Content, cmodels.ContentTypeText, true); err != nil {
		if deleteErr := app.conversation.DeleteConversation(conversationUUID); deleteErr != nil {
			app.lo.Error("deleting failed portal conversation", "conversation_uuid", conversationUUID, "error", deleteErr)
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingMessage"), nil, envelope.GeneralError)
	}

	created, err := app.conversation.GetPortalConversation(user.ID, conversationUUID)
	if err != nil {
		app.lo.Error("loading created portal conversation", "conversation_id", conversationID, "error", err)
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(created)
}

func handleGetPortalConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	item, err := app.conversation.GetPortalConversation(user.ID, r.RequestCtx.UserValue("uuid").(string))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(item)
}

func handleGetPortalMessages(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	page, pageSize := getPagination(r)
	items, pageSize, err := app.conversation.GetPortalMessages(user.ID, r.RequestCtx.UserValue("uuid").(string), page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(envelope.PageResults{Results: items, Page: page, PerPage: pageSize})
}
