package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/httputil"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/zerodha/fastglue"
)

type inboxReq struct {
	InboxID int `json:"inbox_id"`
}

// widgetOrigin middleware validates the Origin header against trusted domains configured in the live chat inbox settings.
func widgetOrigin(next fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Get the Origin header from the request
		origin := string(r.RequestCtx.Request.Header.Peek("Origin"))

		// If no origin header is present, allow direct access.
		if origin == "" {
			return next(r)
		}

		// Extract inbox ID from request
		var inboxID int

		// Search for inbox_id in query parameters first.
		if qInboxID := r.RequestCtx.QueryArgs().GetUintOrZero("inbox_id"); qInboxID > 0 {
			inboxID = qInboxID
		} else {
			// For POST/PUT requests, try to decode the body to get `inbox_id`, every request sends it.
			if r.RequestCtx.IsPost() || r.RequestCtx.IsPut() {
				inboxReq := inboxReq{}
				bodyBytes := r.RequestCtx.Request.Body()
				if len(bodyBytes) > 0 {
					if err := json.Unmarshal(bodyBytes, &inboxReq); err == nil {
						inboxID = inboxReq.InboxID
					}
				}
			}

			// Check multipart form data for `inbox_id` if not found in query or body.
			if inboxID == 0 {
				if r.RequestCtx.Request.Header.IsPost() {
					form, err := r.RequestCtx.MultipartForm()
					if err == nil && form != nil && form.Value != nil {
						if inboxIDValues, exists := form.Value["inbox_id"]; exists && len(inboxIDValues) > 0 {
							if parsedID, parseErr := strconv.Atoi(inboxIDValues[0]); parseErr == nil && parsedID > 0 {
								inboxID = parsedID
							}
						}
					}
				}
			}
		}

		// If we can't determine the inbox ID, disallow the request
		if inboxID <= 0 {
			app.lo.Warn("widget request without inbox_id blocked", "path", string(r.RequestCtx.Path()))
			return r.SendErrorEnvelope(http.StatusBadRequest, app.i18n.Ts("globals.messages.required", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
		}

		// Get inbox configuration
		inbox, err := app.inbox.GetDBRecord(inboxID)
		if err != nil {
			app.lo.Error("error fetching inbox for origin check", "inbox_id", inboxID, "error", err)
			return r.SendErrorEnvelope(http.StatusNotFound, app.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.inbox}"), nil, envelope.NotFoundError)
		}

		if !inbox.Enabled {
			return r.SendErrorEnvelope(http.StatusBadRequest, app.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil, envelope.InputError)
		}

		// Parse the live chat config
		var config livechat.Config
		if err := json.Unmarshal(inbox.Config, &config); err != nil {
			app.lo.Error("error parsing live chat config for origin check", "error", err)
			return r.SendErrorEnvelope(http.StatusInternalServerError, app.i18n.Ts("globals.messages.invalid", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
		}

		// If trusted domains list is empty, allow all origins
		if len(config.TrustedDomains) == 0 {
			return next(r)
		}

		// Check if the origin matches any of the trusted domains
		if !httputil.IsOriginTrusted(origin, config.TrustedDomains) {
			app.lo.Warn("widget request from untrusted origin blocked",
				"origin", origin,
				"inbox_id", inboxID,
				"trusted_domains", config.TrustedDomains)
			return r.SendErrorEnvelope(http.StatusForbidden, "Widget not allowed from this origin: "+origin, nil, envelope.PermissionError)
		}

		app.lo.Debug("widget request from trusted origin allowed", "origin", origin, "inbox_id", inboxID)

		return next(r)
	}
}
