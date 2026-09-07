package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zerodha/logf"
)

const testToken = "TOKEN123"

func TestAccountVersion(t *testing.T) {
	if got := (Account{}).Version(); got != DefaultAPIVersion {
		t.Fatalf("expected the default version, got %q", got)
	}
	if got := (Account{APIVersion: "v21.0"}).Version(); got != "v21.0" {
		t.Fatalf("expected v21.0, got %q", got)
	}
}

func TestMetaAPIErrorMessage(t *testing.T) {
	if got := (&MetaAPIError{Message: "internal"}).Error(); got != "internal" {
		t.Fatalf("expected the message, got %q", got)
	}
	// The user message is what an agent should see when Meta provides one.
	if got := (&MetaAPIError{Message: "internal", UserMsg: "for the agent"}).Error(); got != "for the agent" {
		t.Fatalf("expected the user message, got %q", got)
	}
}

func TestSetBaseURLTrimsTrailingSlash(t *testing.T) {
	c := New(testLogger())
	c.SetBaseURL("https://graph.example.test/")
	if c.baseURL != "https://graph.example.test" {
		t.Fatalf("unexpected base url %q", c.baseURL)
	}
}

func TestValidateCredentials(t *testing.T) {
	var paths []string
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/phone_numbers") {
			writeJSON(w, 200, map[string]any{"data": []map[string]string{{"id": "PN1"}}})
			return
		}
		writeJSON(w, 200, map[string]any{"id": "PN1"})
	})
	if err := c.ValidateCredentials(context.Background(), testAccount()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// A token scoped to only the number must not pass the WABA check.
	want := []string{"/v25.0/PN1", "/v25.0/WABA1/phone_numbers"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("expected %v, got %v", want, paths)
	}
}

// A phone number ID from a different WABA reachable with the same token must be rejected.
func TestValidateCredentialsRejectsForeignPhoneNumber(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/phone_numbers") {
			writeJSON(w, 200, map[string]any{"data": []map[string]string{{"id": "OTHER"}}})
			return
		}
		writeJSON(w, 200, map[string]any{"id": "PN1"})
	})
	err := c.ValidateCredentials(context.Background(), testAccount())
	if err == nil || !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("expected a membership error, got %v", err)
	}
}

func TestValidateCredentialsFailsOnPhoneNumber(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		metaError(w, 400, "bad id", 803)
	})
	if err := c.ValidateCredentials(context.Background(), testAccount()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestValidateCredentialsFailsOnWABA(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/phone_numbers") {
			metaError(w, 400, "no waba", 803)
			return
		}
		writeJSON(w, 200, map[string]any{"id": "PN1"})
	})
	if err := c.ValidateCredentials(context.Background(), testAccount()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSendText(t *testing.T) {
	var body map[string]any
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v25.0/PN1/messages" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Errorf("unexpected auth header %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("unexpected content type %q", got)
		}
		decode(t, r, &body)
		writeJSON(w, 200, sendResponse("wamid.1"))
	})

	id, err := c.SendText(context.Background(), testAccount(), "919876543210", "hello", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wamid.1" {
		t.Fatalf("unexpected message id %q", id)
	}
	text := body["text"].(map[string]any)
	if body["type"] != "text" || text["body"] != "hello" || text["preview_url"] != false {
		t.Fatalf("unexpected payload: %v", body)
	}
	if _, ok := body["context"]; ok {
		t.Fatal("a reply context must be omitted when there is nothing to reply to")
	}
}

func TestSendTextAsReply(t *testing.T) {
	var body map[string]any
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		decode(t, r, &body)
		writeJSON(w, 200, sendResponse("wamid.2"))
	})
	if _, err := c.SendText(context.Background(), testAccount(), "919876543210", "hello", "wamid.orig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ctx := body["context"].(map[string]any)
	if ctx["message_id"] != "wamid.orig" {
		t.Fatalf("unexpected context: %v", body["context"])
	}
}

func TestSendMedia(t *testing.T) {
	tests := []struct {
		name        string
		mediaType   string
		caption     string
		filename    string
		wantCaption bool
		wantName    bool
	}{
		{"image keeps the caption", "image", "look", "photo.jpg", true, false},
		{"video keeps the caption", "video", "look", "clip.mp4", true, false},
		{"document keeps both", "document", "invoice", "invoice.pdf", true, true},
		{"audio takes neither", "audio", "look", "note.ogg", false, false},
		{"empty caption is omitted", "image", "", "photo.jpg", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body map[string]any
			c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				decode(t, r, &body)
				writeJSON(w, 200, sendResponse("wamid.3"))
			})
			id, err := c.SendMedia(context.Background(), testAccount(), "919876543210", tc.mediaType, "MEDIA1", tc.caption, tc.filename, "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != "wamid.3" {
				t.Fatalf("unexpected id %q", id)
			}
			media := body[tc.mediaType].(map[string]any)
			if media["id"] != "MEDIA1" {
				t.Fatalf("unexpected media id: %v", media)
			}
			if _, ok := media["caption"]; ok != tc.wantCaption {
				t.Fatalf("caption present=%v, want %v", ok, tc.wantCaption)
			}
			if _, ok := media["filename"]; ok != tc.wantName {
				t.Fatalf("filename present=%v, want %v", ok, tc.wantName)
			}
		})
	}
}

func TestSendMediaAsReply(t *testing.T) {
	var body map[string]any
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		decode(t, r, &body)
		writeJSON(w, 200, sendResponse("wamid.4"))
	})
	if _, err := c.SendMedia(context.Background(), testAccount(), "919876543210", "image", "MEDIA1", "", "", "wamid.orig"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["context"] == nil {
		t.Fatal("expected a reply context")
	}
}

func TestSendTemplate(t *testing.T) {
	var body map[string]any
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		decode(t, r, &body)
		writeJSON(w, 200, sendResponse("wamid.5"))
	})
	components := []map[string]any{{"type": "body", "parameters": []map[string]any{{"type": "text", "text": "Ravi"}}}}
	if _, err := c.SendTemplate(context.Background(), testAccount(), "919876543210", "order_update", "en_US", components); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tmpl := body["template"].(map[string]any)
	if tmpl["name"] != "order_update" {
		t.Fatalf("unexpected template: %v", tmpl)
	}
	if tmpl["language"].(map[string]any)["code"] != "en_US" {
		t.Fatalf("unexpected language: %v", tmpl["language"])
	}
	if len(tmpl["components"].([]any)) != 1 {
		t.Fatalf("unexpected components: %v", tmpl["components"])
	}
}

func TestSendTemplateWithoutComponents(t *testing.T) {
	var body map[string]any
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		decode(t, r, &body)
		writeJSON(w, 200, sendResponse("wamid.6"))
	})
	if _, err := c.SendTemplate(context.Background(), testAccount(), "919876543210", "hello_world", "en_US", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := body["template"].(map[string]any)["components"]; ok {
		t.Fatal("expected no components key")
	}
}

func TestSendMessageWithoutMessageID(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"messaging_product": "whatsapp", "messages": []any{}})
	})
	if _, err := c.SendText(context.Background(), testAccount(), "919876543210", "hello", ""); err == nil {
		t.Fatal("expected an error when Meta returns no message id")
	}
}

func TestSendMessageWithUndecodableResponse(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	})
	if _, err := c.SendText(context.Background(), testAccount(), "919876543210", "hello", ""); err == nil {
		t.Fatal("expected a decode error")
	}
}

func TestSubscribeWebhook(t *testing.T) {
	var bodies []string
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v25.0/WABA1/subscribed_apps" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(raw))
		writeJSON(w, 200, map[string]bool{"success": true})
	})
	if err := c.SubscribeWebhook(context.Background(), testAccount(), "https://desk.example.test/webhooks/whatsapp/1", "verify-me"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bodies) != 2 {
		t.Fatalf("expected a subscribe then an override, got %d calls", len(bodies))
	}
	if bodies[0] != "" {
		t.Fatalf("expected the subscribe call to carry no body, got %q", bodies[0])
	}
	if !strings.Contains(bodies[1], "override_callback_uri") || !strings.Contains(bodies[1], "verify-me") {
		t.Fatalf("unexpected override body %q", bodies[1])
	}
}

func TestSubscribeWebhookFailures(t *testing.T) {
	t.Run("subscribe fails", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			metaError(w, 400, "cannot subscribe", 100)
		})
		err := c.SubscribeWebhook(context.Background(), testAccount(), "https://desk.example.test/x", "v")
		if err == nil || !strings.Contains(err.Error(), "subscribing app to waba") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("override fails", func(t *testing.T) {
		calls := 0
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				writeJSON(w, 200, map[string]bool{"success": true})
				return
			}
			metaError(w, 400, "cannot override", 100)
		})
		err := c.SubscribeWebhook(context.Background(), testAccount(), "https://desk.example.test/x", "v")
		if err == nil || !strings.Contains(err.Error(), "overriding waba callback") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestMarkRead(t *testing.T) {
	var body map[string]any
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		decode(t, r, &body)
		writeJSON(w, 200, map[string]bool{"success": true})
	})
	if err := c.MarkRead(context.Background(), testAccount(), "wamid.9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["status"] != "read" || body["message_id"] != "wamid.9" {
		t.Fatalf("unexpected payload: %v", body)
	}
}

func TestGetMediaURL(t *testing.T) {
	c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"url": "https://mmg.whatsapp.net/x", "mime_type": "image/png", "file_size": 12, "id": "MEDIA1"})
	})
	_ = srv
	info, err := c.GetMediaURL(context.Background(), testAccount(), "MEDIA1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.URL != "https://mmg.whatsapp.net/x" || info.MimeType != "image/png" || info.FileSize != 12 {
		t.Fatalf("unexpected media info: %+v", info)
	}
}

func TestGetMediaURLErrors(t *testing.T) {
	t.Run("meta error", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			metaError(w, 404, "gone", 100)
		})
		if _, err := c.GetMediaURL(context.Background(), testAccount(), "MEDIA1"); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("undecodable body", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("not json"))
		})
		if _, err := c.GetMediaURL(context.Background(), testAccount(), "MEDIA1"); err == nil {
			t.Fatal("expected a decode error")
		}
	})
}

func TestDownloadMedia(t *testing.T) {
	c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Errorf("media download must carry the token, got %q", got)
		}
		w.Write([]byte("filebytes"))
	})
	body, err := c.DownloadMedia(context.Background(), testAccount(), srv.URL+"/media/MEDIA1", maxMediaDownloadBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "filebytes" {
		t.Fatalf("unexpected body %q", body)
	}
}

func TestDownloadMediaErrors(t *testing.T) {
	t.Run("meta error", func(t *testing.T) {
		c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			metaError(w, 404, "expired", 100)
		})
		if _, err := c.DownloadMedia(context.Background(), testAccount(), srv.URL+"/media/x", maxMediaDownloadBytes); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("unexpected host", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {})
		_, err := c.DownloadMedia(context.Background(), testAccount(), "https://evil.example.com/media/x", maxMediaDownloadBytes)
		if err == nil || !strings.Contains(err.Error(), "unexpected host") {
			t.Fatalf("expected a host check failure, got %v", err)
		}
	})

	t.Run("unreachable host", func(t *testing.T) {
		c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {})
		url := srv.URL + "/media/x"
		srv.Close()
		if _, err := c.DownloadMedia(context.Background(), testAccount(), url, maxMediaDownloadBytes); err == nil {
			t.Fatal("expected a transport error")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := c.DownloadMedia(ctx, testAccount(), srv.URL+"/media/x", maxMediaDownloadBytes); err == nil {
			t.Fatal("expected the cancelled context to fail the request")
		}
	})
}

// A body over Meta's 100MB cap must be refused rather than buffered whole.
func TestDownloadMediaSizeCap(t *testing.T) {
	c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		chunk := make([]byte, 1<<20)
		for range (maxMediaDownloadBytes / len(chunk)) + 1 {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	})
	_, err := c.DownloadMedia(context.Background(), testAccount(), srv.URL+"/media/big", maxMediaDownloadBytes)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected a size error, got %v", err)
	}
}

func TestUploadMedia(t *testing.T) {
	var (
		gotType     string
		gotFilename string
		gotPartType string
		gotContent  string
	)
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v25.0/PN1/media" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.MultipartForm.Value["messaging_product"][0] != "whatsapp" {
			t.Error("messaging_product must be whatsapp")
		}
		gotType = r.MultipartForm.Value["type"][0]
		fh := r.MultipartForm.File["file"][0]
		gotFilename = fh.Filename
		gotPartType = fh.Header.Get("Content-Type")
		f, _ := fh.Open()
		raw, _ := io.ReadAll(f)
		gotContent = string(raw)
		writeJSON(w, 200, map[string]string{"id": "MEDIAUP1"})
	})

	id, err := c.UploadMedia(context.Background(), testAccount(), []byte("filebytes"), "image/png", "photo.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "MEDIAUP1" {
		t.Fatalf("unexpected media id %q", id)
	}
	if gotType != "image/png" || gotFilename != "photo.png" || gotContent != "filebytes" {
		t.Fatalf("unexpected upload: type=%q name=%q content=%q", gotType, gotFilename, gotContent)
	}
	// Meta validates the file part's own content type, not just the form field.
	if gotPartType != "image/png" {
		t.Fatalf("expected the part content type to be image/png, got %q", gotPartType)
	}
}

func TestUploadMediaErrors(t *testing.T) {
	t.Run("meta error", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			metaError(w, 400, "too big", 100)
		})
		if _, err := c.UploadMedia(context.Background(), testAccount(), []byte("x"), "image/png", "a.png"); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("undecodable body", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("not json"))
		})
		if _, err := c.UploadMedia(context.Background(), testAccount(), []byte("x"), "image/png", "a.png"); err == nil {
			t.Fatal("expected a decode error")
		}
	})

	t.Run("unreachable host", func(t *testing.T) {
		c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {})
		srv.Close()
		if _, err := c.UploadMedia(context.Background(), testAccount(), []byte("x"), "image/png", "a.png"); err == nil {
			t.Fatal("expected a transport error")
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := c.UploadMedia(ctx, testAccount(), []byte("x"), "image/png", "a.png"); err == nil {
			t.Fatal("expected the cancelled context to fail the request")
		}
	})
}

func TestFetchTemplatesPaginates(t *testing.T) {
	var srv *httptest.Server
	page := 0
	c, s := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			writeJSON(w, 200, map[string]any{
				"data":   []map[string]any{{"id": "1", "name": "first"}},
				"paging": map[string]any{"next": srv.URL + "/v25.0/WABA1/message_templates?after=cursor"},
			})
			return
		}
		writeJSON(w, 200, map[string]any{"data": []map[string]any{{"id": "2", "name": "second"}}})
	})
	srv = s

	out, err := c.FetchTemplates(context.Background(), testAccount())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 || out[0].Name != "first" || out[1].Name != "second" {
		t.Fatalf("unexpected templates: %+v", out)
	}
}

// A next link pointing somewhere other than Meta would leak the access token.
func TestFetchTemplatesRejectsForeignNextLink(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"data":   []map[string]any{{"id": "1", "name": "first"}},
			"paging": map[string]any{"next": "https://evil.example.com/steal"},
		})
	})
	_, err := c.FetchTemplates(context.Background(), testAccount())
	if err == nil || !strings.Contains(err.Error(), "unexpected host") {
		t.Fatalf("expected a host check failure, got %v", err)
	}
}

func TestFetchTemplatesStopsAtPageCap(t *testing.T) {
	var srv *httptest.Server
	pages := 0
	c, s := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		writeJSON(w, 200, map[string]any{
			"data":   []map[string]any{{"id": fmt.Sprint(pages), "name": "t"}},
			"paging": map[string]any{"next": srv.URL + "/v25.0/WABA1/message_templates?after=loop"},
		})
	})
	srv = s

	out, err := c.FetchTemplates(context.Background(), testAccount())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pages != maxTemplatePages || len(out) != maxTemplatePages {
		t.Fatalf("expected to stop at %d pages, made %d and got %d templates", maxTemplatePages, pages, len(out))
	}
}

func TestFetchTemplatesErrors(t *testing.T) {
	t.Run("meta error", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			metaError(w, 401, "bad token", 190)
		})
		if _, err := c.FetchTemplates(context.Background(), testAccount()); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("undecodable body", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("not json"))
		})
		if _, err := c.FetchTemplates(context.Background(), testAccount()); err == nil {
			t.Fatal("expected a decode error")
		}
	})
}

func TestSubmitTemplate(t *testing.T) {
	var body map[string]any
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v25.0/WABA1/message_templates" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		decode(t, r, &body)
		writeJSON(w, 200, map[string]any{"id": "999", "status": "PENDING", "category": "UTILITY"})
	})

	id, err := c.SubmitTemplate(context.Background(), testAccount(), TemplateSubmission{
		Name:     "order_update",
		Language: "en_US",
		Category: "UTILITY",
		Components: []TemplateComponent{
			{Type: "BODY", Text: "Hi {{1}}", Example: map[string]any{"body_text": [][]string{{"Ravi"}}}},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "999" {
		t.Fatalf("unexpected template id %q", id)
	}
	if body["name"] != "order_update" || body["language"] != "en_US" {
		t.Fatalf("unexpected submission: %v", body)
	}
}

func TestSubmitTemplateErrors(t *testing.T) {
	t.Run("meta error", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			metaError(w, 400, "policy violation", 100)
		})
		if _, err := c.SubmitTemplate(context.Background(), testAccount(), TemplateSubmission{Name: "x"}); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("undecodable body", func(t *testing.T) {
		c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("not json"))
		})
		if _, err := c.SubmitTemplate(context.Background(), testAccount(), TemplateSubmission{Name: "x"}); err == nil {
			t.Fatal("expected a decode error")
		}
	})
}

func TestDeleteTemplateEscapesName(t *testing.T) {
	var gotQuery string
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method %s", r.Method)
		}
		gotQuery = r.URL.RawQuery
		writeJSON(w, 200, map[string]bool{"success": true})
	})
	if err := c.DeleteTemplate(context.Background(), testAccount(), "name with space&more", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "name=name+with+space%26more" {
		t.Fatalf("unexpected query %q", gotQuery)
	}
}

func TestDeleteTemplateByIDTargetsOneVariant(t *testing.T) {
	var gotQuery string
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		writeJSON(w, 200, map[string]bool{"success": true})
	})
	if err := c.DeleteTemplate(context.Background(), testAccount(), "promo", "12345"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotQuery != "name=promo&hsm_id=12345" {
		t.Fatalf("unexpected query %q", gotQuery)
	}
}

func TestDeleteTemplateError(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		metaError(w, 400, "not found", 100)
	})
	if err := c.DeleteTemplate(context.Background(), testAccount(), "x", ""); err == nil {
		t.Fatal("expected an error")
	}
}

func TestEditTemplate(t *testing.T) {
	var (
		gotPath string
		body    map[string]any
	)
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		decode(t, r, &body)
		writeJSON(w, 200, map[string]bool{"success": true})
	})
	err := c.EditTemplate(context.Background(), testAccount(), "TID9", TemplateEdit{
		Category:   "UTILITY",
		Components: []TemplateComponent{{Type: "BODY", Text: "new copy"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/v25.0/TID9" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if _, ok := body["name"]; ok {
		t.Fatal("an edit must not send the template name")
	}
}

func TestEditTemplateError(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		metaError(w, 400, "cannot edit while pending", 100)
	})
	err := c.EditTemplate(context.Background(), testAccount(), "TID9", TemplateEdit{})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestDoRequestTransportFailure(t *testing.T) {
	c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {})
	srv.Close()
	err := c.MarkRead(context.Background(), testAccount(), "wamid.1")
	if err == nil || !strings.Contains(err.Error(), "calling meta api") {
		t.Fatalf("expected a transport error, got %v", err)
	}
}

func TestDoRequestUnbuildableRequest(t *testing.T) {
	c := New(testLogger())
	c.SetBaseURL("http://\x7f invalid")
	if err := c.MarkRead(context.Background(), testAccount(), "wamid.1"); err == nil {
		t.Fatal("expected a request build error")
	}
}

func TestDoRequestUnmarshalableBody(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {})
	// Channels cannot be marshalled, so the encode step must fail before any call is made.
	_, err := c.SubmitTemplate(context.Background(), testAccount(), TemplateSubmission{
		Components: []TemplateComponent{{Type: "BODY", Example: map[string]any{"bad": make(chan int)}}},
	})
	if err == nil || !strings.Contains(err.Error(), "encoding request body") {
		t.Fatalf("expected an encode error, got %v", err)
	}
}

func TestParseMetaError(t *testing.T) {
	t.Run("structured error", func(t *testing.T) {
		raw := []byte(`{"error":{"message":"Invalid parameter","type":"OAuthException","code":100,"error_subcode":2494010,"error_user_msg":"Template does not exist","fbtrace_id":"ABC"}}`)
		err := parseMetaError(400, raw)
		me, ok := err.(*MetaAPIError)
		if !ok {
			t.Fatalf("expected a MetaAPIError, got %T", err)
		}
		if me.StatusCode != 400 || me.Code != 100 || me.Subcode != 2494010 || me.Type != "OAuthException" || me.FBTraceID != "ABC" {
			t.Fatalf("unexpected error: %+v", me)
		}
		if me.Error() != "Template does not exist" {
			t.Fatalf("expected the user message, got %q", me.Error())
		}
	})

	t.Run("unstructured body", func(t *testing.T) {
		err := parseMetaError(502, []byte("<html>bad gateway</html>"))
		me := err.(*MetaAPIError)
		if me.StatusCode != 502 || !strings.Contains(me.Message, "bad gateway") {
			t.Fatalf("unexpected error: %+v", me)
		}
	})

	t.Run("json without a message", func(t *testing.T) {
		err := parseMetaError(500, []byte(`{"error":{}}`))
		me := err.(*MetaAPIError)
		if !strings.Contains(me.Message, "status 500") {
			t.Fatalf("unexpected error: %+v", me)
		}
	})
}

func TestCheckAuthenticatedHost(t *testing.T) {
	c := New(testLogger())
	c.SetBaseURL("https://graph.facebook.com")

	allowed := []string{
		"https://graph.facebook.com/v25.0/x",
		"https://lookaside.fbsbx.com/media",
		"https://scontent.xx.fbcdn.net/file",
		"https://mmg.whatsapp.net/file",
		"https://media.whatsapp.com/file",
		"https://FACEBOOK.COM/upper",
	}
	for _, u := range allowed {
		if err := c.checkAuthenticatedHost(u); err != nil {
			t.Errorf("%s should be allowed: %v", u, err)
		}
	}

	rejected := []string{
		"http://graph.facebook.com/v25.0/x",
		"https://evil.com/x",
		"https://notfacebook.com/x",
		"https://facebook.com.evil.com/x",
		"://broken",
		"",
		"/relative/path",
	}
	for _, u := range rejected {
		if err := c.checkAuthenticatedHost(u); err == nil {
			t.Errorf("%s should be rejected", u)
		}
	}
}

// A stand-in Graph API (tests, on-prem gateway) is trusted on its own scheme, but only for its host.
func TestCheckAuthenticatedHostWithPlainHTTPBaseURL(t *testing.T) {
	c := New(testLogger())
	c.SetBaseURL("http://127.0.0.1:9099")

	if err := c.checkAuthenticatedHost("http://127.0.0.1:9099/media/abc"); err != nil {
		t.Fatalf("the configured host must be allowed: %v", err)
	}
	if err := c.checkAuthenticatedHost("http://evil.example.com/media/abc"); err == nil {
		t.Fatal("another plain-http host must still be refused")
	}
	// Meta's own CDNs stay allowed, and only over https.
	if err := c.checkAuthenticatedHost("https://mmg.whatsapp.net/x"); err != nil {
		t.Fatalf("meta cdn must be allowed: %v", err)
	}
	if err := c.checkAuthenticatedHost("http://mmg.whatsapp.net/x"); err == nil {
		t.Fatal("plain http to a meta cdn must be refused")
	}
}

// The hook lets the app flag an inbox whose token Meta no longer accepts.
func TestAuthErrorHook(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		code     int
		wantFire bool
	}{
		{"401 fires", http.StatusUnauthorized, 0, true},
		{"code 190 fires", http.StatusBadRequest, 190, true},
		{"other errors do not", http.StatusBadRequest, 100, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				metaError(w, tc.status, "nope", tc.code)
			})
			var fired []Account
			c.SetAuthErrorHook(func(acc Account) { fired = append(fired, acc) })
			c.SendText(context.Background(), testAccount(), "919876543210", "hi", "")
			if got := len(fired) > 0; got != tc.wantFire {
				t.Fatalf("hook fired=%v, want %v", got, tc.wantFire)
			}
			if tc.wantFire && fired[0].PhoneNumberID != "PN1" {
				t.Fatalf("hook got the wrong account: %+v", fired[0])
			}
		})
	}
}

func TestAuthErrorHookOnMediaDownload(t *testing.T) {
	c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		metaError(w, http.StatusUnauthorized, "expired", 190)
	})
	fired := 0
	c.SetAuthErrorHook(func(acc Account) { fired++ })
	c.DownloadMedia(context.Background(), testAccount(), srv.URL+"/media/x", maxMediaDownloadBytes)
	if fired != 1 {
		t.Fatalf("expected the hook to fire once, got %d", fired)
	}
}

func TestAuthErrorHookOnUpload(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		metaError(w, http.StatusUnauthorized, "expired", 190)
	})
	fired := 0
	c.SetAuthErrorHook(func(acc Account) { fired++ })
	c.UploadMedia(context.Background(), testAccount(), []byte("x"), "image/png", "a.png")
	if fired != 1 {
		t.Fatalf("expected the hook to fire once, got %d", fired)
	}
}

func TestNoAuthErrorHookIsSafe(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		metaError(w, http.StatusUnauthorized, "expired", 190)
	})
	if _, err := c.SendText(context.Background(), testAccount(), "919876543210", "hi", ""); err == nil {
		t.Fatal("expected an error")
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := New(testLogger())
	if c.baseURL != defaultGraphURL {
		t.Fatalf("unexpected base url %q", c.baseURL)
	}
	if c.httpClient.Timeout != defaultTimeout {
		t.Fatalf("unexpected timeout %s", c.httpClient.Timeout)
	}
	if c.lo == nil {
		t.Fatal("expected a logger")
	}
}

func TestRequestHonoursContextTimeout(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		writeJSON(w, 200, map[string]bool{"success": true})
	})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := c.MarkRead(ctx, testAccount(), "wamid.1"); err == nil {
		t.Fatal("expected the request to time out")
	}
}

// A truncated response body must surface as an error, not as a silently short file.
func TestDownloadMediaTruncatedBody(t *testing.T) {
	c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(200)
		w.Write([]byte("short"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		panic(http.ErrAbortHandler)
	})
	if _, err := c.DownloadMedia(context.Background(), testAccount(), srv.URL+"/media/x", maxMediaDownloadBytes); err == nil {
		t.Fatal("expected a read error")
	}
}

func TestDoRequestTruncatedBody(t *testing.T) {
	c, _ := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(200)
		w.Write([]byte("short"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		panic(http.ErrAbortHandler)
	})
	err := c.MarkRead(context.Background(), testAccount(), "wamid.1")
	if err == nil || !strings.Contains(err.Error(), "reading meta response") {
		t.Fatalf("expected a read error, got %v", err)
	}
}

func testClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewTLSServer(handler)
	t.Cleanup(srv.Close)
	c := New(testLogger())
	c.SetBaseURL(srv.URL)
	c.httpClient = srv.Client()
	c.httpClient.Timeout = 5 * time.Second
	c.mediaHTTPClient = c.httpClient
	return c, srv
}

func testAccount() Account {
	return Account{PhoneNumberID: "PN1", WABAID: "WABA1", AccessToken: testToken, AppSecret: "SECRET"}
}

func testLogger() *logf.Logger {
	l := logf.New(logf.Opts{Level: logf.FatalLevel})
	return &l
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func metaError(w http.ResponseWriter, code int, msg string, errCode int) {
	writeJSON(w, code, map[string]any{"error": map[string]any{"message": msg, "code": errCode, "type": "OAuthException"}})
}

func sendResponse(id string) map[string]any {
	return map[string]any{
		"messaging_product": "whatsapp",
		"contacts":          []map[string]string{{"input": "919876543210", "wa_id": "919876543210"}},
		"messages":          []map[string]string{{"id": id, "message_status": "accepted"}},
	}
}

func decode(t *testing.T, r *http.Request, out any) {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal body %q: %v", raw, err)
	}
}

func TestDownloadMediaHonoursTheCallersCap(t *testing.T) {
	c, srv := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, 4096))
	})

	if _, err := c.DownloadMedia(context.Background(), testAccount(), srv.URL+"/media/x", 8192); err != nil {
		t.Fatalf("a body inside the cap must download: %v", err)
	}

	_, err := c.DownloadMedia(context.Background(), testAccount(), srv.URL+"/media/x", 1024)
	if err == nil {
		t.Fatal("a body over the caller's cap must be rejected")
	}
	if !errors.Is(err, ErrMediaTooLarge) {
		t.Fatalf("expected ErrMediaTooLarge so the caller can store a placeholder instead of retrying, got %v", err)
	}
}

func TestMediaExceedsCap(t *testing.T) {
	if !MediaExceedsCap(MediaInfo{FileSize: 20 << 20}, 10<<20) {
		t.Fatal("a declared size over the cap must be refused up front")
	}
	if MediaExceedsCap(MediaInfo{FileSize: 5 << 20}, 10<<20) {
		t.Fatal("a declared size inside the cap must be allowed")
	}
	// Meta does not always send file_size.
	if MediaExceedsCap(MediaInfo{}, 10<<20) {
		t.Fatal("an unknown size must not be treated as oversized")
	}
	if MediaExceedsCap(MediaInfo{FileSize: 20 << 20}, 0) {
		t.Fatal("an unset cap must not reject everything")
	}
}
