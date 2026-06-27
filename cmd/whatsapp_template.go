package main

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	whatsappChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/whatsapp"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	wtmodels "github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const whatsAppTemplateSyncInterval = 6 * time.Hour

// whatsappTemplateSyncWorker periodically mirrors template status from Meta.
func whatsappTemplateSyncWorker(ctx context.Context, app *App) {
	initial := time.NewTimer(2 * time.Minute)
	defer initial.Stop()
	select {
	case <-ctx.Done():
		return
	case <-initial.C:
		syncAllWhatsAppTemplates(ctx, app)
	}

	ticker := time.NewTicker(whatsAppTemplateSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncAllWhatsAppTemplates(ctx, app)
		}
	}
}

func syncAllWhatsAppTemplates(ctx context.Context, app *App) {
	if app.whatsappTemplate == nil {
		return
	}
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return
	}
	for _, rec := range inboxes {
		if rec.Channel != whatsappChannel.ChannelWhatsApp || !rec.Enabled {
			continue
		}
		syncCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		if _, err := app.whatsappTemplate.SyncFromMeta(syncCtx, rec.ID); err != nil {
			app.lo.Warn("periodic whatsapp template sync failed", "inbox_id", rec.ID, "error", err)
		}
		cancel()
	}
}

// makeWhatsAppAuthErrorHook flags the matching inbox when Meta rejects its token.
func makeWhatsAppAuthErrorHook(app *App) func(acc whatsapp.Account) {
	return func(acc whatsapp.Account) {
		inboxes, err := app.inbox.GetAll()
		if err != nil {
			return
		}
		for _, rec := range inboxes {
			if rec.Channel != whatsappChannel.ChannelWhatsApp {
				continue
			}
			var cfg whatsappChannel.Config
			if err := json.Unmarshal(rec.Config, &cfg); err != nil || cfg.PhoneNumberID != acc.PhoneNumberID {
				continue
			}
			if _, flagged := app.inboxAuthErrors.LoadOrStore(rec.ID, time.Now()); !flagged {
				app.lo.Error("whatsapp access token rejected by meta, sends and media downloads will fail until the token is replaced", "inbox_id", rec.ID)
			}
			return
		}
	}
}

// handleListWhatsAppTemplates lists templates for a given inbox (?inbox_id=123).
func handleListWhatsAppTemplates(r *fastglue.Request) error {
	app := r.Context.(*App)
	if whatsAppTemplateUnavailable(r, app) {
		return nil
	}
	inboxIDRaw := string(r.RequestCtx.QueryArgs().Peek("inbox_id"))
	inboxID, err := strconv.Atoi(inboxIDRaw)
	if err != nil || inboxID == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "inbox_id is required", nil, envelope.InputError)
	}
	templates, err := app.whatsappTemplate.GetByInbox(inboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(templates)
}

// handleGetWhatsAppTemplate returns a single template by id.
func handleGetWhatsAppTemplate(r *fastglue.Request) error {
	app := r.Context.(*App)
	if whatsAppTemplateUnavailable(r, app) {
		return nil
	}
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid id", nil, envelope.InputError)
	}
	t, err := app.whatsappTemplate.GetByID(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

// handleCreateWhatsAppTemplate stores a new template and submits it to Meta.
func handleCreateWhatsAppTemplate(r *fastglue.Request) error {
	app := r.Context.(*App)
	if whatsAppTemplateUnavailable(r, app) {
		return nil
	}
	var t wtmodels.Template
	if err := r.Decode(&t, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid request", nil, envelope.InputError)
	}
	if t.InboxID == 0 || t.Name == "" || t.Language == "" || t.Category == "" || t.BodyContent == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "inbox_id, name, language, category, body_content are required", nil, envelope.InputError)
	}
	created, err := app.whatsappTemplate.Create(r.RequestCtx, t)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(created)
}

// handleDeleteWhatsAppTemplate removes a template locally and on Meta.
func handleDeleteWhatsAppTemplate(r *fastglue.Request) error {
	app := r.Context.(*App)
	if whatsAppTemplateUnavailable(r, app) {
		return nil
	}
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "invalid id", nil, envelope.InputError)
	}
	if err := app.whatsappTemplate.Delete(r.RequestCtx, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]string{"status": "deleted"})
}

// handleSyncWhatsAppTemplates pulls templates from Meta and upserts locally.
func handleSyncWhatsAppTemplates(r *fastglue.Request) error {
	app := r.Context.(*App)
	if whatsAppTemplateUnavailable(r, app) {
		return nil
	}
	inboxIDRaw := string(r.RequestCtx.QueryArgs().Peek("inbox_id"))
	inboxID, err := strconv.Atoi(inboxIDRaw)
	if err != nil || inboxID == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "inbox_id is required", nil, envelope.InputError)
	}
	count, err := app.whatsappTemplate.SyncFromMeta(r.RequestCtx, inboxID)
	if err != nil {
		app.lo.Error("error syncing whatsapp templates", "inbox_id", inboxID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, err.Error(), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(map[string]int{"synced": count})
}

func whatsAppTemplateUnavailable(r *fastglue.Request, app *App) bool {
	if app.whatsappTemplate != nil {
		return false
	}
	r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "whatsapp not configured", nil, envelope.GeneralError)
	return true
}
