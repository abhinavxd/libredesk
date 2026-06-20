package twitter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

var ErrUnsupportedProvider = errors.New("unsupported twitter provider")

type Provider interface {
	FetchDMEvents(ctx context.Context, sinceCursor string) ([]DMEvent, string, error)
	SendDM(ctx context.Context, recipientUserID, text string, media []MediaRef) (string, error)
	FetchMentions(ctx context.Context, sinceID string) ([]Tweet, string, error)
	Reply(ctx context.Context, inReplyToStatusID, text string, media []MediaRef) (string, error)
	LookupUser(ctx context.Context, userID string) (XUser, error)
}

func NewProvider(cfg Config) (Provider, error) {
	provider := cfg.Provider
	if provider == "" {
		provider = ProviderOfficial
	}
	switch provider {
	case ProviderOfficial:
		return newOfficialProvider(cfg)
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedProvider, provider)
	}
}

type officialProvider struct {
	baseURL       string
	accountUserID string
	client        *http.Client
}

func newOfficialProvider(cfg Config) (*officialProvider, error) {
	if cfg.OAuth == nil || cfg.OAuth.AccessToken == "" {
		return nil, fmt.Errorf("twitter OAuth access token is required")
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.x.com"
	}
	token := &oauth2.Token{
		AccessToken:  cfg.OAuth.AccessToken,
		RefreshToken: cfg.OAuth.RefreshToken,
		Expiry:       cfg.OAuth.ExpiresAt,
	}
	return &officialProvider{
		baseURL:       baseURL,
		accountUserID: cfg.AccountUserID,
		client:        oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token)),
	}, nil
}

func (p *officialProvider) FetchDMEvents(ctx context.Context, sinceCursor string) ([]DMEvent, string, error) {
	values := url.Values{}
	values.Set("dm_event.fields", "id,text,created_at,sender_id,dm_conversation_id,attachments")
	if sinceCursor != "" {
		values.Set("pagination_token", sinceCursor)
	}
	var resp dmEventsResponse
	if err := p.do(ctx, http.MethodGet, "/2/dm_events?"+values.Encode(), nil, &resp); err != nil {
		return nil, "", err
	}
	events := make([]DMEvent, 0, len(resp.Data))
	for _, item := range resp.Data {
		events = append(events, DMEvent{
			ID:        item.ID,
			ThreadID:  item.DMConversationID,
			SenderID:  item.SenderID,
			Text:      item.Text,
			CreatedAt: item.CreatedAt,
		})
	}
	return events, resp.Meta.NextToken, nil
}

func (p *officialProvider) SendDM(ctx context.Context, recipientUserID, text string, media []MediaRef) (string, error) {
	payload := dmMessageCreateRequest{
		DMEvent: dmMessageCreateEvent{
			EventType: "MessageCreate",
			Text:      text,
		},
	}
	var resp idResponse
	path := fmt.Sprintf("/2/dm_conversations/with/%s/messages", url.PathEscape(recipientUserID))
	if err := p.do(ctx, http.MethodPost, path, payload, &resp); err != nil {
		return "", err
	}
	return resp.Data.ID, nil
}

func (p *officialProvider) FetchMentions(ctx context.Context, sinceID string) ([]Tweet, string, error) {
	values := url.Values{}
	values.Set("tweet.fields", "id,text,author_id,conversation_id,in_reply_to_user_id,referenced_tweets,created_at")
	values.Set("user.fields", "id,name,username,profile_image_url")
	values.Set("expansions", "author_id")
	if sinceID != "" {
		values.Set("since_id", sinceID)
	}
	var resp mentionsResponse
	path := fmt.Sprintf("/2/users/%s/mentions?%s", url.PathEscape(p.accountUserID), values.Encode())
	if err := p.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, "", err
	}

	users := make(map[string]XUser, len(resp.Includes.Users))
	for _, u := range resp.Includes.Users {
		users[u.ID] = XUser{ID: u.ID, Name: u.Name, Username: u.Username, AvatarURL: u.ProfileImageURL}
	}
	tweets := make([]Tweet, 0, len(resp.Data))
	for _, item := range resp.Data {
		inReplyToID := ""
		for _, ref := range item.Referenced {
			if ref.Type == "replied_to" {
				inReplyToID = ref.ID
				break
			}
		}
		author := users[item.AuthorID]
		tweets = append(tweets, Tweet{
			ID:                item.ID,
			ConversationID:    item.ConversationID,
			AuthorID:          item.AuthorID,
			AuthorUsername:    author.Username,
			AuthorName:        author.Name,
			AuthorAvatarURL:   author.AvatarURL,
			Text:              item.Text,
			InReplyToStatusID: inReplyToID,
			CreatedAt:         item.CreatedAt,
		})
	}
	return tweets, resp.Meta.NewestID, nil
}

func (p *officialProvider) Reply(ctx context.Context, inReplyToStatusID, text string, media []MediaRef) (string, error) {
	payload := tweetCreateRequest{
		Text: text,
		Reply: tweetReply{
			InReplyToTweetID: inReplyToStatusID,
		},
	}
	var resp idResponse
	if err := p.do(ctx, http.MethodPost, "/2/tweets", payload, &resp); err != nil {
		return "", err
	}
	return resp.Data.ID, nil
}

func (p *officialProvider) LookupUser(ctx context.Context, userID string) (XUser, error) {
	values := url.Values{}
	values.Set("user.fields", "id,name,username,profile_image_url")
	var resp lookupUserResponse
	path := fmt.Sprintf("/2/users/%s?%s", url.PathEscape(userID), values.Encode())
	if err := p.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return XUser{}, err
	}
	return XUser{ID: resp.Data.ID, Name: resp.Data.Name, Username: resp.Data.Username, AvatarURL: resp.Data.ProfileImageURL}, nil
}

func (p *officialProvider) do(ctx context.Context, method, path string, payload any, out any) error {
	body := bytes.NewReader(nil)
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("twitter API returned %s", resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

var _ Provider = (*officialProvider)(nil)
