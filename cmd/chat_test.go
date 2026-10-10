package main

import (
	"bytes"
	"math"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestHandleChatSendMessageRejectsMoreThanFiveAttachments(t *testing.T) {
	app := testI18nApp(t)
	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}, Context: app}
	req.RequestCtx.SetUserValue("uuid", "00000000-0000-0000-0000-000000000001")
	req.RequestCtx.Request.Header.SetContentType("application/json")
	req.RequestCtx.Request.SetBodyString(`{"message":"hello","attachments":[1,2,3,4,5,6]}`)

	if err := handleChatSendMessage(req); err != nil {
		t.Fatalf("handleChatSendMessage returned error: %v", err)
	}
	if status := req.RequestCtx.Response.StatusCode(); status != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, fasthttp.StatusBadRequest)
	}
	if body := string(req.RequestCtx.Response.Body()); !strings.Contains(body, "at most 5 attachments") {
		t.Fatalf("response %q does not clearly state the attachment limit", body)
	}
}

func TestHandleWidgetMediaUploadRejectsMissingConversation(t *testing.T) {
	app := testI18nApp(t)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := form.WriteField("conversation_uuid", "   "); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}, Context: app}
	req.RequestCtx.Request.Header.SetContentType(form.FormDataContentType())
	req.RequestCtx.Request.SetBody(body.Bytes())
	if err := handleWidgetMediaUpload(req); err != nil {
		t.Fatalf("handleWidgetMediaUpload returned error: %v", err)
	}
	if status := req.RequestCtx.Response.StatusCode(); status != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, fasthttp.StatusBadRequest)
	}
	if response := string(req.RequestCtx.Response.Body()); !strings.Contains(response, "An active conversation is required") {
		t.Fatalf("response %q does not explain the missing conversation", response)
	}
}

func TestValidChatInitMessage(t *testing.T) {
	tests := []struct {
		name               string
		message            string
		attachments        int
		preChatFormEnabled bool
		want               bool
	}{
		{name: "attachment-only init", attachments: 1, want: true},
		{name: "empty init", want: false},
		{name: "empty init with pre-chat form", attachments: 1, preChatFormEnabled: true, want: false},
		{name: "text-only init with pre-chat form", message: "hello", preChatFormEnabled: true, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := validChatInitMessage(tc.message, tc.attachments, tc.preChatFormEnabled); got != tc.want {
				t.Fatalf("validChatInitMessage() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDecodeChatInitRequestWithAttachmentOnlyMultipart(t *testing.T) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("files", "attachment.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("attachment")); err != nil {
		t.Fatal(err)
	}
	if err := form.Close(); err != nil {
		t.Fatal(err)
	}

	req := &fastglue.Request{RequestCtx: &fasthttp.RequestCtx{}}
	req.RequestCtx.Request.Header.SetContentType(form.FormDataContentType())
	req.RequestCtx.Request.SetBody(body.Bytes())
	decoded, files, err := decodeChatInitRequest(req)
	if err != nil {
		t.Fatalf("decodeChatInitRequest returned error: %v", err)
	}
	if decoded.Message != "" || len(files) != 1 || !validChatInitMessage(decoded.Message, len(files), false) {
		t.Fatalf("decoded attachment-only init = (%q, %d files), want empty message with one allowed file", decoded.Message, len(files))
	}
}

func TestIsFormFieldValuePresent(t *testing.T) {
	tests := []struct {
		name  string
		field livechat.PreChatFormField
		value any
		want  bool
	}{
		{name: "text", field: livechat.PreChatFormField{Type: "text"}, value: "answer", want: true},
		{name: "blank text", field: livechat.PreChatFormField{Type: "text"}, value: "  "},
		{name: "wrong text type", field: livechat.PreChatFormField{Type: "text"}, value: []string{"answer"}},
		{name: "number", field: livechat.PreChatFormField{Type: "number"}, value: float64(0), want: true},
		{name: "invalid number", field: livechat.PreChatFormField{Type: "number"}, value: math.NaN()},
		{name: "checkbox", field: livechat.PreChatFormField{Type: "checkbox"}, value: false, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isFormFieldValuePresent(tc.field, tc.value); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInitialChatConfigDisablesHandoffOnlyForm(t *testing.T) {
	var config livechat.Config
	config.PreChatForm.Enabled = true
	config.PreChatForm.HandoffOnly = true

	initial := resolveInitialChatConfig(config, true)
	if initial.PreChatForm.Enabled {
		t.Fatal("handoff-only form remained enabled for initial chat")
	}
	if !config.PreChatForm.Enabled {
		t.Fatal("input config was mutated")
	}
}

func TestInitialChatConfigUsesAudienceSetting(t *testing.T) {
	var config livechat.Config
	config.PreChatForm.Enabled = true
	showVisitors := true
	showUsers := false
	config.PreChatForm.Visitors = &livechat.AudiencePreChatFormConfig{Enabled: &showVisitors}
	config.PreChatForm.Users = &livechat.AudiencePreChatFormConfig{Enabled: &showUsers}

	if !resolveInitialChatConfig(config, true).PreChatForm.Enabled {
		t.Fatal("visitor pre-chat form was disabled")
	}
	if resolveInitialChatConfig(config, false).PreChatForm.Enabled {
		t.Fatal("user pre-chat form remained enabled")
	}
}

func TestValidateHandoffFormUsesSubmittedContactDetails(t *testing.T) {
	var config livechat.Config
	config.PreChatForm.Enabled = true
	config.PreChatForm.Fields = []livechat.PreChatFormField{{
		Key:       "name",
		Type:      "text",
		Enabled:   true,
		Required:  true,
		IsDefault: true,
	}}

	name, _, _, _, err := validateFormData(nil, map[string]any{"name": "QA Visitor"}, config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "QA Visitor" {
		t.Fatalf("got name %q", name)
	}
}
