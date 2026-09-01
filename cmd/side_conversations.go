package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

type sideConversationReq struct {
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Content string   `json:"content"`
}

func handleListSideConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	uuid := r.RequestCtx.UserValue("uuid").(string)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if _, err := enforceConversationAccess(app, uuid, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	list, err := app.conversation.ListSideConversations(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(list)
}

func handleCreateSideConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	uuid := r.RequestCtx.UserValue("uuid").(string)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if _, err := enforceConversationAccess(app, uuid, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	var req sideConversationReq
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	created, err := app.conversation.CreateSideConversation(uuid, user.ID, req.To, req.Subject, req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(created)
}

func handleReplySideConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	convUUID := r.RequestCtx.UserValue("uuid").(string)
	sideUUID := r.RequestCtx.UserValue("side_uuid").(string)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if _, err := enforceConversationAccess(app, convUUID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	var req sideConversationReq
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	updated, err := app.conversation.ReplySideConversation(sideUUID, user.ID, req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(updated)
}
