package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/inbox"
	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"
	"github.com/zerodha/logf"
)

type syncMemoryStore struct {
	cursor    map[string]uint32
	failures  map[string]map[uint32]bool
	messages  []models.IncomingMessage
	failUID   uint32
	capacity  int
	notifyAt  int
	delivered chan struct{}
}

func (s *syncMemoryStore) IMAPState(_ int, key string, v uint32) (uint32, []uint32, error) {
	k := fmt.Sprintf("%s/%d", key, v)
	failed := []uint32{}
	for uid := range s.failures[k] {
		failed = append(failed, uid)
	}
	return s.cursor[k], failed, nil
}
func (s *syncMemoryStore) RecordIMAPResult(_ int, key string, v, uid uint32, err error) error {
	k := fmt.Sprintf("%s/%d", key, v)
	if uid > s.cursor[k] {
		s.cursor[k] = uid
	}
	if s.failures[k] == nil {
		s.failures[k] = map[uint32]bool{}
	}
	if err != nil {
		s.failures[k][uid] = true
	} else {
		delete(s.failures[k], uid)
	}
	return nil
}
func (s *syncMemoryStore) EnqueueIncoming(m models.IncomingMessage) error {
	if s.capacity > 0 && len(s.messages) >= s.capacity {
		return inbox.ErrIncomingQueueFull
	}
	if m.UID == s.failUID {
		return errors.New("durable queue unavailable")
	}
	s.messages = append(s.messages, m)
	if s.delivered != nil && len(s.messages) == s.notifyAt {
		close(s.delivered)
	}
	return nil
}

type mailLiteral struct{ *strings.Reader }

func (l mailLiteral) Size() int64 { return int64(l.Len()) }
func TestUIDSyncBackfillsOldMailRetriesFailuresAndHandlesReset(t *testing.T) {
	backend := imapmemserver.New()
	account := imapmemserver.NewUser("test", "secret")
	backend.AddUser(account)
	if err := account.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := imapserver.New(&imapserver.Options{InsecureAuth: true, NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
		return backend.NewSession(), nil, nil
	}})
	go server.Serve(listener)
	t.Cleanup(func() { server.Close() })
	appendMail := func(id, extra string) {
		t.Helper()
		raw := fmt.Sprintf("From: Sender <sender@example.test>\r\nTo: a@example.test\r\nReply-To: replies@example.test\r\nSubject: Old mail\r\n%s%s\r\nBody\r\n", id, extra)
		_, err := account.Append("INBOX", mailLiteral{strings.NewReader(raw)}, &imap.AppendOptions{Time: time.Now().Add(-30 * 24 * time.Hour)})
		if err != nil {
			t.Fatal(err)
		}
	}
	appendMail("Message-ID: <old@test>\r\n", "Auto-Submitted: auto-generated\r\n")
	appendMail("", "X-Autoreply: yes\r\n")
	host, portText, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portText)
	cfg := imodels.IMAPConfig{Host: host, Port: port, Username: "test", Password: "secret", Mailbox: "INBOX", TLSType: "none"}
	store := &syncMemoryStore{cursor: map[string]uint32{}, failures: map[string]map[uint32]bool{}, failUID: 1}
	lo := logf.New(logf.Opts{})
	receiver := &Email{id: 1, messageStore: store, lo: &lo}
	if err = receiver.processMailbox(context.Background(), time.Hour, cfg); err != nil {
		t.Fatal(err)
	}
	if len(store.messages) != 1 || store.messages[0].UID != 2 || store.messages[0].SourceID.String == "" {
		t.Fatalf("old/no-ID/automated mail lost: %+v", store.messages)
	}
	store.failUID = 0
	if err = receiver.processMailbox(context.Background(), time.Hour, cfg); err != nil {
		t.Fatal(err)
	}
	if len(store.messages) != 2 || store.messages[1].UID != 1 {
		t.Fatalf("failed UID not retried past cursor: %+v", store.messages)
	}
	var meta struct {
		ReplyTo   []string `json:"reply_to"`
		AutoReply bool     `json:"auto_reply"`
	}
	if err = json.Unmarshal(store.messages[1].Meta, &meta); err != nil || len(meta.ReplyTo) != 1 || meta.ReplyTo[0] != "replies@example.test" || !meta.AutoReply {
		t.Fatalf("headers not preserved: %+v %v", meta, err)
	}
	if err = receiver.processMailbox(context.Background(), time.Hour, cfg); err != nil {
		t.Fatal(err)
	}
	if len(store.messages) != 2 {
		t.Fatal("completed UIDs fetched again")
	}
	if err = account.Delete("INBOX"); err != nil {
		t.Fatal(err)
	}
	if err = account.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	appendMail("Message-ID: <after-reset@test>\r\n", "")
	if err = receiver.processMailbox(context.Background(), time.Hour, cfg); err != nil {
		t.Fatal(err)
	}
	if len(store.messages) != 3 || store.messages[2].UID != 1 || store.messages[2].UIDValidity == store.messages[0].UIDValidity {
		t.Fatalf("UIDVALIDITY reset skipped mail: %+v", store.messages)
	}
	// Queue backpressure leaves the deferred UID unacknowledged, while already
	// staged UIDs remain committed. Retrying later imports only the deferred one.
	appendMail("Message-ID: <capacity-one@test>\r\n", "")
	appendMail("Message-ID: <capacity-two@test>\r\n", "")
	store.capacity = 4
	if err = receiver.processMailbox(context.Background(), time.Hour, cfg); !errors.Is(err, inbox.ErrIncomingQueueFull) {
		t.Fatal(err)
	}
	if len(store.messages) != 4 {
		t.Fatalf("backpressure accepted %d", len(store.messages))
	}
	store.capacity = 0
	if err = receiver.processMailbox(context.Background(), time.Hour, cfg); err != nil {
		t.Fatal(err)
	}
	if len(store.messages) != 5 || store.messages[4].SourceID.String != "capacity-two@test" {
		t.Fatal("deferred UID skipped or already-staged UID imported again")
	}
	for i := range 125 {
		appendMail(fmt.Sprintf("Message-ID: <batch-%d@test>\r\n", i), "")
	}
	if err = receiver.processMailbox(context.Background(), time.Hour, cfg); !errors.Is(err, errIMAPBacklog) {
		t.Fatal(err)
	}
	if len(store.messages) != 105 {
		t.Fatalf("expected bounded 100-message fetch, got %d", len(store.messages)-5)
	}
	if err = receiver.processMailbox(context.Background(), time.Hour, cfg); err != nil {
		t.Fatal(err)
	}
	if len(store.messages) != 130 {
		t.Fatal("next batch did not resume at last staged UID")
	}
	// The normal receiver immediately continues bounded batches instead of
	// spending an hour between them during the initial historical backfill.
	for i := range 105 {
		appendMail(fmt.Sprintf("Message-ID: <catchup-%d@test>\r\n", i), "")
	}
	store.notifyAt = 235
	store.delivered = make(chan struct{})
	cfg.ReadInterval = "1h"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopped := make(chan struct{})
	go func() { _ = receiver.ReadIncomingMessages(ctx, cfg); close(stopped) }()
	select {
	case <-store.delivered:
	case <-time.After(5 * time.Second):
		cancel()
		<-stopped
		t.Fatal("backlog waited for normal long polling interval")
	}
	cancel()
	<-stopped

}

func TestMissingMessageIDUsesStableDeliveryIdentity(t *testing.T) {
	env := &imap.Envelope{From: []imap.Address{{Mailbox: "sender", Host: "example.test"}}}
	one, err := incomingFromEnvelope(env, 1, "account/INBOX", 4, 12)
	if err != nil {
		t.Fatal(err)
	}
	same, _ := incomingFromEnvelope(env, 1, "account/INBOX", 4, 12)
	other, _ := incomingFromEnvelope(env, 1, "account/INBOX", 4, 13)
	reset, _ := incomingFromEnvelope(env, 1, "account/INBOX", 5, 12)
	if one.SourceID.String == "" || one.SourceID != same.SourceID || one.SourceID == other.SourceID || one.SourceID == reset.SourceID {
		t.Fatal("missing Message-ID fallback must be stable and scoped to UID identity")
	}
}

func TestInvalidTransportConfigurationReturnsErrorWithoutPanic(t *testing.T) {
	lo := logf.New(logf.Opts{})
	for _, authType := range []string{"", imodels.AuthTypeOAuth2} {
		sender := &Email{id: 1, lo: &lo, authType: authType}
		if err := sender.Send(models.OutboundMessage{From: "support@example.test", To: []string{"recipient@example.test"}, Content: "Reply"}); err == nil {
			t.Fatalf("invalid transport %q accepted send", authType)
		}
	}
}
