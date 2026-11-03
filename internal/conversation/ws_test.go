package conversation

import (
	"encoding/json"
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/volatiletech/null/v9"
)

func TestConvToBroadcastOmitsPerUserFields(t *testing.T) {
	c := &cmodels.ConversationListItem{
		ID:                   42,
		UnreadMessageCount:   7,
		MentionedMessageUUID: null.StringFrom("abc-123"),
		Subject:              null.StringFrom("hello"),
		AssignedUserID:       null.IntFrom(7),
		Assignee:             json.RawMessage(`{"id":7,"first_name":"Agent","last_name":"One","avatar_url":"/avatar.png","type":"agent","availability_status":"online"}`),
	}
	b, err := json.Marshal(convToBroadcast(c))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "unread_message_count") || strings.Contains(s, "mentioned_message_uuid") {
		t.Fatalf("per-user field leaked: %s", s)
	}
	if !strings.Contains(s, `"id":42`) || !strings.Contains(s, "hello") {
		t.Fatalf("normal fields missing: %s", s)
	}
	var payload struct {
		Assignee json.RawMessage `json:"assignee"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatal(err)
	}
	wantAssignee, err := json.Marshal(c.Assignee)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload.Assignee) != string(wantAssignee) {
		t.Fatalf("assignee details missing: %s", payload.Assignee)
	}
	for _, assignee := range []json.RawMessage{nil, json.RawMessage(`null`)} {
		c.Assignee = assignee
		b, err := json.Marshal(convToBroadcast(c))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), `"assignee":null`) {
			t.Fatalf("unassigned conversation has a non-null assignee: %s", b)
		}
	}
	if got, _ := json.Marshal(convToBroadcast(nil)); string(got) != "null" {
		t.Fatalf("nil conv = %s", got)
	}
}
