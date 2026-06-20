package twitter

import "time"

type dmEventsResponse struct {
	Data []dmEventResponse `json:"data"`
	Meta paginationMeta    `json:"meta"`
}

type dmEventResponse struct {
	ID               string    `json:"id"`
	Text             string    `json:"text"`
	CreatedAt        time.Time `json:"created_at"`
	SenderID         string    `json:"sender_id"`
	DMConversationID string    `json:"dm_conversation_id"`
}

type paginationMeta struct {
	NextToken string `json:"next_token"`
	NewestID  string `json:"newest_id"`
}

type dmMessageCreateRequest struct {
	DMEvent dmMessageCreateEvent `json:"dm_event"`
}

type dmMessageCreateEvent struct {
	EventType string `json:"event_type"`
	Text      string `json:"text"`
}

type idResponse struct {
	Data idResponseData `json:"data"`
}

type idResponseData struct {
	ID string `json:"id"`
}

type mentionsResponse struct {
	Data     []tweetResponse `json:"data"`
	Includes tweetIncludes   `json:"includes"`
	Meta     paginationMeta  `json:"meta"`
}

type tweetResponse struct {
	ID             string            `json:"id"`
	Text           string            `json:"text"`
	AuthorID       string            `json:"author_id"`
	ConversationID string            `json:"conversation_id"`
	CreatedAt      time.Time         `json:"created_at"`
	Referenced     []referencedTweet `json:"referenced_tweets"`
}

type referencedTweet struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type tweetIncludes struct {
	Users []userResponse `json:"users"`
}

type userResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Username        string `json:"username"`
	ProfileImageURL string `json:"profile_image_url"`
}

type tweetCreateRequest struct {
	Text  string     `json:"text"`
	Reply tweetReply `json:"reply"`
}

type tweetReply struct {
	InReplyToTweetID string `json:"in_reply_to_tweet_id"`
}

type lookupUserResponse struct {
	Data userResponse `json:"data"`
}
