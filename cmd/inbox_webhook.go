package main

import (
	"net/http"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleInboundEmailWebhook receives inbound-email webhook calls from an HTTPS email API
// provider (e.g. Resend) configured on an inbox using the http_api transport. It verifies
// the provider's signature and enqueues the parsed message for processing, the same way
// IMAP polling does for the smtp_imap transport.
//
// The response is deliberately generic (404/400/200) so this endpoint doesn't leak which
// inbox UUIDs exist or why a given payload was rejected.
func handleInboundEmailWebhook(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)

	inboxRecord, err := app.inbox.GetDBRecord(uuid)
	if err != nil || inboxRecord.Channel != inbox.ChannelEmail || !inboxRecord.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("validation.notFoundInbox"), nil, envelope.InputError)
	}

	liveInbox, err := app.inbox.Get(inboxRecord.ID)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("validation.notFoundInbox"), nil, envelope.InputError)
	}

	webhookReceiver, ok := liveInbox.(inbox.WebhookReceiver)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("validation.notFoundInbox"), nil, envelope.InputError)
	}

	headers := make(http.Header)
	r.RequestCtx.Request.Header.VisitAll(func(key, value []byte) {
		headers.Add(string(key), string(value))
	})
	body := append([]byte(nil), r.RequestCtx.PostBody()...)

	if err := webhookReceiver.ReceiveWebhook(headers, body); err != nil {
		app.lo.Error("error processing inbound email webhook", "inbox_id", inboxRecord.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	return r.SendEnvelope(true)
}
