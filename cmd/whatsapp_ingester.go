package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/streamqueue"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
)

const (
	whatsAppStream      = "libredesk:whatsapp:inbound"
	whatsAppStreamGroup = "libredesk"
	whatsAppConsumer    = "ingester"

	// Must exceed the worst-case media-download budget so the reclaimer never re-runs a still-in-flight delivery.
	whatsAppReclaimMinIdle = 5 * time.Minute
)

// whatsAppJob is the durable envelope persisted to the stream; Body is the raw Meta POST body, parsed in the worker.
type whatsAppJob struct {
	InboxID int             `json:"inbox_id"`
	Body    json.RawMessage `json:"body"`
}

// WhatsAppIngester is the durable inbound pipeline: a Redis-stream work queue plus per-sender serialization.
type WhatsAppIngester struct {
	queue       *streamqueue.Queue
	sourceLocks *keyedLock
}

// keyedLock serializes work per string key; entries are refcounted and dropped once the last holder releases.
type keyedLock struct {
	mu      sync.Mutex
	entries map[string]*keyedLockEntry
}

type keyedLockEntry struct {
	mu   sync.Mutex
	refs int
}

func newWhatsAppIngester(app *App) (*WhatsAppIngester, error) {
	ing := &WhatsAppIngester{
		sourceLocks: &keyedLock{entries: make(map[string]*keyedLockEntry)},
	}
	q, err := streamqueue.New(streamqueue.Opts{
		Redis:        app.redis,
		Logger:       app.lo,
		Stream:       whatsAppStream,
		Group:        whatsAppStreamGroup,
		Consumer:     whatsAppConsumer,
		Handler:      ing.handle(app),
		ClaimMinIdle: whatsAppReclaimMinIdle,
	})
	if err != nil {
		return nil, err
	}
	ing.queue = q
	return ing, nil
}

// Run consumes the stream until Close is called.
func (i *WhatsAppIngester) Run() { i.queue.Run() }

// Close stops the queue and waits for in-flight work; un-acked deliveries stay durable for the next start.
func (i *WhatsAppIngester) Close() { i.queue.Close() }

// Enqueue durably stores a raw webhook body for the inbox.
func (i *WhatsAppIngester) Enqueue(inboxID int, body []byte) error {
	data, err := json.Marshal(whatsAppJob{InboxID: inboxID, Body: json.RawMessage(body)})
	if err != nil {
		return err
	}
	return i.queue.Enqueue(context.Background(), data)
}

// lockSender blocks until the per-sender-phone lock is held, returning the release func.
func (i *WhatsAppIngester) lockSender(from string) func() {
	return i.sourceLocks.lock(from)
}

// handle returns nil for an unparseable job so it is dropped rather than retried forever; a processing error keeps the entry pending for retry.
func (i *WhatsAppIngester) handle(app *App) streamqueue.Handler {
	return func(ctx context.Context, payload []byte) error {
		var job whatsAppJob
		if err := json.Unmarshal(payload, &job); err != nil {
			app.lo.Error("error decoding whatsapp stream job, dropping", "error", err)
			return nil
		}
		parsed, err := whatsapp.ParsePayload(job.Body)
		if err != nil {
			app.lo.Error("error parsing whatsapp webhook payload from stream, dropping", "inbox_id", job.InboxID, "error", err)
			return nil
		}
		return processWhatsAppPayload(ctx, app, job.InboxID, parsed)
	}
}

func (k *keyedLock) lock(key string) func() {
	k.mu.Lock()
	e := k.entries[key]
	if e == nil {
		e = &keyedLockEntry{}
		k.entries[key] = e
	}
	e.refs++
	k.mu.Unlock()

	e.mu.Lock()

	return func() {
		e.mu.Unlock()
		k.mu.Lock()
		e.refs--
		if e.refs == 0 {
			delete(k.entries, key)
		}
		k.mu.Unlock()
	}
}
