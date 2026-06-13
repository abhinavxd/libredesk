package twitter

import (
	"encoding/json"
	"time"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
)

const (
	ChannelTwitter       = "twitter"
	ProviderOfficial     = "official"
	DeliveryModePolling  = "polling"
	DeliveryModeWebhook  = "webhook"
	TwitterSourceDM      = "dm"
	TwitterSourceMention = "mention"
)

type Config struct {
	AccountUserID  string               `json:"account_user_id"`
	ScreenName     string               `json:"screen_name"`
	AuthType       string               `json:"auth_type"`
	OAuth          *imodels.OAuthConfig `json:"oauth"`
	Provider       string               `json:"provider"`
	BaseURL        string               `json:"base_url"`
	DeliveryMode   string               `json:"delivery_mode"`
	FilteredStream FilteredStreamConfig `json:"filtered_stream"`
	Webhook        WebhookConfig        `json:"webhook"`
	Activity       ActivityConfig       `json:"activity"`
	IngestDMs      bool                 `json:"ingest_dms"`
	IngestMentions bool                 `json:"ingest_mentions"`
	PollInterval   string               `json:"poll_interval"`
	ScanSince      string               `json:"scan_since"`
	MentionsCursor string               `json:"mentions_cursor"`
	DMCursor       string               `json:"dm_cursor"`
}

type FilteredStreamConfig struct {
	Rules []StreamRule `json:"rules"`
}

type StreamRule struct {
	ID      string `json:"id"`
	Value   string `json:"value"`
	Tag     string `json:"tag"`
	Enabled bool   `json:"enabled"`
}

type WebhookConfig struct {
	ID             string `json:"id"`
	URL            string `json:"url"`
	ConsumerSecret string `json:"consumer_secret"`
}

type ActivityConfig struct {
	SubscriptionID string   `json:"subscription_id"`
	EventTypes     []string `json:"event_types"`
}

type MediaRef struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type XUser struct {
	ID        string
	Name      string
	Username  string
	AvatarURL string
}

type DMEvent struct {
	ID        string
	ThreadID  string
	SenderID  string
	Text      string
	CreatedAt time.Time
	Media     []MediaRef
}

type Tweet struct {
	ID                string
	ConversationID    string
	AuthorID          string
	AuthorUsername    string
	AuthorName        string
	AuthorAvatarURL   string
	Text              string
	InReplyToStatusID string
	CreatedAt         time.Time
}

type MessageMeta struct {
	TwitterSource     string `json:"twitter_source,omitempty"`
	DMThreadID        string `json:"dm_thread_id,omitempty"`
	ConversationID    string `json:"conversation_id,omitempty"`
	InReplyToStatusID string `json:"in_reply_to_status_id,omitempty"`
	TweetID           string `json:"tweet_id,omitempty"`
	TwitterUserID     string `json:"twitter_user_id,omitempty"`
	TwitterHandle     string `json:"twitter_handle,omitempty"`
	TweetURL          string `json:"tweet_url,omitempty"`
}

func (m MessageMeta) MarshalJSONRaw() json.RawMessage {
	b, _ := json.Marshal(m)
	return b
}

func ParseMessageMeta(raw json.RawMessage) (MessageMeta, error) {
	var meta MessageMeta
	if len(raw) == 0 {
		return meta, nil
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

type TokenRefreshCallback func(inboxID int, updatedConfig Config) error
