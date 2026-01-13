package importer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

// Job represents the status of an import job.
type Job struct {
	Running   bool      `json:"running"`
	Logs      []string  `json:"logs"`
	Total     int       `json:"total"`
	Success   int       `json:"success"`
	Errors    int       `json:"errors"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

// Importer manages background import jobs.
type Importer struct {
	lo     *logf.Logger
	i18n   *i18n.I18n
	jobs   map[string]*Job
	mu     sync.RWMutex
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// Opts contains options for initializing the Importer.
type Opts struct {
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// New creates and returns a new instance of the Importer.
func New(opts Opts) *Importer {
	ctx, cancel := context.WithCancel(context.Background())
	i := &Importer{
		lo:     opts.Lo,
		i18n:   opts.I18n,
		jobs:   make(map[string]*Job),
		ctx:    ctx,
		cancel: cancel,
	}
	i.wg.Add(1)
	go i.cleanUp()
	return i
}

// Submit submits a new import job for execution.
func (i *Importer) Submit(namespace string, fn func() error) error {
	i.mu.Lock()

	if status, exists := i.jobs[namespace]; exists && status.Running {
		i.mu.Unlock()
		return envelope.NewError(envelope.ConflictError,
			i.i18n.T("importer.importAlreadyInProgress"), nil)
	}

	status := &Job{
		Running:   true,
		Logs:      []string{},
		StartedAt: time.Now(),
	}
	i.jobs[namespace] = status
	i.mu.Unlock()

	i.lo.Info("starting import job", "namespace", namespace)

	go func() {
		defer func() {
			// Recover from panics
			if r := recover(); r != nil {
				i.mu.Lock()
				status.Logs = append(status.Logs, fmt.Sprintf("Panic: %v", r))
				i.mu.Unlock()
				i.lo.Error("import job panicked", "namespace", namespace, "panic", r)
			}

			i.mu.Lock()
			status.Running = false
			status.EndedAt = time.Now()
			i.mu.Unlock()

			i.lo.Info("import job completed", "namespace", namespace,
				"total", status.Total, "success", status.Success, "errors", status.Errors)
		}()

		if err := fn(); err != nil {
			i.mu.Lock()
			status.Logs = append(status.Logs, fmt.Sprintf("Error: %v", err))
			i.mu.Unlock()
			i.lo.Error("import job failed", "namespace", namespace, "error", err)
		}
	}()

	return nil
}

// GetStatus returns the status of an import job.
func (i *Importer) GetStatus(namespace string) (*Job, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	status, exists := i.jobs[namespace]
	if !exists {
		return nil, envelope.NewError(envelope.NotFoundError,
			i.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.import}"), nil)
	}

	return status, nil
}

// AddLog appends a log message to the job status.
func (i *Importer) AddLog(namespace, message string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if status, exists := i.jobs[namespace]; exists {
		status.Logs = append(status.Logs, message)
	}
}

// UpdateCounts updates the success/error counts and total.
func (i *Importer) UpdateCounts(namespace string, total, success, errors int) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if status, exists := i.jobs[namespace]; exists {
		if total > 0 {
			status.Total = total
		}
		if success > 0 {
			status.Success += success
		}
		if errors > 0 {
			status.Errors += errors
		}
	}
}

// Close gracefully shuts down the importer.
func (i *Importer) Close() {
	i.cancel()
	i.wg.Wait()
}

// cleanUp periodically removes old completed jobs.
func (i *Importer) cleanUp() {
	defer i.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-i.ctx.Done():
			return
		case <-ticker.C:
			i.mu.Lock()
			now := time.Now()
			for namespace, status := range i.jobs {
				if !status.Running && now.Sub(status.EndedAt) > 1*time.Hour {
					delete(i.jobs, namespace)
					i.lo.Debug("cleaned up old import job", "namespace", namespace)
				}
			}
			i.mu.Unlock()
		}
	}
}
