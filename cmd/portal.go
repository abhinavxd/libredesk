package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type createPortalConversationRequest struct {
	InboxID          int            `json:"inbox_id"`
	Subject          string         `json:"subject"`
	Content          string         `json:"content"`
	CC               []string       `json:"cc"`
	CustomAttributes map[string]any `json:"custom_attributes"`
}

type portalInbox struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func handleGetPortalMe(r *fastglue.Request) error {
	return r.SendEnvelope(r.RequestCtx.UserValue("user").(amodels.User))
}

func handleGetPortalInboxes(r *fastglue.Request) error {
	app := r.Context.(*App)
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result := make([]portalInbox, 0, len(inboxes))
	for _, candidate := range inboxes {
		if candidate.Enabled && candidate.Channel == inbox.ChannelEmail {
			result = append(result, portalInbox{ID: candidate.ID, Name: candidate.Name})
		}
	}
	return r.SendEnvelope(result)
}

func handleGetPortalCustomAttributes(r *fastglue.Request) error {
	app := r.Context.(*App)
	attributes, err := app.customAttribute.GetAll("conversation")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result := attributes[:0]
	for _, attribute := range attributes {
		if attribute.PortalRequired {
			result = append(result, attribute)
		}
	}
	return r.SendEnvelope(result)
}

func handleGetPortalConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	page, pageSize := getPagination(r)
	items, total, err := app.conversation.GetPortalConversations(user.ID, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(envelope.PageResults{
		Total: total, Results: items, Page: page, PerPage: pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	})
}

func handleCreatePortalConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	req := createPortalConversationRequest{}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	req.Subject = strings.TrimSpace(req.Subject)
	req.Content = strings.TrimSpace(req.Content)
	if req.Subject == "" || req.Content == "" || len(req.Subject) > 200 || len(req.Content) > 10000 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	cc := make([]string, 0, len(req.CC))
	seen := map[string]struct{}{strings.ToLower(user.Email): {}}
	for _, address := range req.CC {
		address = strings.ToLower(strings.TrimSpace(address))
		if address == "" {
			continue
		}
		if !stringutil.ValidEmail(address) || len(cc) >= 20 {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidEmail"), nil, envelope.InputError)
		}
		if _, exists := seen[address]; !exists {
			seen[address] = struct{}{}
			cc = append(cc, address)
		}
	}
	attributes, err := app.customAttribute.GetAll("conversation")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	validAttributes := make(map[string]any)
	for _, attribute := range attributes {
		value, exists := req.CustomAttributes[attribute.Key]
		if attribute.PortalRequired && (!exists || strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(toString(value)), "false", "")) == "") {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, attribute.Name+" is required.", nil, envelope.InputError)
		}
		if !exists {
			continue
		}
		if attribute.Regex != "" {
			pattern, compileErr := regexp.Compile(attribute.Regex)
			if compileErr != nil || !pattern.MatchString(toString(value)) {
				message := attribute.RegexHint
				if message == "" {
					message = attribute.Name + " is invalid."
				}
				return r.SendErrorEnvelope(fasthttp.StatusBadRequest, message, nil, envelope.InputError)
			}
		}
		validAttributes[attribute.Key] = value
	}

	selectedInbox, err := app.inbox.GetDBRecord(req.InboxID)
	if err != nil || !selectedInbox.Enabled || selectedInbox.Channel != inbox.ChannelEmail {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "Support is not configured to receive new tickets.", nil, envelope.GeneralError)
	}

	conversationID, conversationUUID, err := app.conversation.CreateConversation(
		user.ID, selectedInbox.ID, "", time.Now(), req.Subject, true, nil, validAttributes, 0, 0,
	)
	if err != nil {
		app.lo.Error("creating portal conversation", "contact_id", user.ID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}
	messageMeta := map[string]any{}
	if len(cc) > 0 {
		messageMeta["cc"] = cc
	}
	if _, err := app.conversation.CreateContactMessageWithMeta(nil, user.ID, conversationUUID, req.Content, cmodels.ContentTypeText, true, messageMeta); err != nil {
		if deleteErr := app.conversation.DeleteConversation(conversationUUID); deleteErr != nil {
			app.lo.Error("deleting failed portal conversation", "conversation_uuid", conversationUUID, "error", deleteErr)
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingMessage"), nil, envelope.GeneralError)
	}

	created, err := app.conversation.GetPortalConversation(user.ID, conversationUUID)
	if err != nil {
		app.lo.Error("loading created portal conversation", "conversation_id", conversationID, "error", err)
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(created)
}

func toString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func handleGetPortalConversation(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	item, err := app.conversation.GetPortalConversation(user.ID, r.RequestCtx.UserValue("uuid").(string))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(item)
}

func handleGetPortalMessages(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	page, pageSize := getPagination(r)
	items, pageSize, err := app.conversation.GetPortalMessages(user.ID, r.RequestCtx.UserValue("uuid").(string), page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(envelope.PageResults{Results: items, Page: page, PerPage: pageSize})
}
