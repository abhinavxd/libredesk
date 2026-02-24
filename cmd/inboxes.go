package main

import (
	"encoding/json"
	"net/mail"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/oauth"
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
	for i := range inboxes {
		if err := inboxes[i].ClearPasswords(); err != nil {
			app.lo.Error("error clearing inbox passwords from response", "error", err)
			return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
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
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}

	// Trim whitespace from inbox fields and config.
	if err := trimInboxFields(&inbox); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}

	createdInbox, err := app.inbox.Create(inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := validateInbox(app, createdInbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := reloadInboxes(app); err != nil {
		app.lo.Error("error reloading inboxes", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning.
	if err := createdInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
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
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	if err := r.Decode(&inbox, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}

	// Trim whitespace from inbox fields and config.
	if err := trimInboxFields(&inbox); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}

	if err := validateInbox(app, inbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	updatedInbox, err := app.inbox.Update(id, inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := reloadInboxes(app); err != nil {
		app.lo.Error("error reloading inboxes", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning.
	if err := updatedInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
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
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	toggledInbox, err := app.inbox.Toggle(id)
	if err != nil {
		return err
	}

	if err := reloadInboxes(app); err != nil {
		app.lo.Error("error reloading inboxes", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning
	if err := toggledInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
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
		app.lo.Error("error reloading inboxes", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(true)
}

// validateInbox validates the inbox
func validateInbox(app *App, inbox imodels.Inbox) error {
	// Validate from address only for email channels.
	if inbox.Channel == "email" {
		if _, err := mail.ParseAddress(inbox.From); err != nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.invalidFromAddress"), nil)
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
				return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
			// Ensure linked inbox is an email channel
			if linkedInbox.Channel != "email" {
				return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
			// Ensure linked inbox is enabled
			if !linkedInbox.Enabled {
				return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)

			}
		}
	}

	// Validate email channel config.
	if inbox.Channel == "email" {
		if err := validateEmailConfig(app, inbox.Config); err != nil {
			return err
		}
	}
	return nil
}

// validateEmailConfig validates the email inbox configuration.
func validateEmailConfig(app *App, configJSON json.RawMessage) error {
	var cfg imodels.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Validate auth_type.
	if cfg.AuthType != "" && cfg.AuthType != imodels.AuthTypePassword && cfg.AuthType != imodels.AuthTypeOAuth2 {
		return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Validate OAuth config if auth_type is oauth2.
	if cfg.AuthType == imodels.AuthTypeOAuth2 {
		if cfg.OAuth == nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "oauth"), nil)
		}
		if cfg.OAuth.Provider != string(oauth.ProviderGoogle) && cfg.OAuth.Provider != string(oauth.ProviderMicrosoft) {
			return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if cfg.OAuth.ClientID == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "oauth.client_id"), nil)
		}
	}

	// Validate SMTP configs.
	for i, smtp := range cfg.SMTP {
		if smtp.Host == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "smtp.host"), nil)
		}
		if smtp.Port <= 0 {
			return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidPortValue"), nil)
		}
		// Validate auth_protocol for password auth.
		if cfg.AuthType != imodels.AuthTypeOAuth2 {
			validAuthProtocols := map[string]bool{"": true, "none": true, "plain": true, "login": true, "cram": true}
			if !validAuthProtocols[cfg.SMTP[i].AuthProtocol] {
				return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
		}
	}

	// Validate IMAP configs.
	for _, imap := range cfg.IMAP {
		if imap.Host == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "imap.host"), nil)
		}
		if imap.Port <= 0 {
			return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if imap.Mailbox == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "imap.mailbox"), nil)
		}
		// Validate tls_type.
		validTLSTypes := map[string]bool{"none": true, "starttls": true, "tls": true}
		if !validTLSTypes[imap.TLSType] {
			return envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}

	return nil
}

// trimInboxFields trims whitespace from inbox fields and its email config if applicable.
func trimInboxFields(inb *imodels.Inbox) error {
	inb.Name = strings.TrimSpace(inb.Name)
	inb.From = strings.TrimSpace(inb.From)

	// Trim email config fields if this is an email channel.
	if inb.Channel == inbox.ChannelEmail && len(inb.Config) > 0 {
		var cfg imodels.Config
		if err := json.Unmarshal(inb.Config, &cfg); err != nil {
			return err
		}
		trimEmailConfig(&cfg)
		trimmedConfig, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		inb.Config = trimmedConfig
	}
	return nil
}

// trimEmailConfig trims whitespace from email configuration fields.
// Passwords and secrets are intentionally NOT trimmed.
func trimEmailConfig(cfg *imodels.Config) {
	// Trim IMAP configs.
	for i := range cfg.IMAP {
		cfg.IMAP[i].Host = strings.TrimSpace(cfg.IMAP[i].Host)
		cfg.IMAP[i].Username = strings.TrimSpace(cfg.IMAP[i].Username)
		cfg.IMAP[i].Mailbox = strings.TrimSpace(cfg.IMAP[i].Mailbox)
	}

	// Trim SMTP configs.
	for i := range cfg.SMTP {
		cfg.SMTP[i].Host = strings.TrimSpace(cfg.SMTP[i].Host)
		cfg.SMTP[i].Username = strings.TrimSpace(cfg.SMTP[i].Username)
		cfg.SMTP[i].HelloHostname = strings.TrimSpace(cfg.SMTP[i].HelloHostname)
	}

	// Trim OAuth config.
	if cfg.OAuth != nil {
		cfg.OAuth.Provider = strings.TrimSpace(cfg.OAuth.Provider)
		cfg.OAuth.ClientID = strings.TrimSpace(cfg.OAuth.ClientID)
		cfg.OAuth.TenantID = strings.TrimSpace(cfg.OAuth.TenantID)
	}
}
