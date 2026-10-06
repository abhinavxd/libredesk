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
	"slices"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/attachment"
)

func TestSendAlbums(t *testing.T) {
	var photo bytes.Buffer
	if err := png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name        string
		types, want []string
	}{
		{"photos", []string{"image/png", "image/png"}, []string{"photo", "photo"}},
		{"mixed visual", []string{"image/png", "video/mp4"}, []string{"photo", "video"}},
		{"audio", []string{"audio/mpeg", "audio/mp4"}, []string{"audio", "audio"}},
		{"documents", []string{"text/plain", "application/pdf"}, []string{"document", "document"}},
		{"incompatible media", []string{"image/png", "audio/mpeg"}, []string{"document", "document"}},
		{"voice", []string{"audio/ogg", "audio/ogg"}, []string{"document", "document"}},
		{"animation", []string{"image/gif", "image/gif"}, []string{"document", "document"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/botsecret/sendMediaGroup" {
					t.Errorf("path=%s", r.URL.Path)
				}
				if err := r.ParseMultipartForm(1024 * 1024); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				if r.FormValue("chat_id") != "123" || r.FormValue("business_connection_id") != "business" || r.FormValue("message_thread_id") != "8" {
					t.Errorf("routing=%v", r.Form)
				}
				var quote ReplyParameters
				if err := json.Unmarshal([]byte(r.FormValue("reply_parameters")), &quote); err != nil || quote.MessageID != 7 || !quote.AllowSendingWithoutReply {
					t.Errorf("quote=%+v err=%v", quote, err)
				}
				var media []InputMedia
				if err := json.Unmarshal([]byte(r.FormValue("media")), &media); err != nil {
					t.Error(err)
					return
				}
				if len(media) != 2 || media[0].Caption != "<b>Album</b>" || media[0].ParseMode != "HTML" || media[1].Caption != "" {
					t.Errorf("media=%+v", media)
					return
				}
				for i, item := range media {
					if item.Type != tc.want[i] {
						t.Errorf("type=%s want=%s", item.Type, tc.want[i])
					}
					part, _, err := r.FormFile(item.Media[len("attach://"):])
					if err != nil {
						t.Error(err)
						continue
					}
					data, _ := io.ReadAll(part)
					part.Close()
					if !bytes.Equal(data, photo.Bytes()) {
						t.Error("attachment changed")
					}
				}
				io.WriteString(w, `{"ok":true,"result":[{"message_id":8},{"message_id":9}]}`)
			}))
			defer server.Close()
			c := New()
			c.SetBaseURL(server.URL)
			files := attachment.Attachments{}
			for _, mime := range tc.types {
				files = append(files, attachment.Attachment{Name: "file", ContentType: mime, Content: photo.Bytes()})
			}
			got, err := c.SendAlbum(t.Context(), "secret", 123, "<b>Album</b>", files, SendOptions{BusinessConnectionID: "business", ThreadID: 8, ReplyToMessageID: 7, ParseMode: "HTML"})
			if err != nil || len(got) != 2 || got[0].ID != 8 || got[1].ID != 9 {
				t.Fatalf("messages=%v err=%v", got, err)
			}
		})
	}
}

func TestAlbumLimitsAndRetries(t *testing.T) {
	c := New()
	for _, files := range []attachment.Attachments{nil, {{}}, make(attachment.Attachments, 11), {{Content: make([]byte, MaxUploadBytes+1)}, {}}} {
		if _, err := c.SendAlbum(t.Context(), "secret", 1, "", files, SendOptions{}); err == nil {
			t.Fatal("invalid album accepted")
		}
	}
	if _, err := c.SendAlbum(t.Context(), "secret", 1, "", attachment.Attachments{{}, {}}, SendOptions{Buttons: []Button{{Text: "Yes", Data: "yes"}}}); err == nil {
		t.Fatal("album keyboard accepted")
	}
	for _, tc := range []struct {
		name      string
		fail      int
		cancel    bool
		wantCalls int
	}{
		{"retry then succeed", 1, false, 2}, {"retry exhausted", 3, false, 3}, {"canceled retry", 1, true, 1}, {"success", 0, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls <= tc.fail {
					w.WriteHeader(429)
					io.WriteString(w, `{"ok":false,"error_code":429,"parameters":{"retry_after":1}}`)
					return
				}
				io.WriteString(w, `{"ok":true,"result":[{"message_id":1},{"message_id":2}]}`)
			}))
			defer server.Close()
			c.SetBaseURL(server.URL)
			c.http = New().http
			if tc.cancel {
				c.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					cancel()
					return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader(`{"ok":false,"error_code":429}`)), Header: make(http.Header)}, nil
				})
			}
			_, err := c.SendAlbum(ctx, "secret", 1, "", attachment.Attachments{{}, {}}, SendOptions{})
			if calls != tc.wantCalls || (err != nil) != (tc.cancel || tc.fail == 3) {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			if tc.cancel && !errors.Is(err, context.Canceled) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestInlineButtons(t *testing.T) {
	valid := []Button{{Text: "Yes", Data: "yes"}, {Text: "Docs", URL: "https://example.com"}, {Text: "Telegram", URL: "tg://user?id=1"}}
	if err := ValidateButtons(valid); err != nil {
		t.Fatal(err)
	}
	if MakeKeyboard(nil) != nil || !slices.Equal(MakeKeyboard(valid).Rows[0], valid[:1]) {
		t.Fatal("keyboard layout")
	}
	for _, buttons := range [][]Button{make([]Button, 11), {{Text: " ", Data: "yes"}}, {{Text: string(bytes.Repeat([]byte("a"), 65)), Data: "yes"}}, {{Text: "Yes"}}, {{Text: "Yes", Data: "yes", URL: "https://example.com"}}, {{Text: "Yes", Data: string(bytes.Repeat([]byte("😀"), 17))}}, {{Text: "Bad", URL: "%"}}, {{Text: "Bad", URL: "/relative"}}, {{Text: "Bad", URL: "javascript://alert"}}} {
		if err := ValidateButtons(buttons); err == nil {
			t.Errorf("invalid buttons accepted: %v", buttons)
		}
	}
	for _, media := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var markup Keyboard
			if media {
				if err := r.ParseMultipartForm(1024); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				json.Unmarshal([]byte(r.FormValue("reply_markup")), &markup)
			} else {
				var payload struct {
					Markup Keyboard `json:"reply_markup"`
				}
				json.NewDecoder(r.Body).Decode(&payload)
				markup = payload.Markup
			}
			if len(markup.Rows) != 3 || markup.Rows[0][0].Data != "yes" || markup.Rows[1][0].URL != "https://example.com" {
				t.Errorf("markup=%v", markup)
			}
			io.WriteString(w, `{"ok":true,"result":{"message_id":1}}`)
		}))
		c := New()
		c.SetBaseURL(server.URL)
		var files attachment.Attachments
		if media {
			files = attachment.Attachments{{Name: "file.txt", Content: []byte("text"), ContentType: "text/plain"}}
		}
		if _, err := c.Send(t.Context(), "secret", 1, "Hello", files, SendOptions{Buttons: valid}); err != nil {
			t.Fatal(err)
		}
		if _, err := c.Send(t.Context(), "secret", 1, "Hello", files, SendOptions{Buttons: []Button{{Text: "invalid"}}}); err == nil {
			t.Fatal("invalid keyboard sent")
		}
		server.Close()
	}
}

func TestRatingButtons(t *testing.T) {
	buttons := RatingButtons()
	if len(buttons) != 5 || buttons[0].Data != RatingCallbackPrefix+"1" || buttons[4].Data != RatingCallbackPrefix+"5" || buttons[4].Text != "5 ★" {
		t.Fatalf("buttons=%v", buttons)
	}
	if err := ValidateButtons(buttons); err != nil {
		t.Fatal(err)
	}
}
