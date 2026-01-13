package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	vmodels "github.com/abhinavxd/libredesk/internal/view/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetUserViews returns all personal views for a user.
func handleGetUserViews(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	v, err := app.view.GetUsersViews(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(v)
}

// handleCreateUserView creates a personal view for a user.
func handleCreateUserView(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		view  = vmodels.View{}
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if err := r.Decode(&view, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if view.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`Name`"), nil, envelope.InputError)
	}
	if string(view.Filters) == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`Filters`"), nil, envelope.InputError)
	}
	createdView, err := app.view.Create(view.Name, view.Filters, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(createdView)
}

// handleDeleteUserView deletes a personal view for a user.
func handleDeleteUserView(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	view, err := app.view.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Only allow deletion of personal views owned by the user
	if err := validatePersonalViewOwnership(app, view, user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err = app.view.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUpdateUserView updates a personal view for a user.
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}
	user, err := app.user.GetAgent(auser.ID, "")
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
	// Only allow update of personal views owned by the user
	if err := validatePersonalViewOwnership(app, v, user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}
	updatedView, err := app.view.Update(id, view.Name, view.Filters, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(updatedView)
}

// handleGetSharedViews returns shared views accessible to the current user.
func handleGetSharedViews(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	views, err := app.view.GetSharedViewsForUser(user.Teams.IDs())
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(views)
}

// handleGetAllSharedViews returns all shared views (admin only).
func handleGetAllSharedViews(r *fastglue.Request) error {
	app := r.Context.(*App)
	views, err := app.view.GetAllSharedViews()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(views)
}

// handleGetSharedView returns a single shared view (admin only).
func handleGetSharedView(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	view, err := app.view.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Ensure it's a shared view (not a personal view)
	if view.Visibility == vmodels.VisibilityUser {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.view}"), nil, envelope.NotFoundError)
	}
	return r.SendEnvelope(view)
}

// handleCreateSharedView creates a shared view (admin only).
func handleCreateSharedView(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		view = vmodels.View{}
	)
	if err := r.Decode(&view, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	// Validation
	if err := validateSharedView(app, view); err != nil {
		return sendErrorEnvelope(r, err)
	}

	createdView, err := app.view.CreateSharedView(view.Name, view.Filters, view.Visibility, view.TeamID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(createdView)
}

// handleUpdateSharedView updates a shared view (admin only).
func handleUpdateSharedView(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		view  = vmodels.View{}
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	if err := r.Decode(&view, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	// Verify view exists and is shared
	existingView, err := app.view.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if existingView.Visibility == vmodels.VisibilityUser {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.view}"), nil, envelope.NotFoundError)
	}

	// Validation
	if err := validateSharedView(app, view); err != nil {
		return sendErrorEnvelope(r, err)
	}

	updatedView, err := app.view.UpdateSharedView(id, view.Name, view.Filters, view.Visibility, view.TeamID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(updatedView)
}

// handleDeleteSharedView deletes a shared view (admin only).
func handleDeleteSharedView(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	// Verify view exists and is shared
	existingView, err := app.view.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if existingView.Visibility == vmodels.VisibilityUser {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.view}"), nil, envelope.NotFoundError)
	}

	if err = app.view.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// validatePersonalViewOwnership checks if the user owns the personal view.
func validatePersonalViewOwnership(app *App, view vmodels.View, userID int) error {
	if view.UserID == nil || *view.UserID != userID || view.Visibility != vmodels.VisibilityUser {
		return envelope.NewError(envelope.PermissionError, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil)
	}
	return nil
}

// validateSharedView validates the fields of a shared view.
func validateSharedView(app *App, view vmodels.View) error {
	if view.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	if string(view.Filters) == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`filters`"), nil)
	}
	if view.Visibility != vmodels.VisibilityAll && view.Visibility != vmodels.VisibilityTeam {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`visibility`"), nil)
	}
	if view.Visibility == vmodels.VisibilityTeam && (view.TeamID == nil || *view.TeamID <= 0) {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.required", "name", "`team_id`"), nil)
	}
	return nil
}
