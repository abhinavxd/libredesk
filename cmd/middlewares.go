package main

import (
	"net/http"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/simplesessions/v3"
)

// tryAuth is a middleware that attempts to authenticate the user and add them to the context
// but doesn't enforce authentication. Handlers can check if user exists in context optionally.
func tryAuth(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Try to validate session without returning error.
		userSession, err := app.auth.ValidateSession(r)
		if err != nil || userSession.ID <= 0 {
			return handler(r)
		}

		// Try to get user.
		user, err := app.user.GetAgent(userSession.ID)
		if err != nil {
			return handler(r)
		}

		// Set user in context if found.
		r.RequestCtx.SetUserValue("user", amodels.User{
			ID:        user.ID,
			Email:     user.Email.String,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		})

		return handler(r)
	}
}

// auth makes sure the user is logged in.
func auth(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var app = r.Context.(*App)

		// Validate session and fetch user.
		userSession, err := app.auth.ValidateSession(r)
		if err != nil || userSession.ID <= 0 {
			app.lo.Error("error validating session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.GeneralError)
		}

		// Set user in the request context.
		user, err := app.user.GetAgent(userSession.ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		r.RequestCtx.SetUserValue("user", amodels.User{
			ID:        user.ID,
			Email:     user.Email.String,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		})

		return handler(r)
	}
}

// perm does session validation, CSRF, and permission enforcement.
func perm(handler fastglue.FastRequestHandler, perm string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			app         = r.Context.(*App)
			cookieToken = string(r.RequestCtx.Request.Header.Cookie("csrf_token"))
			hdrToken    = string(r.RequestCtx.Request.Header.Peek("X-CSRFTOKEN"))
		)

		if cookieToken == "" || hdrToken == "" || cookieToken != hdrToken {
			app.lo.Error("csrf token mismatch", "cookie_token", cookieToken, "header_token", hdrToken)
			return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.T("auth.csrfTokenMismatch"), nil, envelope.PermissionError)
		}

		// Validate session and fetch user.
		sessUser, err := app.auth.ValidateSession(r)
		if err != nil || sessUser.ID <= 0 {
			app.lo.Error("error validating session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.GeneralError)
		}

		// Get user from DB.
		user, err := app.user.GetAgent(sessUser.ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		// Destroy session if user is disabled.
		if !user.Enabled {
			if err := app.auth.DestroySession(r); err != nil {
				app.lo.Error("error destroying session", "error", err)
			}
			return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("user.accountDisabled"), nil, envelope.PermissionError)
		}

		// Split the permission string into object and action and enforce it.
		parts := strings.Split(perm, ":")
		if len(parts) != 2 {
			return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "{globals.entities.permission}"), nil, envelope.GeneralError)
		}
		object, action := parts[0], parts[1]
		ok, err := app.authz.Enforce(user, object, action)
		if err != nil {
			return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorChecking", "name", "{globals.entities.permission}"), nil, envelope.GeneralError)
		}
		if !ok {
			return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.T("authz.permissionDenied"), nil, envelope.PermissionError)
		}

		// Set user in the request context.
		r.RequestCtx.SetUserValue("user", amodels.User{
			ID:        user.ID,
			Email:     user.Email.String,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		})

		return handler(r)
	}
}

// authPage ensures the user is logged in; otherwise, redirects to the login page.
func authPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Validate session.
		user, err := app.auth.ValidateSession(r)
		if err != nil {
			// Session is not valid, destroy it and redirect to login.
			if err != simplesessions.ErrInvalidSession {
				app.lo.Error("error validating session", "error", err)
				return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.Ts("globals.messages.errorValidating", "name", "{globals.entities.session}"), nil, envelope.GeneralError)
			}
			if err := app.auth.DestroySession(r); err != nil {
				app.lo.Error("error destroying session", "error", err)
			}
		}

		// User is authenticated.
		if user.ID > 0 {
			return handler(r)
		}

		nextURI := r.RequestCtx.QueryArgs().Peek("next")
		if len(nextURI) == 0 {
			nextURI = r.RequestCtx.RequestURI()
		}
		return r.RedirectURI("/", fasthttp.StatusFound, map[string]any{
			"next": string(nextURI),
		}, "")
	}
}

// notAuthPage allows access only if the user is not authenticated; otherwise, redirects to the user inbox.
func notAuthPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Validate session.
		user, err := app.auth.ValidateSession(r)
		if err != nil {
			app.lo.Error("error validating session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.T("auth.invalidOrExpiredSessionClearCookie"), nil, envelope.GeneralError)
		}

		if user.ID != 0 {
			nextURI := string(r.RequestCtx.QueryArgs().Peek("next"))
			if nextURI == "" {
				nextURI = "/inboxes/assigned"
			}
			return r.RedirectURI(nextURI, fasthttp.StatusFound, nil, "")
		}
		return handler(r)
	}
}
