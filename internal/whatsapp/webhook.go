package whatsapp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WebhookPayload is the top-level shape Meta delivers.
type WebhookPayload struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

type WebhookEntry struct {
	ID      string          `json:"id"`
	Changes []WebhookChange `json:"changes"`
}

type WebhookChange struct {
	Field string       `json:"field"`
	Value WebhookValue `json:"value"`
}

type WebhookValue struct {
	MessagingProduct        string           `json:"messaging_product"`
	Metadata                WebhookMetadata  `json:"metadata"`
	Contacts                []WebhookContact `json:"contacts"`
	Messages                []WebhookMessage `json:"messages"`
	Statuses                []WebhookStatus  `json:"statuses"`
	Event                   string           `json:"event"`
	MessageTemplateID       any              `json:"message_template_id"`
	MessageTemplateName     string           `json:"message_template_name"`
	MessageTemplateLanguage string           `json:"message_template_language"`
	Reason                  string           `json:"reason"`
	Errors                  []WebhookError   `json:"errors"`
}

type WebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WebhookContact struct {
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
	WAID string `json:"wa_id"`
}

type WebhookMessage struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`

	Text *struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`

	Image    *WebhookMedia `json:"image,omitempty"`
	Video    *WebhookMedia `json:"video,omitempty"`
	Audio    *WebhookMedia `json:"audio,omitempty"`
	Document *WebhookMedia `json:"document,omitempty"`
	Sticker  *WebhookMedia `json:"sticker,omitempty"`
	Voice    *WebhookMedia `json:"voice,omitempty"`

	Interactive *struct {
		Type        string `json:"type"`
		ButtonReply *struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"button_reply,omitempty"`
		ListReply *struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"list_reply,omitempty"`
	} `json:"interactive,omitempty"`

	Button *struct {
		Text    string `json:"text"`
		Payload string `json:"payload"`
	} `json:"button,omitempty"`

	Context *struct {
		From string `json:"from"`
		ID   string `json:"id"`
	} `json:"context,omitempty"`
}

type WebhookMedia struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
	Voice    bool   `json:"voice,omitempty"`
}

type WebhookStatus struct {
	ID           string         `json:"id"`
	Status       string         `json:"status"`
	Timestamp    string         `json:"timestamp"`
	RecipientID  string         `json:"recipient_id"`
	Errors       []WebhookError `json:"errors,omitempty"`
	Conversation any            `json:"conversation,omitempty"`
	Pricing      any            `json:"pricing,omitempty"`
}

type WebhookError struct {
	Code      int    `json:"code"`
	Subcode   int    `json:"error_subcode"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	UserMsg   string `json:"error_user_msg"`
	ErrorData struct {
		Details string `json:"details"`
	} `json:"error_data"`
	FBTraceID string `json:"fbtrace_id"`
}

// ParsePayload deserializes the webhook body.
func ParsePayload(body []byte) (*WebhookPayload, error) {
	var p WebhookPayload
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("decoding webhook payload: %w", err)
	}
	return &p, nil
}

// ExtractMessages flattens inbound messages out of the nested webhook payload.
func (p *WebhookPayload) ExtractMessages() []ParsedMessage {
	var out []ParsedMessage
	for _, e := range p.Entry {
		for _, c := range e.Changes {
			if c.Field != "messages" {
				continue
			}
			contactName := ""
			if len(c.Value.Contacts) > 0 {
				contactName = c.Value.Contacts[0].Profile.Name
			}
			for _, m := range c.Value.Messages {
				pm := ParsedMessage{
					From:          m.From,
					ID:            m.ID,
					Timestamp:     parseUnixSeconds(m.Timestamp),
					Type:          m.Type,
					ContactName:   contactName,
					PhoneNumberID: c.Value.Metadata.PhoneNumberID,
				}
				if m.Context != nil {
					pm.ContextID = m.Context.ID
				}
				switch m.Type {
				case "text":
					if m.Text != nil {
						pm.Text = m.Text.Body
					}
				case "image":
					applyMedia(&pm, m.Image)
				case "video":
					applyMedia(&pm, m.Video)
				case "audio":
					applyMedia(&pm, m.Audio)
				case "voice":
					applyMedia(&pm, m.Voice)
				case "document":
					applyMedia(&pm, m.Document)
				case "sticker":
					applyMedia(&pm, m.Sticker)
				case "interactive":
					if m.Interactive != nil {
						if m.Interactive.ButtonReply != nil {
							pm.ButtonReplyID = m.Interactive.ButtonReply.ID
							pm.Text = m.Interactive.ButtonReply.Title
						}
						if m.Interactive.ListReply != nil {
							pm.ListReplyID = m.Interactive.ListReply.ID
							pm.Text = m.Interactive.ListReply.Title
						}
					}
				case "button":
					if m.Button != nil {
						pm.ButtonReplyID = m.Button.Payload
						pm.Text = m.Button.Text
					}
				}
				out = append(out, pm)
			}
		}
	}
	return out
}

// ExtractStatuses pulls status (delivered / read / failed) events.
func (p *WebhookPayload) ExtractStatuses() []ParsedStatus {
	var out []ParsedStatus
	for _, e := range p.Entry {
		for _, c := range e.Changes {
			if c.Field != "messages" {
				continue
			}
			for _, s := range c.Value.Statuses {
				ps := ParsedStatus{
					MessageID: s.ID,
					Status:    s.Status,
					Timestamp: parseUnixSeconds(s.Timestamp),
				}
				if len(s.Errors) > 0 {
					err := s.Errors[0]
					ps.UserMsg = firstNonEmptyStr(err.ErrorData.Details, err.UserMsg, err.Message)
				}
				out = append(out, ps)
			}
		}
	}
	return out
}

// ExtractTemplateStatusUpdates pulls template approval/rejection events.
func (p *WebhookPayload) ExtractTemplateStatusUpdates() []ParsedTemplateStatus {
	var out []ParsedTemplateStatus
	for _, e := range p.Entry {
		for _, c := range e.Changes {
			if c.Field != "message_template_status_update" {
				continue
			}
			out = append(out, ParsedTemplateStatus{
				WABAID:         e.ID,
				Event:          c.Value.Event,
				TemplateName:   c.Value.MessageTemplateName,
				Language:       c.Value.MessageTemplateLanguage,
				Reason:         c.Value.Reason,
				MetaTemplateID: stringifyTemplateID(c.Value.MessageTemplateID),
			})
		}
	}
	return out
}

// VerifySignature validates Meta's full X-Hub-Signature-256 header value (e.g. "sha256=abc...") against the raw body.
func VerifySignature(body []byte, signatureHeader, appSecret string) bool {
	if appSecret == "" || signatureHeader == "" {
		return false
	}
	parts := strings.SplitN(signatureHeader, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(parts[1]))
}

func applyMedia(pm *ParsedMessage, m *WebhookMedia) {
	if m == nil {
		return
	}
	pm.MediaID = m.ID
	pm.MediaMimeType = m.MimeType
	pm.Caption = m.Caption
	pm.Filename = m.Filename
}

func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseUnixSeconds(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}

func stringifyTemplateID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}
