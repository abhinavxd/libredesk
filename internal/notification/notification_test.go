package notifier

import (
	"testing"
	"time"
)

type fakeProvider struct {
	name    string
	got     []Message
	closed  bool
	started chan struct{} // when set, closed once Send has been entered
	release chan struct{} // when set, Send blocks until it is closed
}

func (f *fakeProvider) Send(m Message) error {
	if f.started != nil {
		close(f.started)
	}
	if f.release != nil {
		<-f.release
	}
	f.got = append(f.got, m)
	return nil
}
func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Close()       { f.closed = true }

// A settings save swaps the provider on the running service: the next message
// reaches the new one, the old one sees nothing more and is closed.
func TestSetProviderRoutesSubsequentSendsAndClosesOld(t *testing.T) {
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
	if !old.closed {
		t.Fatal("old provider was not closed after the swap")
	}
	if len(next.got) != 1 || next.got[0].Subject != "after" {
		t.Fatalf("new provider got %+v, want only the message sent after the swap", next.got)
	}
	if next.closed {
		t.Fatal("new provider must not be closed")
	}
}

// A swap must not close a provider while one of its sends is still running.
func TestSetProviderWaitsForInFlightSend(t *testing.T) {
	old := &fakeProvider{name: ProviderEmail, started: make(chan struct{}), release: make(chan struct{})}
	s := NewService(map[string]Notifier{ProviderEmail: old}, 1, 1, nil)

	sendDone := make(chan struct{})
	go func() {
		_ = s.SendSync(Message{Provider: ProviderEmail})
		close(sendDone)
	}()
	// Wait until the send is inside the provider and blocked there.
	<-old.started

	swapDone := make(chan struct{})
	go func() {
		s.SetProvider(&fakeProvider{name: ProviderEmail})
		close(swapDone)
	}()

	select {
	case <-swapDone:
		t.Fatal("SetProvider returned while a send was still in flight on the old provider")
	case <-time.After(50 * time.Millisecond):
	}
	if old.closed {
		t.Fatal("old provider closed under an in-flight send")
	}

	close(old.release)
	<-sendDone
	select {
	case <-swapDone:
	case <-time.After(time.Second):
		t.Fatal("SetProvider did not complete once the in-flight send finished")
	}
	if !old.closed {
		t.Fatal("old provider not closed after its send drained")
	}
}

// The email channel flag can be flipped on a running dispatcher.
func TestDispatcherSetEmailEnabled(t *testing.T) {
	d := NewDispatcher(DispatcherOpts{EmailEnabled: false})
	if d.emailEnabled.Load() {
		t.Fatal("expected email disabled")
	}
	d.SetEmailEnabled(true)
	if !d.emailEnabled.Load() {
		t.Fatal("expected email enabled after SetEmailEnabled(true)")
	}
}
