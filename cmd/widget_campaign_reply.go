package main

import (
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	realip "github.com/ferluci/fast-realip"
	"github.com/zerodha/fastglue"
)

// handleWidgetCampaignReply runs when a visitor replies to a campaign pop-up, adding the reply to its conversation or starting a new one.
func handleWidgetCampaignReply(r *fastglue.Request, req chatInitReq, inbox imodels.Inbox, config livechat.Config) error {
	app := r.Context.(*App)
	if !validCampaignKey(req.DeliveryID) || !validCampaignKey(req.BrowserKey) {
		return sendErrorEnvelope(r, campaignInputError(app))
	}
	unlock := app.proactive.Lock()
	defer unlock()
	contact, err := widgetContact(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	delivery, err := app.proactive.Get(req.DeliveryID, inbox.ID, req.BrowserKey, contact.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if delivery.Replied && delivery.ConversationUUID == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.NotFoundError, app.i18n.T("globals.messages.notFound"), nil))
	}
	if !delivery.Displayed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.ConflictError, app.i18n.T("widget.invitationExpired"), nil))
	}
	var sessionToken string
	var attrs map[string]any
	visitor := getWidgetIsVisitor(r)
	if delivery.ConversationUUID != "" {
		conversation, err := app.conversation.GetConversation(0, delivery.ConversationUUID, "")
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if contact.ID != conversation.ContactID {
			if contact.ID > 0 {
				return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, app.i18n.T("widget.conversationNotPermitted"), nil))
			}
			contact, err = app.user.GetContactOrVisitor(conversation.ContactID, "")
			if err != nil {
				return sendErrorEnvelope(r, err)
			}
			if contact.Type != "visitor" {
				return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, app.i18n.T("widget.conversationNotPermitted"), nil))
			}
			sessionToken, err = generateSessionToken(app, contact.ID, inbox.ID, true, "", defaultSessionTTL)
			if err != nil {
				return sendErrorEnvelope(r, err)
			}
			visitor = true
		}
		if err := canReply(r, conversation); err != nil {
			return sendErrorEnvelope(r, err)
		}
		message := cmodels.Message{ConversationUUID: conversation.UUID, ConversationID: conversation.ID, SenderID: contact.ID, Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, Status: cmodels.MessageStatusReceived, Content: req.Message, ContentType: cmodels.ContentTypeText}
		if _, err := app.conversation.ProcessIncomingLiveChatMessage(message); err != nil {
			return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.errorSendingMessage"), nil))
		}
	} else {
		if contact.ID == 0 {
			contact, sessionToken, attrs, err = createVisitorContact(app, req.FormData, config, inbox)
			if err != nil {
				return sendErrorEnvelope(r, err)
			}
			visitor = true
		} else {
			attrs = saveContactAttrsAndCollectConvoAttrs(app, contact.ID, nil, req.FormData, config)
		}
		if err := checkConversationPermissions(app, config, visitor, contact.ID, inbox.ID); err != nil {
			return sendErrorEnvelope(r, err)
		}
		meta := map[string]any{"ip": realip.FromRequest(r.RequestCtx), "user_agent": string(r.RequestCtx.Request.Header.Peek("User-Agent")), "campaign_id": delivery.CampaignID}
		reply, err := app.conversation.CreateProactiveConversation(delivery, contact.ID, req.Message, attrs, meta, maxChatConversationsPerContact, chatConversationRateLimitWindow)
		if err != nil {
			if _, ok := err.(envelope.Error); ok {
				return sendErrorEnvelope(r, err)
			}
			return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.errorSendingMessage"), nil))
		}
		delivery.ConversationUUID = reply.ConversationUUID
		if reply.ID > 0 {
			if err := app.conversation.ProcessIncomingMessageHooks(reply, true); err != nil {
				app.lo.Error("error processing proactive reply hooks", "error", err)
			}
		}
	}
	conversation, err := app.conversation.GetConversation(0, delivery.ConversationUUID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	resp, err := buildConversationResponseWithBusinessHours(app, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	response := map[string]any{"conversation": resp.Conversation, "messages": resp.Messages, "business_hours_id": resp.BusinessHoursID, "working_hours_utc_offset": resp.WorkingHoursUTCOffset}
	if sessionToken != "" {
		response["session_token"] = sessionToken
		response["user"] = map[string]any{"user_id": contact.ID, "is_visitor": visitor, "first_name": contact.FirstName, "last_name": contact.LastName}
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	return r.SendEnvelope(response)
}
