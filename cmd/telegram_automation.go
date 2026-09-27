package main

import (
	"time"

	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

func sendTelegramAutomaticReply(app *App, rec imodels.Inbox, cfg telegramChannel.Config, sourceID string) error {
	if cfg.GreetingMessage == "" && cfg.AwayMessage == "" {
		return nil
	}
	timezone := stringutil.NormalizeTimezone(cfg.Timezone)
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
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return err
		}
		open, err := hours.IsOpen(time.Now(), loc)
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
		stringutil.PlainTextToHTML(content), nil, nil, nil, map[string]any{"is_automated": true, "telegram_automatic_reply": kind})
	return err
}
