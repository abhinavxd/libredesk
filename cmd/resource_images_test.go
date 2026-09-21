package main

import (
	"encoding/json"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/resourcepolicy"
	"github.com/volatiletech/null/v9"
)

func TestImageSourcesOnlyIncludeValidMessageImages(t *testing.T) {
	msg := cmodels.Message{Content: `<img src="https://images.example/a?x=1&amp;y=2"><a href="https://images.example/link">Link</a><img src="http://images.example/plain"><img src="https://127.0.0.1/private"><div style="background-image:url(https://images.example/css)"></div>`}
	sources := messageImageSources(msg)
	want := "https://images.example/a?x=1&y=2"
	if len(sources) != 1 || sources[imageSourceID(want)] != want {
		t.Fatalf("unexpected sources: %v", sources)
	}
	msg.Content = strings.Repeat(`<img src="https://images.example/a">`, 100)
	if len(messageImageSources(msg)) != 1 {
		t.Fatal("duplicate images counted separately")
	}
}

func TestBlockAllIgnoresImageResolver(t *testing.T) {
	display := (resourcepolicy.Policy{}).PrepareDisplayWithImages(`<img src="https://images.example/a">`, nil, func(string) string { t.Fatal("block_all invoked resolver"); return "/allowed" })
	if display.BlockedImages != 1 || strings.Contains(display.HTML, "<img") {
		t.Fatal("block_all allowed an external image")
	}
}

func TestImageSenderUsesExactContactAddress(t *testing.T) {
	msg := cmodels.Message{Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, Meta: json.RawMessage(`{"from":["Person <person@EXAMPLE.COM>"]}`), Author: cmodels.MessageAuthor{Email: null.StringFrom("changed@example.com")}}
	if imageSender(msg) != "person@example.com" {
		t.Fatal("incorrect exact sender")
	}
	msg.SenderType = cmodels.SenderTypeAgent
	if imageSender(msg) != "" {
		t.Fatal("agent treated as incoming sender")
	}
	msg.SenderType = cmodels.SenderTypeContact
	msg.Meta = json.RawMessage(`{"from":["one@example.com", "two@example.com"]}`)
	if imageSender(msg) != "" {
		t.Fatal("ambiguous sender accepted")
	}
}

func TestImageSenderRequiresAnUnambiguousStoredAddress(t *testing.T) {
	for _, meta := range []string{`{}`, `null`, `{"from":[]}`, `{"from":"sender@example.com"}`, `{"from":["one@example.com,two@example.com"]}`, `invalid`} {
		msg := cmodels.Message{Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact, Meta: json.RawMessage(meta), Author: cmodels.MessageAuthor{Email: null.StringFrom("trusted@example.com")}}
		if imageSender(msg) != "" {
			t.Fatalf("fell back to mutable contact address for %s", meta)
		}
	}
}

func TestResourceImagesCannotUseGenericMediaRoute(t *testing.T) {
	for _, model := range []string{mmodels.ModelResourceImages, mmodels.ModelResourceAvatars} {
		r := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
		media := &mmodels.Media{Model: null.StringFrom(model)}
		if err := serveMediaFile(r, &App{}, "unused", media); err != nil {
			t.Fatal(err)
		}
		if r.RequestCtx.Response.StatusCode() != fasthttp.StatusForbidden {
			t.Fatal("generic media route bypassed image permissions")
		}
	}
}

func TestInlineMediaTypesExcludeActiveContent(t *testing.T) {
	for _, kind := range []string{"text/html", "image/svg+xml", "application/pdf", "audio/x-mpegurl", "application/vnd.apple.mpegurl"} {
		if inlineMediaType(kind) {
			t.Fatalf("active preview allowed: %s", kind)
		}
	}
	for _, kind := range []string{"image/png", "audio/mpeg", "video/mp4"} {
		if !inlineMediaType(kind) {
			t.Fatalf("safe preview blocked: %s", kind)
		}
	}
}

func TestImageLoadingModes(t *testing.T) {
	for _, mode := range []string{resourcepolicy.BlockAll, resourcepolicy.Allowlist, resourcepolicy.LoadOnReceipt, "invalid"} {
		policy, _ := resourcepolicy.New(resourcepolicy.Config{Mode: mode, AllowedDomains: []string{"trusted.example"}})
		for _, consent := range []bool{false, true} {
			for _, source := range []string{"https://trusted.example/a", "https://unknown.example/a", "https://127.0.0.1/a"} {
				want := source != "https://127.0.0.1/a" && (mode == resourcepolicy.LoadOnReceipt || (mode == resourcepolicy.Allowlist && (consent || source == "https://trusted.example/a")))
				if imageLoadingAllowed(policy, source, consent) != want {
					t.Fatalf("mode %s, consent %v, source %s", mode, consent, source)
				}
			}
		}
	}
}
