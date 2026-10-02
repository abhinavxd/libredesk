package ws

import (
	"encoding/json"
	"testing"

	"github.com/jakedolan443/fernmail/internal/ws/models"
	"github.com/zerodha/logf"
)

func TestTypingEnforcesSenderAndRecipientAccess(t *testing.T) {
	lo := logf.New(logf.Opts{})
	hub := NewHub(&lo, nil)
	policy := &mailboxAccessStub{allowed: map[int]bool{1: true, 2: true, 3: false}}
	hub.SetConversationStore(policy)
	clients := []*Client{
		{ID: 1, Name: "Alice", Hub: hub, Send: make(chan models.WSMessage, 4)},
		{ID: 1, Name: "Alice", Hub: hub, Send: make(chan models.WSMessage, 4)},
		{ID: 2, Name: "Bob", Hub: hub, Send: make(chan models.WSMessage, 4)},
		{ID: 3, Name: "Hidden", Hub: hub, Send: make(chan models.WSMessage, 4)},
	}
	for _, client := range clients {
		hub.SubscribeOpenConv(client, "thread")
	}
	frame := json.RawMessage(`{"conversation_uuid":"thread","is_typing":true,"is_private_message":true,"user_id":3,"user_name":"Spoofed"}`)
	clients[0].handleTyping(frame)
	if len(clients[0].Send) != 0 || len(clients[1].Send) != 0 || len(clients[3].Send) != 0 {
		t.Fatal("typing leaked to self or denied reader")
	}
	select {
	case msg := <-clients[2].Send:
		var event struct {
			Type string               `json:"type"`
			Data models.TypingMessage `json:"data"`
		}
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			t.Fatal(err)
		}
		if event.Type != "typing" || event.Data.UserID != 1 || event.Data.UserName != "Alice" || !event.Data.IsPrivateMessage {
			t.Fatalf("wrong event: %+v", event)
		}
	default:
		t.Fatal("authorized teammate received no typing event")
	}
	clients[3].handleTyping(frame)
	if len(clients[2].Send) != 0 {
		t.Fatal("unauthorized sender broadcast typing")
	}
	policy.allowed[2] = false
	clients[0].handleTyping(frame)
	if len(clients[2].Send) != 0 {
		t.Fatal("revoked recipient still received typing")
	}
}
