package telegram

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestInboundMessageSelection(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`{"message_id":1,"chat":{"id":4500000000000,"type":"private"},"from":{"id":4500000000000}}`, true},
		{`{"message_id":0,"chat":{"id":1,"type":"private"},"from":{"id":1}}`, false},
		{`{"message_id":-1,"chat":{"id":1,"type":"private"},"from":{"id":1}}`, false},
		{`{"message_id":1,"chat":{"id":-1,"type":"group"},"from":{"id":1}}`, false},
		{`{"message_id":1,"chat":{"id":1,"type":"channel"},"from":{"id":1}}`, false},
		{`{"message_id":1,"chat":{"id":1,"type":"private"},"from":{"id":1,"is_bot":true}}`, false},
		{`{"message_id":1,"chat":{"id":1,"type":"private"}}`, false},
		{`{"message_id":1,"chat":{"id":1,"type":"private"},"from":{"id":2}}`, false},
	} {
		var message Message
		if err := json.Unmarshal([]byte(tc.body), &message); err != nil {
			t.Fatal(err)
		}
		if message.Inbound() != tc.want {
			t.Errorf("body=%s", tc.body)
		}
	}
	for _, raw := range []string{`{"edited_message":{"message_id":1}}`, `{"channel_post":{"message_id":1}}`, `{"callback_query":{}}`} {
		var update Update
		json.Unmarshal([]byte(raw), &update)
		if update.Message != nil {
			t.Errorf("unexpected message: %s", raw)
		}
	}
}

func TestMessageContentAndAttachments(t *testing.T) {
	for _, tc := range []struct{ body, text, kind, file string }{
		{`{"text":"hello","caption":"ignored"}`, "hello", "", ""},
		{`{"caption":"caption"}`, "caption", "", ""},
		{`{"photo":[{"file_id":"a","width":20,"height":20},{"file_id":"b","width":80,"height":80},{"file_id":"c","width":10,"height":10}]}`, "", "photo", "b"},
		{`{"document":{"file_id":"d"}}`, "", "document", "d"},
		{`{"animation":{"file_id":"a"}}`, "", "animation", "a"},
		{`{"video":{"file_id":"v"}}`, "", "video", "v"},
		{`{"audio":{"file_id":"a"}}`, "", "audio", "a"},
		{`{"voice":{"file_id":"v"}}`, "", "voice", "v"},
		{`{"sticker":{"file_id":"s"}}`, "", "sticker", "s"},
		{`{"video_note":{"file_id":"v"}}`, "", "video", "v"},
		{`{"location":{"latitude":12.1,"longitude":77.2}}`, "https://maps.google.com/?q=12.100000,77.200000", "", ""},
		{`{"contact":{"first_name":"A","last_name":"B","phone_number":"123"}}`, "A B\n123", "", ""},
		{`{"venue":{"title":"Place","address":"Street","location":{"latitude":12.1,"longitude":77.2}}}`, "Place\nStreet\nhttps://maps.google.com/?q=12.100000,77.200000", "", ""},
		{`{}`, "", "", ""},
	} {
		var message Message
		json.Unmarshal([]byte(tc.body), &message)
		if got := message.Content(); got != tc.text {
			t.Errorf("content=%q want=%q", got, tc.text)
		}
		file, kind := message.Attachment()
		if file.ID != tc.file || kind != tc.kind {
			t.Errorf("attachment=%v kind=%s", file, kind)
		}
	}
	if SourceID(1, 2, 3) == SourceID(2, 2, 3) || SourceID(1, 2, 3) == SourceID(1, 3, 3) || !strings.Contains(SourceID(1, 2, 3), "telegram:") {
		t.Fatal("source IDs must be scoped by inbox and chat")
	}
}

func TestBusinessMessageRouting(t *testing.T) {
	msg := Message{ID: 3, From: &User{ID: 2}, Chat: Chat{ID: 2, Type: "private"}}
	if msg.Outgoing() || msg.SourceID(1) != SourceID(1, 2, 3) {
		t.Fatal("ordinary chat routing changed")
	}
	msg.BusinessConnectionID = "business-a"
	if !msg.Inbound() || msg.Outgoing() {
		t.Fatal("business customer message rejected")
	}
	if msg.SourceID(1) != BusinessSourceID(1, "business-a", 2, 3) || msg.SourceID(1) == SourceID(1, 2, 3) {
		t.Fatal("business and bot messages collided")
	}
	msg.From.ID = 4
	if !msg.Inbound() || !msg.Outgoing() {
		t.Fatal("business owner message not recognized")
	}
	msg.From = nil
	if msg.Outgoing() || msg.Inbound() {
		t.Fatal("missing sender accepted")
	}
}

func TestMessageIDFromSource(t *testing.T) {
	for _, tc := range []struct {
		source, business string
		want             int64
	}{
		{"telegram:1:2:3", "", 3},
		{"telegram:1:business:a:2:3", "a", 3},
		{"telegram:1:business:b:2:3", "a", 0},
		{"telegram:2:2:3", "", 0},
		{"telegram:1:4:3", "", 0},
		{"telegram:1:2:0", "", 0},
		{"telegram:1:2:-1", "", 0},
		{"invalid", "", 0},
	} {
		id, err := MessageIDFromSource(tc.source, 1, 2, tc.business)
		if id != tc.want || (err != nil) != (tc.want == 0) {
			t.Fatalf("source=%s id=%d err=%v", tc.source, id, err)
		}
	}
}

func TestCallbackQuery(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{"id":"query-1","data":"Billing","from":{"id":42},"message":{"message_id":1,"chat":{"id":42,"type":"private"}}}`, true},
		{`{"id":"query-1","data":"Billing","from":{"id":42},"message":{"message_id":1,"business_connection_id":"business","message_thread_id":7,"chat":{"id":42,"type":"private"}}}`, true},
		{`{"data":"Billing"}`, false},
		{`{"id":"query-1","data":"  "}`, false},
		{`{"id":"query-1","data":"Billing"}`, false},
		{`{"id":"query-1","data":"Billing","from":{"id":43},"message":{"message_id":1,"chat":{"id":42,"type":"private"}}}`, false},
		{`{"id":"query-1","data":"Billing","from":{"id":42,"is_bot":true},"message":{"message_id":1,"chat":{"id":42,"type":"private"}}}`, false},
		{`{"id":"query-1","data":"Billing","from":{"id":42},"message":{"message_id":1,"chat":{"id":-42,"type":"group"}}}`, false},
	} {
		var query CallbackQuery
		if err := json.Unmarshal([]byte(tc.body), &query); err != nil {
			t.Fatal(err)
		}
		msg, valid := query.IncomingMessage()
		if valid != tc.valid {
			t.Fatalf("valid=%v for %s", valid, tc.body)
		}
		if !valid {
			continue
		}
		if msg.Text != "Billing" || msg.CallbackID != query.ID || msg.ThreadID != query.Message.ThreadID || msg.BusinessConnectionID != query.Message.BusinessConnectionID {
			t.Fatalf("message=%+v", msg)
		}
		expected := fmt.Sprintf("telegram:3:%s:42:callback:query-1", msg.BusinessConnectionID)
		if msg.SourceID(3) != expected {
			t.Fatalf("source=%s", msg.SourceID(3))
		}
		if _, err := MessageIDFromSource(msg.SourceID(3), 3, 42, msg.BusinessConnectionID); err == nil {
			t.Fatal("callback accepted as a quoted message")
		}
	}
}
