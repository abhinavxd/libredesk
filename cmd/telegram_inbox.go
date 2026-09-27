package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/abhinavxd/libredesk/internal/telegram"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/envelope"
	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

var (
	telegramSetupMu      sync.Mutex
	telegramTokenPattern = regexp.MustCompile(`^[0-9]+:[A-Za-z0-9_-]+$`)
)

func prepareTelegramInbox(ctx context.Context, app *App, inb *imodels.Inbox, id int) error {
	var cfg telegramChannel.Config
	if err := json.Unmarshal(inb.Config, &cfg); err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.invalidConfig"), nil)
	}
	for _, message := range []string{cfg.GreetingMessage, cfg.AwayMessage, cfg.CSATMessage} {
		if utf8.RuneCountInString(message) > telegram.MaxTextLength {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.maxLength", "max", "4096"), nil)
		}
	}
	if cfg.Timezone == "" {
		cfg.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(cfg.Timezone); err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.timezone"), nil)
	}
	if cfg.BusinessHoursID < 0 || (cfg.AwayMessage != "" && cfg.BusinessHoursID == 0) {
		return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.businessHours"), nil)
	}
	if cfg.BusinessHoursID > 0 {
		if _, err := app.businessHours.Get(cfg.BusinessHoursID); err != nil {
			return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.businessHours"), nil)
		}
	}
	var previous telegramChannel.Config
	if id > 0 {
		old, err := app.inbox.GetDBRecord(id)
		if err != nil {
			return err
		}
		if old.Channel != telegramChannel.ChannelTelegram {
			return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.changeBot"), nil)
		}
		if err := json.Unmarshal(old.Config, &previous); err != nil {
			return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if cfg.BotToken == "" || strings.Contains(cfg.BotToken, stringutil.PasswordDummy) {
			cfg.BotToken = previous.BotToken
		}
	}
	cfg.BotToken = strings.TrimSpace(cfg.BotToken)
	if !telegramTokenPattern.MatchString(cfg.BotToken) {
		return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.invalidToken"), nil)
	}
	if inb.ReopenWindowHours < 0 || inb.ReopenWindowHours > 2147483647 {
		return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.reopenWindow"), nil)
	}
	cfg.SecretToken = previous.SecretToken
	if cfg.SecretToken == "" {
		cfg.SecretToken = rand.Text()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	bot, err := app.telegramClient.GetMe(ctx, cfg.BotToken)
	if err != nil {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("admin.inbox.telegram.error.credentials", "error", err.Error()), nil)
	}
	if !bot.IsBot || bot.ID <= 0 {
		return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.invalidToken"), nil)
	}
	if previous.BotID != 0 && previous.BotID != bot.ID {
		return envelope.NewError(envelope.InputError, app.i18n.T("admin.inbox.telegram.error.changeBot"), nil)
	}
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return err
	}
	for _, existing := range inboxes {
		if existing.ID == id || existing.Channel != telegramChannel.ChannelTelegram {
			continue
		}
		var config telegramChannel.Config
		if err := json.Unmarshal(existing.Config, &config); err != nil {
			return envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if config.BotID == bot.ID {
			return envelope.NewError(envelope.ConflictError, app.i18n.T("admin.inbox.telegram.error.duplicateBot"), nil)
		}
	}
	cfg.BotID, cfg.BotUsername = bot.ID, bot.Username
	inb.Config, err = json.Marshal(cfg)
	return err
}

func configureTelegramWebhook(app *App, rec imodels.Inbox) {
	var cfg telegramChannel.Config
	if err := json.Unmarshal(rec.Config, &cfg); err != nil {
		app.telegramHookErrors.Store(rec.ID, app.i18n.T("admin.inbox.telegram.error.invalidConfig"))
		return
	}
	ctx, cancel := context.WithTimeout(app.ctx, 30*time.Second)
	defer cancel()
	var err error
	if !rec.Enabled {
		err = app.telegramClient.DeleteWebhook(ctx, cfg.BotToken)
	} else {
		root, rootErr := app.setting.GetAppRootURL()
		if rootErr != nil || !isPublicWebhookURL(root) {
			err = fmt.Errorf("%s", app.i18n.T("admin.inbox.telegram.error.rootURL"))
		} else {
			err = app.telegramClient.SetWebhook(ctx, cfg.BotToken, telegramCallbackURL(root, rec.ID), cfg.SecretToken)
		}
	}
	if err != nil {
		app.lo.Error("error configuring telegram webhook", "inbox_id", rec.ID, "error", err)
		app.telegramHookErrors.Store(rec.ID, app.i18n.Ts("admin.inbox.telegram.error.webhook", "error", err.Error()))
	} else {
		app.telegramHookErrors.Delete(rec.ID)
	}
}

func reconcileTelegramWebhooks(app *App) {
	telegramSetupMu.Lock()
	defer telegramSetupMu.Unlock()
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		app.lo.Error("error listing telegram inboxes", "error", err)
		return
	}
	for _, rec := range inboxes {
		if rec.Channel == telegramChannel.ChannelTelegram && rec.Enabled {
			configureTelegramWebhook(app, rec)
		}
	}
}
