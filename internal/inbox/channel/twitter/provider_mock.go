package twitter

import (
	"context"
	"fmt"
)

type MockProvider struct {
	DMEvents       []DMEvent
	DMCursor       string
	Mentions       []Tweet
	MentionsCursor string
	Users          map[string]XUser
	SentDMs        []DMEvent
	SentReplies    []Tweet
	Err            error
}

func (m *MockProvider) FetchDMEvents(ctx context.Context, sinceCursor string) ([]DMEvent, string, error) {
	if m.Err != nil {
		return nil, "", m.Err
	}
	return m.DMEvents, m.DMCursor, nil
}

func (m *MockProvider) SendDM(ctx context.Context, recipientUserID, text string, media []MediaRef) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	id := fmt.Sprintf("mock-dm-%d", len(m.SentDMs)+1)
	m.SentDMs = append(m.SentDMs, DMEvent{ID: id, SenderID: recipientUserID, Text: text, Media: media})
	return id, nil
}

func (m *MockProvider) FetchMentions(ctx context.Context, sinceID string) ([]Tweet, string, error) {
	if m.Err != nil {
		return nil, "", m.Err
	}
	return m.Mentions, m.MentionsCursor, nil
}

func (m *MockProvider) Reply(ctx context.Context, inReplyToStatusID, text string, media []MediaRef) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	id := fmt.Sprintf("mock-tweet-%d", len(m.SentReplies)+1)
	m.SentReplies = append(m.SentReplies, Tweet{ID: id, InReplyToStatusID: inReplyToStatusID, Text: text})
	return id, nil
}

func (m *MockProvider) LookupUser(ctx context.Context, userID string) (XUser, error) {
	if m.Err != nil {
		return XUser{}, m.Err
	}
	if m.Users == nil {
		return XUser{ID: userID}, nil
	}
	if u, ok := m.Users[userID]; ok {
		return u, nil
	}
	return XUser{ID: userID}, nil
}

var _ Provider = (*MockProvider)(nil)
