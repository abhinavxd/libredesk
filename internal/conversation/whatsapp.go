package conversation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
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

// WhatsAppStatus values mirror Meta's delivery lifecycle, kept in message.meta since the message_status enum collapses delivered/read into sent.
const (
	WhatsAppStatusSent      = "sent"
	WhatsAppStatusDelivered = "delivered"
	WhatsAppStatusRead      = "read"
	WhatsAppStatusFailed    = "failed"
)

var templatePlaceholderPattern = regexp.MustCompile(`\{\{[A-Za-z0-9_]+\}\}`)

// MessageUUIDBySourceID resolves a transport-side message ID to a local message UUID, "" with nil error when no match.
func (m *Manager) MessageUUIDBySourceID(sourceID string) (string, error) {
	if sourceID == "" {
		return "", nil
	}
	var uuid string
	if err := m.q.GetMessageUUIDBySourceID.Get(&uuid, sourceID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return uuid, nil
}

// WhatsAppReadReceiptTarget returns the inbox ID and wamid of the latest inbound message the user has not yet seen on a WhatsApp conversation, or empty values when there is nothing to mark read.
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

// RecordWhatsAppStatus stamps a Meta status event (sent/delivered/read/failed) into the message meta and broadcasts a partial message_update.
func (m *Manager) RecordWhatsAppStatus(sourceID, metaStatus string, eventAt time.Time, errorMsg string) error {
	if sourceID == "" || metaStatus == "" {
		return nil
	}
	if eventAt.IsZero() {
		eventAt = time.Now().UTC()
	}
	ts := eventAt.Format(time.RFC3339)

	patch := map[string]any{
		"wa_status":            metaStatus,
		"wa_status_updated_at": ts,
	}
	switch metaStatus {
	case WhatsAppStatusSent:
		patch["wa_sent_at"] = ts
	case WhatsAppStatusDelivered:
		patch["wa_delivered_at"] = ts
	case WhatsAppStatusRead:
		patch["wa_read_at"] = ts
	case WhatsAppStatusFailed:
		patch["wa_failed_at"] = ts
		if errorMsg != "" {
			patch["wa_failure_reason"] = errorMsg
		}
	}

	return m.mergeWhatsAppMeta(m.q.MergeMessageMetaBySourceID, sourceID, patch)
}

// RecordWhatsAppSendFailure stamps a send-time Meta error into the message meta and broadcasts it.
func (m *Manager) RecordWhatsAppSendFailure(messageUUID, errorMsg string) error {
	if messageUUID == "" || errorMsg == "" {
		return nil
	}
	patch := map[string]any{
		"wa_status":         WhatsAppStatusFailed,
		"wa_failed_at":      time.Now().UTC().Format(time.RFC3339),
		"wa_failure_reason": errorMsg,
	}
	return m.mergeWhatsAppMeta(m.q.MergeMessageMetaByUUID, messageUUID, patch)
}

// SetWhatsAppSentAttachments records the attachment UUIDs already delivered for a message so retrying a partially-sent multi-attachment message does not re-deliver them.
func (m *Manager) SetWhatsAppSentAttachments(messageUUID string, attachmentUUIDs []string) error {
	if messageUUID == "" {
		return nil
	}
	patch, err := json.Marshal(map[string]any{"whatsapp_sent_attachments": attachmentUUIDs})
	if err != nil {
		return err
	}
	var row struct {
		UUID             string          `db:"uuid"`
		ConversationUUID string          `db:"conversation_uuid"`
		Meta             json.RawMessage `db:"meta"`
	}
	if err := m.q.MergeMessageMetaByUUID.Get(&row, messageUUID, patch); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}

// mergeWhatsAppMeta is a no-op on an unmatched key, e.g. the 2nd..nth media of a multi-attachment send shares one message row.
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

// csatTemplateLanguage returns the inbox's configured CSAT template language so the send-time lookup matches the language the template was registered under.
func (m *Manager) csatTemplateLanguage(inboxID int) string {
	inb, err := m.inboxStore.GetDBRecord(inboxID)
	if err != nil {
		return whatsappChannel.DefaultCSATTemplateLanguage
	}
	var cfg whatsappChannel.Config
	if err := json.Unmarshal(inb.Config, &cfg); err != nil {
		return whatsappChannel.DefaultCSATTemplateLanguage
	}
	return cfg.CSATLanguage()
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

// prepareWhatsAppOutbound validates the send, writes channel fields into metaMap, and returns the rendered template body or unchanged free-form content.
func (m *Manager) prepareWhatsAppOutbound(inboxRecord imodels.Inbox, contactID int, conversationUUID string, content string, hasAttachments bool, metaMap map[string]any) (string, error) {
	conv, err := m.GetConversation(0, conversationUUID, "")
	if err != nil {
		return content, err
	}
	if conv.InboxID != inboxRecord.ID {
		return content, envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// The channel identity is the wa_id Meta routes by; the phone columns are display data an agent may edit freely.
	toPhone, err := m.userStore.GetChannelIdentity(contactID, whatsappChannel.ChannelWhatsApp)
	if err != nil {
		return content, err
	}
	if toPhone == "" {
		contact, err := m.userStore.Get(contactID, "", nil)
		if err != nil {
			return content, err
		}
		if contact.PhoneNumber.String == "" {
			return content, envelope.NewError(envelope.InputError, "contact has no phone number", nil)
		}
		dialCode := countries.DialCodeForISO(contact.PhoneNumberCountryCode.String)
		if dialCode == "" {
			return content, envelope.NewError(envelope.InputError, "contact's phone country code is missing or invalid", nil)
		}
		toPhone = stringutil.NormalizeWhatsAppPhone(dialCode + contact.PhoneNumber.String)
	}
	if toPhone == "" {
		return content, envelope.NewError(envelope.InputError, "contact has no phone number", nil)
	}

	templateID := extractInt(metaMap, "whatsapp_template_id")
	templateParams := extractStringMap(metaMap, "whatsapp_template_params")

	send := whatsappChannel.SendMeta{
		ToPhone: toPhone,
	}

	rendered := content

	if templateID > 0 {
		if m.whatsappTemplate == nil {
			return content, envelope.NewError(envelope.GeneralError, "whatsapp template store unavailable", nil)
		}
		t, err := m.whatsappTemplate.GetByID(templateID)
		if err != nil {
			return content, err
		}
		if t.InboxID != inboxRecord.ID {
			return content, envelope.NewError(envelope.InputError, "template does not belong to this inbox", nil)
		}
		if !strings.EqualFold(t.Status, wtmodels.StatusApproved) {
			return content, envelope.NewError(envelope.InputError, fmt.Sprintf("template not approved (status: %s)", t.Status), nil)
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
		if err := validateTemplateParams(t, templateParams); err != nil {
			return content, err
		}
		rendered = renderTemplateBody(t.BodyContent, templateParams)
	} else {
		if !m.whatsAppWindowOpen(conv.ContactID, conv.InboxID) {
			return content, envelope.NewError(envelope.InputError, "24h customer service window has expired; reply with an approved template", nil)
		}
		if strings.TrimSpace(content) == "" && !hasAttachments {
			return content, envelope.NewError(envelope.InputError, "message content is required", nil)
		}
		if utf8.RuneCountInString(stringutil.HTML2Text(content)) > whatsAppMaxTextLength {
			return content, envelope.NewError(envelope.InputError, fmt.Sprintf("message exceeds WhatsApp's %d character limit", whatsAppMaxTextLength), nil)
		}
	}

	encoded, err := json.Marshal(send)
	if err != nil {
		return content, err
	}

	metaMap["whatsapp"] = json.RawMessage(encoded)
	return rendered, nil
}

// validateTemplateParams rejects a template send whose body or text-header placeholders have no supplied value, so Meta's parameter-mismatch error surfaces locally as a clear input error.
func validateTemplateParams(t wtmodels.Template, params map[string]string) error {
	for _, key := range whatsapp.OrderedPlaceholders(t.BodyContent) {
		if strings.TrimSpace(params["body:"+key]) == "" {
			return envelope.NewError(envelope.InputError, fmt.Sprintf("missing value for template parameter {{%s}}", key), nil)
		}
	}
	if t.HeaderType.Valid && strings.EqualFold(t.HeaderType.String, "TEXT") {
		for _, key := range whatsapp.OrderedPlaceholders(t.HeaderContent.String) {
			if strings.TrimSpace(params["header:"+key]) == "" {
				return envelope.NewError(envelope.InputError, fmt.Sprintf("missing value for template header parameter {{%s}}", key), nil)
			}
		}
	}
	return nil
}

// renderTemplateBody fills {{name}} placeholders from "body:"+name params, leaving unmatched ones verbatim to expose missing params in the timeline.
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
