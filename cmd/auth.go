package main

import (
	"errors"
	"strconv"
	"strings"

	auth_ "github.com/abhinavxd/libredesk/internal/auth"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	oidcErrLoginFailed     = "oidc_login_failed"
	oidcErrSessionExpired  = "oidc_session_expired"
	oidcErrInvalidClient   = "oidc_invalid_client"
	oidcErrAccessDenied    = "oidc_access_denied"
	oidcErrNoAccount       = "oidc_no_account"
	oidcErrAccountDisabled = "oidc_account_disabled"
)

var (
	oidcStateSessKey = "oidc_state"
	oidcNextSessKey  = "oidc_next"
)

// handleOIDCLogin redirects to the OIDC provider for login.
func handleOIDCLogin(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		next            = string(r.RequestCtx.QueryArgs().Peek("next"))
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return redirectLoginError(r, oidcErrLoginFailed, next)
	}

	// Set a state and save it in the session, to prevent CSRF attacks.
	state, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		app.lo.Error("error generating state", "error", err)
		return redirectLoginError(r, oidcErrLoginFailed, next)
	}

	sessionValues := map[string]any{
		oidcStateSessKey: state,
		// For redirecting after login
		oidcNextSessKey: next,
	}

	if err = app.auth.SetSessionValues(r, sessionValues); err != nil {
		app.lo.Error("error saving state in session", "error", err)
		return redirectLoginError(r, oidcErrLoginFailed, next)
	}

	authURL, err := app.auth.LoginURL(providerID, state)
	if err != nil {
		app.lo.Error("error getting oidc login url", "provider_id", providerID, "error", err)
		return redirectLoginError(r, oidcErrLoginFailed, next)
	}
	app.lo.Debug("redirecting to oidc provider for login", "provider_id", providerID)
	return r.Redirect(authURL, fasthttp.StatusFound, nil, "")
}

// handleOIDCCallback receives the redirect callback from the OIDC provider and completes the handshake.
func handleOIDCCallback(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		code            = string(r.RequestCtx.QueryArgs().Peek("code"))
		state           = string(r.RequestCtx.QueryArgs().Peek("state"))
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		ip              = realip.FromRequest(r.RequestCtx)
	)
	next, _ := app.auth.GetSessionValue(r, oidcNextSessKey)
	nextStr, _ := next.(string)

	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return redirectLoginError(r, oidcErrLoginFailed, nextStr)
	}

	app.lo.Debug("oidc callback received", "provider_id", providerID, "has_code", code != "")

	// Providers redirect back with an error param instead of a code when the handshake fails on their side (RFC 6749 4.1.2.1).
	if oauthErr := string(r.RequestCtx.QueryArgs().Peek("error")); oauthErr != "" {
		desc := string(r.RequestCtx.QueryArgs().Peek("error_description"))
		if oauthErr == "access_denied" {
			app.lo.Warn("oidc sign-in cancelled or denied at provider", "provider_id", providerID, "description", desc)
			return redirectLoginError(r, oidcErrAccessDenied, nextStr)
		}
		app.lo.Error("oidc provider returned an error on callback", "provider_id", providerID, "oauth_error", oauthErr, "description", desc)
		return redirectLoginError(r, oidcErrLoginFailed, nextStr)
	}

	// Compare the state from the session with the state from the query.
	sessionState, err := app.auth.GetSessionValue(r, oidcStateSessKey)
	if err != nil {
		app.lo.Error("error getting oidc state from session, the session cookie may be missing or expired", "provider_id", providerID, "error", err)
		return redirectLoginError(r, oidcErrSessionExpired, nextStr)
	}
	if state != sessionState {
		app.lo.Error("oidc state mismatch, the session cookie may be missing or expired or the callback URL is stale", "provider_id", providerID)
		return redirectLoginError(r, oidcErrSessionExpired, nextStr)
	}

	_, claims, err := app.auth.ExchangeOIDCToken(r.RequestCtx, providerID, code)
	if err != nil {
		if errors.Is(err, auth_.ErrOIDCInvalidClient) {
			return redirectLoginError(r, oidcErrInvalidClient, nextStr)
		}
		return redirectLoginError(r, oidcErrLoginFailed, nextStr)
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))

	user, err := app.user.GetAgent(0, email)
	if err != nil {
		if e, ok := err.(envelope.Error); ok && e.ErrorType == envelope.NotFoundError {
			app.lo.Warn("no agent account matching oidc email", "provider_id", providerID, "email", email)
			return redirectLoginError(r, oidcErrNoAccount, nextStr)
		}
		return redirectLoginError(r, oidcErrLoginFailed, nextStr)
	}

	if !user.Enabled {
		app.lo.Warn("oidc login rejected for disabled account", "provider_id", providerID, "user_id", user.ID)
		return redirectLoginError(r, oidcErrAccountDisabled, nextStr)
	}
	// Only agents can log in; GetAgent also resolves ai_assistant identity users.
	if user.Type != models.UserTypeAgent {
		app.lo.Warn("oidc login rejected for non-agent user", "provider_id", providerID, "user_id", user.ID)
		return redirectLoginError(r, oidcErrNoAccount, nextStr)
	}

	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		app.lo.Error("error saving session for oidc login", "user_id", user.ID, "error", err)
		return redirectLoginError(r, oidcErrLoginFailed, nextStr)
	}

	if err := app.user.UpdateLastLoginAt(user.ID); err != nil {
		app.lo.Error("error updating last login at for oidc login", "user_id", user.ID, "error", err)
	}

	app.user.InvalidateAgentCache(user.ID)

	// Insert activity log.
	if err := app.activityLog.Login(user.ID, user.Email.String, ip); err != nil {
		app.lo.Error("error creating login activity log", "error", err)
	}

	app.lo.Info("oidc login successful", "provider_id", providerID, "user_id", user.ID, "email", user.Email.String)

	redirectURL := "/"
	if nextStr != "" {
		redirectURL = nextStr
	}

	return r.RedirectURI(redirectURL, fasthttp.StatusFound, nil, "")
}

func redirectLoginError(r *fastglue.Request, code, next string) error {
	args := map[string]any{"error": code}
	if next != "" {
		args["next"] = next
	}
	return r.RedirectURI("/", fasthttp.StatusFound, args, "")
}
