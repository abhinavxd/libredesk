package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/httputil"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/oauth"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	whatsappChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/whatsapp"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	wtmodels "github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
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
		setWebhookURL(app, &inboxes[i])
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
	setWebhookURL(app, &inbox)
	return r.SendEnvelope(inbox)
}

// setWebhookURL populates the computed webhook_url field for channels that receive inbound events over HTTP (currently WhatsApp only).
func setWebhookURL(app *App, inb *imodels.Inbox) {
	if inb.Channel != whatsappChannel.ChannelWhatsApp {
		return
	}
	url := whatsAppCallbackURL(app, inb.ID)
	if url == "" {
		return
	}
	inb.WebhookURL = url
	_, inb.TokenInvalid = app.whatsappAuthErrors.Load(inb.ID)
}

func whatsAppCallbackURL(app *App, inboxID int) string {
	root, err := app.setting.GetAppRootURL()
	if err != nil || root == "" {
		return ""
	}
	return strings.TrimRight(root, "/") + "/webhooks/whatsapp/" + strconv.Itoa(inboxID)
}

// subscribeWhatsAppWebhook best-effort points the WABA's webhook at this inbox; the manual Meta dashboard setup stays as fallback.
func subscribeWhatsAppWebhook(app *App, inboxID int) {
	cfg, err := whatsAppConfigForInbox(app, inboxID)
	if err != nil || app.whatsappClient == nil {
		return
	}
	callbackURL := whatsAppCallbackURL(app, inboxID)
	if callbackURL == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.whatsappClient.SubscribeWebhook(ctx, cfg.Account(), callbackURL, cfg.WebhookVerifyToken); err != nil {
		app.lo.Warn("automatic whatsapp webhook subscription failed, configure the webhook manually in the Meta dashboard", "inbox_id", inboxID, "callback_url", callbackURL, "error", err)
		return
	}
	app.lo.Info("whatsapp webhook subscribed automatically", "inbox_id", inboxID, "callback_url", callbackURL)
}

func validateWhatsAppCredentials(r *fastglue.Request, app *App, inb imodels.Inbox) error {
	if inb.Channel != whatsappChannel.ChannelWhatsApp || app.whatsappClient == nil {
		return nil
	}
	var cfg whatsappChannel.Config
	if err := json.Unmarshal(inb.Config, &cfg); err != nil {
		return nil
	}
	if err := app.whatsappClient.ValidateCredentials(r.RequestCtx, cfg.Account()); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, fmt.Sprintf("meta credential check failed: %s", err.Error()), nil, envelope.InputError)
	}
	return nil
}

// ensureWhatsAppCSATTemplate provisions the inbox's reserved CSAT template on Meta; approval arrives via webhook/sync.
func ensureWhatsAppCSATTemplate(app *App, inboxID int) {
	if app.whatsappTemplate == nil {
		return
	}
	name := wtmodels.CSATTemplateName(inboxID)
	exists, err := app.whatsappTemplate.Exists(inboxID, name, wtmodels.CSATTemplateLanguage)
	if err != nil || exists {
		return
	}
	root, err := app.setting.GetAppRootURL()
	if err != nil || root == "" {
		return
	}
	buttons, err := json.Marshal([]map[string]string{{
		"type": "URL",
		"text": app.i18n.T("conversation.whatsapp.csatTemplateButton"),
		"url":  strings.TrimRight(root, "/") + "/csat/{{1}}",
	}})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := app.whatsappTemplate.Create(ctx, wtmodels.Template{
		InboxID:     inboxID,
		Name:        name,
		Language:    wtmodels.CSATTemplateLanguage,
		Category:    wtmodels.CategoryUtility,
		BodyContent: app.i18n.T("conversation.whatsapp.csatTemplateBody"),
		Buttons:     buttons,
	}); err != nil {
		app.lo.Warn("error provisioning whatsapp csat template", "inbox_id", inboxID, "error", err)
		return
	}
	app.lo.Info("whatsapp csat template submitted to meta", "inbox_id", inboxID, "name", name)
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

	if err := validateInbox(app, inbox, false); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := validateWhatsAppCredentials(r, app, inbox); err != nil {
		return err
	}

	createdInbox, err := app.inbox.Create(inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := reloadInbox(app, createdInbox.ID); err != nil {
		app.lo.Error("error reloading inbox", "id", createdInbox.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	if createdInbox.Channel == whatsappChannel.ChannelWhatsApp {
		subscribeWhatsAppWebhook(app, createdInbox.ID)
		if createdInbox.CSATEnabled {
			ensureWhatsAppCSATTemplate(app, createdInbox.ID)
		}
	}

	// Clear passwords before returning.
	if err := createdInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	setWebhookURL(app, &createdInbox)

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

	if err := validateInbox(app, inbox, true); err != nil {
		return sendErrorEnvelope(r, err)
	}

	updatedInbox, err := app.inbox.Update(id, inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Validated after Update merges back the masked secrets; a failure keeps the new config in the DB so the agent can re-edit, matching the create path.
	if err := validateWhatsAppCredentials(r, app, updatedInbox); err != nil {
		return err
	}

	if err := reloadInbox(app, id); err != nil {
		app.lo.Error("error reloading inbox", "id", id, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	if updatedInbox.Channel == whatsappChannel.ChannelWhatsApp {
		subscribeWhatsAppWebhook(app, id)
		app.whatsappAuthErrors.Delete(id)
		if updatedInbox.CSATEnabled {
			ensureWhatsAppCSATTemplate(app, id)
		}
	}

	// Clear passwords before returning.
	if err := updatedInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	setWebhookURL(app, &updatedInbox)

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

	if err := reloadInbox(app, id); err != nil {
		app.lo.Error("error reloading inbox", "id", id, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Clear passwords before returning
	if err := toggledInbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing inbox passwords from response", "error", err)
		return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	setWebhookURL(app, &toggledInbox)

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
	if err := reloadInbox(app, id); err != nil {
		app.lo.Error("error reloading inbox", "id", id, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	return r.SendEnvelope(true)
}

// validateInbox validates the inbox
func validateInbox(app *App, inbox imodels.Inbox, isUpdate bool) error {
	// Validate from address only for email channels.
	if inbox.Channel == "email" {
		if _, err := mail.ParseAddress(inbox.From); err != nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.invalidFromAddress"), nil)
		}
		var cfg imodels.Config
		if len(inbox.Config) > 0 {
			if err := json.Unmarshal(inbox.Config, &cfg); err == nil && cfg.ReplyTo != "" {
				if _, err := mail.ParseAddress(cfg.ReplyTo); err != nil {
					return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidEmail"), nil)
				}
			}
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

	// Live credential check against Meta runs in the handler where request context is available.
	if inbox.Channel == whatsappChannel.ChannelWhatsApp {
		var cfg whatsappChannel.Config
		if err := json.Unmarshal(inbox.Config, &cfg); err != nil {
			return envelope.NewError(envelope.InputError, "invalid whatsapp config", nil)
		}
		if cfg.PhoneNumberID == "" || cfg.WABAID == "" {
			return envelope.NewError(envelope.InputError, "phone_number_id and waba_id are required", nil)
		}
		if cfg.WebhookVerifyToken == "" {
			return envelope.NewError(envelope.InputError, "webhook_verify_token is required", nil)
		}
		// On edit secrets arrive masked/empty and the config merge restores them.
		if !isUpdate {
			if cfg.AccessToken == "" {
				return envelope.NewError(envelope.InputError, "access_token is required", nil)
			}
			if cfg.AppSecret == "" {
				return envelope.NewError(envelope.InputError, "app_secret is required", nil)
			}
		}
	}

	// Validate livechat-specific configuration
	if inbox.Channel == livechat.ChannelLiveChat {
		var config livechat.Config
		if err := json.Unmarshal(inbox.Config, &config); err == nil {
			// ShowOfficeHoursAfterAssignment cannot be enabled if ShowOfficeHoursInChat is disabled
			if config.ShowOfficeHoursAfterAssignment && !config.ShowOfficeHoursInChat {
				return envelope.NewError(envelope.InputError, "`show_office_hours_after_assignment` cannot be enabled when `show_office_hours_in_chat` is disabled", nil)
			}
			// Validate continuity settings - required when linked email inbox is set.
			if inbox.LinkedEmailInboxID.Valid && inbox.LinkedEmailInboxID.Int > 0 {
				if config.Continuity.OfflineThreshold == "" {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "offline_threshold"), nil)
				}
				if config.Continuity.MinEmailInterval == "" {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "min_email_interval"), nil)
				}
				if config.Continuity.MaxMessagesPerEmail == 0 {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "max_messages_per_email"), nil)
				}
			}
			if config.Continuity.OfflineThreshold != "" {
				d, err := time.ParseDuration(config.Continuity.OfflineThreshold)
				if err != nil {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.invalidDuration", "name", "offline_threshold"), nil)
				}
				if d < time.Minute {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.minDuration", "name", "offline_threshold", "min", "1m"), nil)
				}
			}
			if config.Continuity.MinEmailInterval != "" {
				d, err := time.ParseDuration(config.Continuity.MinEmailInterval)
				if err != nil {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.invalidDuration", "name", "min_email_interval"), nil)
				}
				if d < time.Minute {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.minDuration", "name", "min_email_interval", "min", "1m"), nil)
				}
			}
			if config.Continuity.MaxMessagesPerEmail != 0 {
				if config.Continuity.MaxMessagesPerEmail < 1 || config.Continuity.MaxMessagesPerEmail > 100 {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.minmaxNumber", "min", "1", "max", "100"), nil)
				}
			}

			// Validate colors.
			hexColorRegex := regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)
			if config.Colors.Primary == "" {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "primary color"), nil)
			}
			if !hexColorRegex.MatchString(config.Colors.Primary) {
				return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidColor"), nil)
			}

			// Validate launcher position.
			if config.Launcher.Position != "left" && config.Launcher.Position != "right" {
				return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
			}

			// Validate launcher spacing: clamp to a sane range so a fat-fingered value doesn't push the launcher off-screen.
			if config.Launcher.Spacing.Side < 0 || config.Launcher.Spacing.Side > 200 ||
				config.Launcher.Spacing.Bottom < 0 || config.Launcher.Spacing.Bottom > 200 {
				return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
			}

			// Validate home apps.
			for _, ha := range config.HomeApps {
				if ha.URL != "" && !httputil.IsValidHTTPURL(ha.URL) {
					return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidUrl"), nil)
				}
				if ha.Type == livechat.HomeAppAnnouncement && ha.ImageURL != "" && !httputil.IsValidHTTPURL(ha.ImageURL) {
					return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidUrl"), nil)
				}
			}

			// Validate home screen background image URL.
			if config.HomeScreen.Background.ImageURL != "" && !httputil.IsValidHTTPURL(config.HomeScreen.Background.ImageURL) {
				return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidUrl"), nil)
			}

			// Validate URLs if set.
			for _, u := range []string{config.LogoURL, config.Launcher.LogoURL, config.WebsiteURL} {
				if u != "" && !httputil.IsValidHTTPURL(u) {
					return envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidUrl"), nil)
				}
			}

			// Validate trusted domains.
			// Valid formats: example.com, *.example.com, sub.example.com, example.com:8080
			for _, domain := range config.TrustedDomains {
				d := strings.TrimSpace(domain)
				if d == "" {
					continue
				}
				if strings.Contains(d, "://") || strings.Contains(d, "/") || strings.Contains(d, " ") {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.invalidDomain", "domain", d), nil)
				}
				// Wildcard must be at the start followed by a dot.
				if strings.Contains(d, "*") && !strings.HasPrefix(d, "*.") {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.invalidDomain", "domain", d), nil)
				}
			}

			// Validate blocked IPs entries.
			for _, entry := range config.BlockedIPs {
				if !httputil.ValidateIPOrCIDR(entry) {
					return envelope.NewError(envelope.InputError, app.i18n.Ts("validation.invalidIPOrCIDR", "entry", entry), nil)
				}
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

	// Meta tokens and IDs never contain whitespace, so pasted trailing spaces and newlines are always junk.
	if inb.Channel == whatsappChannel.ChannelWhatsApp && len(inb.Config) > 0 {
		var cfg whatsappChannel.Config
		if err := json.Unmarshal(inb.Config, &cfg); err != nil {
			return err
		}
		cfg.PhoneNumberID = strings.TrimSpace(cfg.PhoneNumberID)
		cfg.WABAID = strings.TrimSpace(cfg.WABAID)
		cfg.AccessToken = strings.TrimSpace(cfg.AccessToken)
		cfg.AppSecret = strings.TrimSpace(cfg.AppSecret)
		cfg.WebhookVerifyToken = strings.TrimSpace(cfg.WebhookVerifyToken)
		cfg.APIVersion = strings.TrimSpace(cfg.APIVersion)
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
	cfg.ReplyTo = strings.TrimSpace(cfg.ReplyTo)

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
