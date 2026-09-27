package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

type brokenBody struct{}

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func (brokenBody) Read([]byte) (int, error)                               { return 0, errors.New("read failed") }
func (brokenBody) Close() error                                           { return nil }

func TestCredentialsAndWebhooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected JSON")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case "/bot123:secret/getMe":
			io.WriteString(w, `{"ok":true,"result":{"id":123,"is_bot":true,"username":"support_bot"}}`)
		case "/bot123:secret/setWebhook":
			if body["url"] != "https://support.example/webhook" || body["secret_token"] != "webhook-secret" || body["drop_pending_updates"] != nil {
				t.Errorf("unexpected registration: %v", body)
			}
			updates := body["allowed_updates"].([]any)
			if len(updates) != 5 || updates[0] != "message" || updates[1] != "edited_message" || updates[2] != "business_message" || updates[3] != "edited_business_message" || updates[4] != "callback_query" {
				t.Errorf("unexpected updates: %v", updates)
			}
			io.WriteString(w, `{"ok":true,"result":true}`)
		case "/bot123:secret/answerCallbackQuery":
			if body["callback_query_id"] != "callback-1" {
				t.Errorf("callback=%v", body)
			}
			io.WriteString(w, `{"ok":true,"result":true}`)
		case "/bot123:secret/deleteWebhook":
			if len(body) != 0 {
				t.Errorf("must preserve pending updates: %v", body)
			}
			io.WriteString(w, `{"ok":true,"result":true}`)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c := New()
	c.SetBaseURL(server.URL + "/")
	user, err := c.GetMe(t.Context(), "123:secret")
	if err != nil || user.ID != 123 || !user.IsBot || user.Username != "support_bot" {
		t.Fatalf("user=%+v err=%v", user, err)
	}
	if err := c.SetWebhook(t.Context(), "123:secret", "https://support.example/webhook", "webhook-secret"); err != nil {
		t.Fatal(err)
	}
	if err := c.AnswerCallbackQuery(t.Context(), "123:secret", "callback-1"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteWebhook(t.Context(), "123:secret"); err != nil {
		t.Fatal(err)
	}
}

func TestSendTextAndMedia(t *testing.T) {
	for _, tc := range []struct{ mime, method, field string }{
		{"", "sendMessage", ""}, {"text/plain", "sendDocument", "document"}, {"image/jpeg", "sendPhoto", "photo"},
		{"image/png", "sendPhoto", "photo"}, {"image/gif", "sendAnimation", "animation"}, {"video/mp4", "sendVideo", "video"},
		{"audio/mpeg", "sendAudio", "audio"}, {"audio/mp4", "sendAudio", "audio"}, {"audio/ogg", "sendVoice", "voice"},
	} {
		t.Run(tc.method+tc.mime, func(t *testing.T) {
			payload := []byte("attachment")
			if tc.field == "photo" {
				var buf bytes.Buffer
				if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
					t.Fatal(err)
				}
				payload = buf.Bytes()
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/bot1:token/"+tc.method {
					t.Errorf("path=%s", r.URL.Path)
				}
				if tc.field == "" {
					var body map[string]any
					json.NewDecoder(r.Body).Decode(&body)
					if body["text"] != "A & B <literal> 😀" || body["chat_id"] != float64(4500000000000) || body["parse_mode"] != nil {
						t.Errorf("body=%v", body)
					}
				} else {
					if err := r.ParseMultipartForm(1024); err != nil {
						t.Fatal(err)
					}
					if r.FormValue("caption") != "A & B <literal> 😀" || r.FormValue("chat_id") != "4500000000000" {
						t.Error("missing recipient or caption")
					}
					file, header, err := r.FormFile(tc.field)
					if err != nil {
						t.Fatal(err)
					}
					defer file.Close()
					content, _ := io.ReadAll(file)
					if !bytes.Equal(content, payload) || header.Filename != "file.txt" {
						t.Errorf("file=%s name=%s", content, header.Filename)
					}
				}
				io.WriteString(w, `{"ok":true,"result":{"message_id":99}}`)
			}))
			defer server.Close()
			c := New()
			c.SetBaseURL(server.URL)
			var files attachment.Attachments
			if tc.mime != "" {
				files = attachment.Attachments{{Name: "file.txt", ContentType: tc.mime, Content: payload}}
			}
			id, err := c.Send(t.Context(), "1:token", 4500000000000, "A & B <literal> 😀", files, SendOptions{})
			if err != nil || id != 99 {
				t.Fatalf("id=%d err=%v", id, err)
			}
		})
	}
}

func TestSendLimitsAndLargePhoto(t *testing.T) {
	c := New()
	if _, err := c.Send(t.Context(), "token", 1, "", attachment.Attachments{{}, {}}, SendOptions{}); err == nil {
		t.Fatal("multiple attachments accepted")
	}
	if _, err := c.Send(t.Context(), "token", 1, "", attachment.Attachments{{Content: make([]byte, MaxUploadBytes+1)}}, SendOptions{}); err == nil {
		t.Fatal("oversize accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sendDocument") {
			t.Error("large photo must send as a document")
		}
		io.Copy(io.Discard, r.Body)
		io.WriteString(w, `{"ok":true,"result":{"message_id":8}}`)
	}))
	defer server.Close()
	c.SetBaseURL(server.URL)
	if _, err := c.Send(t.Context(), "token", 1, "", attachment.Attachments{{ContentType: "image/jpeg", Content: make([]byte, 10*1024*1024+1)}}, SendOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestRateLimitRetriesOnlyExplicitRefusals(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(429)
			io.WriteString(w, `{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":1}}`)
			return
		}
		io.WriteString(w, `{"ok":true,"result":{"message_id":9}}`)
	}))
	defer server.Close()
	c := New()
	c.SetBaseURL(server.URL)
	if id, err := c.Send(t.Context(), "token", 1, "hi", nil, SendOptions{}); err != nil || id != 9 || calls != 2 {
		t.Fatalf("id=%d calls=%d err=%v", id, calls, err)
	}
	for _, code := range []int{400, 401, 403, 500} {
		calls = 0
		c.http = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(`{"ok":false,"description":"refused"}`))}, nil
		})}
		if _, err := c.Send(t.Context(), "token", 1, "hi", nil, SendOptions{}); err == nil || calls != 1 {
			t.Errorf("code=%d calls=%d err=%v", code, calls, err)
		}
	}
	c.http = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader(`{"ok":false,"error_code":429}`))}, nil
	})}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Send(ctx, "token", 1, "hi", nil, SendOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation: %v", err)
	}
	calls = 0
	c.http = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader(`{"ok":false,"error_code":429}`))}, nil
	})}
	if _, err := c.Send(t.Context(), "token", 1, "hi", nil, SendOptions{}); err == nil || calls != 3 {
		t.Fatalf("retry bound: calls=%d err=%v", calls, err)
	}
}

func TestDownload(t *testing.T) {
	for _, tc := range []struct {
		name, metadata string
		status         int
		body           string
		wantError      bool
	}{
		{"normal", `{"file_id":"a","file_path":"documents/a.txt","file_size":3}`, 200, "abc", false},
		{"metadata cap", `{"file_path":"a","file_size":20971521}`, 200, "", true},
		{"missing path", `{}`, 200, "", true},
		{"absolute path", `{"file_path":"/a"}`, 200, "", true},
		{"traversal", `{"file_path":"a/../b"}`, 200, "", true},
		{"doubled separator", `{"file_path":"a//b"}`, 200, "", true},
		{"hidden traversal", `{"file_path":"..file"}`, 200, "", true},
		{"missing file", `{"file_path":"a"}`, 404, "", true},
		{"stream cap", `{"file_path":"a"}`, 200, strings.Repeat("a", MaxDownloadBytes+1), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "getFile") {
					io.WriteString(w, `{"ok":true,"result":`+tc.metadata+`}`)
					return
				}
				if !strings.HasPrefix(r.URL.Path, "/file/bot1:token/") {
					t.Errorf("download path=%s", r.URL.Path)
				}
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer server.Close()
			c := New()
			c.SetBaseURL(server.URL)
			_, body, err := c.Download(t.Context(), "1:token", "a")
			if (err != nil) != tc.wantError {
				t.Fatalf("err=%v", err)
			}
			if !tc.wantError && string(body) != tc.body {
				t.Error("wrong download")
			}
		})
	}
}

func TestRequestAndDownloadFailures(t *testing.T) {
	c := New()
	if err := c.call(t.Context(), "token", "getMe", func() {}, nil); err == nil {
		t.Fatal("unencodable payload accepted")
	}
	c.SetBaseURL(":invalid")
	if _, err := c.GetMe(t.Context(), "token"); err == nil {
		t.Fatal("invalid URL accepted")
	}
	c.SetBaseURL("https://example.test")
	for _, body := range []string{"not JSON", `{"ok":false,"description":"token rejected"}`, `{"ok":true,"result":true}`} {
		c.http = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
		_, err := c.GetMe(t.Context(), "token")
		if err == nil || strings.Contains(err.Error(), "token") {
			t.Fatalf("unsafe error: %v", err)
		}
	}
	c.http = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("token unavailable") })}
	if _, err := c.GetMe(t.Context(), "token"); err == nil || strings.Contains(err.Error(), "token") {
		t.Fatalf("unsafe transport error: %v", err)
	}
	if _, _, err := c.Download(t.Context(), "token", "a"); err == nil {
		t.Fatal("download ignored getFile failure")
	}
	for _, mode := range []string{"url", "transport", "read"} {
		t.Run(mode, func(t *testing.T) {
			c := New()
			c.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.URL.Path, "getFile") {
					if mode == "url" {
						c.baseURL = ":invalid"
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":{"file_path":"a"}}`))}, nil
				}
				if mode == "transport" {
					return nil, errors.New("token unavailable")
				}
				return &http.Response{StatusCode: 200, Body: brokenBody{}}, nil
			})}
			if _, _, err := c.Download(t.Context(), "token", "a"); err == nil || strings.Contains(err.Error(), "token") {
				t.Fatalf("unsafe download error: %v", err)
			}
		})
	}
	if got := redactError(errors.New("token"), "token"); got.Error() != "[redacted]" {
		t.Fatal(got)
	}
}

func TestRedirectDoesNotForwardToken(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("redirect followed") }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer server.Close()
	c := New()
	c.SetBaseURL(server.URL)
	if _, err := c.GetMe(t.Context(), "token"); err == nil {
		t.Fatal("redirect accepted")
	}
}

func TestCanceledRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	if _, err := New().GetMe(ctx, "1:token"); err == nil {
		t.Fatal("expected timeout")
	}
}

func TestSendFormattedTextAndCaption(t *testing.T) {
	for _, withMedia := range []bool{false, true} {
		t.Run(strconv.FormatBool(withMedia), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if withMedia {
					if err := r.ParseMultipartForm(1024); err != nil {
						t.Fatal(err)
					}
					var reply ReplyParameters
					json.Unmarshal([]byte(r.FormValue("reply_parameters")), &reply)
					if reply.MessageID != 7 || !reply.AllowSendingWithoutReply {
						t.Fatal("reply reference lost")
					}
					if r.FormValue("business_connection_id") != "connection" || r.FormValue("message_thread_id") != "12" || r.FormValue("parse_mode") != "HTML" || r.FormValue("caption") != "<b>Bold</b>" {
						t.Fatal("caption formatting lost")
					}
				} else {
					var payload map[string]any
					json.NewDecoder(r.Body).Decode(&payload)
					reply := payload["reply_parameters"].(map[string]any)
					if reply["message_id"] != float64(7) || reply["allow_sending_without_reply"] != true {
						t.Fatal("reply reference lost")
					}
					if payload["business_connection_id"] != "connection" || payload["message_thread_id"] != float64(12) || payload["parse_mode"] != "HTML" || payload["text"] != "<b>Bold</b>" {
						t.Fatalf("payload=%v", payload)
					}
				}
				io.WriteString(w, `{"ok":true,"result":{"message_id":1}}`)
			}))
			defer server.Close()
			client := New()
			client.SetBaseURL(server.URL)
			var files attachment.Attachments
			if withMedia {
				files = attachment.Attachments{{Name: "file.txt", Content: []byte("file"), ContentType: "text/plain"}}
			}
			if _, err := client.Send(t.Context(), "123:secret", 1, "<b>Bold</b>", files, SendOptions{ParseMode: "HTML", BusinessConnectionID: "connection", ThreadID: 12, ReplyToMessageID: 7}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetProfilePhoto(t *testing.T) {
	for _, tc := range []struct {
		response, want string
		wantErr        bool
	}{
		{`{"ok":true,"result":{"photos":[[{"file_id":"small"},{"file_id":"large"}]]}}`, "large", false},
		{`{"ok":true,"result":{"photos":[]}}`, "", false},
		{`{"ok":true,"result":{"photos":[[]]}}`, "", false},
		{`{"ok":false,"error_code":401}`, "", true},
	} {
		t.Run(tc.response, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]any
				json.NewDecoder(r.Body).Decode(&payload)
				if payload["user_id"] != float64(4500000000000) || payload["limit"] != float64(1) {
					t.Fatalf("payload=%v", payload)
				}
				io.WriteString(w, tc.response)
			}))
			defer server.Close()
			client := New()
			client.SetBaseURL(server.URL)
			got, err := client.GetProfilePhoto(t.Context(), "123:secret", 4500000000000)
			if got != tc.want || (err != nil) != tc.wantErr {
				t.Fatalf("photo=%q err=%v", got, err)
			}
		})
	}
}

func TestSendPhotoDimensions(t *testing.T) {
	for _, tc := range []struct {
		width, height int
		method        string
	}{
		{6000, 5000, "sendDocument"}, {1000, 10, "sendDocument"}, {10, 1000, "sendDocument"},
		{100, 100, "sendPhoto"}, {200, 10, "sendPhoto"}, {0, 0, "sendDocument"},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/"+tc.method) {
				t.Errorf("dimensions=%dx%d method=%s", tc.width, tc.height, r.URL.Path)
			}
			io.Copy(io.Discard, r.Body)
			io.WriteString(w, `{"ok":true,"result":{"message_id":1}}`)
		}))
		c := New()
		c.SetBaseURL(server.URL)
		var buf bytes.Buffer
		if tc.width > 0 {
			if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, tc.width, tc.height))); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := c.Send(t.Context(), "token", 42, "", attachment.Attachments{{Name: "image.png", ContentType: "image/png", Content: buf.Bytes()}}, SendOptions{}); err != nil {
			t.Fatal(err)
		}
		server.Close()
	}
}
