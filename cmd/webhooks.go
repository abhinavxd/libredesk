package main

import (
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetWebhooks returns all webhooks from the database.
func handleGetWebhooks(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	webhooks, err := app.webhook.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Hide secrets.
	for i := range webhooks {
		if webhooks[i].Secret != "" {
			webhooks[i].Secret = strings.Repeat(stringutil.PasswordDummy, 10)
		}
	}
	return r.SendEnvelope(webhooks)
}

// handleGetWebhook returns a specific webhook by ID.
func handleGetWebhook(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	webhook, err := app.webhook.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Hide secret in the response.
	if webhook.Secret != "" {
		webhook.Secret = strings.Repeat(stringutil.PasswordDummy, 10)
	}

	return r.SendEnvelope(webhook)
}

// handleCreateWebhook creates a new webhook in the database.
func handleCreateWebhook(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		webhook = models.Webhook{}
	)
	if err := r.Decode(&webhook, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	// Validate webhook fields
	if err := validateWebhook(app, webhook); err != nil {
		return r.SendEnvelope(err)
	}

	_, err := app.webhook.Create(webhook)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateWebhook updates an existing webhook in the database.
func handleUpdateWebhook(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		webhook = models.Webhook{}
		id, _   = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&webhook, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	// Validate webhook fields
	if err := validateWebhook(app, webhook); err != nil {
		return r.SendEnvelope(err)
	}

	// If secret is empty or contains dummy characters, fetch existing webhook and preserve the secret
	if webhook.Secret == "" || strings.Contains(webhook.Secret, stringutil.PasswordDummy) {
		existingWebhook, err := app.webhook.Get(id)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		webhook.Secret = existingWebhook.Secret
	}

	if err := app.webhook.Update(id, webhook); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleDeleteWebhook deletes a webhook from the database.
func handleDeleteWebhook(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := app.webhook.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleToggleWebhook toggles the active status of a webhook.
func handleToggleWebhook(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := app.webhook.Toggle(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleTestWebhook sends a test payload to a webhook.
func handleTestWebhook(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := app.webhook.SendTestWebhook(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// validateWebhook validates the webhook data.
func validateWebhook(app *App, webhook models.Webhook) error {
	if webhook.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	if webhook.URL == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`url`"), nil)
	}
	if len(webhook.Events) == 0 {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`events`"), nil)
	}
	return nil
}
