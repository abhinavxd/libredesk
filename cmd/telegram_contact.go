package main

import (
	"encoding/json"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	telegramChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/telegram"
	"github.com/abhinavxd/libredesk/internal/telegram"
	"github.com/zerodha/fastglue"
)

func handleSaveTelegramContact(r *fastglue.Request) error {
	app := r.Context.(*App)
	cuuid := r.RequestCtx.UserValue("cuuid").(string)
	uuid := r.RequestCtx.UserValue("uuid").(string)
	actor := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(actor.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	message, err := app.conversation.GetMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if message.ConversationUUID != cuuid || message.Private || conversation.InboxChannel != telegramChannel.ChannelTelegram {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}
	var meta struct {
		Contact *telegram.Contact `json:"telegram_contact"`
	}
	if err := json.Unmarshal(message.Meta, &meta); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if meta.Contact == nil || meta.Contact.Phone == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("conversation.telegram.error.noContact"), nil))
	}
	id, err := app.user.SaveSharedContact(meta.Contact.FirstName, meta.Contact.LastName, meta.Contact.Phone)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]int{"id": id})
}
