package importer

import (
    "fmt"
    "io"
    "sync"
    "time"
)

type JobStatus struct {
    Running   bool
    Logs      []string
    Total     int
    Success   int
    Errors    int
    StartedAt time.Time
    EndedAt   time.Time
}

type CsvParserFunc func(io.Reader) ([]map[string]string, error)

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

func (i *Importer) Submit(
    namespace string,
    r io.Reader,
    parse CsvParserFunc,
    processRow func(map[string]string) error,
) error {
    i.mu.Lock()
    if _, exists := i.jobs[namespace]; exists {
        i.mu.Unlock()
        return fmt.Errorf("job %s already exists", namespace)
    }

    status := &JobStatus{
        Running:   true,
        StartedAt: time.Now(),
    }
    i.jobs[namespace] = status
		fmt.Printf("Current jobs: %+v\n", i.jobs)
    i.mu.Unlock()

    go func() {
        defer func() {
            i.mu.Lock()
            status.Running = false
            status.EndedAt = time.Now()
            i.mu.Unlock()
        }()

        records, err := parse(r)
        if err != nil {
            i.mu.Lock()
            status.Errors++
            status.Logs = append(status.Logs, "CSV parse error: "+err.Error())
            i.mu.Unlock()
            return
        }

        for _, row := range records {
            err := processRow(row)
            i.mu.Lock()
            status.Total++
            if err != nil {
                status.Errors++
                status.Logs = append(status.Logs, fmt.Sprintf("Error: %v", err))
            } else {
                status.Success++
            }
            i.mu.Unlock()
        }
    }()

    return nil
}

func (i *Importer) Get(namespace string) (*JobStatus, bool) {
    i.mu.RLock()
    defer i.mu.RUnlock()
    job, exists := i.jobs[namespace]
    return job, exists
}

func (i *Importer) cleanUp() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        cutoff := time.Now().Add(-6 * time.Hour)
        i.mu.Lock()
        for ns, job := range i.jobs {
            if !job.Running && job.EndedAt.Before(cutoff) {
                delete(i.jobs, ns)
            }
        }
        i.mu.Unlock()
    }
}