package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

var (
	oidcStateSessKey = "oidc_state"
	oidcNextSessKey  = "oidc_next"
)

// handleOIDCLogin redirects to the OIDC provider for login.
func handleOIDCLogin(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.GeneralError)
	}

	// Set a state and save it in the session, to prevent CSRF attacks.
	state, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		app.lo.Error("error generating state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorGenerating", "name", "state"), nil, envelope.GeneralError)
	}

	sessionValues := map[string]any{
		oidcStateSessKey: state,
		// For redirecting after login
		oidcNextSessKey: string(r.RequestCtx.QueryArgs().Peek("next")),
	}

	if err = app.auth.SetSessionValues(r, sessionValues); err != nil {
		app.lo.Error("error saving state in session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorSaving", "name", "{globals.terms.session}"), nil, envelope.GeneralError)
	}

	authURL, err := app.auth.LoginURL(providerID, state)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.Redirect(authURL, fasthttp.StatusFound, nil, "")
}

// handleOIDCCallback receives the redirect callback from the OIDC provider and completes the handshake.
func handleOIDCCallback(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		code            = string(r.RequestCtx.QueryArgs().Peek("code"))
		state           = string(r.RequestCtx.QueryArgs().Peek("state"))
		providerID, err = strconv.Atoi(string(r.RequestCtx.UserValue("id").(string)))
		ip              = realip.FromRequest(r.RequestCtx)
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.GeneralError)
	}

	// Compare the state from the session with the state from the query.
	sessionState, err := app.auth.GetSessionValue(r, oidcStateSessKey)
	if err != nil {
		app.lo.Error("error getting state from session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.session}"), nil, envelope.GeneralError)
	}
	if state != sessionState {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.Ts("globals.messages.mismatch", "name", "{globals.terms.state}"), nil, envelope.GeneralError)
	}

	_, claims, err := app.auth.ExchangeOIDCToken(r.RequestCtx, providerID, code)
	if err != nil {
		app.lo.Error("error exchanging oidc token", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			app.i18n.T("globals.messages.errorExchangingToken"), nil, envelope.GeneralError)
	}

	// Lookup the user by email and set the session.
	user, err := app.user.GetAgent(0, claims.Email)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			app.i18n.Ts("globals.messages.errorSaving", "name", "{globals.terms.session}"), nil, envelope.GeneralError)
	}

	// Update last login time.
	if err := app.user.UpdateLastLoginAt(user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Insert activity log.
	if err := app.activityLog.Login(user.ID, user.Email.String, ip); err != nil {
		app.lo.Error("error creating login activity log", "error", err)
	}

	// Read the 'next' parameter from session to redirect after login.
	nextParam, _ := app.auth.GetSessionValue(r, oidcNextSessKey)
	redirectURL := "/"
	if nextStr, ok := nextParam.(string); ok && nextStr != "" {
		redirectURL = nextStr
	}

	return r.RedirectURI(redirectURL, fasthttp.StatusFound, nil, "")
}
