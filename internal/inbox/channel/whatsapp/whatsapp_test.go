package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/whatsapp"
	"github.com/zerodha/logf"
)

func TestConfigAccount(t *testing.T) {
	cfg := Config{
		PhoneNumberID: "PN1",
		WABAID:        "WABA1",
		AccessToken:   "TOKEN",
		AppSecret:     "SECRET",
		APIVersion:    "v25.0",
	}
	acc := cfg.Account()
	if acc.PhoneNumberID != "PN1" || acc.WABAID != "WABA1" || acc.AccessToken != "TOKEN" || acc.AppSecret != "SECRET" || acc.APIVersion != "v25.0" {
		t.Fatalf("unexpected account: %+v", acc)
	}
	// The CSAT fields are libredesk-side and must not leak into Meta calls.
	if acc.Version() != "v25.0" {
		t.Fatalf("unexpected version %q", acc.Version())
	}
}

func TestNew(t *testing.T) {
	client := whatsapp.New(testLogger())
	tests := []struct {
		name    string
		opts    Opts
		wantErr string
	}{
		{
			name:    "no client",
			opts:    Opts{ID: 1, Config: Config{PhoneNumberID: "PN1", AccessToken: "T"}, Lo: testLogger()},
			wantErr: "client is required",
		},
		{
			name:    "no phone number id",
			opts:    Opts{ID: 1, Config: Config{AccessToken: "T"}, Client: client, Lo: testLogger()},
			wantErr: "phone_number_id and access_token are required",
		},
		{
			name:    "no access token",
			opts:    Opts{ID: 1, Config: Config{PhoneNumberID: "PN1"}, Client: client, Lo: testLogger()},
			wantErr: "phone_number_id and access_token are required",
		},
		{
			name:    "no logger",
			opts:    Opts{ID: 1, Config: Config{PhoneNumberID: "PN1", AccessToken: "T"}, Client: client},
			wantErr: "logger is required",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(nil, tc.opts); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestInboxAccessors(t *testing.T) {
	inb := testInbox(t, nil, nil)
	if inb.Identifier() != 7 {
		t.Fatalf("unexpected id %d", inb.Identifier())
	}
	if inb.Name() != "WA Inbox" {
		t.Fatalf("unexpected name %q", inb.Name())
	}
	if inb.Channel() != ChannelWhatsApp {
		t.Fatalf("unexpected channel %q", inb.Channel())
	}
	if inb.Config().PhoneNumberID != "PN1" {
		t.Fatalf("unexpected config %+v", inb.Config())
	}
	// The email template machinery reads these, and WhatsApp has no address to give it.
	if inb.FromAddress() != "" || inb.ReplyToAddress() != "" || inb.FromNameTemplate() != "" {
		t.Fatal("expected empty email fields")
	}
	if err := inb.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// Inbound arrives over the webhook, so Receive does nothing.
	if err := inb.Receive(context.Background()); err != nil {
		t.Fatalf("receive: %v", err)
	}
}

func TestSendText(t *testing.T) {
	var body map[string]any
	updater := &fakeSourceUpdater{}
	inb := testInbox(t, func(w http.ResponseWriter, r *http.Request) {
		decodeBody(t, r, &body)
		writeSendResponse(w, "wamid.OUT1")
	}, updater)

	err := inb.Send(models.OutboundMessage{
		UUID:        "msg-uuid",
		Content:     "<p>hello <b>there</b></p>",
		ContentType: models.ContentTypeHTML,
		Meta:        json.RawMessage(`{"whatsapp":{"to_phone":"919876543210"}}`),
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if body["type"] != "text" {
		t.Fatalf("expected a text send, got %v", body)
	}
	if got := body["text"].(map[string]any)["body"].(string); strings.Contains(got, "<") {
		t.Fatalf("HTML must be flattened before it reaches WhatsApp, got %q", got)
	}
	// The Meta message id is what later status webhooks are matched on.
	if updater.uuid != "msg-uuid" || updater.sourceID != "wamid.OUT1" {
		t.Fatalf("unexpected source id update: %+v", updater)
	}
}

func TestSendTemplate(t *testing.T) {
	var body map[string]any
	inb := testInbox(t, func(w http.ResponseWriter, r *http.Request) {
		decodeBody(t, r, &body)
		writeSendResponse(w, "wamid.OUT2")
	}, &fakeSourceUpdater{})

	meta := SendMeta{
		ToPhone:             "919876543210",
		TemplateName:        "order_update",
		TemplateLanguage:    "en_US",
		TemplateBodyContent: "Hi {{name}}",
		TemplateParams:      map[string]string{"body:name": "Ravi"},
	}
	raw, _ := json.Marshal(map[string]any{"whatsapp": meta})
	if err := inb.Send(models.OutboundMessage{UUID: "u", Meta: raw}); err != nil {
		t.Fatalf("send: %v", err)
	}
	tmpl := body["template"].(map[string]any)
	if tmpl["name"] != "order_update" {
		t.Fatalf("unexpected template: %v", tmpl)
	}
	if len(tmpl["components"].([]any)) != 1 {
		t.Fatalf("expected the body component, got %v", tmpl["components"])
	}
}

// A template send wins over any free-form content on the same message.
func TestSendTemplateTakesPrecedenceOverContent(t *testing.T) {
	var body map[string]any
	inb := testInbox(t, func(w http.ResponseWriter, r *http.Request) {
		decodeBody(t, r, &body)
		writeSendResponse(w, "wamid.OUT3")
	}, nil)
	raw, _ := json.Marshal(map[string]any{"whatsapp": SendMeta{ToPhone: "91", TemplateName: "t", TemplateLanguage: "en_US"}})
	if err := inb.Send(models.OutboundMessage{UUID: "u", TextContent: "free form", Meta: raw}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if body["type"] != "template" {
		t.Fatalf("expected a template send, got %v", body["type"])
	}
}

func TestSendAttachment(t *testing.T) {
	var (
		uploaded string
		sent     map[string]any
	)
	inb := testInbox(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/media") {
			r.ParseMultipartForm(1 << 20)
			uploaded = r.MultipartForm.File["file"][0].Filename
			writeJSONBody(w, map[string]string{"id": "MEDIAUP1"})
			return
		}
		decodeBody(t, r, &sent)
		writeSendResponse(w, "wamid.OUT4")
	}, nil)

	err := inb.Send(models.OutboundMessage{
		UUID:        "u",
		TextContent: "see attached",
		Meta:        json.RawMessage(`{"whatsapp":{"to_phone":"919876543210"}}`),
		Attachments: attachment.Attachments{{Name: "invoice.pdf", ContentType: "application/pdf", Content: []byte("pdfbytes"), Size: 8}},
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if uploaded != "invoice.pdf" {
		t.Fatalf("unexpected upload %q", uploaded)
	}
	doc := sent["document"].(map[string]any)
	if sent["type"] != "document" || doc["id"] != "MEDIAUP1" || doc["caption"] != "see attached" || doc["filename"] != "invoice.pdf" {
		t.Fatalf("unexpected send payload: %v", sent)
	}
}

func TestSendAttachmentFailures(t *testing.T) {
	tests := []struct {
		name        string
		message     models.OutboundMessage
		handler     http.HandlerFunc
		wantErrPart string
	}{
		{
			name: "two attachments",
			message: models.OutboundMessage{
				UUID: "u",
				Meta: json.RawMessage(`{"whatsapp":{"to_phone":"91"}}`),
				Attachments: attachment.Attachments{
					{Name: "a.pdf", ContentType: "application/pdf", Size: 1},
					{Name: "b.pdf", ContentType: "application/pdf", Size: 1},
				},
			},
			wantErrPart: "one attachment per message",
		},
		{
			name: "unsupported type",
			message: models.OutboundMessage{
				UUID:        "u",
				Meta:        json.RawMessage(`{"whatsapp":{"to_phone":"91"}}`),
				Attachments: attachment.Attachments{{Name: "logs.zip", ContentType: "application/zip", Size: 1}},
			},
			wantErrPart: "can't send these files",
		},
		{
			name: "upload rejected by meta",
			message: models.OutboundMessage{
				UUID:        "u",
				Meta:        json.RawMessage(`{"whatsapp":{"to_phone":"91"}}`),
				Attachments: attachment.Attachments{{Name: "a.pdf", ContentType: "application/pdf", Content: []byte("x"), Size: 1}},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(400)
				w.Write([]byte(`{"error":{"message":"too big","code":100}}`))
			},
			wantErrPart: "uploading attachment to meta",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inb := testInbox(t, tc.handler, nil)
			err := inb.Send(tc.message)
			if err == nil || !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Fatalf("expected %q, got %v", tc.wantErrPart, err)
			}
		})
	}
}

func TestSendRejectsUnusableMessages(t *testing.T) {
	tests := []struct {
		name        string
		message     models.OutboundMessage
		wantErrPart string
	}{
		{
			name:        "no recipient",
			message:     models.OutboundMessage{UUID: "u", TextContent: "hi"},
			wantErrPart: "missing recipient phone number",
		},
		{
			name:        "malformed meta",
			message:     models.OutboundMessage{UUID: "u", Meta: json.RawMessage(`{"whatsapp":"not an object"}`)},
			wantErrPart: "parsing whatsapp send meta",
		},
		{
			name:        "nothing to send",
			message:     models.OutboundMessage{UUID: "u", Meta: json.RawMessage(`{"whatsapp":{"to_phone":"91"}}`)},
			wantErrPart: "no content",
		},
		{
			name:        "whitespace only",
			message:     models.OutboundMessage{UUID: "u", TextContent: "   ", Meta: json.RawMessage(`{"whatsapp":{"to_phone":"91"}}`)},
			wantErrPart: "no content",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inb := testInbox(t, nil, nil)
			err := inb.Send(tc.message)
			if err == nil || !strings.Contains(err.Error(), tc.wantErrPart) {
				t.Fatalf("expected %q, got %v", tc.wantErrPart, err)
			}
		})
	}
}

// Meta accepted the message, so the id has to be stored even though the update itself failed.
func TestSendKeepsGoingWhenSourceIDUpdateFails(t *testing.T) {
	updater := &fakeSourceUpdater{err: errors.New("db down")}
	inb := testInbox(t, func(w http.ResponseWriter, r *http.Request) {
		writeSendResponse(w, "wamid.OUT5")
	}, updater)
	err := inb.Send(models.OutboundMessage{UUID: "u", TextContent: "hi", Meta: json.RawMessage(`{"whatsapp":{"to_phone":"91"}}`)})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if updater.calls != 1 {
		t.Fatalf("expected one update attempt, got %d", updater.calls)
	}
}

func TestSendWithoutSourceUpdater(t *testing.T) {
	inb := testInbox(t, func(w http.ResponseWriter, r *http.Request) {
		writeSendResponse(w, "wamid.OUT6")
	}, nil)
	if err := inb.Send(models.OutboundMessage{UUID: "u", TextContent: "hi", Meta: json.RawMessage(`{"whatsapp":{"to_phone":"91"}}`)}); err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestSendSurfacesMetaError(t *testing.T) {
	inb := testInbox(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":{"message":"outside window","code":131047,"error_user_msg":"more than 24 hours"}}`))
	}, &fakeSourceUpdater{})
	err := inb.Send(models.OutboundMessage{UUID: "u", TextContent: "hi", Meta: json.RawMessage(`{"whatsapp":{"to_phone":"91"}}`)})
	if err == nil || !strings.Contains(err.Error(), "more than 24 hours") {
		t.Fatalf("expected the Meta user message, got %v", err)
	}
}

func TestHumanBytes(t *testing.T) {
	tests := map[int]string{
		512:              "512 B",
		2048:             "2 KB",
		5 * 1024 * 1024:  "5.0 MB",
		16 * 1024 * 1024: "16.0 MB",
	}
	for size, want := range tests {
		if got := humanBytes(size); got != want {
			t.Errorf("%d: expected %q, got %q", size, want, got)
		}
	}
}

func TestMaxMediaBytes(t *testing.T) {
	if maxMediaBytes("audio") != effectiveLimit(maxAudioBytes) {
		t.Fatal("audio must use the audio cap")
	}
	if maxMediaBytes("sticker") != effectiveLimit(maxDocumentBytes) {
		t.Fatal("an unknown media type must fall back to the document cap")
	}
}

func TestRejectMediaReason(t *testing.T) {
	tests := []struct {
		name        string
		file        string
		contentType string
		size        int
		wantMatch   string
	}{
		{"pdf under cap", "invoice.pdf", "application/pdf", 1024, ""},
		{"jpeg under cap", "photo.jpg", "image/jpeg", 1024, ""},
		{"jpeg with charset parameter", "photo.jpg", "image/jpeg; charset=binary", 1024, ""},
		{"uppercase mime", "photo.jpg", "IMAGE/JPEG", 1024, ""},
		{"webp", "sticker.webp", "image/webp", 1024, "WebP"},
		{"zip", "logs.zip", "application/zip", 1024, "unsupported type"},
		{"oversized image", "big.jpg", "image/jpeg", 6 * 1024 * 1024, "image limit"},
		{"oversized video", "big.mp4", "video/mp4", 20 * 1024 * 1024, "video limit"},
		{"oversized document", "big.pdf", "application/pdf", 101 * 1024 * 1024, "document limit"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RejectMediaReason(tc.file, tc.contentType, tc.size)
			if tc.wantMatch == "" {
				if got != "" {
					t.Fatalf("expected the file to be accepted, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.wantMatch) {
				t.Fatalf("expected reason to mention %q, got %q", tc.wantMatch, got)
			}
		})
	}
}

// An image just under Meta's cap must pass while the 2% headroom keeps one at the cap out.
func TestRejectMediaReasonImageBoundary(t *testing.T) {
	if got := RejectMediaReason("photo.jpg", "image/jpeg", effectiveLimit(maxImageBytes)); got != "" {
		t.Fatalf("expected the file at the effective limit to be accepted, got %q", got)
	}
	if got := RejectMediaReason("photo.jpg", "image/jpeg", maxImageBytes); got == "" {
		t.Fatal("expected a file at Meta's hard cap to be rejected by the headroom")
	}
}

func TestMediaTypeForAttachment(t *testing.T) {
	tests := map[string]string{
		"image/jpeg":       "image",
		"image/png":        "image",
		"video/mp4":        "video",
		"video/3gpp":       "video",
		"audio/ogg":        "audio",
		"audio/mpeg":       "audio",
		"application/pdf":  "document",
		"text/plain":       "document",
		"application/zip":  "document",
		"IMAGE/PNG ":       "image",
		"image/png; x=y":   "image",
		"application/json": "document",
	}
	for contentType, want := range tests {
		if got := mediaTypeForAttachment(attachment.Attachment{ContentType: contentType}); got != want {
			t.Errorf("%s: expected %s, got %s", contentType, want, got)
		}
	}
}

func TestSupportsCaption(t *testing.T) {
	if SupportsCaption("audio/ogg") {
		t.Fatal("audio must not advertise caption support")
	}
	for _, contentType := range []string{"image/jpeg", "video/mp4", "application/pdf"} {
		if !SupportsCaption(contentType) {
			t.Errorf("%s should support a caption", contentType)
		}
	}
}

func TestParseSendMeta(t *testing.T) {
	t.Run("envelope key wins", func(t *testing.T) {
		meta, err := parseSendMeta(json.RawMessage(`{"to":["a@b.c"],"whatsapp":{"to_phone":"919876543210","template_name":"order_update"}}`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if meta.ToPhone != "919876543210" || meta.TemplateName != "order_update" {
			t.Fatalf("unexpected meta: %+v", meta)
		}
	})

	t.Run("flat payload", func(t *testing.T) {
		meta, err := parseSendMeta(json.RawMessage(`{"to_phone":"919876543210"}`))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if meta.ToPhone != "919876543210" {
			t.Fatalf("unexpected meta: %+v", meta)
		}
	})

	t.Run("empty meta", func(t *testing.T) {
		meta, err := parseSendMeta(nil)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		if meta.ToPhone != "" {
			t.Fatalf("expected a zero meta, got %+v", meta)
		}
	})

	t.Run("malformed meta", func(t *testing.T) {
		if _, err := parseSendMeta(json.RawMessage(`not json`)); err == nil {
			t.Fatal("expected an error for malformed meta")
		}
	})
}

func TestTextBody(t *testing.T) {
	if got := textBody(models.OutboundMessage{TextContent: "plain", Content: "<p>html</p>"}); got != "plain" {
		t.Fatalf("expected the stored text content, got %q", got)
	}
	got := textBody(models.OutboundMessage{ContentType: models.ContentTypeHTML, Content: "<p>hello <b>there</b></p>"})
	if strings.Contains(got, "<") {
		t.Fatalf("expected HTML to be flattened, got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("expected the text to survive, got %q", got)
	}
}

func testInbox(t *testing.T, handler http.HandlerFunc, updater SourceIDUpdater) *WhatsApp {
	t.Helper()
	if handler == nil {
		handler = func(w http.ResponseWriter, r *http.Request) { writeSendResponse(w, "wamid.DEFAULT") }
	}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := whatsapp.New(testLogger())
	client.SetBaseURL(srv.URL)

	inb, err := New(nil, Opts{
		ID:            7,
		Name:          "WA Inbox",
		Config:        Config{PhoneNumberID: "PN1", WABAID: "WABA1", AccessToken: "TOKEN", APIVersion: "v25.0"},
		Client:        client,
		Lo:            testLogger(),
		SourceUpdater: updater,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return inb
}

func testLogger() *logf.Logger {
	l := logf.New(logf.Opts{Level: logf.FatalLevel})
	return &l
}

func writeJSONBody(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeSendResponse(w http.ResponseWriter, id string) {
	writeJSONBody(w, map[string]any{
		"messaging_product": "whatsapp",
		"messages":          []map[string]string{{"id": id}},
	})
}

func decodeBody(t *testing.T, r *http.Request, out any) {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal %q: %v", raw, err)
	}
}

type fakeSourceUpdater struct {
	uuid     string
	sourceID string
	calls    int
	err      error
}

func (f *fakeSourceUpdater) UpdateMessageSourceID(messageUUID, sourceID string) error {
	f.calls++
	f.uuid, f.sourceID = messageUUID, sourceID
	return f.err
}
