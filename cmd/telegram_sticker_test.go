package main

import (
	"bytes"
	"testing"

	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
)

func TestTelegramStickerStorage(t *testing.T) {
	app, _, _ := newTelegramIntegrationApp(t)
	data := []byte("sticker")
	if _, _, err := app.media.Upload("sticker", "application/octet-stream", bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.media.Upload("oversize", "application/octet-stream", bytes.NewReader(make([]byte, 65537))); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		size    int
		wantErr bool
	}{
		{"sticker", len(data), false}, {"missing", 1, true}, {"sticker", 65537, true}, {"oversize", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := telegramStickerData(app, &mmodels.Media{UUID: tc.name, Size: tc.size})
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v", err)
			}
			if !tc.wantErr && !bytes.Equal(got, data) {
				t.Fatalf("data=%q", got)
			}
		})
	}
	r := telegramTestRequest(app, "1", "", "{}")
	if err := serveMediaFile(r, app, "sticker", &mmodels.Media{UUID: "sticker", Filename: "animation.tgs", Size: len(data)}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r.RequestCtx.Response.Body(), data) || string(r.RequestCtx.Response.Header.Peek("Content-Type")) != "application/x-tgsticker" {
		t.Fatal("sticker not served inline")
	}
	r = telegramTestRequest(app, "1", "", "{}")
	if err := serveMediaFile(r, app, "missing", &mmodels.Media{UUID: "missing", Filename: "animation.tgs", Size: 1}); err != nil {
		t.Fatal(err)
	}
	if r.RequestCtx.Response.StatusCode() != 500 {
		t.Fatalf("status=%d", r.RequestCtx.Response.StatusCode())
	}
}
