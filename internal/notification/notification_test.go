package notifier

import (
	"testing"
)

type fakeProvider struct {
	name string
	got  []Message
}

func (f *fakeProvider) Send(m Message) error { f.got = append(f.got, m); return nil }
func (f *fakeProvider) Name() string         { return f.name }

// A settings save swaps the provider on the running service; the next message
// must reach the new one and the old one must see nothing more.
func TestSetProviderRoutesSubsequentSends(t *testing.T) {
	old := &fakeProvider{name: ProviderEmail}
	s := NewService(map[string]Notifier{ProviderEmail: old}, 1, 1, nil)

	if err := s.SendSync(Message{Provider: ProviderEmail, Subject: "before"}); err != nil {
		t.Fatal(err)
	}

	next := &fakeProvider{name: ProviderEmail}
	s.SetProvider(next)

	if err := s.SendSync(Message{Provider: ProviderEmail, Subject: "after"}); err != nil {
		t.Fatal(err)
	}
	if len(old.got) != 1 || old.got[0].Subject != "before" {
		t.Fatalf("old provider got %+v, want only the message sent before the swap", old.got)
	}
	if len(next.got) != 1 || next.got[0].Subject != "after" {
		t.Fatalf("new provider got %+v, want only the message sent after the swap", next.got)
	}
}

// The enabled flag is a closure, so flipping it between sends changes the
// outcome without rebuilding the dispatcher. A nil closure means never.
func TestDispatcherEmailEnabledIsResolvedPerSend(t *testing.T) {
	enabled := false
	d := NewDispatcher(DispatcherOpts{EmailEnabled: func() bool { return enabled }})
	if d.emailEnabled() {
		t.Fatal("expected email disabled")
	}
	enabled = true
	if !d.emailEnabled() {
		t.Fatal("expected email enabled after the flag flipped")
	}
	if NewDispatcher(DispatcherOpts{}).emailEnabled() {
		t.Fatal("nil EmailEnabled must mean disabled, not a nil-func panic")
	}
}
