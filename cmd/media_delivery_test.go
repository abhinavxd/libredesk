package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/media"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/media/stores/s3"
	"github.com/abhinavxd/libredesk/internal/resourceimage"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestS3MediaServedWithoutBrowserRedirect(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte("stored media"))
	}))
	defer server.Close()
	store, err := s3.New(s3.Opt{URL: server.URL, AccessKey: "test", SecretKey: "test", Bucket: "test", BucketType: "private"})
	if err != nil {
		t.Fatal(err)
	}
	db := testutil.NewDB(t, "media_delivery")
	lo := logf.New(logf.Opts{})
	manager, err := media.New(media.Opts{Store: store, DB: db, Lo: &lo, SigningKey: "key", RootURL: func() string { return "https://desk.example" }})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{media: manager}
	app.consts.Store(&constants{UploadProvider: "s3"})
	for _, kind := range []string{"image/png", "audio/mpeg", "video/mp4", "image/svg+xml", "text/html"} {
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
		r.RequestCtx.Request.SetRequestURI("https://desk.example/uploads/12345678-1234-4234-9234-123456789abc")
		metadata := &mmodels.Media{ContentType: kind, Filename: "test", Private: true}
		if err := serveMediaFile(r, app, "12345678-1234-4234-9234-123456789abc", metadata); err != nil {
			t.Fatal(err)
		}
		if got := string(r.RequestCtx.Response.Body()); got != "stored media" {
			t.Fatalf("wrong body: %q", got)
		}
		if len(r.RequestCtx.Response.Header.Peek("Location")) != 0 {
			t.Fatal("browser redirected to storage")
		}
		if string(r.RequestCtx.Response.Header.Peek("Content-Security-Policy")) != "default-src 'none'; sandbox" {
			t.Fatal("missing sandbox")
		}
		if (kind == "image/svg+xml" || kind == "text/html") && !strings.HasPrefix(string(r.RequestCtx.Response.Header.Peek("Content-Disposition")), "attachment") {
			t.Fatal("active content served inline")
		}
	}
	if calls != 5 {
		t.Fatalf("storage fetched %d times", calls)
	}
}

func TestAvatarGatewayRejectsBlockedAndUnconfiguredSources(t *testing.T) {
	db := testutil.NewDB(t, "avatar_gateway")
	lo := logf.New(logf.Opts{})
	settings, err := setting.New(setting.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{setting: settings, resourceImages: resourceimage.NewStore(db, nil, "fs", settings.GetResourcePolicyTx), lo: &lo}
	request := func(source string) int {
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}, Context: app}
		r.RequestCtx.QueryArgs().Set("source", source)
		if err := handleResourceAvatar(r); err != nil {
			t.Fatal(err)
		}
		return r.RequestCtx.Response.StatusCode()
	}
	if err := settings.SetResourcePolicy(resourcepolicy.Blocked()); err != nil {
		t.Fatal(err)
	}
	if got := request("https://avatars.example/person"); got != 403 {
		t.Fatalf("block_all returned %d", got)
	}
	if err := settings.SetResourcePolicy(resourcepolicy.Config{Mode: resourcepolicy.Allowlist, MaxCacheBytes: resourcepolicy.DefaultMaxCacheBytes}); err != nil {
		t.Fatal(err)
	}
	if got := request("https://avatars.example/person"); got != 404 {
		t.Fatalf("unconfigured source returned %d", got)
	}
	if got := request("https://127.0.0.1/image"); got != 403 {
		t.Fatalf("unsafe source returned %d", got)
	}
}

func TestStoredMediaRanges(t *testing.T) {
	for _, tc := range []struct {
		value, want string
		status      int
	}{
		{"bytes=3-5", "345", 206},
		{"bytes=-3", "789", 206},
		{"bytes=6-", "6789", 206},
		{"bytes=100-", "", 416},
		{"bytes=-0", "", 416},
	} {
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
		r.RequestCtx.Request.Header.Set("Range", tc.value)
		if err := streamStoredMedia(r, io.NopCloser(strings.NewReader("0123456789")), 10, false); err != nil {
			t.Fatal(err)
		}
		if got := string(r.RequestCtx.Response.Body()); got != tc.want || r.RequestCtx.Response.StatusCode() != tc.status {
			t.Fatalf("%s: %q, %d", tc.value, got, r.RequestCtx.Response.StatusCode())
		}
	}
}
