package webhook

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/zerodha/logf"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

type receivedDelivery struct {
	deliveryID string
	attempt    string
	event      string
	signature  string
	body       string
}

type recordingReceiver struct {
	mu         sync.Mutex
	deliveries []receivedDelivery
	statuses   []int
	arrived    chan struct{}
}

func newRecordingReceiver(statuses ...int) *recordingReceiver {
	return &recordingReceiver{statuses: statuses, arrived: make(chan struct{}, 16)}
}

func (r *recordingReceiver) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	r.deliveries = append(r.deliveries, receivedDelivery{
		deliveryID: req.Header.Get("X-Libredesk-Delivery-Id"),
		attempt:    req.Header.Get("X-Libredesk-Delivery-Attempt"),
		event:      req.Header.Get("X-Libredesk-Event"),
		signature:  req.Header.Get("X-Libredesk-Signature"),
		body:       string(body),
	})
	status := http.StatusOK
	if len(r.statuses) > 0 {
		status = r.statuses[0]
		r.statuses = r.statuses[1:]
	}
	r.mu.Unlock()
	w.WriteHeader(status)
	r.arrived <- struct{}{}
}

func (r *recordingReceiver) received() []receivedDelivery {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]receivedDelivery(nil), r.deliveries...)
}

func (r *recordingReceiver) waitFor(t *testing.T, count int) []receivedDelivery {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for len(r.received()) < count {
		select {
		case <-r.arrived:
		case <-deadline:
			t.Fatalf("received %d deliveries, want %d", len(r.received()), count)
		}
	}
	return r.received()
}

func newDeliveryManager(maxAttempts int) *Manager {
	lo := logf.New(logf.Opts{})
	return &Manager{
		lo:            &lo,
		deliveryQueue: make(chan DeliveryTask, 4),
		httpClient:    &http.Client{Timeout: 2 * time.Second},
		maxAttempts:   max(maxAttempts, 1),
		retryBackoff:  time.Millisecond,
	}
}

func scheduledRetry(t *testing.T, m *Manager) (DeliveryTask, bool) {
	t.Helper()
	select {
	case task := <-m.deliveryQueue:
		return task, true
	case <-time.After(100 * time.Millisecond):
		return DeliveryTask{}, false
	}
}

func deliverUntilSettled(t *testing.T, m *Manager, webhook models.Webhook, first DeliveryTask) {
	t.Helper()
	m.deliverSingleWebhook(webhook, first)
	for {
		next, ok := scheduledRetry(t, m)
		if !ok {
			return
		}
		if next.WebhookID != webhook.ID {
			t.Fatalf("retry targets webhook %d, want %d", next.WebhookID, webhook.ID)
		}
		m.deliverSingleWebhook(webhook, next)
	}
}

func TestRetriedDeliveriesKeepTheirIdentityBodyAndSignature(t *testing.T) {
	receiver := newRecordingReceiver(http.StatusServiceUnavailable, http.StatusTooManyRequests, http.StatusOK)
	server := httptest.NewServer(receiver)
	defer server.Close()
	m := newDeliveryManager(4)
	webhook := models.Webhook{ID: 7, URL: server.URL, Secret: "receiver-secret", IsActive: true}

	deliverUntilSettled(t, m, webhook, DeliveryTask{Event: models.EventConversationCreated, Payload: map[string]any{"uuid": "c-1"}})

	deliveries := receiver.received()
	if len(deliveries) != 3 {
		t.Fatalf("received %d deliveries, want 3", len(deliveries))
	}
	first := deliveries[0]
	if first.deliveryID == "" || first.signature == "" {
		t.Fatalf("the first delivery lacks its id or signature: %+v", first)
	}
	if first.event != string(models.EventConversationCreated) {
		t.Fatalf("X-Libredesk-Event = %q", first.event)
	}
	for index, delivery := range deliveries {
		if delivery.deliveryID != first.deliveryID || delivery.body != first.body || delivery.signature != first.signature {
			t.Fatalf("attempt %d differs from the first one: %+v", index+1, delivery)
		}
		if want := []string{"1", "2", "3"}[index]; delivery.attempt != want {
			t.Fatalf("attempt header = %q, want %q", delivery.attempt, want)
		}
	}
}

func TestEveryDeliveryGetsItsOwnIdentity(t *testing.T) {
	receiver := newRecordingReceiver()
	server := httptest.NewServer(receiver)
	defer server.Close()
	m := newDeliveryManager(4)
	webhook := models.Webhook{ID: 7, URL: server.URL, IsActive: true}

	m.deliverSingleWebhook(webhook, DeliveryTask{Event: models.EventConversationCreated, Payload: map[string]any{"uuid": "c-1"}})
	m.deliverSingleWebhook(webhook, DeliveryTask{Event: models.EventConversationCreated, Payload: map[string]any{"uuid": "c-2"}})

	deliveries := receiver.received()
	if len(deliveries) != 2 || deliveries[0].deliveryID == deliveries[1].deliveryID {
		t.Fatalf("two events share a delivery id: %+v", deliveries)
	}
}

func TestOnlyTransientFailuresAreRetried(t *testing.T) {
	for _, tt := range []struct {
		status int
		want   int
	}{
		{http.StatusBadRequest, 1},
		{http.StatusUnauthorized, 1},
		{http.StatusNotFound, 1},
		{http.StatusRequestTimeout, 2},
		{http.StatusTooEarly, 2},
		{http.StatusTooManyRequests, 2},
		{http.StatusInternalServerError, 2},
		{http.StatusBadGateway, 2},
	} {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			receiver := newRecordingReceiver(tt.status, http.StatusOK)
			server := httptest.NewServer(receiver)
			defer server.Close()
			m := newDeliveryManager(4)

			deliverUntilSettled(t, m, models.Webhook{ID: 7, URL: server.URL, IsActive: true}, DeliveryTask{Event: models.EventConversationCreated})

			if got := len(receiver.received()); got != tt.want {
				t.Fatalf("received %d deliveries after %d, want %d", got, tt.status, tt.want)
			}
		})
	}
}

func TestUnreachableReceiversAreRetried(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL
	server.Close()
	m := newDeliveryManager(2)
	edited := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

	m.deliverSingleWebhook(models.Webhook{ID: 7, URL: url, IsActive: true, UpdatedAt: edited}, DeliveryTask{Event: models.EventConversationCreated})

	next, ok := scheduledRetry(t, m)
	if !ok || next.attempt != 2 {
		t.Fatalf("no second attempt was scheduled after a connection error: %+v", next)
	}
	if !next.webhookUpdatedAt.Equal(edited) {
		t.Fatalf("the retry remembers revision %v, want %v", next.webhookUpdatedAt, edited)
	}
}

func TestARetryIsStaleOnlyWhenTheWebhookChanged(t *testing.T) {
	first := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	later := first.Add(time.Minute)
	for _, tt := range []struct {
		name    string
		task    DeliveryTask
		webhook models.Webhook
		stale   bool
	}{
		{"first attempt", DeliveryTask{}, models.Webhook{UpdatedAt: later}, false},
		{"unchanged webhook", DeliveryTask{webhookUpdatedAt: first}, models.Webhook{UpdatedAt: first}, false},
		{"edited webhook", DeliveryTask{webhookUpdatedAt: first}, models.Webhook{UpdatedAt: later}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.task.staleFor(tt.webhook); got != tt.stale {
				t.Fatalf("staleFor = %v, want %v", got, tt.stale)
			}
		})
	}
}

func TestDeliveryStopsAfterMaxAttempts(t *testing.T) {
	receiver := newRecordingReceiver(http.StatusInternalServerError, http.StatusInternalServerError, http.StatusInternalServerError)
	server := httptest.NewServer(receiver)
	defer server.Close()
	m := newDeliveryManager(2)

	deliverUntilSettled(t, m, models.Webhook{ID: 7, URL: server.URL, IsActive: true}, DeliveryTask{Event: models.EventConversationCreated})

	if got := len(receiver.received()); got != 2 {
		t.Fatalf("received %d deliveries with max_attempts 2, want 2", got)
	}
}

func TestOneAttemptKeepsTheFormerBehaviour(t *testing.T) {
	receiver := newRecordingReceiver(http.StatusServiceUnavailable)
	server := httptest.NewServer(receiver)
	defer server.Close()
	m := newDeliveryManager(0)

	deliverUntilSettled(t, m, models.Webhook{ID: 7, URL: server.URL, IsActive: true}, DeliveryTask{Event: models.EventConversationCreated})

	if got := len(receiver.received()); got != 1 {
		t.Fatalf("received %d deliveries without max_attempts, want 1", got)
	}
}

func TestTestDeliveriesAreNotRetried(t *testing.T) {
	receiver := newRecordingReceiver(http.StatusServiceUnavailable)
	server := httptest.NewServer(receiver)
	defer server.Close()
	m := newDeliveryManager(4)

	deliverUntilSettled(t, m, models.Webhook{ID: 7, URL: server.URL, IsActive: true}, DeliveryTask{Event: models.EventWebhookTest})

	if got := len(receiver.received()); got != 1 {
		t.Fatalf("received %d test deliveries, want 1", got)
	}
}

func TestNoRetryIsQueuedAfterClose(t *testing.T) {
	m := newDeliveryManager(4)
	m.Close()

	m.enqueueRetry(DeliveryTask{Event: models.EventConversationCreated, WebhookID: 7, attempt: 2})

	if _, ok := <-m.deliveryQueue; ok {
		t.Fatal("a retry was queued after Close")
	}
}

func TestRetryDelayDoublesUpToTheCap(t *testing.T) {
	m := &Manager{retryBackoff: time.Second}
	for attempt, want := range map[int]time.Duration{1: time.Second, 2: 2 * time.Second, 3: 4 * time.Second, 20: maxRetryBackoff} {
		if got := m.retryDelay(attempt); got != want {
			t.Fatalf("retryDelay(%d) = %v, want %v", attempt, got, want)
		}
	}
}

func newStoredWebhook(t *testing.T, url string, retryBackoff time.Duration) (*Manager, models.Webhook) {
	t.Helper()
	db := testutil.NewDB(t, "webhook_delivery")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{
		DB:            db,
		Lo:            &lo,
		I18n:          testutil.NewI18n(t),
		Workers:       2,
		QueueSize:     16,
		Timeout:       2 * time.Second,
		EncryptionKey: testEncryptionKey,
		MaxAttempts:   3,
		RetryBackoff:  retryBackoff,
	})
	if err != nil {
		t.Fatal(err)
	}
	webhook, err := m.Create(models.Webhook{
		Name:     "receiver",
		URL:      url,
		Events:   []string{string(models.EventConversationCreated)},
		Secret:   "receiver-secret",
		IsActive: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.Run(ctx)
	t.Cleanup(func() {
		cancel()
		m.Close()
	})
	return m, webhook
}

func TestRetriesGoThroughTheWorkerAndReadTheWebhookAgain(t *testing.T) {
	receiver := newRecordingReceiver(http.StatusServiceUnavailable, http.StatusOK)
	server := httptest.NewServer(receiver)
	defer server.Close()
	m, _ := newStoredWebhook(t, server.URL, 10*time.Millisecond)

	m.TriggerEvent(models.EventConversationCreated, map[string]any{"uuid": "c-1"})

	deliveries := receiver.waitFor(t, 2)
	if deliveries[0].deliveryID != deliveries[1].deliveryID || deliveries[1].attempt != "2" {
		t.Fatalf("the retry is not the same delivery: %+v", deliveries)
	}
}

func TestARetryIsDroppedWhenTheWebhookWasEdited(t *testing.T) {
	oldReceiver := newRecordingReceiver(http.StatusServiceUnavailable, http.StatusOK)
	oldServer := httptest.NewServer(oldReceiver)
	defer oldServer.Close()
	newReceiver := newRecordingReceiver()
	newServer := httptest.NewServer(newReceiver)
	defer newServer.Close()
	m, webhook := newStoredWebhook(t, oldServer.URL, 300*time.Millisecond)

	m.TriggerEvent(models.EventConversationCreated, map[string]any{"uuid": "c-1"})
	oldReceiver.waitFor(t, 1)
	webhook.URL = newServer.URL
	if _, err := m.Update(webhook.ID, webhook); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)

	if got := len(oldReceiver.received()) + len(newReceiver.received()); got != 1 {
		t.Fatalf("received %d deliveries after the webhook was edited, want only the first one", got)
	}
}
