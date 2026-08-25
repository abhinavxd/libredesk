package conversation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/countries"
	"github.com/abhinavxd/libredesk/internal/envelope"
	whatsappChannel "github.com/abhinavxd/libredesk/internal/inbox/channel/whatsapp"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	wtmodels "github.com/abhinavxd/libredesk/internal/whatsapp_template/models"
	"github.com/jmoiron/sqlx"
)

// WhatsAppWindowDuration is Meta's customer service window for free-form messages.
const WhatsAppWindowDuration = 24 * time.Hour

// whatsAppMaxTextLength is Meta's cap on a text message body.
const whatsAppMaxTextLength = 4096

// WhatsAppStatus values mirror Meta's delivery lifecycle, kept in message.meta.
const (
	WhatsAppStatusSent      = "sent"
	WhatsAppStatusDelivered = "delivered"
	WhatsAppStatusRead      = "read"
	WhatsAppStatusFailed    = "failed"
)

// A media header needs a media ID the sender can't supply, so only text (or no) headers can go out.
var sendableTemplateHeaderTypes = []string{"", "NONE", "TEXT"}

var templatePlaceholderPattern = regexp.MustCompile(`\{\{[A-Za-z0-9_]+\}\}`)

// WhatsAppReadReceiptTarget returns the inbox ID and wamid of the latest unseen inbound message, or empty values when there is nothing to mark read.
func (m *Manager) WhatsAppReadReceiptTarget(uuid string, userID int) (int, string, error) {
	var row struct {
		SourceID string `db:"source_id"`
		InboxID  int    `db:"inbox_id"`
	}
	if err := m.q.GetWhatsAppReadReceiptTarget.Get(&row, uuid, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil
		}
		return 0, "", err
	}
	return row.InboxID, row.SourceID, nil
}

func (m *Manager) ApplyWhatsAppStatus(sourceID, metaStatus string, eventAt time.Time, errorMsg string) error {
	if sourceID == "" || metaStatus == "" {
		return nil
	}
	if eventAt.IsZero() {
		eventAt = time.Now().UTC()
	}
	ts := eventAt.Format(time.RFC3339)

	patch := map[string]any{
		"provider_status":            metaStatus,
		"provider_status_updated_at": ts,
	}
	switch metaStatus {
	case WhatsAppStatusSent:
		patch["provider_sent_at"] = ts
	case WhatsAppStatusDelivered:
		patch["provider_delivered_at"] = ts
	case WhatsAppStatusRead:
		patch["provider_read_at"] = ts
	case WhatsAppStatusFailed:
		patch["provider_failed_at"] = ts
		if errorMsg != "" {
			patch["provider_failure_reason"] = errorMsg
		}
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	// The message_status enum collapses delivered/read into sent; the full lifecycle lives in meta.
	dbStatus := models.MessageStatusSent
	if metaStatus == WhatsAppStatusFailed {
		dbStatus = models.MessageStatusFailed
	}

	var row struct {
		UUID             string          `db:"uuid"`
		ConversationUUID string          `db:"conversation_uuid"`
		Status           string          `db:"status"`
		Meta             json.RawMessage `db:"meta"`
	}
	if err := m.q.ApplyWhatsAppMessageStatus.Get(&row, sourceID, dbStatus, patchBytes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("status update for source_id=%s status=%s: %w", sourceID, metaStatus, ErrMessageNotFound)
		}
		m.lo.Error("error applying whatsapp message status", "source_id", sourceID, "error", err)
		return err
	}
	m.BroadcastMessageUpdate(row.ConversationUUID, row.UUID, map[string]any{"status": row.Status, "meta": stripCSATUUID(row.Meta)})
	return nil
}

func (m *Manager) RecordWhatsAppSendFailure(messageUUID, errorMsg string) error {
	if messageUUID == "" || errorMsg == "" {
		return nil
	}
	patch := map[string]any{
		"provider_status":         WhatsAppStatusFailed,
		"provider_failed_at":      time.Now().UTC().Format(time.RFC3339),
		"provider_failure_reason": errorMsg,
	}
	return m.mergeWhatsAppMeta(m.q.MergeMessageMetaByUUID, messageUUID, patch)
}

// mergeWhatsAppMeta is a no-op on an unmatched key.
func (m *Manager) mergeWhatsAppMeta(stmt *sqlx.Stmt, key string, patch map[string]any) error {
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	var row struct {
		UUID             string          `db:"uuid"`
		ConversationUUID string          `db:"conversation_uuid"`
		Meta             json.RawMessage `db:"meta"`
	}
	if err := stmt.Get(&row, key, patchBytes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		m.lo.Error("error merging whatsapp meta", "key", key, "error", err)
		return err
	}
	m.BroadcastMessageUpdate(row.ConversationUUID, row.UUID, map[string]any{"meta": stripCSATUUID(row.Meta)})
	return nil
}

// sendWhatsAppCSAT sends CSAT via the reserved template, falls back to a link inside the 24h window, else records a not-sent activity.
func (m *Manager) sendWhatsAppCSAT(actorUserID int, conversation models.Conversation, csatUUID, csatURL string) error {
	meta := map[string]any{
		"is_csat":      true,
		"is_automated": true,
		"csat_uuid":    csatUUID,
	}

	if m.whatsappTemplate != nil {
		t, err := m.whatsappTemplate.GetApproved(conversation.InboxID, wtmodels.CSATTemplateName(conversation.InboxID), m.csatTemplateLanguage(conversation.InboxID))
		if err == nil {
			meta["whatsapp_template_id"] = t.ID
			meta["whatsapp_template_params"] = map[string]string{"button_url_0": csatUUID}
			if _, err := m.QueueReply(nil, conversation.InboxID, actorUserID, conversation.ContactID, conversation.UUID, "", nil, nil, nil, meta); err != nil {
				m.lo.Error("error sending whatsapp CSAT template", "conversation_uuid", conversation.UUID, "error", err)
				return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
			return nil
		}
	}

	if m.whatsAppWindowOpen(conversation.ContactID, conversation.InboxID) {
		content := m.i18n.Ts("conversation.whatsapp.csatMessage", "link", csatURL)
		if m.whatsappTemplate != nil {
			if tmpl, err := m.whatsappTemplate.GetByName(conversation.InboxID, wtmodels.CSATTemplateName(conversation.InboxID)); err == nil && tmpl.BodyContent != "" {
				content = tmpl.BodyContent + "\n" + csatURL
			}
		}
		if _, err := m.QueueReply(nil, conversation.InboxID, actorUserID, conversation.ContactID, conversation.UUID, content, nil, nil, nil, meta); err != nil {
			m.lo.Error("error sending whatsapp CSAT link", "conversation_uuid", conversation.UUID, "error", err)
			return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		return nil
	}

	actor, err := m.userStore.GetSystemUser()
	if err != nil {
		m.lo.Error("error fetching system user for whatsapp CSAT activity", "conversation_uuid", conversation.UUID, "error", err)
		return nil
	}
	return m.InsertConversationActivity(models.ActivityCSATNotSent, conversation.UUID, "", actor)
}

func (m *Manager) csatTemplateLanguage(inboxID int) string {
	inb, err := m.inboxStore.GetDBRecord(inboxID)
	if err != nil {
		return ""
	}
	var cfg whatsappChannel.Config
	if err := json.Unmarshal(inb.Config, &cfg); err != nil {
		return ""
	}
	return cfg.CSATTemplateLanguage
}

// whatsAppWindowOpen reports whether the contact is inside Meta's 24h window. Scoped to (contact, inbox), not a single conversation.
func (m *Manager) whatsAppWindowOpen(contactID, inboxID int) bool {
	var ts sql.NullTime
	if err := m.q.GetContactWindowInboundAt.Get(&ts, contactID, inboxID); err != nil {
		m.lo.Error("error getting contact whatsapp window", "contact_id", contactID, "inbox_id", inboxID, "error", err)
		return false
	}
	return ts.Valid && time.Since(ts.Time) < WhatsAppWindowDuration
}

// prepareWhatsAppOutbound writes channel fields into metaMap and returns the rendered template body, or free-form content unchanged.
func (m *Manager) prepareWhatsAppOutbound(inboxRecord imodels.Inbox, conversationUUID string, content string, hasAttachments bool, metaMap map[string]any) (string, error) {
	var conv struct {
		InboxID   int `db:"inbox_id"`
		ContactID int `db:"contact_id"`
	}
	if err := m.q.GetConversationInboxContact.Get(&conv, conversationUUID); err != nil {
		m.lo.Error("error fetching conversation inbox and contact", "conversation_uuid", conversationUUID, "error", err)
		return content, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if conv.InboxID != inboxRecord.ID {
		return content, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// The channel identity is the wa_id Meta routes by; the phone columns are display data an agent may edit freely.
	toPhone, err := m.userStore.GetChannelIdentity(conv.ContactID, whatsappChannel.ChannelWhatsApp)
	if err != nil {
		return content, err
	}
	if toPhone == "" {
		contact, err := m.userStore.Get(conv.ContactID, "", nil)
		if err != nil {
			return content, err
		}
		if contact.PhoneNumber.String == "" {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.contactNoPhone"), nil)
		}
		dialCode := countries.DialCodeForISO(contact.PhoneNumberCountryCode.String)
		if dialCode == "" {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.contactCountryCodeInvalid"), nil)
		}
		toPhone = stringutil.NormalizeWhatsAppPhone(dialCode + contact.PhoneNumber.String)
		if toPhone == "" {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.contactNoPhone"), nil)
		}
		// Link the wa_id now so the contact's reply threads back to this contact instead of forking a duplicate.
		linkedID, err := m.userStore.LinkChannelIdentity(conv.ContactID, whatsappChannel.ChannelWhatsApp, toPhone)
		if err != nil {
			return content, err
		}
		if linkedID != conv.ContactID {
			return content, envelope.NewError(envelope.ConflictError, m.i18n.T("conversation.whatsapp.error.numberLinkedToAnotherContact"), nil)
		}
	}

	templateID := extractInt(metaMap, "whatsapp_template_id")
	templateParams := extractStringMap(metaMap, "whatsapp_template_params")

	send := whatsappChannel.SendMeta{
		ToPhone: toPhone,
	}

	rendered := content

	if templateID > 0 {
		if m.whatsappTemplate == nil {
			return content, envelope.NewError(envelope.GeneralError, m.i18n.T("conversation.whatsapp.error.templateStoreUnavailable"), nil)
		}
		t, err := m.whatsappTemplate.GetByID(templateID)
		if err != nil {
			return content, err
		}
		if t.InboxID != inboxRecord.ID {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.templateWrongInbox"), nil)
		}
		if !strings.EqualFold(t.Status, wtmodels.StatusApproved) {
			return content, envelope.NewError(envelope.InputError, m.i18n.Ts("conversation.whatsapp.error.templateNotApproved", "status", t.Status), nil)
		}
		if t.HeaderType.Valid && !slices.Contains(sendableTemplateHeaderTypes, strings.ToUpper(t.HeaderType.String)) {
			return content, envelope.NewError(envelope.InputError, m.i18n.Ts("conversation.whatsapp.error.templateHeaderUnsupported", "type", strings.ToUpper(t.HeaderType.String)), nil)
		}
		send.TemplateName = t.Name
		send.TemplateLanguage = t.Language
		send.TemplateParams = templateParams
		send.TemplateBodyContent = t.BodyContent
		if t.HeaderType.Valid {
			send.TemplateHeaderType = t.HeaderType.String
		}
		if t.HeaderContent.Valid {
			send.TemplateHeaderContent = t.HeaderContent.String
		}
		if len(t.Buttons) > 0 {
			var btns []whatsapp.TemplateButton
			if err := json.Unmarshal(t.Buttons, &btns); err == nil {
				send.TemplateButtons = btns
			}
		}
		if err := m.validateTemplateParams(t, templateParams); err != nil {
			return content, err
		}
		rendered = renderTemplateBody(t.BodyContent, templateParams)
	} else {
		if !m.whatsAppWindowOpen(conv.ContactID, conv.InboxID) {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.windowClosed"), nil)
		}
		if strings.TrimSpace(content) == "" && !hasAttachments {
			return content, envelope.NewError(envelope.InputError, m.i18n.T("conversation.whatsapp.error.contentRequired"), nil)
		}
		if utf8.RuneCountInString(stringutil.HTML2Text(content)) > whatsAppMaxTextLength {
			return content, envelope.NewError(envelope.InputError, m.i18n.Ts("conversation.whatsapp.error.tooLong", "limit", strconv.Itoa(whatsAppMaxTextLength)), nil)
		}
	}

	encoded, err := json.Marshal(send)
	if err != nil {
		return content, err
	}

	metaMap["whatsapp"] = json.RawMessage(encoded)
	return rendered, nil
}

// validateTemplateParams rejects unfilled body and text-header placeholders locally, ahead of Meta's opaque parameter-mismatch error.
func (m *Manager) validateTemplateParams(t wtmodels.Template, params map[string]string) error {
	for _, key := range whatsapp.OrderedPlaceholders(t.BodyContent) {
		if strings.TrimSpace(params["body:"+key]) == "" {
			return envelope.NewError(envelope.InputError, m.i18n.Ts("conversation.whatsapp.error.missingBodyParam", "placeholder", "{{"+key+"}}"), nil)
		}
	}
	if t.HeaderType.Valid && strings.EqualFold(t.HeaderType.String, "TEXT") {
		for _, key := range whatsapp.OrderedPlaceholders(t.HeaderContent.String) {
			if strings.TrimSpace(params["header:"+key]) == "" {
				return envelope.NewError(envelope.InputError, m.i18n.Ts("conversation.whatsapp.error.missingHeaderParam", "placeholder", "{{"+key+"}}"), nil)
			}
		}
	}
	if len(t.Buttons) > 0 {
		var btns []whatsapp.TemplateButton
		if err := json.Unmarshal(t.Buttons, &btns); err == nil {
			for i, b := range btns {
				if !strings.EqualFold(b.Type, "URL") || len(whatsapp.OrderedPlaceholders(b.URL)) == 0 {
					continue
				}
				if strings.TrimSpace(params["button_url_"+strconv.Itoa(i)]) == "" {
					return envelope.NewError(envelope.InputError, m.i18n.Ts("conversation.whatsapp.error.missingButtonParam", "button", b.Text), nil)
				}
			}
		}
	}
	return nil
}

// renderTemplateBody fills {{name}} placeholders from "body:"+name params; unmatched ones stay verbatim so missing params show in the timeline.
func renderTemplateBody(body string, params map[string]string) string {
	if body == "" || len(params) == 0 {
		return body
	}
	return templatePlaceholderPattern.ReplaceAllStringFunc(body, func(match string) string {
		name := match[2 : len(match)-2]
		if v, ok := params["body:"+name]; ok {
			return v
		}
		return match
	})
}

// extractInt pulls an int out of a meta map regardless of the JSON decoder's numeric type.
func extractInt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}

// extractStringMap pulls a string map out of a meta map, tolerating both map[string]string and decoded map[string]any.
func extractStringMap(m map[string]any, key string) map[string]string {
	switch raw := m[key].(type) {
	case map[string]string:
		if len(raw) == 0 {
			return nil
		}
		out := make(map[string]string, len(raw))
		maps.Copy(out, raw)
		return out
	case map[string]any:
		if len(raw) == 0 {
			return nil
		}
		out := make(map[string]string, len(raw))
		for k, v := range raw {
			switch t := v.(type) {
			case string:
				out[k] = t
			case json.Number:
				out[k] = t.String()
			case float64:
				out[k] = fmt.Sprintf("%v", t)
			case bool:
				out[k] = fmt.Sprintf("%v", t)
			}
		}
		return out
	}
	return nil
}

func stripCSATUUID(meta json.RawMessage) json.RawMessage {
	if len(meta) == 0 {
		return meta
	}
	var m map[string]any
	if err := json.Unmarshal(meta, &m); err != nil {
		return meta
	}
	if _, ok := m["csat_uuid"]; !ok {
		return meta
	}
	delete(m, "csat_uuid")
	stripped, err := json.Marshal(m)
	if err != nil {
		return meta
	}
	return stripped
}
