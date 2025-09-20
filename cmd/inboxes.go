package main

import (
	"encoding/json"
	"net/mail"
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetInboxes returns all inboxes
func handleGetInboxes(r *fastglue.Request) error {
	var app = r.Context.(*App)
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(inboxes)
}

// handleGetInbox returns an inbox by ID
func handleGetInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	inbox, err := app.inbox.GetDBRecord(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := inbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil)
	}
	return r.SendEnvelope(inbox)
}

// handleCreateInbox creates a new inbox
func handleCreateInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		inbox = imodels.Inbox{}
	)
	if err := r.Decode(&inbox, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	createdInbox, err := app.inbox.Create(inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := validateInbox(app, createdInbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning.
	if err := createdInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil)
	}

	return r.SendEnvelope(createdInbox)
}

// handleUpdateInbox updates an inbox
func handleUpdateInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		inbox = imodels.Inbox{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&inbox, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	if err := validateInbox(app, inbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	updatedInbox, err := app.inbox.Update(id, inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning.
	if err := updatedInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil)
	}

	return r.SendEnvelope(updatedInbox)
}

// handleToggleInbox toggles an inbox
func handleToggleInbox(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	toggledInbox, err := app.inbox.Toggle(id)
	if err != nil {
		return err
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning
	if err := toggledInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.inbox}"), nil)
	}

	return r.SendEnvelope(toggledInbox)
}

// handleDeleteInbox deletes an inbox
func handleDeleteInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	err := app.inbox.SoftDelete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.couldNotReload", "name", "{globals.terms.inbox}"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(true)
}

// validateInbox validates the inbox
func validateInbox(app *App, inbox imodels.Inbox) error {
	// Validate from address only for email channels.
	if inbox.Channel == "email" {
		if _, err := mail.ParseAddress(inbox.From); err != nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalidFromAddress"), nil)
		}
	}
	if len(inbox.Config) == 0 {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "config"), nil)
	}
	if inbox.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "name"), nil)
	}
	if inbox.Channel == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "channel"), nil)
	}

	// Validate livechat-specific configuration
	if inbox.Channel == livechat.ChannelLiveChat {
		var config livechat.Config
		if err := json.Unmarshal(inbox.Config, &config); err == nil {
			// ShowOfficeHoursAfterAssignment cannot be enabled if ShowOfficeHoursInChat is disabled
			if config.ShowOfficeHoursAfterAssignment && !config.ShowOfficeHoursInChat {
				return envelope.NewError(envelope.InputError, "`show_office_hours_after_assignment` cannot be enabled when `show_office_hours_in_chat` is disabled", nil)
			}
		}

		// Validate linked email inbox if specified
		if inbox.LinkedEmailInboxID.Valid {
			linkedInbox, err := app.inbox.GetDBRecord(int(inbox.LinkedEmailInboxID.Int))
			if err != nil {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "linked_email_inbox_id"), nil)
			}
			// Ensure linked inbox is an email channel
			if linkedInbox.Channel != "email" {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "linked_email_inbox_id"), nil)
			}
			// Ensure linked inbox is enabled
			if !linkedInbox.Enabled {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "linked_email_inbox_id"), nil)
			}
		}
	}

	return nil
}
