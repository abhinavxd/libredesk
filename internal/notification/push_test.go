package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

type fakePushSettings struct {
	values  map[string]string
	updates int
}

func (s *fakePushSettings) Get(key string) (types.JSONText, error) {
	b, _ := json.Marshal(s.values[key])
	return b, nil
}

func (s *fakePushSettings) Update(value any) error {
	s.updates++
	for key, setting := range value.(map[string]string) {
		s.values[key] = setting
	}
	return nil
}

type fakePushStore struct {
	subscriptions []PushSubscription
	deletedID     int
}

func (s *fakePushStore) List(int) ([]PushSubscription, error) {
	return s.subscriptions, nil
}

func (s *fakePushStore) Upsert(int, PushSubscriptionInput) error { return nil }

func (s *fakePushStore) Delete(int, string) error { return nil }

func (s *fakePushStore) DeleteByID(id int) error {
	s.deletedID = id
	return nil
}

func TestPushManagerDeliverSendsEverySubscription(t *testing.T) {
	store := &fakePushStore{subscriptions: []PushSubscription{
		{ID: 11, Endpoint: "https://push.example/one", P256DH: "key-one", Auth: "auth-one"},
		{ID: 12, Endpoint: "https://push.example/two", P256DH: "key-two", Auth: "auth-two"},
	}}
	var endpoints []string
	m := &PushManager{
		store:      store,
		publicKey:  "public",
		privateKey: "private",
		subject:    "https://desk.example",
		sender: func(_ context.Context, payload []byte, subscription PushSubscription, _, _, _ string) (*http.Response, error) {
			if !strings.Contains(string(payload), `"url":"/inboxes/assigned/conversation/conv-1?scrollTo=msg-1"`) {
				t.Fatalf("unexpected payload: %s", payload)
			}
			endpoints = append(endpoints, subscription.Endpoint)
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(""))}, nil
		},
	}

	err := m.deliver(t.Context(), pushDelivery{UserID: 7, Payload: PushPayload{
		Title: "Assigned conversation",
		Body:  "A conversation was assigned to you",
		Tag:   "assignment_conv-1_msg-1",
		URL:   "/inboxes/assigned/conversation/conv-1?scrollTo=msg-1",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("sent to %d endpoints, want 2", len(endpoints))
	}
}

func TestPushManagerDeliverDeletesExpiredSubscription(t *testing.T) {
	store := &fakePushStore{subscriptions: []PushSubscription{{ID: 23, Endpoint: "https://push.example/expired"}}}
	m := &PushManager{
		store: store,
		sender: func(context.Context, []byte, PushSubscription, string, string, string) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusGone, Body: io.NopCloser(strings.NewReader(""))}, nil
		},
	}

	if err := m.deliver(t.Context(), pushDelivery{UserID: 7, Payload: PushPayload{Title: "Mention"}}); err != nil {
		t.Fatal(err)
	}
	if store.deletedID != 23 {
		t.Fatalf("deleted subscription %d, want 23", store.deletedID)
	}
}

func TestValidPushSubscription(t *testing.T) {
	tests := []struct {
		name  string
		input PushSubscriptionInput
		want  bool
	}{
		{"valid", PushSubscriptionInput{Endpoint: "https://push.example/id", P256DH: "key", Auth: "secret"}, true},
		{"empty endpoint", PushSubscriptionInput{P256DH: "key", Auth: "secret"}, false},
		{"non HTTPS endpoint", PushSubscriptionInput{Endpoint: "http://push.example/id", P256DH: "key", Auth: "secret"}, false},
		{"missing public key", PushSubscriptionInput{Endpoint: "https://push.example/id", Auth: "secret"}, false},
		{"missing auth", PushSubscriptionInput{Endpoint: "https://push.example/id", P256DH: "key"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validPushSubscription(tt.input); got != tt.want {
				t.Fatalf("validPushSubscription(%#v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPushHTTPClientBlocksPrivateEndpoints(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("private endpoint received push request")
	}))
	defer server.Close()
	lo := logf.New(logf.Opts{})
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := newPushHTTPClient(&lo).Do(request); err == nil {
		t.Fatal("private endpoint request succeeded")
	}
}

func TestVAPIDSubject(t *testing.T) {
	if got := vapidSubject("https://desk.example/"); got != "https://desk.example" {
		t.Fatalf("HTTPS subject = %q", got)
	}
	if got := vapidSubject("http://localhost:9000"); got != "support@libredesk.io" {
		t.Fatalf("local subject = %q", got)
	}
}

func TestLoadVAPIDKeysReusesStoredPair(t *testing.T) {
	settings := &fakePushSettings{values: map[string]string{
		pushPrivateKeySetting: "stored-private",
		pushPublicKeySetting:  "stored-public",
	}}

	privateKey, publicKey, err := loadVAPIDKeys(settings)
	if err != nil {
		t.Fatal(err)
	}
	if privateKey != "stored-private" || publicKey != "stored-public" {
		t.Fatalf("got %q, %q", privateKey, publicKey)
	}
	if settings.updates != 0 {
		t.Fatalf("updated stored VAPID keys %d times", settings.updates)
	}
}

func TestLoadVAPIDKeysCreatesMissingPair(t *testing.T) {
	settings := &fakePushSettings{values: map[string]string{}}

	privateKey, publicKey, err := loadVAPIDKeys(settings)
	if err != nil {
		t.Fatal(err)
	}
	if privateKey == "" || publicKey == "" {
		t.Fatal("generated VAPID keys are empty")
	}
	if settings.updates != 1 {
		t.Fatalf("updated VAPID keys %d times, want 1", settings.updates)
	}
}
