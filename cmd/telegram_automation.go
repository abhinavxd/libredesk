package main

import (
	"html"
	"strings"
	"time"

	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/telegram"
)

func sendTelegramAutomaticReply(app *App, rec imodels.Inbox, cfg telegramChannel.Config, sourceID string) error {
	if cfg.GreetingMessage == "" && cfg.AwayMessage == "" {
		return nil
	}
	timezone := cfg.Timezone
	if timezone == "" {
		timezone = "UTC"
	}
	state, err := app.conversation.GetTelegramAutoReplyState(sourceID, timezone)
	if err != nil {
		return err
	}
	if state.ConversationUUID == "" {
		return nil
	}
	content, kind := "", ""
	if cfg.AwayMessage != "" && cfg.BusinessHoursID > 0 && !state.RecentReply && !state.AwaySentToday {
		hours, err := app.businessHours.Get(cfg.BusinessHoursID)
		if err != nil {
			return err
		}
		open, err := telegram.WithinBusinessHours(time.Now(), hours, timezone)
		if err != nil {
			return err
		}
		if !open {
			content, kind = cfg.AwayMessage, "away"
		}
	}
	if content == "" && !state.HasOutgoing && cfg.GreetingMessage != "" {
		content, kind = cfg.GreetingMessage, "greeting"
	}
	if content == "" {
		return nil
	}
	actor, err := app.user.GetSystemUser()
	if err != nil {
		return err
	}
	_, err = app.conversation.QueueReply(nil, rec.ID, actor.ID, state.ContactID, state.ConversationUUID,
		strings.ReplaceAll(html.EscapeString(content), "\n", "<br>"), nil, nil, nil, map[string]any{"is_automated": true, "telegram_automatic_reply": kind})
	return err
}
