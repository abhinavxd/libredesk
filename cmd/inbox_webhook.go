package main

import (
	"context"
	"net/http"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// inboundWebhookTimeout bounds a single inbound webhook, including fetching the full message
// from the provider's API.
const inboundWebhookTimeout = 60 * time.Second

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

	ctx, cancel := context.WithTimeout(context.Background(), inboundWebhookTimeout)
	defer cancel()

	if err := webhookReceiver.ReceiveWebhook(ctx, headers, body); err != nil {
		app.lo.Error("error processing inbound email webhook", "inbox_id", inboxRecord.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	return r.SendEnvelope(true)
}
