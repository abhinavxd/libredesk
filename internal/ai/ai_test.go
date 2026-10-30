package ai

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestProviderErrorKeepsTransportDetailsPrivate(t *testing.T) {
	translations := testutil.NewI18n(t)
	for _, tc := range []struct {
		name    string
		err     error
		message string
		logs    bool
	}{
		{name: "canceled", err: context.Canceled, message: "The AI provider could not be reached. Try again in a few minutes, or ask your administrator to check the AI settings."},
		{name: "wrapped cancellation", err: fmt.Errorf("%w: POST https://private-provider.example: %w", ErrProviderUnavailable, context.Canceled), message: "The AI provider could not be reached. Try again in a few minutes, or ask your administrator to check the AI settings."},
		{name: "unavailable", err: fmt.Errorf("%w: private transport details", ErrProviderUnavailable), message: "The AI provider could not be reached. Try again in a few minutes, or ask your administrator to check the AI settings.", logs: true},
		{name: "timeout", err: fmt.Errorf("%w: %w", ErrProviderUnavailable, context.DeadlineExceeded), message: "The AI provider took too long to respond. Try again, or with a shorter request.", logs: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := logf.New(logf.Opts{Writer: &logs})
			manager := &Manager{i18n: translations, lo: &logger}
			err := manager.providerError(tc.err)
			response, ok := err.(envelope.Error)
			if !ok || response.Message != tc.message || response.ErrorType != envelope.GeneralError {
				t.Fatalf("unexpected provider error response: %v", err)
			}
			if got := strings.Contains(logs.String(), "level=error"); got != tc.logs {
				t.Fatalf("error log present = %v, want %v: %s", got, tc.logs, logs.String())
			}
		})
	}
}

func TestStripCodeFence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "html fence unwrapped",
			in:   "```html\n<p>Hello <a href=\"https://example.com\">link</a></p>\n```",
			want: `<p>Hello <a href="https://example.com">link</a></p>`,
		},
		{
			name: "bare fence unwrapped",
			in:   "```\n<p>Hi</p>\n```",
			want: "<p>Hi</p>",
		},
		{
			name: "fence with surrounding whitespace",
			in:   "  ```html\n<p>Hi</p>\n```  ",
			want: "<p>Hi</p>",
		},
		{
			name: "no fence untouched",
			in:   "<p>Hi</p>",
			want: "<p>Hi</p>",
		},
		{
			name: "fence in the middle untouched",
			in:   "Use ```code``` like this",
			want: "Use ```code``` like this",
		},
		{
			name: "inner fences untouched",
			in:   "```html\n<p>a</p>\n```\ntext\n```html\n<p>b</p>\n```",
			want: "```html\n<p>a</p>\n```\ntext\n```html\n<p>b</p>\n```",
		},
		{
			name: "single line fence untouched",
			in:   "``````",
			want: "``````",
		},
		{
			name: "content on opening fence line untouched",
			in:   "```<p>Hello,</p>\n<p>Thanks.</p>\n```",
			want: "```<p>Hello,</p>\n<p>Thanks.</p>\n```",
		},
		{
			name: "c++ fence unwrapped",
			in:   "```c++\nint main() {}\n```",
			want: "int main() {}",
		},
		{
			name: "text/plain fence unwrapped",
			in:   "```text/plain\nHello there.\n```",
			want: "Hello there.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripCodeFence(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureHTMLFragment(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "html untouched",
			in:   "<p>Hello <a href=\"https://example.com\">link</a></p>",
			want: "<p>Hello <a href=\"https://example.com\">link</a></p>",
		},
		{
			name: "plain text escaped with line breaks",
			in:   "Hello,\nHow are you? 1 < 2",
			want: "Hello,<br>How are you? 1 &lt; 2",
		},
		{
			name: "angle-bracketed email is not html",
			in:   "Contact <customer@example.com>",
			want: "Contact &lt;customer@example.com&gt;",
		},
		{
			name: "self-closing tag untouched",
			in:   "line one<br/>line two",
			want: "line one<br/>line two",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ensureHTMLFragment(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCustomToolHTTPFailureLogsOutcome(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("private customer response"))
	}))
	defer server.Close()
	var logs bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &logs, Level: logf.DebugLevel})
	tool := newHTTPTool(models.Tool{ID: 42, Name: "lookup", URL: server.URL, Method: http.MethodGet}, "" /** encryptionKey **/, &lo, server.Client(), ToolContext{ConversationUUID: "conversation"})
	manager := &Manager{lo: &lo}
	result := manager.executeToolCall(t.Context(), map[string]Tool{"lookup": tool}, models.ToolCall{Function: models.ToolCallFunction{Name: "lookup", Arguments: `{"otp":"private-otp-value"}`}})
	if !strings.Contains(result, "503") {
		t.Fatalf("model tool response changed: %q", result)
	}
	if !strings.Contains(logs.String(), "status=503") || !strings.Contains(logs.String(), "tool_id=42") {
		t.Fatalf("missing HTTP outcome log: %s", logs.String())
	}
}
