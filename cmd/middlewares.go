package main

import (
	"net/http"
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/simplesessions/v3"
)

// authenticateUser handles both API key and session-based authentication
// Returns the authenticated user or an error
// For session-based auth, CSRF is checked for POST/PUT/DELETE requests
func authenticateUser(r *fastglue.Request, app *App) (models.User, error) {
	var user models.User

	// Check for Authorization header first (API key authentication)
	apiKey, apiSecret, err := r.ParseAuthHeader(fastglue.AuthBasic | fastglue.AuthToken)
	if err == nil && len(apiKey) > 0 && len(apiSecret) > 0 {
		user, err = app.user.ValidateAPIKey(string(apiKey), string(apiSecret))
		if err != nil {
			return user, err
		}
		return user, nil
	}

	// Session-based authentication - Check CSRF first.
	method := string(r.RequestCtx.Method())
	if method == "POST" || method == "PUT" || method == "DELETE" {
		cookieToken := string(r.RequestCtx.Request.Header.Cookie("csrf_token"))
		hdrToken := string(r.RequestCtx.Request.Header.Peek("X-CSRFTOKEN"))

		// Match CSRF token from cookie and header.
		if cookieToken == "" || hdrToken == "" || cookieToken != hdrToken {
			app.lo.Error("csrf token mismatch", "method", method, "cookie_token", cookieToken, "header_token", hdrToken)
			return user, envelope.NewError(envelope.PermissionError, app.i18n.T("auth.csrfTokenMismatch"), nil)
		}
	}

	// Validate session and fetch user.
	sessUser, err := app.auth.ValidateSession(r)
	if err != nil || sessUser.ID <= 0 {
		app.lo.Error("error validating session", "error", err)
		return user, envelope.NewError(envelope.GeneralError, app.i18n.T("auth.invalidOrExpiredSession"), nil)
	}

	// Get agent user from cache or load it.
	user, err = app.user.GetAgentCachedOrLoad(sessUser.ID)
	if err != nil {
		return user, err
	}

	// Destroy session if user is disabled.
	if !user.Enabled {
		if err := app.auth.DestroySession(r); err != nil {
			app.lo.Error("error destroying session", "error", err)
		}
		return user, envelope.NewError(envelope.PermissionError, app.i18n.T("user.accountDisabled"), nil)
	}

	return user, nil
}

// tryAuth attempts to authenticate the user and add them to the context but doesn't enforce authentication.
// Handlers can check if user exists in context optionally.
// Supports both API key authentication (Authorization header) and session-based authentication.
func tryAuth(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Try to authenticate user using shared authentication logic, but don't return errors
		user, err := authenticateUser(r, app)
		if err != nil {
			// Authentication failed, but this is optional, so continue without user
			return handler(r)
		}

		// Set user in context if authentication succeeded.
		r.RequestCtx.SetUserValue("user", amodels.User{
			ID:        user.ID,
			Email:     user.Email.String,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		})

		return handler(r)
	}
}

// auth validates the session or API key and adds the user to the request context.
// Supports both API key authentication (Authorization header) and session-based authentication.
func auth(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var app = r.Context.(*App)

		// Authenticate user using shared authentication logic
		user, err := authenticateUser(r, app)
		if err != nil {
			if envErr, ok := err.(envelope.Error); ok {
				if envErr.ErrorType == envelope.PermissionError {
					return r.SendErrorEnvelope(http.StatusForbidden, envErr.Message, nil, envelope.PermissionError)
				}
				return r.SendErrorEnvelope(http.StatusUnauthorized, envErr.Message, nil, envelope.GeneralError)
			}
			return sendErrorEnvelope(r, err)
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

// perm checks if the user has the required permission to access the endpoint.
// Supports both API key authentication (Authorization header) and session-based authentication.
func perm(handler fastglue.FastRequestHandler, perm string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var app = r.Context.(*App)

		// Authenticate user using shared authentication logic
		user, err := authenticateUser(r, app)
		if err != nil {
			if envErr, ok := err.(envelope.Error); ok {
				if envErr.ErrorType == envelope.PermissionError {
					return r.SendErrorEnvelope(http.StatusForbidden, envErr.Message, nil, envelope.PermissionError)
				}
				return r.SendErrorEnvelope(http.StatusUnauthorized, envErr.Message, nil, envelope.GeneralError)
			}
			return sendErrorEnvelope(r, err)
		}

		// Split the permission string into object and action and enforce it.
		parts := strings.Split(perm, ":")
		if len(parts) != 2 {
			return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.permission}"), nil, envelope.GeneralError)
		}
		object, action := parts[0], parts[1]
		ok, err := app.authz.Enforce(user, object, action)
		if err != nil {
			return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.errorChecking", "name", "{globals.terms.permission}"), nil, envelope.GeneralError)
		}
		if !ok {
			return r.SendErrorEnvelope(http.StatusForbidden, app.i18n.Ts("globals.messages.denied", "name", "{globals.terms.permission}"), nil, envelope.PermissionError)
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
				return r.SendErrorEnvelope(http.StatusUnauthorized, app.i18n.Ts("globals.messages.errorValidating", "name", "{globals.terms.session}"), nil, envelope.GeneralError)
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

// authOrSignedURL allows access if user is authenticated OR if URL has valid signature.
// Used for media endpoints that support both access methods.
func authOrSignedURL(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// First, try to authenticate normally.
		user, err := authenticateUser(r, app)
		if err == nil && user.ID > 0 {
			// User is authenticated, set user context and proceed.
			r.RequestCtx.SetUserValue("user", amodels.User{
				ID:        user.ID,
				Email:     user.Email.String,
				FirstName: user.FirstName,
				LastName:  user.LastName,
			})
			r.RequestCtx.SetUserValue("auth_method", "session")
			return handler(r)
		}

		// Authentication failed, check for signed URL.
		validator := app.media.SignedURLValidator()
		if validator == nil {
			// Store doesn't support signed URLs, require auth.
			return r.SendErrorEnvelope(http.StatusUnauthorized,
				app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.GeneralError)
		}

		// Parse signature and expiry from query params.
		sig := string(r.RequestCtx.QueryArgs().Peek("sig"))
		expStr := string(r.RequestCtx.QueryArgs().Peek("exp"))

		if sig == "" || expStr == "" {
			return r.SendErrorEnvelope(http.StatusUnauthorized,
				app.i18n.T("auth.invalidOrExpiredSession"), nil, envelope.GeneralError)
		}

		exp, err := strconv.ParseInt(expStr, 10, 64)
		if err != nil {
			return r.SendErrorEnvelope(http.StatusBadRequest,
				app.i18n.Ts("globals.messages.invalid", "name", "expiry"), nil, envelope.InputError)
		}

		// Get the UUID from the route.
		uuid := r.RequestCtx.UserValue("uuid").(string)

		// Validate signature.
		if !validator(uuid, sig, exp) {
			return r.SendErrorEnvelope(http.StatusForbidden,
				app.i18n.T("media.invalidOrExpiredURL"), nil, envelope.PermissionError)
		}

		// Mark as signed URL access (no user context).
		r.RequestCtx.SetUserValue("auth_method", "signed_url")
		return handler(r)
	}
}
