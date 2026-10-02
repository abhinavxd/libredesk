package main

import (
	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"
	"github.com/knadh/koanf/v2"
	"testing"
)

func TestEmailInboxConfigurationDoesNotInheritAnotherMailbox(t *testing.T) {
	previous := ko
	ko = koanf.New(".")
	t.Cleanup(func() { ko = previous })
	if err := ko.Set("app.log_level", "error"); err != nil {
		t.Fatal(err)
	}
	if err := ko.Set("app.env", "test"); err != nil {
		t.Fatal(err)
	}
	if err := ko.Set("reply_to", "global@example.test"); err != nil {
		t.Fatal(err)
	}
	first, err := initEmailInbox(imodels.Inbox{ID: 1, Name: "First", From: "first@example.test", Config: []byte(`{"reply_to":"private@example.test","email_aliases":[{"address":"private@example.test","enabled":true}]}`)}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := initEmailInbox(imodels.Inbox{ID: 2, Name: "Second", From: "second@example.test", Config: []byte(`{}`)}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if first.ReplyToAddress() != "private@example.test" || second.ReplyToAddress() != "" {
		t.Fatalf("mailbox routing crossed config boundaries: %q %q", first.ReplyToAddress(), second.ReplyToAddress())
	}
	if ko.String("reply_to") != "global@example.test" || ko.Exists("email_aliases") {
		t.Fatal("mailbox config mutated global app configuration")
	}
}
