package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

func handleGetPortalMe(r *fastglue.Request) error {
	return r.SendEnvelope(r.RequestCtx.UserValue("user").(amodels.User))
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
