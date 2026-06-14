package whatsapp

import "testing"

func TestExtractMessages_Text(t *testing.T) {
	body := []byte(`{
		"object": "whatsapp_business_account",
		"entry": [{
			"id": "WABA-1",
			"changes": [{
				"field": "messages",
				"value": {
					"messaging_product": "whatsapp",
					"metadata": {"display_phone_number": "+1", "phone_number_id": "PN-1"},
					"contacts": [{"profile": {"name": "Jane Doe"}, "wa_id": "919876543210"}],
					"messages": [{
						"from": "919876543210",
						"id": "wamid.ABC",
						"timestamp": "1716000000",
						"type": "text",
						"text": {"body": "hello"}
					}]
				}
			}]
		}]
	}`)
	p, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	msgs := p.ExtractMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.From != "919876543210" || m.ID != "wamid.ABC" || m.Type != "text" || m.Text != "hello" {
		t.Fatalf("unexpected parsed message: %+v", m)
	}
	if m.ContactName != "Jane Doe" {
		t.Fatalf("expected contact name Jane Doe, got %q", m.ContactName)
	}
}

func TestExtractMessages_ImageWithCaption(t *testing.T) {
	body := []byte(`{
		"entry": [{
			"changes": [{
				"field": "messages",
				"value": {
					"messages": [{
						"from": "1", "id": "id1", "timestamp": "1716000000", "type": "image",
						"image": {"id": "media-1", "mime_type": "image/jpeg", "caption": "see this"}
					}]
				}
			}]
		}]
	}`)
	p, _ := ParsePayload(body)
	m := p.ExtractMessages()[0]
	if m.MediaID != "media-1" || m.MediaMimeType != "image/jpeg" || m.Caption != "see this" {
		t.Fatalf("media not extracted correctly: %+v", m)
	}
}

func TestExtractMessages_TemplateQuickReplyButton(t *testing.T) {
	body := []byte(`{
		"entry": [{
			"changes": [{
				"field": "messages",
				"value": {
					"messages": [{
						"from": "1", "id": "id1", "timestamp": "1716000000", "type": "button",
						"button": {"text": "Yes, confirm", "payload": "confirm-payload"}
					}]
				}
			}]
		}]
	}`)
	p, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	m := p.ExtractMessages()[0]
	if m.Text != "Yes, confirm" || m.ButtonReplyID != "confirm-payload" {
		t.Fatalf("button reply not extracted: %+v", m)
	}
}

func TestExtractStatuses(t *testing.T) {
	body := []byte(`{
		"entry": [{
			"changes": [{
				"field": "messages",
				"value": {
					"statuses": [{
						"id": "wamid.OUT",
						"status": "delivered",
						"timestamp": "1716000000",
						"recipient_id": "919876543210"
					}]
				}
			}]
		}]
	}`)
	p, _ := ParsePayload(body)
	st := p.ExtractStatuses()
	if len(st) != 1 || st[0].MessageID != "wamid.OUT" || st[0].Status != "delivered" {
		t.Fatalf("unexpected statuses: %+v", st)
	}
}

func TestExtractTemplateStatusUpdate(t *testing.T) {
	body := []byte(`{
		"entry": [{
			"id": "WABA-1",
			"changes": [{
				"field": "message_template_status_update",
				"value": {
					"event": "APPROVED",
					"message_template_id": 1234567890,
					"message_template_name": "order_status",
					"message_template_language": "en_US"
				}
			}]
		}]
	}`)
	p, _ := ParsePayload(body)
	ts := p.ExtractTemplateStatusUpdates()
	if len(ts) != 1 {
		t.Fatalf("expected 1 template status, got %d", len(ts))
	}
	if ts[0].Event != "APPROVED" || ts[0].TemplateName != "order_status" ||
		ts[0].Language != "en_US" || ts[0].MetaTemplateID != "1234567890" {
		t.Fatalf("unexpected template status: %+v", ts[0])
	}
}

func TestExtractTemplateStatusUpdate_LargeID(t *testing.T) {
	body := []byte(`{
		"entry": [{
			"id": "WABA-1",
			"changes": [{
				"field": "message_template_status_update",
				"value": {
					"event": "APPROVED",
					"message_template_id": 123456789012345678,
					"message_template_name": "order_status",
					"message_template_language": "en_US"
				}
			}]
		}]
	}`)
	p, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ts := p.ExtractTemplateStatusUpdates()
	if len(ts) != 1 || ts[0].MetaTemplateID != "123456789012345678" {
		t.Fatalf("large template id lost precision: %+v", ts)
	}
}

// wrapValue wraps a single change "value" object in the full webhook envelope and parses it.
func wrapValue(t *testing.T, valueJSON string) *WebhookPayload {
	t.Helper()
	body := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"WABA-1","changes":[{"field":"messages","value":` + valueJSON + `}]}]}`)
	p, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return p
}

func TestParsePayload_InvalidJSON(t *testing.T) {
	if _, err := ParsePayload([]byte(`{"entry": [`)); err == nil {
		t.Fatal("expected error for malformed json, got nil")
	}
}

func TestParsePayload_EmptyBody(t *testing.T) {
	if _, err := ParsePayload(nil); err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

func TestExtractMessages_MediaTypes(t *testing.T) {
	cases := []struct {
		name     string
		msgType  string
		field    string
		mime     string
		filename string
	}{
		{name: "document", msgType: "document", field: "document", mime: "application/pdf", filename: "invoice.pdf"},
		{name: "audio", msgType: "audio", field: "audio", mime: "audio/ogg"},
		{name: "voice", msgType: "voice", field: "voice", mime: "audio/ogg"},
		{name: "video", msgType: "video", field: "video", mime: "video/mp4"},
		{name: "sticker", msgType: "sticker", field: "sticker", mime: "image/webp"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			media := `{"id":"media-x","mime_type":"` + c.mime + `"`
			if c.filename != "" {
				media += `,"filename":"` + c.filename + `"`
			}
			media += `}`
			p := wrapValue(t, `{"messages":[{"from":"1","id":"id1","timestamp":"1716000000","type":"`+c.msgType+`","`+c.field+`":`+media+`}]}`)
			m := p.ExtractMessages()[0]
			if m.Type != c.msgType || m.MediaID != "media-x" || m.MediaMimeType != c.mime {
				t.Fatalf("%s not extracted: %+v", c.name, m)
			}
			if c.filename != "" && m.Filename != c.filename {
				t.Fatalf("%s filename = %q, want %q", c.name, m.Filename, c.filename)
			}
		})
	}
}

func TestExtractMessages_InteractiveButtonReply(t *testing.T) {
	p := wrapValue(t, `{"messages":[{"from":"1","id":"id1","timestamp":"1716000000","type":"interactive",
		"interactive":{"type":"button_reply","button_reply":{"id":"btn-1","title":"Track order"}}}]}`)
	m := p.ExtractMessages()[0]
	if m.ButtonReplyID != "btn-1" || m.Text != "Track order" {
		t.Fatalf("interactive button_reply not extracted: %+v", m)
	}
}

func TestExtractMessages_InteractiveListReply(t *testing.T) {
	p := wrapValue(t, `{"messages":[{"from":"1","id":"id1","timestamp":"1716000000","type":"interactive",
		"interactive":{"type":"list_reply","list_reply":{"id":"opt-2","title":"Refund","description":"Request a refund"}}}]}`)
	m := p.ExtractMessages()[0]
	if m.ListReplyID != "opt-2" || m.Text != "Refund" {
		t.Fatalf("interactive list_reply not extracted: %+v", m)
	}
}

func TestExtractMessages_ReplyContext(t *testing.T) {
	p := wrapValue(t, `{"messages":[{"from":"1","id":"id2","timestamp":"1716000000","type":"text",
		"text":{"body":"replying"},"context":{"from":"1","id":"wamid.QUOTED"}}]}`)
	m := p.ExtractMessages()[0]
	if m.ContextID != "wamid.QUOTED" {
		t.Fatalf("reply context not extracted: %+v", m)
	}
}

func TestExtractMessages_UnsupportedTypePreserved(t *testing.T) {
	p := wrapValue(t, `{"messages":[{"from":"1","id":"id3","timestamp":"1716000000","type":"unsupported",
		"errors":[{"code":131051,"title":"Unsupported message type"}]}]}`)
	m := p.ExtractMessages()[0]
	if m.Type != "unsupported" {
		t.Fatalf("expected type preserved as unsupported, got %q", m.Type)
	}
}

func TestExtractMessages_MultipleMessagesAndEntries(t *testing.T) {
	body := []byte(`{"entry":[
		{"id":"WABA-1","changes":[{"field":"messages","value":{"messages":[
			{"from":"1","id":"a","timestamp":"1716000000","type":"text","text":{"body":"one"}},
			{"from":"1","id":"b","timestamp":"1716000001","type":"text","text":{"body":"two"}}
		]}}]},
		{"id":"WABA-1","changes":[{"field":"messages","value":{"messages":[
			{"from":"2","id":"c","timestamp":"1716000002","type":"text","text":{"body":"three"}}
		]}}]}
	]}`)
	p, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	msgs := p.ExtractMessages()
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages across entries, got %d", len(msgs))
	}
	if msgs[0].ID != "a" || msgs[1].ID != "b" || msgs[2].ID != "c" {
		t.Fatalf("messages out of order or missing: %+v", msgs)
	}
}

func TestExtractMessages_NonMessageFieldSkipped(t *testing.T) {
	body := []byte(`{"entry":[{"id":"WABA-1","changes":[{"field":"message_template_status_update",
		"value":{"event":"APPROVED","message_template_name":"x"}}]}]}`)
	p, err := ParsePayload(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if msgs := p.ExtractMessages(); len(msgs) != 0 {
		t.Fatalf("expected no messages for non-messages field, got %d", len(msgs))
	}
}

func TestExtractStatuses_FailedWithError(t *testing.T) {
	p := wrapValue(t, `{"statuses":[{"id":"wamid.OUT","status":"failed","timestamp":"1716000000","recipient_id":"919876543210",
		"errors":[{"code":131047,"error_subcode":2655000,"title":"Re-engagement message",
		"error_data":{"details":"Message failed to send because more than 24 hours have passed."},"fbtrace_id":"trace-1"}]}]}`)
	st := p.ExtractStatuses()
	if len(st) != 1 {
		t.Fatalf("expected 1 status, got %d", len(st))
	}
	s := st[0]
	if s.Status != "failed" {
		t.Fatalf("failed status fields not extracted: %+v", s)
	}
	if s.UserMsg != "Message failed to send because more than 24 hours have passed." {
		t.Fatalf("expected error_data.details as UserMsg, got %q", s.UserMsg)
	}
}

func TestExtractStatuses_UnknownStatusDoesNotBreak(t *testing.T) {
	p := wrapValue(t, `{"statuses":[{"id":"wamid.OUT","status":"deleted","timestamp":"1716000000","recipient_id":"1"}]}`)
	st := p.ExtractStatuses()
	if len(st) != 1 || st[0].Status != "deleted" {
		t.Fatalf("unknown status not passed through: %+v", st)
	}
}
