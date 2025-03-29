package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleLogin logs in the user and returns the user.
func handleLogin(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		email    = string(r.RequestCtx.PostArgs().Peek("email"))
		password = r.RequestCtx.PostArgs().Peek("password")
	)

	// Verify email and password.
	user, err := app.user.VerifyPassword(email, password)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check if user is enabled.
	if !user.Enabled {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.accountDisabled"), nil))
	}

	// Set user availability status to online.
	if err := app.user.UpdateAvailability(user.ID, umodels.Online); err != nil {
		return sendErrorEnvelope(r, err)
	}
	user.AvailabilityStatus = umodels.Online

	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		app.lo.Error("error saving session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorSavingSession"), nil))
	}
	// Set CSRF cookie if not already set.
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorSavingSession"), nil))
	}
	return r.SendEnvelope(user)
}

// handleLogout logs out the user and redirects to the dashboard.
func handleLogout(r *fastglue.Request) error {
	var app = r.Context.(*App)
	if err := app.auth.DestroySession(r); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorDestroying", "name", "{globals.entities.session}"), nil))
	}
	// Add no-cache headers.
	r.RequestCtx.Response.Header.Add("Cache-Control",
		"no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")
	return r.RedirectURI("/", fasthttp.StatusFound, nil, "")
}
