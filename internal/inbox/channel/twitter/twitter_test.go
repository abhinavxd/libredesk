package twitter

import (
	"context"
	"encoding/json"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/stretchr/testify/require"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type testMessageStore struct {
	incoming []cmodels.IncomingMessage
}

func (s *testMessageStore) MessageExists(sourceID string) (bool, error) {
	return false, nil
}

func (s *testMessageStore) EnqueueIncoming(msg cmodels.IncomingMessage) error {
	s.incoming = append(s.incoming, msg)
	return nil
}

type testUserStore struct {
	externalID string
}

func (s testUserStore) GetAgent(id int, email string) (umodels.User, error) {
	return umodels.User{}, nil
}

func (s testUserStore) IsEmailBlocked(email string) (bool, error) {
	return false, nil
}

func (s testUserStore) GetContactExternalID(id int) (string, error) {
	return s.externalID, nil
}

func TestFetchDMsBuildsIncomingMessagesAndSkipsSelf(t *testing.T) {
	store := &testMessageStore{}
	provider := &MockProvider{
		DMCursor: "next-cursor",
		DMEvents: []DMEvent{
			{ID: "self", SenderID: "account", ThreadID: "thread-1", Text: "ignore"},
			{ID: "dm-1", SenderID: "123", ThreadID: "thread-1", Text: "hello"},
		},
		Users: map[string]XUser{
			"123": {ID: "123", Name: "Jane Doe", Username: "jane", AvatarURL: "https://img.example/jane.jpg"},
		},
	}
	var persisted Config
	tw, err := New(store, testUserStore{}, Opts{
		ID:       10,
		Name:     "X Support",
		Config:   Config{AccountUserID: "account", IngestDMs: true, PollInterval: "30s"},
		Provider: provider,
		Lo:       testLogger(),
		TokenRefreshCallback: func(inboxID int, updatedConfig Config) error {
			persisted = updatedConfig
			return nil
		},
	})
	require.NoError(t, err)

	tw.fetchDMs(context.Background())

	require.Len(t, store.incoming, 1)
	msg := store.incoming[0]
	require.Equal(t, ChannelTwitter, msg.Channel)
	require.Equal(t, null.StringFrom("dm-1"), msg.SourceID)
	require.Equal(t, "hello", msg.Content)
	require.Equal(t, "Jane", msg.Contact.FirstName)
	require.Equal(t, "Doe", msg.Contact.LastName)
	require.Equal(t, null.StringFrom("x:123"), msg.Contact.ExternalUserID)

	var meta map[string]any
	require.NoError(t, json.Unmarshal(msg.Meta, &meta))
	require.Equal(t, TwitterSourceDM, meta["twitter_source"])
	require.Equal(t, "thread-1", meta["dm_thread_id"])
	require.Equal(t, "jane", meta["twitter_handle"])
	require.Equal(t, "next-cursor", persisted.DMCursor)
}

func TestFetchMentionsBuildsIncomingMessagesAndSkipsSelf(t *testing.T) {
	store := &testMessageStore{}
	provider := &MockProvider{
		MentionsCursor: "newest",
		Mentions: []Tweet{
			{ID: "self", AuthorID: "account", ConversationID: "thread-a", Text: "ignore"},
			{ID: "tweet-1", AuthorID: "456", ConversationID: "thread-a", AuthorName: "John Smith", AuthorUsername: "john", Text: "@support hi", InReplyToStatusID: "root"},
		},
	}
	var persisted Config
	tw, err := New(store, testUserStore{}, Opts{
		ID:       10,
		Name:     "X Support",
		Config:   Config{AccountUserID: "account", ScreenName: "support", IngestMentions: true, PollInterval: "30s"},
		Provider: provider,
		Lo:       testLogger(),
		TokenRefreshCallback: func(inboxID int, updatedConfig Config) error {
			persisted = updatedConfig
			return nil
		},
	})
	require.NoError(t, err)

	tw.fetchMentions(context.Background())

	require.Len(t, store.incoming, 1)
	msg := store.incoming[0]
	require.Equal(t, null.StringFrom("tweet-1"), msg.SourceID)
	require.Equal(t, null.StringFrom("x:456"), msg.Contact.ExternalUserID)

	var meta map[string]any
	require.NoError(t, json.Unmarshal(msg.Meta, &meta))
	require.Equal(t, TwitterSourceMention, meta["twitter_source"])
	require.Equal(t, "thread-a", meta["conversation_id"])
	require.Equal(t, "root", meta["in_reply_to_status_id"])
	require.Equal(t, "https://x.com/support/status/tweet-1", meta["tweet_url"])
	require.Equal(t, "newest", persisted.MentionsCursor)
}

func TestSendRoutesDMsAndMentions(t *testing.T) {
	provider := &MockProvider{}
	tw, err := New(&testMessageStore{}, testUserStore{externalID: "x:123"}, Opts{
		ID:       10,
		Name:     "X Support",
		Config:   Config{PollInterval: "30s"},
		Provider: provider,
		Lo:       testLogger(),
	})
	require.NoError(t, err)

	dmMeta, _ := json.Marshal(map[string]any{"twitter_source": TwitterSourceDM})
	require.NoError(t, tw.Send(cmodels.OutboundMessage{
		MessageReceiverID: 42,
		TextContent:       "private reply",
		Meta:              dmMeta,
	}))
	require.Len(t, provider.SentDMs, 1)
	require.Equal(t, "123", provider.SentDMs[0].SenderID)
	require.Equal(t, "private reply", provider.SentDMs[0].Text)

	mentionMeta, _ := json.Marshal(map[string]any{"twitter_source": TwitterSourceMention, "tweet_id": "tweet-1"})
	require.NoError(t, tw.Send(cmodels.OutboundMessage{
		TextContent: "public reply",
		Meta:        mentionMeta,
	}))
	require.Len(t, provider.SentReplies, 1)
	require.Equal(t, "tweet-1", provider.SentReplies[0].InReplyToStatusID)
	require.Equal(t, "public reply", provider.SentReplies[0].Text)
}

var _ inbox.MessageStore = (*testMessageStore)(nil)
var _ inbox.UserStore = (*testUserStore)(nil)

func testLogger() *logf.Logger {
	lo := logf.New(logf.Opts{})
	return &lo
}
