package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/user"
	realip "github.com/ferluci/fast-realip"
	"github.com/zerodha/fastglue"
)

type previewContactMergeReq struct {
	RemoveID int `json:"remove_id"`
}

func handleSearchMergeContacts(r *fastglue.Request) error {
	app, _, term, err := searchTerm(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	contacts, err := app.user.SearchMergeContacts(term)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(contacts)
}

func handlePreviewContactMerge(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	var req previewContactMergeReq
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	preview, err := app.user.PreviewContactMerge(id, req.RemoveID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(preview)
}

func handleMergeContacts(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	actor := r.RequestCtx.UserValue("user").(amodels.User)
	var req user.ContactMergeRequest
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	merged, err := app.user.MergeContacts(id, req, actor.ID, actor.Email, realip.FromRequest(r.RequestCtx))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		app.lo.Error("error fetching inboxes after contact merge", "error", err)
	}
	for _, inbox := range inboxes {
		initialized, err := app.inbox.Get(inbox.ID)
		if err != nil {
			continue
		}
		if chat, ok := initialized.(*livechat.LiveChat); ok {
			chat.DisconnectContact(strconv.Itoa(req.RemoveID))
		}
	}
	contact, err := app.user.GetContactOrVisitor(id, "")
	if err == nil {
		identities, identityErr := app.user.GetChannelIdentities(id)
		if identityErr != nil {
			app.lo.Error("error fetching merged contact identities", "error", identityErr)
		}
		fields := map[string]any{"id": id, "first_name": contact.FirstName, "last_name": contact.LastName,
			"email": contact.Email, "custom_attributes": contact.CustomAttributes, "enabled": contact.Enabled,
			"avatar_url": contact.AvatarURL, "type": contact.Type, "external_user_id": contact.ExternalUserID,
			"phone_number": contact.PhoneNumber, "phone_number_country_code": contact.PhoneNumberCountryCode,
			"country": contact.Country, "channel_identities": identities}
		app.conversation.BroadcastContactUpdate(id, fields)
		for _, uuid := range merged.ConversationUUIDs {
			app.conversation.BroadcastConversationUpdate(uuid, map[string]any{"contact_id": id, "contact": fields})
		}
	}

	return r.SendEnvelope(map[string]int{"id": id})
}
