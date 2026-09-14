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
	if got, _ := json.Marshal(convToBroadcast(nil)); string(got) != "null" {
		t.Fatalf("nil conv = %s", got)
	}
}
