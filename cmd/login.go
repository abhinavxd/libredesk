package main

import (
	amodels "github.com/jakedolan443/fernmail/internal/auth/models"
	"github.com/jakedolan443/fernmail/internal/envelope"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleLogin logs in the user and returns the user.
func handleLogin(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)

		loginReq loginRequest
	)

	if localLoginDisabledInConfig() {
		oidcProviders, err := app.oidc.GetAll()
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if !isLocalLoginEnabled(oidcProviders) {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("auth.localLoginDisabled"), nil, envelope.PermissionError)
		}
		app.lo.Warn("local login is disabled in config but no OIDC provider is enabled, allowing password login")
	}

	// Decode JSON request.
	if err := r.Decode(&loginReq, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	if loginReq.Email == "" || loginReq.Password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Verify email and password.
	user, err := app.user.VerifyPassword(loginReq.Email, []byte(loginReq.Password))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check if user is enabled.
	if !user.Enabled {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.accountDisabled"), nil))
	}

	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		app.lo.Error("error saving session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	// Set CSRF cookie if not already set.
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}

	// Update last login time.
	if err := app.user.UpdateLastLoginAt(user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.user.InvalidateAgentCache(user.ID)

	// Insert activity log.

	return r.SendEnvelope(user)
}

// handleLogout logs out the user and redirects to the dashboard.
func handleLogout(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	// Insert activity log.

	if err := app.auth.DestroySession(r); err != nil {
		app.lo.Error("error destroying session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	// Add no-cache headers.
	r.RequestCtx.Response.Header.Add("Cache-Control",
		"no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")
	return r.RedirectURI("/", fasthttp.StatusFound, nil, "")
}
