// Package resourceusage reports resource usage visible to the Libredesk
// process. It intentionally reports container-local values: a process inside
// a container cannot reliably discover Docker limits for other containers or
// mounted volumes without access to the Docker socket.
package resourceusage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	// DefaultMaxIncomingMessageSize is the safety ceiling for raw incoming
	// messages when no administrator override has been saved.
	DefaultMaxIncomingMessageSize int64 = 100 << 20
	// MaxConfigurableBytes is the largest byte value the settings UI can
	// represent exactly in a JavaScript number.
	MaxConfigurableBytes int64 = (1 << 53) - 1
)

// Limits contains the application-level resource budgets editable by an
// administrator. A zero durable-storage limit means unlimited.
type Limits struct {
	MaxIncomingMessageSize int64 `json:"max_incoming_message_size"`
	MaxStorageBytes        int64 `json:"max_storage_bytes"`
}

// Normalize validates and returns a copy of the configured application
// resource budgets.
func Normalize(limits Limits) (Limits, error) {
	if limits.MaxIncomingMessageSize < 0 || limits.MaxIncomingMessageSize > MaxConfigurableBytes {
		return Limits{}, fmt.Errorf("maximum incoming email size must be between 0 and %d bytes", MaxConfigurableBytes)
	}
	if limits.MaxStorageBytes < 0 || limits.MaxStorageBytes > MaxConfigurableBytes {
		return Limits{}, fmt.Errorf("maximum durable storage must be between 0 and %d bytes", MaxConfigurableBytes)
	}
	return limits, nil
}

// Snapshot is the resource data exposed to administrators.
type Snapshot struct {
	SampledAt time.Time   `json:"sampled_at"`
	Memory    MemoryUsage `json:"memory"`
	Disk      DiskUsage   `json:"disk"`
}

// MemoryUsage contains the memory usage and cgroup limit visible to the app.
type MemoryUsage struct {
	CurrentBytes uint64   `json:"current_bytes"`
	LimitBytes   *uint64  `json:"limit_bytes"`
	PeakBytes    *uint64  `json:"peak_bytes,omitempty"`
	UsagePercent *float64 `json:"usage_percent"`
	LimitSource  string   `json:"limit_source"`
}

// DiskUsage describes the filesystem containing the configured upload path.
// It is not a claim about total VPS disk usage or the Postgres volume.
type DiskUsage struct {
	Path           string   `json:"path"`
	UsedBytes      uint64   `json:"used_bytes"`
	AvailableBytes uint64   `json:"available_bytes"`
	LimitBytes     uint64   `json:"limit_bytes"`
	UsagePercent   *float64 `json:"usage_percent"`
	LimitSource    string   `json:"limit_source"`
	Error          string   `json:"error,omitempty"`
}

// Collect returns the resource values visible from this process. Collection
// is best-effort so that a missing cgroup or an unavailable upload path does
// not make the admin UI unusable.
func Collect(diskPath string) Snapshot {
	return Snapshot{
		SampledAt: time.Now().UTC(),
		Memory:    collectMemory(),
		Disk:      collectDisk(diskPath),
	}
}

func collectMemory() MemoryUsage {
	current, currentOK := readUint64FromFiles(
		"/sys/fs/cgroup/memory.current",
		"/sys/fs/cgroup/memory/memory.usage_in_bytes",
	)
	limit, limitOK := readMemoryLimit()
	peak, peakOK := readUint64FromFiles(
		"/sys/fs/cgroup/memory.peak",
		"/sys/fs/cgroup/memory/memory.max_usage_in_bytes",
	)

	source := "cgroup"
	if !currentOK {
		current, currentOK = readProcessRSS()
		source = "process"
	}
	if !currentOK {
		current = 0
	}

	usagePercent := percent(current, limit, limitOK)
	result := MemoryUsage{
		CurrentBytes: current,
		LimitSource:  source,
		UsagePercent: usagePercent,
	}
	if limitOK {
		result.LimitBytes = &limit
	}
	if peakOK {
		result.PeakBytes = &peak
	}
	return result
}

func collectDisk(path string) DiskUsage {
	if path == "" {
		path = "."
	}

	result := DiskUsage{
		Path:        path,
		LimitSource: "filesystem",
	}

	absolutePath, err := filepath.Abs(path)
	if err == nil {
		result.Path = absolutePath
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		result.Error = err.Error()
		result.UsagePercent = nil
		return result
	}

	blockSize := uint64(stat.Bsize)
	total := stat.Blocks * blockSize
	available := stat.Bavail * blockSize
	used := uint64(0)
	if total > available {
		used = total - available
	}

	result.LimitBytes = total
	result.AvailableBytes = available
	result.UsedBytes = used
	result.UsagePercent = percent(used, total, total > 0)
	return result
}

func readMemoryLimit() (uint64, bool) {
	for _, path := range []string{
		"/sys/fs/cgroup/memory.max",
		"/sys/fs/cgroup/memory/memory.limit_in_bytes",
	} {
		value, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(value))
		if text == "" || text == "max" {
			return 0, false
		}
		parsed, err := strconv.ParseUint(text, 10, 64)
		if err != nil || parsed == 0 || parsed >= 1<<60 {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}

func readUint64FromFiles(paths ...string) (uint64, bool) {
	for _, path := range paths {
		value, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parsed, err := strconv.ParseUint(strings.TrimSpace(string(value)), 10, 64)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func readProcessRSS() (uint64, bool) {
	value, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(value), "\n") {
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, false
		}
		kilobytes, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, false
		}
		return kilobytes * 1024, true
	}
	return 0, false
}

func percent(value, limit uint64, hasLimit bool) *float64 {
	if !hasLimit || limit == 0 {
		return nil
	}
	percentage := float64(value) * 100 / float64(limit)
	return &percentage
}
