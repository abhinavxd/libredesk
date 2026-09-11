package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	gmodels "github.com/abhinavxd/libredesk/internal/guidedform/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetGuidedForms returns all guided forms.
func handleGetGuidedForms(r *fastglue.Request) error {
	app := r.Context.(*App)
	forms, err := app.guidedForm.GetForms()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(forms)
}

// handleGetGuidedForm returns one guided form by id.
func handleGetGuidedForm(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	form, err := app.guidedForm.GetForm(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(form)
}

// handleCreateGuidedForm creates a new guided form.
func handleCreateGuidedForm(r *fastglue.Request) error {
	app := r.Context.(*App)
	var req gmodels.Form
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	form, err := app.guidedForm.CreateForm(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(form)
}

// handleUpdateGuidedForm updates an existing guided form.
func handleUpdateGuidedForm(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	var req gmodels.Form
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	form, err := app.guidedForm.UpdateForm(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(form)
}

// handleDeleteGuidedForm deletes a guided form.
func handleDeleteGuidedForm(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if _, err := app.guidedForm.DeleteForm(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
