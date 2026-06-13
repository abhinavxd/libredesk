package main

import (
	"context"
	"errors"
	"sync"

	"github.com/abhinavxd/libredesk/internal/whatsapp"
)

const (
	whatsAppQueueSize   = 1000
	whatsAppWorkerCount = 4
)

var ErrWhatsAppQueueFull = errors.New("whatsapp ingest queue is full")

// whatsAppJob is one Meta webhook delivery scheduled for processing.
type whatsAppJob struct {
	inboxID int
	payload *whatsapp.WebhookPayload
}

// WhatsAppIngester drains webhook payloads through a bounded worker pool.
type WhatsAppIngester struct {
	queue       chan whatsAppJob
	workers     int
	app         *App
	wg          sync.WaitGroup
	closed      bool
	closedMu    sync.RWMutex
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

// newWhatsAppIngester returns an ingester; zero values fall back to the package defaults.
func newWhatsAppIngester(app *App, queueSize, workers int) *WhatsAppIngester {
	if queueSize <= 0 {
		queueSize = whatsAppQueueSize
	}
	if workers <= 0 {
		workers = whatsAppWorkerCount
	}
	return &WhatsAppIngester{
		queue:       make(chan whatsAppJob, queueSize),
		workers:     workers,
		app:         app,
		sourceLocks: &keyedLock{entries: make(map[string]*keyedLockEntry)},
	}
}

// lockSender blocks until the per-sender-phone lock is held, returning the release func.
func (i *WhatsAppIngester) lockSender(from string) func() {
	return i.sourceLocks.lock(from)
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

// Run starts the worker goroutines and blocks until ctx is cancelled or Close is called.
func (i *WhatsAppIngester) Run(ctx context.Context) {
	for w := 0; w < i.workers; w++ {
		i.wg.Add(1)
		go func() {
			defer i.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-i.queue:
					if !ok {
						return
					}
					processWhatsAppPayload(i.app, job.inboxID, job.payload)
				}
			}
		}()
	}
	i.wg.Wait()
}

// Close signals the workers to drain and stops accepting new jobs.
func (i *WhatsAppIngester) Close() {
	i.closedMu.Lock()
	if i.closed {
		i.closedMu.Unlock()
		return
	}
	i.closed = true
	close(i.queue)
	i.closedMu.Unlock()
	i.wg.Wait()
}

// Enqueue adds a job to the pool, returning ErrWhatsAppQueueFull when saturated.
func (i *WhatsAppIngester) Enqueue(inboxID int, payload *whatsapp.WebhookPayload) error {
	i.closedMu.RLock()
	defer i.closedMu.RUnlock()
	if i.closed {
		return ErrWhatsAppQueueFull
	}
	select {
	case i.queue <- whatsAppJob{inboxID: inboxID, payload: payload}:
		return nil
	default:
		return ErrWhatsAppQueueFull
	}
}
