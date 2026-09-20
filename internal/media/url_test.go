package media

import (
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGatewayMediaURLs(t *testing.T) {
	m := &Manager{signingKey: "test-key", urlExpiry: time.Hour, rootURL: func() string { return "https://desk.example" }}
	id := "12345678-1234-4234-9234-123456789abc"
	for _, raw := range []string{m.GetURL(id, "image/png", "image.png"), m.GetSignedURL(id), m.GetThumbnailURL(id), m.GetURLForDownload(id, "image.png")} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if u.Host != "desk.example" || !strings.HasPrefix(u.Path, "/uploads/") {
			t.Fatalf("external storage exposed: %s", raw)
		}
		exp, _ := strconv.ParseInt(u.Query().Get("exp"), 10, 64)
		if !m.SignedURLValidator()(id, u.Query().Get("sig"), exp) {
			t.Fatalf("invalid gateway signature: %s", raw)
		}
		if m.SignedURLValidator()("another", u.Query().Get("sig"), exp) {
			t.Fatal("signature crossed media boundary")
		}
		if m.SignedURLValidator()(id, u.Query().Get("sig"), 1) {
			t.Fatal("expired signature accepted")
		}
	}
	if m.GetSignedURL("../other") != "" {
		t.Fatal("invalid media identifier accepted")
	}
}
