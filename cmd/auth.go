package main

import (
	"net/url"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	// The mobile app registers this scheme on both platforms; it is the app's identity, not
	// operator configuration.
	mobileCallbackURL      = "libredesk://callback"
	oidcClientMobile       = "mobile"
	maxCodeChallengeLength = 128
)

var (
	oidcStateSessKey     = "oidc_state"
	oidcNextSessKey      = "oidc_next"
	oidcClientSessKey    = "oidc_client"
	oidcChallengeSessKey = "oidc_code_challenge"
)

// handleOIDCLogin redirects to the OIDC provider for login.
func handleOIDCLogin(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Set a state and save it in the session, to prevent CSRF attacks.
	state, err := stringutil.RandomAlphanumeric(32)
	if err != nil {
		app.lo.Error("error generating state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// The mobile app cannot hold a session cookie, so it proves ownership of the login with PKCE
	// instead and receives a one-time code on the callback.
	client := string(r.RequestCtx.QueryArgs().Peek("client"))
	challenge := string(r.RequestCtx.QueryArgs().Peek("code_challenge"))
	if client == oidcClientMobile {
		if !app.consts.Load().(*constants).AllowMobileApp {
			return redirectToApp(r, "mobile_disabled")
		}
		if challenge == "" || len(challenge) > maxCodeChallengeLength {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
		}
	}

	sessionValues := map[string]any{
		oidcStateSessKey: state,
		// For redirecting after login
		oidcNextSessKey:      string(r.RequestCtx.QueryArgs().Peek("next")),
		oidcClientSessKey:    client,
		oidcChallengeSessKey: challenge,
	}

	if err = app.auth.SetSessionValues(r, sessionValues); err != nil {
		app.lo.Error("error saving state in session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Compare the state from the session with the state from the query.
	sessionState, err := app.auth.GetSessionValue(r, oidcStateSessKey)
	if err != nil {
		app.lo.Error("error getting state from session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	if state != sessionState {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	clientValue, _ := app.auth.GetSessionValue(r, oidcClientSessKey)
	client, _ := clientValue.(string)
	isMobile := client == oidcClientMobile

	_, claims, err := app.auth.ExchangeOIDCToken(r.RequestCtx, providerID, code)
	if err != nil {
		app.lo.Error("error exchanging oidc token", "error", err)
		if isMobile {
			return redirectToApp(r, "server_error")
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Lookup the user by email and set the session.
	user, err := app.user.GetAgent(0, claims.Email)
	if err != nil {
		if isMobile {
			return redirectToApp(r, "unknown_agent")
		}
		return sendErrorEnvelope(r, err)
	}
	// Only agents can log in; GetAgent also resolves ai_assistant identity users.
	if user.Type != models.UserTypeAgent || !user.Enabled {
		if isMobile {
			return redirectToApp(r, "access_denied")
		}
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.PermissionError)
	}

	// Issue the credential the client asked for: a session cookie for the browser, a one-time code
	// the app trades for a device token.
	var appRedirect string
	if isMobile {
		challengeValue, _ := app.auth.GetSessionValue(r, oidcChallengeSessKey)
		challenge, _ := challengeValue.(string)
		if challenge == "" {
			return redirectToApp(r, "server_error")
		}
		loginCode, err := app.auth.MintLoginCode(r.RequestCtx, user.ID, challenge)
		if err != nil {
			return redirectToApp(r, "server_error")
		}
		appRedirect = mobileCallbackURL + "?code=" + url.QueryEscape(loginCode)
	} else if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Bookkeeping only: the login already succeeded, so a failure here must not fail the request.
	if err := app.user.UpdateLastLoginAt(user.ID); err != nil {
		app.lo.Error("error updating last login time", "error", err, "user_id", user.ID)
	}

	app.user.InvalidateAgentCache(user.ID)

	// Insert activity log.
	if err := app.activityLog.Login(user.ID, user.Email.String, ip); err != nil {
		app.lo.Error("error creating login activity log", "error", err)
	}

	if isMobile {
		return r.Redirect(appRedirect, fasthttp.StatusFound, nil, "")
	}

	// Read the 'next' parameter from session to redirect after login.
	nextParam, _ := app.auth.GetSessionValue(r, oidcNextSessKey)
	redirectURL := "/"
	if nextStr, ok := nextParam.(string); ok && nextStr != "" {
		redirectURL = nextStr
	}

	return r.RedirectURI(redirectURL, fasthttp.StatusFound, nil, "")
}

// redirectToApp hands a failure back to the mobile app, which is waiting on the custom scheme and
// would otherwise sit on a raw JSON error page inside the system browser.
func redirectToApp(r *fastglue.Request, reason string) error {
	return r.Redirect(mobileCallbackURL+"?error="+url.QueryEscape(reason), fasthttp.StatusFound, nil, "")
}
