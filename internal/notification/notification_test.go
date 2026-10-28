package notifier

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/notification/models"
)

type fakeProvider struct {
	name    string
	got     []Message
	closed  chan struct{} // closed by Close
	started chan struct{} // when set, closed once Send has been entered
	release chan struct{} // when set, Send blocks until it is closed
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{name: ProviderEmail, closed: make(chan struct{})}
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
func (f *fakeProvider) Close()       { close(f.closed) }

func (f *fakeProvider) isClosed() bool {
	select {
	case <-f.closed:
		return true
	default:
		return false
	}
}

func waitClosed(t *testing.T, f *fakeProvider) {
	t.Helper()
	select {
	case <-f.closed:
	case <-time.After(time.Second):
		t.Fatal("old provider was not closed")
	}
}

// A settings save swaps the provider on the running service: the next message
// reaches the new one, the old one sees nothing more and is closed.
func TestSetProviderRoutesSubsequentSendsAndClosesOld(t *testing.T) {
	old := newFakeProvider()
	s := NewService(map[string]Notifier{ProviderEmail: old}, 1, 1, nil)

	if err := s.SendSync(Message{Provider: ProviderEmail, Subject: "before"}); err != nil {
		t.Fatal(err)
	}
	next := newFakeProvider()
	s.SetProvider(next)
	if err := s.SendSync(Message{Provider: ProviderEmail, Subject: "after"}); err != nil {
		t.Fatal(err)
	}

	waitClosed(t, old)
	if len(old.got) != 1 || old.got[0].Subject != "before" {
		t.Fatalf("old provider got %+v, want only the message sent before the swap", old.got)
	}
	if len(next.got) != 1 || next.got[0].Subject != "after" {
		t.Fatalf("new provider got %+v, want only the message sent after the swap", next.got)
	}
	if next.isClosed() {
		t.Fatal("new provider must not be closed")
	}
}

// A send stuck on a hung relay must not block the swap: SetProvider returns, new
// sends use the new provider, and the old one is closed only once its send finishes.
func TestSetProviderDoesNotWaitForInFlightSend(t *testing.T) {
	old := newFakeProvider()
	old.started, old.release = make(chan struct{}), make(chan struct{})
	s := NewService(map[string]Notifier{ProviderEmail: old}, 1, 1, nil)

	sendDone := make(chan struct{})
	go func() {
		_ = s.SendSync(Message{Provider: ProviderEmail, Subject: "stuck"})
		close(sendDone)
	}()
	// Wait until the send is inside the provider and blocked there.
	<-old.started

	next := newFakeProvider()
	swapDone := make(chan struct{})
	go func() {
		s.SetProvider(next)
		close(swapDone)
	}()
	select {
	case <-swapDone:
	case <-time.After(time.Second):
		t.Fatal("SetProvider blocked behind a send in flight on the old provider")
	}

	if err := s.SendSync(Message{Provider: ProviderEmail, Subject: "after"}); err != nil {
		t.Fatal(err)
	}
	if len(next.got) != 1 || next.got[0].Subject != "after" {
		t.Fatalf("new provider got %+v, want the message sent after the swap", next.got)
	}

	time.Sleep(50 * time.Millisecond)
	if old.isClosed() {
		t.Fatal("old provider closed under an in-flight send")
	}

	close(old.release)
	<-sendDone
	waitClosed(t, old)
	if len(old.got) != 1 || old.got[0].Subject != "stuck" {
		t.Fatalf("old provider got %+v, want only the in-flight message", old.got)
	}
}

// sliceProvider is a non-comparable value type: comparing two of them with == panics.
type sliceProvider struct{ sent []string }

func (sliceProvider) Send(Message) error { return nil }
func (sliceProvider) Name() string       { return ProviderEmail }

// Swapping between non-comparable value providers must not panic.
func TestSetProviderNonComparable(t *testing.T) {
	s := NewService(map[string]Notifier{ProviderEmail: sliceProvider{}}, 1, 1, nil)
	s.SetProvider(sliceProvider{})
	if err := s.SendSync(Message{Provider: ProviderEmail}); err != nil {
		t.Fatal(err)
	}
}

// Notification emails can be switched off and on again on a running queue.
func TestEmailQueueSetEnabled(t *testing.T) {
	provider := newFakeProvider()
	q := &EmailQueue{outbound: NewService(map[string]Notifier{ProviderEmail: provider}, 1, 1, nil)}
	email := models.Email{Recipient: "agent@example.com", Subject: "Reply"}

	q.SetEnabled(false)
	if q.Enabled() || q.Send(email) || q.SendAfter(email, time.Minute) {
		t.Fatal("disabled queue accepted an email")
	}
	if len(provider.got) != 0 {
		t.Fatalf("disabled queue sent %+v", provider.got)
	}

	q.SetEnabled(true)
	if !q.Enabled() || !q.Send(email) {
		t.Fatal("enabled queue refused an email")
	}
}
