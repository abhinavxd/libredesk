package main

import (
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/httputil"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat/proactive"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/fastglue"
)

// handleWidgetCampaignReply runs when a visitor replies to a campaign pop-up, adding the reply to its conversation or starting a new one.
func handleWidgetCampaignReply(r *fastglue.Request, req chatInitReq, inbox imodels.Inbox, config livechat.Config) error {
	app := r.Context.(*App)
	if !validCampaignKey(req.DeliveryID) || !validCampaignKey(req.BrowserKey) {
		return sendErrorEnvelope(r, campaignInputError(app))
	}

	unlock := app.proactive.Lock()
	contact, delivery, err := loadCampaignReply(r, app, req, inbox.ID)
	if err != nil {
		unlock()
		return sendErrorEnvelope(r, err)
	}
	if delivery.ConversationUUID != "" {
		defer unlock()
		return finishCampaignReply(r, app, req, inbox, contact, delivery)
	}

	// Siteverify can take seconds. This lock is shared by every inbox, so a reply to an
	// existing conversation must not wait on it. Recheck the delivery after the call.
	contactID := contact.ID
	isVisitor := getWidgetIsVisitor(r)
	if contactID == 0 {
		isVisitor = true
	}
	unlock()
	if err := enforceChatInitTurnstile(r, app, config, req.TurnstileToken, httputil.ClientIP(r.RequestCtx, app.clientIPHeader), contactID, isVisitor, inbox.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	unlock = app.proactive.Lock()
	defer unlock()
	contact, delivery, err = loadCampaignReply(r, app, req, inbox.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if delivery.ConversationUUID != "" {
		return finishCampaignReply(r, app, req, inbox, contact, delivery)
	}
	return createCampaignReply(r, app, req, inbox, config, contact, delivery)
}

func loadCampaignReply(r *fastglue.Request, app *App, req chatInitReq, inboxID int) (umodels.User, proactive.Delivery, error) {
	contact, err := widgetContact(r)
	if err != nil {
		return umodels.User{}, proactive.Delivery{}, err
	}
	delivery, err := app.proactive.Get(req.DeliveryID, inboxID, req.BrowserKey, contact.ID)
	if err != nil {
		return umodels.User{}, proactive.Delivery{}, err
	}
	if delivery.Replied && delivery.ConversationUUID == "" {
		return umodels.User{}, proactive.Delivery{}, envelope.NewError(envelope.NotFoundError, app.i18n.T("globals.messages.notFound"), nil)
	}
	if !delivery.Displayed {
		return umodels.User{}, proactive.Delivery{}, envelope.NewError(envelope.ConflictError, app.i18n.T("widget.invitationExpired"), nil)
	}
	return contact, delivery, nil
}

func finishCampaignReply(r *fastglue.Request, app *App, req chatInitReq, inbox imodels.Inbox, contact umodels.User, delivery proactive.Delivery) error {
	var sessionToken string
	visitor := getWidgetIsVisitor(r)
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
	return writeCampaignReply(r, app, delivery.ConversationUUID, sessionToken, contact, visitor)
}

func createCampaignReply(r *fastglue.Request, app *App, req chatInitReq, inbox imodels.Inbox, config livechat.Config, contact umodels.User, delivery proactive.Delivery) error {
	var sessionToken string
	var attrs map[string]any
	visitor := getWidgetIsVisitor(r)
	var err error
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
	meta := map[string]any{"ip": httputil.ClientIP(r.RequestCtx, app.clientIPHeader), "user_agent": string(r.RequestCtx.Request.Header.Peek("User-Agent")), "campaign_id": delivery.CampaignID}
	reply, err := app.conversation.CreateProactiveConversation(delivery, contact.ID, req.Message, attrs, meta, maxChatConversationsPerContact, chatConversationRateLimitWindow)
	if err != nil {
		if _, ok := err.(envelope.Error); ok {
			return sendErrorEnvelope(r, err)
		}
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.errorSendingMessage"), nil))
	}
	if reply.ID > 0 {
		if err := app.conversation.ProcessIncomingMessageHooks(reply, true); err != nil {
			app.lo.Error("error processing proactive reply hooks", "error", err)
		}
	}
	return writeCampaignReply(r, app, reply.ConversationUUID, sessionToken, contact, visitor)
}

func writeCampaignReply(r *fastglue.Request, app *App, conversationUUID, sessionToken string, contact umodels.User, visitor bool) error {
	conversation, err := app.conversation.GetConversation(0, conversationUUID, "")
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
