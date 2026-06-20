package twitter

import (
	"context"
	"strings"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/volatiletech/null/v9"
)

const minPollInterval = 30 * time.Second

func (t *Twitter) pollDMs(ctx context.Context) {
	ticker := time.NewTicker(t.pollInterval())
	defer ticker.Stop()

	t.fetchDMs(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.fetchDMs(ctx)
		}
	}
}

func (t *Twitter) pollMentions(ctx context.Context) {
	ticker := time.NewTicker(t.pollInterval())
	defer ticker.Stop()

	t.fetchMentions(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.fetchMentions(ctx)
		}
	}
}

func (t *Twitter) fetchDMs(ctx context.Context) {
	cfg := t.currentConfig()
	events, nextCursor, err := t.provider.FetchDMEvents(ctx, cfg.DMCursor)
	if err != nil {
		t.lo.Error("error fetching twitter DM events", "inbox_id", t.id, "error", err)
		return
	}
	for _, event := range events {
		if event.ID == "" || event.SenderID == "" || event.SenderID == cfg.AccountUserID {
			continue
		}
		if skipFirstScanItem(event.CreatedAt, cfg.ScanSince, cfg.DMCursor) {
			continue
		}
		user, err := t.provider.LookupUser(ctx, event.SenderID)
		if err != nil {
			t.lo.Error("error looking up twitter DM sender", "inbox_id", t.id, "sender_id", event.SenderID, "error", err)
			user = XUser{ID: event.SenderID}
		}
		meta := MessageMeta{
			TwitterSource: TwitterSourceDM,
			DMThreadID:    event.ThreadID,
			TwitterUserID: event.SenderID,
			TwitterHandle: user.Username,
		}
		if err := t.messageStore.EnqueueIncoming(cmodels.IncomingMessage{
			Channel:     ChannelTwitter,
			InboxID:     t.id,
			Contact:     incomingContact(user),
			SourceID:    null.StringFrom(event.ID),
			Content:     event.Text,
			ContentType: cmodels.ContentTypeText,
			Meta:        meta.MarshalJSONRaw(),
		}); err != nil {
			t.lo.Error("error enqueueing twitter DM", "inbox_id", t.id, "event_id", event.ID, "error", err)
		}
	}
	if nextCursor != "" && nextCursor != cfg.DMCursor {
		t.updateCursor(func(c *Config) {
			c.DMCursor = nextCursor
		})
	}
}

func (t *Twitter) fetchMentions(ctx context.Context) {
	cfg := t.currentConfig()
	tweets, nextCursor, err := t.provider.FetchMentions(ctx, cfg.MentionsCursor)
	if err != nil {
		t.lo.Error("error fetching twitter mentions", "inbox_id", t.id, "error", err)
		return
	}
	for _, tweet := range tweets {
		if tweet.ID == "" || tweet.AuthorID == "" || tweet.AuthorID == cfg.AccountUserID {
			continue
		}
		if skipFirstScanItem(tweet.CreatedAt, cfg.ScanSince, cfg.MentionsCursor) {
			continue
		}
		user := XUser{
			ID:        tweet.AuthorID,
			Name:      tweet.AuthorName,
			Username:  tweet.AuthorUsername,
			AvatarURL: tweet.AuthorAvatarURL,
		}
		if user.Name == "" || user.Username == "" {
			lookedUp, err := t.provider.LookupUser(ctx, tweet.AuthorID)
			if err != nil {
				t.lo.Error("error looking up twitter mention author", "inbox_id", t.id, "author_id", tweet.AuthorID, "error", err)
			} else {
				user = lookedUp
			}
		}
		meta := MessageMeta{
			TwitterSource:     TwitterSourceMention,
			ConversationID:    tweet.ConversationID,
			InReplyToStatusID: tweet.InReplyToStatusID,
			TweetID:           tweet.ID,
			TwitterUserID:     tweet.AuthorID,
			TwitterHandle:     user.Username,
			TweetURL:          tweetURL(cfg.ScreenName, tweet.ID),
		}
		if err := t.messageStore.EnqueueIncoming(cmodels.IncomingMessage{
			Channel:     ChannelTwitter,
			InboxID:     t.id,
			Contact:     incomingContact(user),
			SourceID:    null.StringFrom(tweet.ID),
			Content:     tweet.Text,
			ContentType: cmodels.ContentTypeText,
			Meta:        meta.MarshalJSONRaw(),
		}); err != nil {
			t.lo.Error("error enqueueing twitter mention", "inbox_id", t.id, "tweet_id", tweet.ID, "error", err)
		}
	}
	if nextCursor != "" && nextCursor != cfg.MentionsCursor {
		t.updateCursor(func(c *Config) {
			c.MentionsCursor = nextCursor
		})
	}
}

func (t *Twitter) pollInterval() time.Duration {
	cfg := t.currentConfig()
	interval, err := time.ParseDuration(cfg.PollInterval)
	if err != nil || interval < minPollInterval {
		return minPollInterval
	}
	return interval
}

func tweetURL(screenName, tweetID string) string {
	handle := strings.TrimPrefix(screenName, "@")
	if handle == "" || tweetID == "" {
		return ""
	}
	return "https://x.com/" + handle + "/status/" + tweetID
}

func skipFirstScanItem(createdAt time.Time, scanSince, cursor string) bool {
	if cursor != "" || scanSince == "" || createdAt.IsZero() {
		return false
	}
	window, err := time.ParseDuration(scanSince)
	if err != nil {
		return false
	}
	return createdAt.Before(time.Now().Add(-window))
}
