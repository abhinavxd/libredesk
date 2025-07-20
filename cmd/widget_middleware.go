package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	// Context keys for storing authenticated widget data
	ctxWidgetClaims    = "widget_claims"
	ctxWidgetInboxID   = "widget_inbox_id"
	ctxWidgetContactID = "widget_contact_id"
	ctxWidgetInbox     = "widget_inbox"

	// Header sent in every widget request to identify the inbox
	hdrWidgetInboxID = "X-Libredesk-Inbox-ID"
)

// widgetAuth middleware authenticates widget requests using JWT and inbox validation.
// It always validates the inbox from X-Libredesk-Inbox-ID header, and conditionally validates JWT.
// For /conversations/init without JWT, it allows visitor creation while still validating inbox.
func widgetAuth(next func(*fastglue.Request) error) func(*fastglue.Request) error {
	return func(r *fastglue.Request) error {
		var (
			app = r.Context.(*App)
		)

		// Always extract and validate inbox_id from custom header
		inboxIDHeader := string(r.RequestCtx.Request.Header.Peek(hdrWidgetInboxID))
		if inboxIDHeader == "" {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
		}

		inboxID, err := strconv.Atoi(inboxIDHeader)
		if err != nil || inboxID <= 0 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
		}

		// Always fetch and validate inbox
		inbox, err := app.inbox.GetDBRecord(inboxID)
		if err != nil {
			app.lo.Error("error fetching inbox", "inbox_id", inboxID, "error", err)
			return sendErrorEnvelope(r, err)
		}

		if !inbox.Enabled {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
		}

		// Check if inbox is the correct type for widget requests
		if inbox.Channel != livechat.ChannelLiveChat {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
		}

		// Always store inbox data in context
		r.RequestCtx.SetUserValue(ctxWidgetInboxID, inboxID)
		r.RequestCtx.SetUserValue(ctxWidgetInbox, inbox)

		// Extract JWT from Authorization header (Bearer token)
		authHeader := string(r.RequestCtx.Request.Header.Peek("Authorization"))

		// For init endpoint, allow requests without JWT (visitor creation)
		if authHeader == "" && strings.Contains(string(r.RequestCtx.Path()), "/conversations/init") {
			return next(r)
		}

		// For all other requests, require JWT
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
		}
		jwtToken := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify JWT using inbox secret
		claims, err := verifyStandardJWT(jwtToken, inbox.Secret.String)
		if err != nil {
			app.lo.Error("invalid JWT", "jwt", jwtToken, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusUnauthorized, app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
		}

		// Resolve user/contact ID from JWT claims
		contactID, err := resolveUserIDFromClaims(app, claims)
		if err != nil {
			envErr, ok := err.(envelope.Error)
			if ok && envErr.ErrorType != envelope.NotFoundError {
				app.lo.Error("error resolving user ID from JWT claims", "error", err)
				return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.user}"), nil, envelope.GeneralError)
			}
		}

		// Store authenticated data in request context for downstream handlers
		r.RequestCtx.SetUserValue(ctxWidgetClaims, claims)
		r.RequestCtx.SetUserValue(ctxWidgetContactID, contactID)

		return next(r)
	}
}

// Helper functions to extract authenticated data from request context

// getWidgetInboxID extracts inbox ID from request context
func getWidgetInboxID(r *fastglue.Request) (int, error) {
	val := r.RequestCtx.UserValue(ctxWidgetInboxID)
	if val == nil {
		return 0, fmt.Errorf("widget middleware not applied: missing inbox ID in context")
	}
	inboxID, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("invalid inbox ID type in context")
	}
	return inboxID, nil
}

// getWidgetContactID extracts contact ID from request context
func getWidgetContactID(r *fastglue.Request) (int, error) {
	val := r.RequestCtx.UserValue(ctxWidgetContactID)
	if val == nil {
		return 0, fmt.Errorf("widget middleware not applied: missing contact ID in context")
	}
	contactID, ok := val.(int)
	if !ok {
		return 0, fmt.Errorf("invalid contact ID type in context")
	}
	return contactID, nil
}

// getWidgetInbox extracts inbox model from request context
func getWidgetInbox(r *fastglue.Request) (imodels.Inbox, error) {
	val := r.RequestCtx.UserValue(ctxWidgetInbox)
	if val == nil {
		return imodels.Inbox{}, fmt.Errorf("widget middleware not applied: missing inbox in context")
	}
	inbox, ok := val.(imodels.Inbox)
	if !ok {
		return imodels.Inbox{}, fmt.Errorf("invalid inbox type in context")
	}
	return inbox, nil
}

// getWidgetClaimsOptional extracts JWT claims from request context, returns nil if not set
func getWidgetClaimsOptional(r *fastglue.Request) *Claims {
	val := r.RequestCtx.UserValue(ctxWidgetClaims)
	if val == nil {
		return nil
	}
	if claims, ok := val.(Claims); ok {
		return &claims
	}
	return nil
}
