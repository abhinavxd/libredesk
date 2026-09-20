package resourceusage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "uploads")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}

	got := collectDisk(path)
	if got.Error != "" {
		t.Fatalf("collectDisk returned an error: %s", got.Error)
	}
	if got.LimitBytes == 0 || got.UsagePercent == nil {
		t.Fatalf("expected filesystem metrics, got %+v", got)
	}
	if got.Path != path {
		t.Fatalf("path = %q, want %q", got.Path, path)
	}
}

func TestCollectDiskMissingPath(t *testing.T) {
	got := collectDisk(filepath.Join(t.TempDir(), "missing"))
	if got.Error == "" {
		t.Fatal("expected a missing-path error")
	}
	if got.UsagePercent != nil {
		t.Fatal("expected no usage percentage for a missing path")
	}
}
