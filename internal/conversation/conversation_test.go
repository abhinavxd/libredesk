package conversation

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAutomationReplyCC(t *testing.T) {
	got, err := automationReplyCC(
		json.RawMessage(`{"cc":["copy@example.com","requester@example.com","copy@example.com",""]}`),
		"requester@example.com",
	)
	if err != nil {
		t.Fatalf("automationReplyCC() error = %v", err)
	}
	want := []string{"copy@example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("automationReplyCC() = %v, want %v", got, want)
	}
}

func TestAutomationReplyCCRejectsMalformedMetadata(t *testing.T) {
	if _, err := automationReplyCC(json.RawMessage(`{"cc":`), "requester@example.com"); err == nil {
		t.Fatal("automationReplyCC() error = nil, want malformed metadata error")
	}
}
