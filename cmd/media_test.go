package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPrepareImageUploadAllowsOversizedImageWithoutThumbnail(t *testing.T) {
	imageData := []byte{'G', 'I', 'F', '8', '9', 'a', 0x10, 0x27, 0x88, 0x13, 0x00, 0x00, 0x00}

	prepared, err := prepareImageUpload(bytes.NewReader(imageData))
	if err != nil {
		t.Fatal(err)
	}
	if prepared.thumbnail != nil {
		t.Fatal("expected thumbnail to be skipped")
	}
	if prepared.thumbnailErr == nil {
		t.Fatal("expected thumbnail error to be retained")
	}

	var meta struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	if err := json.Unmarshal(prepared.meta, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Width != 10_000 || meta.Height != 5_000 {
		t.Fatalf("dimensions = %dx%d, want 10000x5000", meta.Width, meta.Height)
	}
}
