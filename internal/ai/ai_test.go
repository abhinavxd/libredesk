package ai

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

type timezoneSettings string

func (s timezoneSettings) GetAppTimezone() string { return string(s) }

func TestWorkspaceLocation(t *testing.T) {
	for _, tc := range []struct {
		settings settingsStore
		want     string
	}{
		{want: "UTC"},
		{settings: timezoneSettings(""), want: "UTC"},
		{settings: timezoneSettings("invalid/timezone"), want: "UTC"},
		{settings: timezoneSettings("Asia/Kolkata"), want: "Asia/Kolkata"},
	} {
		m := &Manager{settings: tc.settings}
		if got := m.WorkspaceLocation().String(); got != tc.want {
			t.Errorf("workspace timezone = %q, want %q", got, tc.want)
		}
	}
}

func TestProviderErrorKeepsTransportDetailsPrivate(t *testing.T) {
	translations := testutil.NewI18n(t)
	for _, tc := range []struct {
		name string
		err  error
		key  string
		logs bool
	}{
		{name: "canceled", err: context.Canceled, key: "ai.providerUnavailable"},
		{name: "wrapped cancellation", err: fmt.Errorf("%w: POST https://private-provider.example: %w", ErrProviderUnavailable, context.Canceled), key: "ai.providerUnavailable"},
		{name: "unavailable", err: fmt.Errorf("%w: private transport details", ErrProviderUnavailable), key: "ai.providerUnavailable", logs: true},
		{name: "timeout", err: fmt.Errorf("%w: %w", ErrProviderUnavailable, context.DeadlineExceeded), key: "ai.providerTimeout", logs: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := logf.New(logf.Opts{Writer: &logs})
			manager := &Manager{i18n: translations, lo: &logger}
			err := manager.providerError(tc.err)
			response, ok := err.(envelope.Error)
			if !ok || response.Message != translations.T(tc.key) || response.ErrorType != envelope.GeneralError {
				t.Fatalf("unexpected provider error response: %v", err)
			}
			if got := strings.Contains(logs.String(), "level=error"); got != tc.logs {
				t.Fatalf("error log present = %v, want %v: %s", got, tc.logs, logs.String())
			}
		})
	}
}

func TestCopilotMessagesRetainRecordedTime(t *testing.T) {
	db := testutil.NewDB(t, "ai_copilot_time")
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{Ctx: t.Context(), DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	var userID, inboxID, conversationID int
	if err := db.Get(&userID, `INSERT INTO users (type, first_name, last_name) VALUES ('agent', 'Agent', '') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Support', 'livechat') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&conversationID, `INSERT INTO conversations (contact_id, inbox_id, status_id)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id`, userID, inboxID); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, time.November, 1, 11, 0, 0, 0, time.UTC)
	if _, err := db.Exec(`INSERT INTO copilot_messages (conversation_id, user_id, role, content, created_at)
		VALUES ($1, $2, 'user', 'What happened yesterday?', $3), ($1, $2, 'assistant', 'Checking the conversation.', $4)`, conversationID, userID, at, at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	msgs, err := m.GetCopilotMessages(conversationID, userID, 50 /** limit **/)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 2 || !msgs[0].CreatedAt.Equal(at) || !msgs[1].CreatedAt.Equal(at.Add(time.Minute)) {
		t.Fatalf("stored timestamps lost or reordered: %#v", msgs)
	}
	keys := testutil.JSONKeys(t, msgs[0])
	if keys["created_at"] || msgs[0].Content != "What happened yesterday?" {
		t.Fatalf("AI context formatting changed the stored chat response: %#v", msgs[0])
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

func TestCustomToolReportsRequestOnlyWhenSent(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	lo := logf.New(logf.Opts{})
	var reported []string
	tctx := ToolContext{OnToolRequest: func(name, displayName string) { reported = append(reported, name+"|"+displayName) }}

	blocked := newHTTPTool(models.Tool{Name: "issue_refund", DisplayName: "Issue refund", URL: server.URL, Method: http.MethodPost, RequiresVerification: true}, "" /** encryptionKey **/, &lo, server.Client(), tctx)
	if _, err := blocked.Execute(t.Context(), `{}`); err != nil {
		t.Fatalf("blocked tool: %v", err)
	}
	if requests != 0 || len(reported) != 0 {
		t.Fatalf("unverified contact: requests=%d reported=%v, want none", requests, reported)
	}

	sent := newHTTPTool(models.Tool{Name: "issue_refund", DisplayName: "Issue refund", URL: server.URL, Method: http.MethodPost}, "" /** encryptionKey **/, &lo, server.Client(), tctx)
	if _, err := sent.Execute(t.Context(), `{}`); err != nil {
		t.Fatalf("sent tool: %v", err)
	}
	if requests != 1 || len(reported) != 1 || reported[0] != "issue_refund|Issue refund" {
		t.Fatalf("requests=%d reported=%v, want one issue_refund|Issue refund", requests, reported)
	}
}
