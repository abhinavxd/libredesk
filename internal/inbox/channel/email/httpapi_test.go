package email

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/httpapi"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

// fakeProvider is a test double for httpapi.Provider.
type fakeProvider struct {
	sentMsg   httpapi.OutboundEmail
	sendErr   error
	inbound   httpapi.InboundEmail
	verifyErr error
}

func (f *fakeProvider) Name() string { return "fake" }

func (f *fakeProvider) Send(ctx context.Context, msg httpapi.OutboundEmail) (string, error) {
	f.sentMsg = msg
	if f.sendErr != nil {
		return "", f.sendErr
	}
	return "msg-id-123", nil
}

func (f *fakeProvider) VerifyAndParseWebhook(ctx context.Context, headers http.Header, body []byte) (httpapi.InboundEmail, error) {
	if f.verifyErr != nil {
		return httpapi.InboundEmail{}, f.verifyErr
	}
	return f.inbound, nil
}

// fakeMessageStore is a test double for inbox.MessageStore.
type fakeMessageStore struct {
	existing   map[string]bool
	enqueued   []models.IncomingMessage
	existsErr  error
	enqueueErr error
}

func (f *fakeMessageStore) MessageExists(id string) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}
	return f.existing[id], nil
}

func (f *fakeMessageStore) EnqueueIncoming(msg models.IncomingMessage) error {
	if f.enqueueErr != nil {
		return f.enqueueErr
	}
	f.enqueued = append(f.enqueued, msg)
	return nil
}

// fakeUserStore is a test double for inbox.UserStore.
type fakeUserStore struct {
	blocked map[string]bool
}

func (f *fakeUserStore) GetAgent(id int, email string) (umodels.User, error) {
	return umodels.User{}, nil
}

func (f *fakeUserStore) IsEmailBlocked(email string) (bool, error) {
	return f.blocked[email], nil
}

func newTestLogger() *logf.Logger {
	lo := logf.New(logf.Opts{})
	return &lo
}

func TestSendViaHTTPAPI(t *testing.T) {
	provider := &fakeProvider{}
	e := &Email{
		transport:            imodels.TransportHTTPAPI,
		httpProvider:         provider,
		from:                 "support@example.com",
		enablePlusAddressing: true,
		headers:              map[string]string{"X-Custom": "1"},
		lo:                   newTestLogger(),
	}

	err := e.Send(models.OutboundMessage{
		From:             "Support <support@example.com>",
		To:               []string{"user@example.com"},
		Subject:          "Hello",
		Content:          "<p>hi</p>",
		ContentType:      "html",
		SourceID:         "msg-1",
		InReplyTo:        "orig-1",
		References:       []string{"ref-1", "ref-2"},
		ConversationUUID: "conv-uuid",
		Attachments: attachment.Attachments{
			{Name: "a.txt", Content: []byte("data"), ContentType: "text/plain"},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "Support <support@example.com>", provider.sentMsg.From)
	assert.Equal(t, []string{"user@example.com"}, provider.sentMsg.To)
	assert.Equal(t, "<p>hi</p>", provider.sentMsg.HTML)
	assert.Equal(t, "1", provider.sentMsg.Headers["X-Custom"])
	assert.Equal(t, "support@example.com", provider.sentMsg.Headers[headerLibredeskLoopPrevention])
	assert.Equal(t, "<msg-1>", provider.sentMsg.Headers[headerMessageID])
	assert.Equal(t, "<orig-1>", provider.sentMsg.Headers[headerInReplyTo])
	assert.Equal(t, "conv-uuid", provider.sentMsg.Headers[headerLibredeskConversationID])
	assert.Contains(t, provider.sentMsg.Headers[headerReferences], "<ref-1>")
	assert.Equal(t, "support+conv-conv-uuid@example.com", provider.sentMsg.ReplyTo)
	require.Len(t, provider.sentMsg.Attachments, 1)
	assert.Equal(t, "a.txt", provider.sentMsg.Attachments[0].Filename)
	assert.Equal(t, []byte("data"), provider.sentMsg.Attachments[0].Content)
}

func TestSendViaHTTPAPIError(t *testing.T) {
	provider := &fakeProvider{sendErr: errors.New("boom")}
	e := &Email{
		transport:    imodels.TransportHTTPAPI,
		httpProvider: provider,
		from:         "a@example.com",
		lo:           newTestLogger(),
	}

	err := e.Send(models.OutboundMessage{From: "a@example.com", Subject: "x"})
	require.Error(t, err)
}

func newWebhookTestEmail(provider *fakeProvider, msgStore *fakeMessageStore, userStore *fakeUserStore) *Email {
	return &Email{
		id:           42,
		transport:    imodels.TransportHTTPAPI,
		httpProvider: provider,
		messageStore: msgStore,
		userStore:    userStore,
		lo:           newTestLogger(),
	}
}

func TestReceiveWebhook(t *testing.T) {
	provider := &fakeProvider{
		inbound: httpapi.InboundEmail{
			MessageID: "msg-abc",
			From:      "Jane Doe <jane@example.com>",
			To:        []string{"support+conv-11111111-1111-4111-8111-111111111111@example.com"},
			Subject:   "Re: Help",
			HTML:      "<p>hi</p>",
		},
	}
	msgStore := &fakeMessageStore{existing: map[string]bool{}}
	userStore := &fakeUserStore{blocked: map[string]bool{}}
	e := newWebhookTestEmail(provider, msgStore, userStore)

	err := e.ReceiveWebhook(context.Background(), http.Header{}, []byte(`{}`))
	require.NoError(t, err)
	require.Len(t, msgStore.enqueued, 1)

	got := msgStore.enqueued[0]
	assert.Equal(t, ChannelEmail, got.Channel)
	assert.Equal(t, 42, got.InboxID)
	assert.Equal(t, "jane@example.com", got.Contact.Email.String)
	assert.Equal(t, "Jane", got.Contact.FirstName)
	assert.Equal(t, "Doe", got.Contact.LastName)
	assert.Equal(t, "msg-abc", got.SourceID.String)
	assert.Equal(t, "11111111-1111-4111-8111-111111111111", got.ConversationUUIDFromReplyTo)
	assert.Equal(t, models.ContentTypeHTML, got.ContentType)
	assert.Equal(t, "<p>hi</p>", got.Content)
}

func TestReceiveWebhookVerifyFailure(t *testing.T) {
	provider := &fakeProvider{verifyErr: errors.New("bad signature")}
	msgStore := &fakeMessageStore{existing: map[string]bool{}}
	userStore := &fakeUserStore{blocked: map[string]bool{}}
	e := newWebhookTestEmail(provider, msgStore, userStore)

	err := e.ReceiveWebhook(context.Background(), http.Header{}, []byte(`{}`))
	require.Error(t, err)
	assert.Empty(t, msgStore.enqueued)
}

func TestReceiveWebhookDuplicateMessageSkipped(t *testing.T) {
	provider := &fakeProvider{inbound: httpapi.InboundEmail{MessageID: "dup-1", From: "a@example.com"}}
	msgStore := &fakeMessageStore{existing: map[string]bool{"dup-1": true}}
	userStore := &fakeUserStore{blocked: map[string]bool{}}
	e := newWebhookTestEmail(provider, msgStore, userStore)

	err := e.ReceiveWebhook(context.Background(), http.Header{}, []byte(`{}`))
	require.NoError(t, err)
	assert.Empty(t, msgStore.enqueued)
}

func TestReceiveWebhookBlockedContactSkipped(t *testing.T) {
	provider := &fakeProvider{inbound: httpapi.InboundEmail{MessageID: "msg-2", From: "blocked@example.com"}}
	msgStore := &fakeMessageStore{existing: map[string]bool{}}
	userStore := &fakeUserStore{blocked: map[string]bool{"blocked@example.com": true}}
	e := newWebhookTestEmail(provider, msgStore, userStore)

	err := e.ReceiveWebhook(context.Background(), http.Header{}, []byte(`{}`))
	require.NoError(t, err)
	assert.Empty(t, msgStore.enqueued)
}

func TestReceiveWebhookWrongTransportRejected(t *testing.T) {
	provider := &fakeProvider{}
	msgStore := &fakeMessageStore{existing: map[string]bool{}}
	userStore := &fakeUserStore{blocked: map[string]bool{}}
	e := newWebhookTestEmail(provider, msgStore, userStore)
	e.transport = imodels.TransportSMTPIMAP

	err := e.ReceiveWebhook(context.Background(), http.Header{}, []byte(`{}`))
	require.Error(t, err)
}

func TestReceiveWebhookFromRawMessage(t *testing.T) {
	raw := "From: Jane Doe <jane@example.com>\r\n" +
		"To: support+conv-11111111-1111-4111-8111-111111111111@example.com\r\n" +
		"Subject: Re: Help\r\n" +
		"Message-ID: <m123@example.com>\r\n" +
		"In-Reply-To: <orig@example.com>\r\n" +
		"References: <orig@example.com> <mid2@example.com>\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n" +
		"\r\n" +
		"hello there\r\n"

	provider := &fakeProvider{inbound: httpapi.InboundEmail{RawMessage: []byte(raw)}}
	msgStore := &fakeMessageStore{existing: map[string]bool{}}
	userStore := &fakeUserStore{blocked: map[string]bool{}}
	e := newWebhookTestEmail(provider, msgStore, userStore)

	err := e.ReceiveWebhook(context.Background(), http.Header{}, []byte(`{}`))
	require.NoError(t, err)
	require.Len(t, msgStore.enqueued, 1)

	got := msgStore.enqueued[0]
	assert.Equal(t, ChannelEmail, got.Channel)
	assert.Equal(t, 42, got.InboxID)
	assert.Equal(t, "jane@example.com", got.Contact.Email.String)
	assert.Equal(t, "Jane", got.Contact.FirstName)
	assert.Equal(t, "Doe", got.Contact.LastName)
	assert.Equal(t, "m123@example.com", got.SourceID.String)
	assert.Equal(t, "orig@example.com", got.InReplyTo)
	assert.Equal(t, []string{"orig@example.com", "mid2@example.com"}, got.References)
	assert.Equal(t, "11111111-1111-4111-8111-111111111111", got.ConversationUUIDFromReplyTo)
	assert.Contains(t, got.Content, "hello there")
}

func TestReceiveWebhookConvUUIDFromEnvelopeRecipient(t *testing.T) {
	// MIME To: is the plain address (rewritten by a forwarder); the plus-addressed conversation
	// UUID is only recoverable from the envelope recipient (received_for).
	raw := "From: Jane <jane@example.com>\r\n" +
		"To: support@example.com\r\n" +
		"Subject: Re: Help\r\n" +
		"Message-ID: <m9@example.com>\r\n\r\nbody\r\n"

	provider := &fakeProvider{inbound: httpapi.InboundEmail{
		RawMessage: []byte(raw),
		Recipients: []string{"support+conv-22222222-2222-4222-8222-222222222222@example.com"},
	}}
	msgStore := &fakeMessageStore{existing: map[string]bool{}}
	userStore := &fakeUserStore{blocked: map[string]bool{}}
	e := newWebhookTestEmail(provider, msgStore, userStore)

	err := e.ReceiveWebhook(context.Background(), http.Header{}, []byte(`{}`))
	require.NoError(t, err)
	require.Len(t, msgStore.enqueued, 1)
	assert.Equal(t, "22222222-2222-4222-8222-222222222222", msgStore.enqueued[0].ConversationUUIDFromReplyTo)
}

func TestReceiveWebhookFromRawMessageBlockedSender(t *testing.T) {
	raw := "From: blocked@example.com\r\nSubject: hi\r\nMessage-ID: <b1@example.com>\r\n\r\nbody\r\n"
	provider := &fakeProvider{inbound: httpapi.InboundEmail{RawMessage: []byte(raw)}}
	msgStore := &fakeMessageStore{existing: map[string]bool{}}
	userStore := &fakeUserStore{blocked: map[string]bool{"blocked@example.com": true}}
	e := newWebhookTestEmail(provider, msgStore, userStore)

	err := e.ReceiveWebhook(context.Background(), http.Header{}, []byte(`{}`))
	require.NoError(t, err)
	assert.Empty(t, msgStore.enqueued)
}
