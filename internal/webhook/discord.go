package webhook

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/webhook/models"
)

const (
	discordTitleLimit = 256
	discordDescLimit  = 4096
	discordFieldLimit = 1024
)

type discordPayload struct {
	Username string         `json:"username,omitempty"`
	Embeds   []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	URL         string          `json:"url,omitempty"`
	Color       int             `json:"color,omitempty"`
	Timestamp   string          `json:"timestamp,omitempty"`
	Fields      []discordField  `json:"fields,omitempty"`
	Footer      *discordFooter  `json:"footer,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type discordFooter struct {
	Text string `json:"text"`
}

func normalizeDelivery(rawURL, delivery string) (string, error) {
	switch delivery {
	case "", models.DeliveryHTTP:
		if models.IsDiscordWebhookURL(rawURL) {
			return models.DeliveryDiscord, nil
		}
		return models.DeliveryHTTP, nil
	case models.DeliveryDiscord:
		if !models.IsDiscordWebhookURL(rawURL) {
			return "", fmt.Errorf("invalid discord webhook url")
		}
		return models.DeliveryDiscord, nil
	default:
		return "", fmt.Errorf("invalid delivery %q", delivery)
	}
}

func usesDiscordPayload(w models.Webhook) bool {
	return w.Delivery == models.DeliveryDiscord || models.IsDiscordWebhookURL(w.URL)
}

func buildDiscordPayload(task DeliveryTask, rootURL string) ([]byte, error) {
	payload := mapFrom(task.Payload)
	conv := mapFrom(payload["conversation"])
	if conv == nil && looksLikeConversation(payload) {
		conv = payload
	}

	title, color := discordEventMeta(task.Event)
	uuid := firstNonEmpty(strVal(payload, "conversation_uuid"), strVal(payload, "uuid"), strVal(conv, "uuid"))
	subject := firstNonEmpty(strVal(conv, "subject"), strVal(payload, "subject"))
	contact := contactName(conv, payload)

	embed := discordEmbed{
		Title:     truncate(title, discordTitleLimit),
		Color:     color,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Footer:    &discordFooter{Text: "Libredesk"},
	}
	if link := conversationURL(rootURL, uuid); link != "" {
		embed.URL = link
	}

	switch task.Event {
	case models.EventMessageCreated, models.EventMessageUpdated:
		body := firstNonEmpty(strVal(payload, "text_content"), strVal(payload, "content"))
		embed.Description = truncate(body, discordDescLimit)
		if subject != "" {
			embed.Fields = appendField(embed.Fields, "Subject", subject, false)
		}
		if author := authorName(payload); author != "" {
			embed.Fields = appendField(embed.Fields, "Author", author, true)
		}
		if strVal(payload, "private") == "true" {
			embed.Fields = appendField(embed.Fields, "Note", "Private", true)
		}
	case models.EventWebhookTest:
		name := firstNonEmpty(strVal(payload, "name"), "Libredesk")
		embed.Description = fmt.Sprintf("Test notification from **%s**.", name)
	case models.EventConversationStatusChanged:
		prev := strVal(payload, "previous_status")
		next := firstNonEmpty(strVal(payload, "new_status"), strVal(conv, "status"))
		embed.Description = truncate(statusChangeDesc(subject, prev, next), discordDescLimit)
	default:
		if subject != "" {
			embed.Description = truncate(subject, discordDescLimit)
		}
	}

	if contact != "" {
		embed.Fields = appendField(embed.Fields, "Contact", contact, true)
	}
	if status := firstNonEmpty(strVal(conv, "status"), strVal(payload, "new_status")); status != "" && task.Event != models.EventConversationStatusChanged {
		embed.Fields = appendField(embed.Fields, "Status", status, true)
	}
	if inbox := firstNonEmpty(strVal(conv, "inbox_name"), strVal(payload, "inbox_name")); inbox != "" {
		embed.Fields = appendField(embed.Fields, "Inbox", inbox, true)
	}
	if ref := firstNonEmpty(strVal(conv, "reference_number"), strVal(payload, "reference_number")); ref != "" {
		embed.Fields = appendField(embed.Fields, "Reference", ref, true)
	}
	if tags := firstNonEmpty(formatTags(payload["new_tags"]), formatTags(conv["tags"])); tags != "" {
		embed.Fields = appendField(embed.Fields, "Tags", tags, false)
	}

	return json.Marshal(discordPayload{
		Username: "Libredesk",
		Embeds:   []discordEmbed{embed},
	})
}

func discordEventMeta(event models.WebhookEvent) (string, int) {
	switch event {
	case models.EventConversationCreated:
		return "New conversation", 0x5865F2
	case models.EventConversationAssigned:
		return "Conversation assigned", 0x57F287
	case models.EventConversationUnassigned:
		return "Conversation unassigned", 0xFEE75C
	case models.EventConversationStatusChanged:
		return "Status changed", 0xEB459E
	case models.EventConversationTagsChanged:
		return "Tags updated", 0x3498DB
	case models.EventMessageCreated:
		return "New message", 0x5865F2
	case models.EventMessageUpdated:
		return "Message updated", 0x99AAB5
	case models.EventWebhookTest:
		return "Test webhook", 0x57F287
	default:
		return string(event), 0x5865F2
	}
}

func conversationURL(root, uuid string) string {
	root = strings.TrimRight(strings.TrimSpace(root), "/")
	if root == "" || uuid == "" {
		return ""
	}
	return root + "/inboxes/all/conversation/" + uuid
}

func looksLikeConversation(m map[string]any) bool {
	if m == nil {
		return false
	}
	_, hasUUID := m["uuid"]
	_, hasRef := m["reference_number"]
	return hasUUID || hasRef
}

func contactName(conv, payload map[string]any) string {
	c := mapFrom(conv["contact"])
	if c == nil {
		c = mapFrom(payload["contact"])
	}
	name := strings.TrimSpace(firstNonEmpty(strVal(c, "first_name")+" "+strVal(c, "last_name"), strVal(c, "email")))
	if name != "" {
		return name
	}
	return ""
}

func authorName(payload map[string]any) string {
	a := mapFrom(payload["author"])
	return strings.TrimSpace(firstNonEmpty(strVal(a, "first_name")+" "+strVal(a, "last_name"), strVal(a, "email")))
}

func statusChangeDesc(subject, prev, next string) string {
	switch {
	case subject != "" && prev != "" && next != "":
		return fmt.Sprintf("%s: %s → %s", subject, prev, next)
	case prev != "" && next != "":
		return fmt.Sprintf("%s → %s", prev, next)
	case next != "":
		return next
	default:
		return subject
	}
}

func formatTags(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case []any:
		parts := make([]string, 0, len(t))
		for _, x := range t {
			if s := stringify(x); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return stringify(v)
	}
}

func appendField(fields []discordField, name, value string, inline bool) []discordField {
	value = strings.TrimSpace(value)
	if value == "" {
		return fields
	}
	return append(fields, discordField{Name: name, Value: truncate(value, discordFieldLimit), Inline: inline})
}

func mapFrom(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return t
	case nil:
		return nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return nil
		}
		return m
	}
}

func strVal(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return stringify(m[key])
}

func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	case map[string]any:
		if s, ok := t["String"].(string); ok {
			if valid, ok := t["Valid"].(bool); ok && !valid {
				return ""
			}
			return s
		}
		if name, ok := t["name"].(string); ok {
			return name
		}
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
