package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	vmodels "github.com/abhinavxd/libredesk/internal/view/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetUserViews returns all views for a user.
func handleGetUserViews(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	v, err := app.view.GetUsersViews(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(v)
}

// handleCreateUserView creates a view for a user.
func handleCreateUserView(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		view  = vmodels.View{}
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if err := r.Decode(&view, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), err.Error(), envelope.InputError)
	}
	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if view.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`Name`"), nil, envelope.InputError)
	}
	if string(view.Filters) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`Filters`"), nil, envelope.InputError)
	}
	if err := app.view.Create(view.Name, view.Filters, user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleDeleteUserView deletes a view for a user.
func handleDeleteUserView(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	view, err := app.view.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if view.UserID != user.ID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("authz.permissionDenied"), nil, envelope.PermissionError)
	}
	if err = app.view.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateUserView updates a view for a user.
func handleUpdateUserView(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		view  = vmodels.View{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&view, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.entities.request}"), err.Error(), envelope.InputError)
	}
	user, err := app.user.GetAgent(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if view.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}
	if string(view.Filters) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`filters`"), nil, envelope.InputError)
	}
	v, err := app.view.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if v.UserID != user.ID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("authz.permissionDenied"), nil, envelope.PermissionError)
	}
	if err = app.view.Update(id, view.Name, view.Filters); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
