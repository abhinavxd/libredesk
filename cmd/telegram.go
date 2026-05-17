package main

import (
	"encoding/json"
	"strconv"

	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleTelegramWebhook(r *fastglue.Request) error {
	var app = r.Context.(*App)

	inboxID, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || inboxID == 0 {
		r.RequestCtx.SetStatusCode(fasthttp.StatusBadRequest)
		return nil
	}

	inb, err := app.inbox.Get(inboxID)
	if err != nil {
		r.RequestCtx.SetStatusCode(fasthttp.StatusNotFound)
		return nil
	}

	if inb.Channel() != inbox.ChannelTelegram {
		r.RequestCtx.SetStatusCode(fasthttp.StatusBadRequest)
		return nil
	}

	tgInbox, ok := inb.(*telegram.Telegram)
	if !ok {
		r.RequestCtx.SetStatusCode(fasthttp.StatusInternalServerError)
		return nil
	}

	// Verify webhook authenticity using X-Telegram-Bot-Api-Secret-Token header.
	secretToken := string(r.RequestCtx.Request.Header.Peek("X-Telegram-Bot-Api-Secret-Token"))
	if !tgInbox.VerifyWebhook(secretToken) {
		r.RequestCtx.SetStatusCode(fasthttp.StatusUnauthorized)
		return nil
	}

	var update telegram.Update
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &update); err != nil {
		r.RequestCtx.SetStatusCode(fasthttp.StatusBadRequest)
		return nil
	}

	if err := tgInbox.ProcessWebhookUpdate(update); err != nil {
		app.lo.Error("telegram webhook: error processing update", "error", err)
	}

	r.RequestCtx.SetStatusCode(fasthttp.StatusOK)
	return nil
}
