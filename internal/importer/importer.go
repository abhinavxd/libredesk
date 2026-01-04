package importer

import (
	"fmt"
	"sync"
	"time"
)

type JobStatus struct {
	Running   bool      `json:"running"`
	Logs      []string  `json:"logs"`
	Total     int       `json:"total"`
	Success   int       `json:"success"`
	Errors    int       `json:"errors"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
}

type Importer struct {
	jobs map[string]*JobStatus
	mu   sync.RWMutex
}

func NewImporter() *Importer {
	i := &Importer{
		jobs: make(map[string]*JobStatus),
	}
	go i.cleanUp()
	return i
}

func (i *Importer) Submit(namespace string, fn func() error) error {
	i.mu.Lock()

	if status, exists := i.jobs[namespace]; exists && status.Running {
		i.mu.Unlock()
		return fmt.Errorf("import already running for namespace: %s", namespace)
	}

	status := &JobStatus{
		Running:   true,
		Logs:      []string{},
		StartedAt: time.Now(),
	}
	i.jobs[namespace] = status
	i.mu.Unlock()

	go func() {
		defer func() {
			i.mu.Lock()
			status.Running = false
			status.EndedAt = time.Now()
			i.mu.Unlock()
		}()

		if err := fn(); err != nil {
			i.mu.Lock()
			status.Logs = append(status.Logs, fmt.Sprintf("Error: %v", err))
			i.mu.Unlock()
		}
	}()

	return nil
}

func (i *Importer) GetStatus(namespace string) (*JobStatus, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	status, exists := i.jobs[namespace]
	if !exists {
		return nil, fmt.Errorf("no import job found for namespace: %s", namespace)
	}

	return status, nil
}

func (i *Importer) cleanUp() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		i.mu.Lock()
		now := time.Now()
		for namespace, status := range i.jobs {
			if !status.Running && now.Sub(status.EndedAt) > 24*time.Hour {
				delete(i.jobs, namespace)
			}
		}
		i.mu.Unlock()
	}
}

// AddLog appends a log message to the job status
func (i *Importer) AddLog(namespace, message string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if status, exists := i.jobs[namespace]; exists {
		status.Logs = append(status.Logs, message)
	}
}

// UpdateCounts updates the success/error counts and total
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
