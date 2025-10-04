package main

import (
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/oidc/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetAllOIDC returns all OIDC records
func handleGetAllOIDC(r *fastglue.Request) error {
	app := r.Context.(*App)
	out, err := app.oidc.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Replace secrets with dummy values.
	for i := range out {
		out[i].ClientSecret = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	return r.SendEnvelope(out)
}

// handleGetOIDC returns an OIDC record by id.
func handleGetOIDC(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalid", "name", "OIDC `id`"), nil, envelope.InputError)
	}
	o, err := app.oidc.Get(id, false)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(o)
}

// handleCreateOIDC creates a new OIDC record.
func handleCreateOIDC(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.OIDC{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	// Test OIDC provider URL by performing a discovery.
	if err := app.auth.TestProvider(req.ProviderURL); err != nil {
		return sendErrorEnvelope(r, err)
	}

	createdOIDC, err := app.oidc.Create(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Reload the auth manager to update the OIDC providers.
	if err := reloadAuth(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "OIDC"), nil, envelope.GeneralError)
	}

	// Clear client secret before returning
	createdOIDC.ClientSecret = strings.Repeat(stringutil.PasswordDummy, 10)

	return r.SendEnvelope(createdOIDC)
}

// handleUpdateOIDC updates an OIDC record.
func handleUpdateOIDC(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.OIDC{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "OIDC `id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	// Test OIDC provider URL by performing a discovery.
	if err := app.auth.TestProvider(req.ProviderURL); err != nil {
		return sendErrorEnvelope(r, err)
	}

	updatedOIDC, err := app.oidc.Update(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Reload the auth manager to update the OIDC providers.
	if err := reloadAuth(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "OIDC"), nil, envelope.GeneralError)
	}

	// Clear client secret before returning
	updatedOIDC.ClientSecret = strings.Repeat(stringutil.PasswordDummy, 10)

	return r.SendEnvelope(updatedOIDC)
}

// handleDeleteOIDC deletes an OIDC record.
func handleDeleteOIDC(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "OIDC `id`"), nil, envelope.InputError)
	}
	if err = app.oidc.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
