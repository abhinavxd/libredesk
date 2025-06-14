// Package webhook handles the management of webhooks and webhook deliveries.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/version"
	"github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles webhook-related operations.
type Manager struct {
	q             queries
	lo            *logf.Logger
	i18n          *i18n.I18n
	db            *sqlx.DB
	deliveryQueue chan DeliveryTask
	httpClient    *http.Client
	workers       int
	closed        bool
	closedMu      sync.RWMutex
	wg            sync.WaitGroup
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB        *sqlx.DB
	Lo        *logf.Logger
	I18n      *i18n.I18n
	Workers   int
	QueueSize int
	Timeout   time.Duration
}

// DeliveryTask represents a webhook delivery task
type DeliveryTask struct {
	Event   models.WebhookEvent
	Payload any
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllWebhooks     *sqlx.Stmt `query:"get-all-webhooks"`
	GetWebhook         *sqlx.Stmt `query:"get-webhook"`
	GetActiveWebhooks  *sqlx.Stmt `query:"get-active-webhooks"`
	GetWebhooksByEvent *sqlx.Stmt `query:"get-webhooks-by-event"`
	InsertWebhook      *sqlx.Stmt `query:"insert-webhook"`
	UpdateWebhook      *sqlx.Stmt `query:"update-webhook"`
	DeleteWebhook      *sqlx.Stmt `query:"delete-webhook"`
	ToggleWebhook      *sqlx.Stmt `query:"toggle-webhook"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:             q,
		lo:            opts.Lo,
		i18n:          opts.I18n,
		db:            opts.DB,
		deliveryQueue: make(chan DeliveryTask, opts.QueueSize),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   3 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout:   3 * time.Second,
				ResponseHeaderTimeout: 3 * time.Second,
			},
		},
		workers: opts.Workers,
	}, nil
}

// GetAll retrieves all webhooks.
func (m *Manager) GetAll() ([]models.Webhook, error) {
	var webhooks = make([]models.Webhook, 0)
	if err := m.q.GetAllWebhooks.Select(&webhooks); err != nil {
		m.lo.Error("error fetching webhooks", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "webhooks"), nil)
	}
	return webhooks, nil
}

// Get retrieves a webhook by ID.
func (m *Manager) Get(id int) (models.Webhook, error) {
	var webhook models.Webhook
	if err := m.q.GetWebhook.Get(&webhook, id); err != nil {
		if err == sql.ErrNoRows {
			return webhook, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "webhook"), nil)
		}
		m.lo.Error("error fetching webhook", "error", err)
		return webhook, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "webhook"), nil)
	}
	return webhook, nil
}

// Create creates a new webhook.
func (m *Manager) Create(webhook models.Webhook) (int, error) {
	var id int
	if err := m.q.InsertWebhook.Get(&id, webhook.Name, webhook.URL, pq.Array(webhook.Events), webhook.Secret, webhook.IsActive); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return 0, envelope.NewError(envelope.ConflictError, m.i18n.Ts("globals.messages.errorAlreadyExists", "name", "webhook"), nil)
		}
		m.lo.Error("error inserting webhook", "error", err)
		return 0, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorCreating", "name", "webhook"), nil)
	}
	return id, nil
}

// Update updates a webhook by ID.
func (m *Manager) Update(id int, webhook models.Webhook) error {
	if _, err := m.q.UpdateWebhook.Exec(id, webhook.Name, webhook.URL, pq.Array(webhook.Events), webhook.Secret, webhook.IsActive); err != nil {
		m.lo.Error("error updating webhook", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "webhook"), nil)
	}
	return nil
}

// Delete deletes a webhook by ID.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteWebhook.Exec(id); err != nil {
		m.lo.Error("error deleting webhook", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "webhook"), nil)
	}
	return nil
}

// Toggle toggles the active status of a webhook by ID.
func (m *Manager) Toggle(id int) error {
	if _, err := m.q.ToggleWebhook.Exec(id); err != nil {
		m.lo.Error("error toggling webhook", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "webhook"), nil)
	}
	return nil
}

// SendTestWebhook sends a test webhook to the specified webhook ID.
func (m *Manager) SendTestWebhook(id int) error {
	webhook, err := m.Get(id)
	if err != nil {
		return envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "webhook"), nil)
	}

	m.deliverSingleWebhook(webhook, DeliveryTask{
		Event: models.EventWebhookTest,
		Payload: map[string]any{
			"id":   webhook.ID,
			"name": webhook.Name,
		},
	})

	return nil
}

// TriggerEvent triggers webhooks for a specific event with the provided data.
func (m *Manager) TriggerEvent(event models.WebhookEvent, data any) {
	m.closedMu.RLock()
	defer m.closedMu.RUnlock()
	if m.closed {
		return
	}

	select {
	case m.deliveryQueue <- DeliveryTask{
		Event:   event,
		Payload: data,
	}:
	default:
		m.lo.Warn("webhook delivery queue is full, dropping webhook delivery", "event", event, "queue_size", len(m.deliveryQueue))
	}
}

// Run starts the webhook delivery worker pool.
func (m *Manager) Run(ctx context.Context) {
	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.worker(ctx)
		}()
	}
}

// Close signals the manager to stop processing and waits for all workers to finish.
func (m *Manager) Close() {
	m.closedMu.Lock()
	defer m.closedMu.Unlock()
	if m.closed {
		return
	}
	m.closed = true
	close(m.deliveryQueue)
	m.wg.Wait()
}

// worker processes webhook delivery tasks from the queue.
func (m *Manager) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-m.deliveryQueue:
			if !ok {
				return
			}
			m.deliverWebhook(task)
		}
	}
}

// deliverWebhook delivers webhooks for an event by making HTTP requests.
func (m *Manager) deliverWebhook(task DeliveryTask) {
	webhooks, err := m.getWebhooksByEvent(string(task.Event))
	if err != nil {
		m.lo.Error("error fetching webhooks for event", "event", task.Event, "error", err)
		return
	}

	for _, webhook := range webhooks {
		m.deliverSingleWebhook(webhook, task)
	}
}

// deliverSingleWebhook delivers a webhook to a single endpoint.
func (m *Manager) deliverSingleWebhook(webhook models.Webhook, task DeliveryTask) {
	basePayload := map[string]any{
		"event":     task.Event,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"payload":   task.Payload,
	}

	payloadBytes, err := json.Marshal(basePayload)
	if err != nil {
		m.lo.Error("error marshaling webhook payload", "webhook_id", webhook.ID, "event", task.Event, "error", err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		m.lo.Error("error creating webhook request", "webhook_id", webhook.ID, "url", webhook.URL, "event", task.Event, "error", err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Libredesk-Webhook/"+version.Version)

	// Add signature if secret is provided
	if webhook.Secret != "" {
		signature := m.generateSignature(payloadBytes, webhook.Secret)
		req.Header.Set("X-Libredesk-Signature", signature)
	}

	m.lo.Debug("delivering webhook",
		"webhook_id", webhook.ID,
		"url", webhook.URL,
		"event", task.Event,
		"payload", string(payloadBytes),
		"headers", req.Header,
	)

	// Make the request
	resp, err := m.httpClient.Do(req)
	if err != nil {
		m.lo.Error("webhook delivery failed - HTTP request error",
			"webhook_id", webhook.ID,
			"url", webhook.URL,
			"event", task.Event,
			"error", err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.lo.Error("error reading webhook response", "webhook_id", webhook.ID, "error", err)
		responseBody = []byte(fmt.Sprintf("Error reading response: %v", err))
	}

	// Check if delivery was successful (2xx status codes)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	if success {
		m.lo.Info("webhook delivered successfully",
			"webhook_id", webhook.ID,
			"event", task.Event,
			"url", webhook.URL,
			"status_code", resp.StatusCode)
	} else {
		m.lo.Error("webhook delivery failed",
			"webhook_id", webhook.ID,
			"event", task.Event,
			"url", webhook.URL,
			"status_code", resp.StatusCode,
			"response", string(responseBody))
	}
}

// generateSignature generates HMAC-SHA256 signature for webhook payload.
func (m *Manager) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// getWebhooksByEvent retrieves active webhooks that are subscribed to a specific event.
func (m *Manager) getWebhooksByEvent(event string) ([]models.Webhook, error) {
	var webhooks = make([]models.Webhook, 0)
	if err := m.q.GetWebhooksByEvent.Select(&webhooks, event); err != nil {
		return nil, err
	}
	return webhooks, nil
}
