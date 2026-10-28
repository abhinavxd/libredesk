package email

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jhillyerd/enmime/v2"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

func TestResolveReplyTo(t *testing.T) {
	tests := []struct {
		name             string
		perMessage       string
		inboxReplyTo     string
		fromEmail        string
		conversationUUID string
		plusAddressing   bool
		want             string
	}{
		{
			name:             "per-message override wins over everything",
			perMessage:       "override@example.com",
			inboxReplyTo:     "support@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			plusAddressing:   true,
			want:             "override@example.com",
		},
		{
			name:             "per-message override is passed through literally with display name",
			perMessage:       "Override Name <override@example.com>",
			inboxReplyTo:     "support@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			plusAddressing:   true,
			want:             "Override Name <override@example.com>",
		},

		{
			name:             "plus-addressing on + inbox reply_to: base taken from reply_to",
			inboxReplyTo:     "support@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			plusAddressing:   true,
			want:             "support+conv-abc-123@example.com",
		},
		{
			name:             "plus-addressing on + inbox reply_to with different local part",
			inboxReplyTo:     "replies@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-x",
			plusAddressing:   true,
			want:             "replies+conv-uuid-x@example.com",
		},
		{
			name:             "plus-addressing on + inbox reply_to with display name: strips name",
			inboxReplyTo:     "example Support <support@example.com>",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "id-1",
			plusAddressing:   true,
			want:             "support+conv-id-1@example.com",
		},
		{
			name:             "plus-addressing on + reply_to with surrounding whitespace: trimmed by parser",
			inboxReplyTo:     "  support@example.com  ",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-w",
			plusAddressing:   true,
			want:             "support+conv-uuid-w@example.com",
		},
		{
			name:             "plus-addressing on + reply_to equals From: plus-addressed on that address",
			inboxReplyTo:     "support@hedwig.example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-same",
			plusAddressing:   true,
			want:             "support+conv-uuid-same@hedwig.example.com",
		},
		{
			name:             "plus-addressing on + reply_to already contains +: concatenates (no normalization)",
			inboxReplyTo:     "support+help@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-p",
			plusAddressing:   true,
			want:             "support+help+conv-uuid-p@example.com",
		},
		{
			name:             "plus-addressing on + invalid inbox reply_to: silently falls back to From",
			inboxReplyTo:     "not-an-email",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "id-2",
			plusAddressing:   true,
			want:             "support+conv-id-2@hedwig.example.com",
		},
		{
			name:             "plus-addressing on + reply_to missing domain: falls back to From",
			inboxReplyTo:     "support@",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "id-3",
			plusAddressing:   true,
			want:             "support+conv-id-3@hedwig.example.com",
		},

		{
			name:             "plus-addressing on + no inbox reply_to: base taken from From",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "uuid-2",
			plusAddressing:   true,
			want:             "support+conv-uuid-2@hedwig.example.com",
		},
		{
			name:             "plus-addressing on + subdomain-chained From",
			fromEmail:        "support@mail.hedwig.example.com",
			conversationUUID: "uuid-sub",
			plusAddressing:   true,
			want:             "support+conv-uuid-sub@mail.hedwig.example.com",
		},
		{
			name:             "plus-addressing on + UUID with typical hyphenated format",
			fromEmail:        "support@example.com",
			conversationUUID: "550e8400-e29b-41d4-a716-446655440000",
			plusAddressing:   true,
			want:             "support+conv-550e8400-e29b-41d4-a716-446655440000@example.com",
		},

		{
			name:           "plus-addressing on + no conversation UUID + inbox reply_to: literal reply_to",
			inboxReplyTo:   "support@example.com",
			fromEmail:      "support@hedwig.example.com",
			plusAddressing: true,
			want:           "support@example.com",
		},
		{
			name:           "plus-addressing on + no conversation UUID + no inbox reply_to: empty",
			fromEmail:      "support@hedwig.example.com",
			plusAddressing: true,
			want:           "",
		},
		{
			name:           "plus-addressing on + no conversation UUID + invalid inbox reply_to: falls back to literal From",
			inboxReplyTo:   "invalid",
			fromEmail:      "support@hedwig.example.com",
			plusAddressing: true,
			want:           "support@hedwig.example.com",
		},

		{
			name:         "plus-addressing off + inbox reply_to: literal reply_to",
			inboxReplyTo: "support@example.com",
			fromEmail:    "support@hedwig.example.com",
			want:         "support@example.com",
		},
		{
			name:         "plus-addressing off + inbox reply_to with display name: stripped",
			inboxReplyTo: "example Support <support@example.com>",
			fromEmail:    "support@hedwig.example.com",
			want:         "support@example.com",
		},
		{
			name:             "plus-addressing off + inbox reply_to + conversation UUID: still literal (no plus-addressing)",
			inboxReplyTo:     "support@example.com",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			want:             "support@example.com",
		},
		{
			name:      "plus-addressing off + no inbox reply_to: empty (customer replies to From)",
			fromEmail: "support@hedwig.example.com",
			want:      "",
		},
		{
			name:             "plus-addressing off + no inbox reply_to + conversation UUID: still empty",
			fromEmail:        "support@hedwig.example.com",
			conversationUUID: "abc-123",
			want:             "",
		},
		{
			name: "everything empty: empty result",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveReplyTo(tt.perMessage, tt.inboxReplyTo, tt.fromEmail, tt.conversationUUID, tt.plusAddressing)
			if got != tt.want {
				t.Errorf("resolveReplyTo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPlusAddress(t *testing.T) {
	tests := []struct {
		name             string
		email            string
		conversationUUID string
		want             string
	}{
		{
			name:             "standard address",
			email:            "support@example.com",
			conversationUUID: "abc-123",
			want:             "support+conv-abc-123@example.com",
		},
		{
			name:             "subdomain preserved",
			email:            "support@hedwig.example.com",
			conversationUUID: "id-1",
			want:             "support+conv-id-1@hedwig.example.com",
		},
		{
			name:             "no @ in input: returned unchanged",
			email:            "not-an-email",
			conversationUUID: "abc",
			want:             "not-an-email",
		},
		{
			name:             "empty email: returned unchanged",
			email:            "",
			conversationUUID: "abc",
			want:             "",
		},
		{
			name:             "empty UUID still produces a valid-shaped address",
			email:            "support@example.com",
			conversationUUID: "",
			want:             "support+conv-@example.com",
		},
		{
			name:             "multiple @ in input: splits only on first",
			email:            "weird@part@example.com",
			conversationUUID: "abc",
			want:             "weird+conv-abc@part@example.com",
		},
		{
			name:             "uuid containing plus: passed through verbatim",
			email:            "support@example.com",
			conversationUUID: "uuid+tag",
			want:             "support+conv-uuid+tag@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPlusAddress(tt.email, tt.conversationUUID)
			if got != tt.want {
				t.Errorf("buildPlusAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveReplyTo_PlusAddressIsRoundTrippable(t *testing.T) {
	const uuid = "550e8400-e29b-41d4-a716-446655440000"

	cases := []struct {
		name         string
		inboxReplyTo string
		fromEmail    string
	}{
		{"reply_to set", "support@example.com", "support@hedwig.example.com"},
		{"reply_to with display name", "example Support <support@example.com>", "support@hedwig.example.com"},
		{"no reply_to", "", "support@hedwig.example.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveReplyTo("", tc.inboxReplyTo, tc.fromEmail, uuid, true)
			if !strings.Contains(got, "+conv-") {
				t.Errorf("expected plus-addressed reply-to, got %q", got)
			}
			if extracted := stringutil.ExtractConvUUID(got); extracted != uuid {
				t.Errorf("UUID round-trip failed: got %q from %q, want %q", extracted, got, uuid)
			}
		})
	}
}

func TestAliasReplyRouting(t *testing.T) {
	for _, tt := range []struct {
		name        string
		from        string
		override    string
		want        string
		replyTo     string
		disablePlus bool
	}{
		{name: "alias", from: "billing@example.com", want: "support+conv-550e8400-e29b-41d4-a716-446655440000@example.com"},
		{name: "primary", from: "support@example.com", want: "support+conv-550e8400-e29b-41d4-a716-446655440000@example.com"},
		{name: "explicit override", from: "billing@example.com", override: "replies@example.com", want: "replies@example.com"},
		{name: "alias with receiving mailbox", from: "Billing <billing@example.com>", replyTo: "Replies <replies@example.net>", want: "replies+conv-550e8400-e29b-41d4-a716-446655440000@example.net"},
		{name: "alias without plus addressing", from: "billing@example.com", replyTo: "replies@example.net", disablePlus: true},
		{name: "alias without plus addressing or reply-to", from: "billing@example.com", disablePlus: true},
		{name: "primary with reply-to", from: "support@example.com", replyTo: "replies@example.net", want: "replies+conv-550e8400-e29b-41d4-a716-446655440000@example.net"},
		{name: "primary with reply-to without plus addressing", from: "support@example.com", replyTo: "replies@example.net", disablePlus: true, want: "replies@example.net"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			message := captureSMTPMessage(t, func(e *Email) error {
				e.replyTo = tt.replyTo
				e.enablePlusAddressing = !tt.disablePlus
				return e.Send(models.OutboundMessage{
					From: tt.from, To: []string{"customer@example.net"}, ReplyTo: tt.override,
					ConversationUUID: "550e8400-e29b-41d4-a716-446655440000", Content: "Reply",
				})
			})
			require.Equal(t, tt.want, message.GetHeader("Reply-To"))
		})
	}
}

func TestAliasVerificationMessageFormat(t *testing.T) {
	message := captureSMTPMessage(t, func(e *Email) error {
		return e.StartAliasVerification("billing@example.com", "test-token")
	})
	require.Contains(t, message.GetHeader("Content-Type"), "text/plain")
	require.Equal(t, "<alias-verification-test-token@example.com>", message.GetHeader("Message-ID"))
	require.Equal(t, "test-token", message.GetHeader(headerAliasVerification))
	require.Equal(t, "<billing@example.com>", message.GetHeader("From"))
	require.Equal(t, "<support@example.com>", message.GetHeader("To"))
}

func TestAliasVerificationReceivingAddress(t *testing.T) {
	for _, replyTo := range []string{"replies@example.net", "Replies <replies@example.net>"} {
		t.Run(replyTo, func(t *testing.T) {
			message := captureSMTPMessage(t, func(e *Email) error {
				e.replyTo = replyTo
				return e.StartAliasVerification("billing@example.com", "test-token")
			})
			addresses, err := message.AddressList("To")
			require.NoError(t, err)
			require.Len(t, addresses, 1)
			require.Equal(t, "replies@example.net", addresses[0].Address)
		})
	}
}

func TestSMTPTransportAndLongReferences(t *testing.T) {
	for _, tlsMode := range []string{"none", "tls", "starttls"} {
		for _, count := range []int{15, 20} {
			for _, idBytes := range []int{66, 90, 130} {
				for _, subject := range []string{"Reply", "é" + strings.Repeat("a", 50)} {
					name := fmt.Sprintf("tls=%s/count=%d/bytes=%d/subject=%q", tlsMode, count, idBytes, subject)
					t.Run(name, func(t *testing.T) {
						references := make([]string, count)
						for i := range count {
							references[i] = fmt.Sprintf("%03d", i) + strings.Repeat("a", idBytes-15) + "@example.com"
						}
						message := captureSMTPMessageWithTLS(t, tlsMode, func(e *Email) error {
							return e.Send(models.OutboundMessage{
								From: "support@example.com", To: []string{"customer@example.net"},
								Subject: subject, References: references, InReplyTo: references[count-1],
								ContentType: "plain", Content: "Reply body\n",
							})
						})
						require.Equal(t, "<"+strings.Join(references, "> <")+">", message.GetHeader(headerReferences))
						require.Equal(t, "<"+references[count-1]+">", message.GetHeader(headerInReplyTo))
						require.Equal(t, subject, message.GetHeader("Subject"))
						require.Equal(t, "Reply body\n", message.Text)
					})
				}
			}
		}
	}
}

func TestNewSmtpPoolDefaultRetries(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	go func() {
		for attempt := range 2 {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.SetDeadline(time.Now().Add(5 * time.Second))
			protocol := textproto.NewConn(conn)
			if attempt == 0 {
				protocol.PrintfLine("421 temporarily unavailable")
				conn.Close()
				continue
			}
			defer conn.Close()
			protocol.PrintfLine("220 localhost SMTP")
			for {
				line, err := protocol.ReadLine()
				if err != nil {
					return
				}
				if line == "DATA" {
					protocol.PrintfLine("354 send message")
					if _, err := protocol.ReadDotBytes(); err != nil {
						return
					}
				}
				protocol.PrintfLine("250 ok")
			}
		}
	}()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	pools, err := NewSmtpPool([]imodels.SMTPConfig{{Host: host, Port: port, TLSType: "none", MaxConns: 1}}, nil /* oauth */)
	require.NoError(t, err)
	logger := logf.New(logf.Opts{Writer: io.Discard})
	e := &Email{smtpPools: pools, lo: &logger}
	t.Cleanup(func() { e.Close() })
	require.NoError(t, e.Send(models.OutboundMessage{
		From: "support@example.com", To: []string{"customer@example.net"}, ContentType: "plain", Content: "Reply\n",
	}))
}

func captureSMTPMessage(t *testing.T, send func(*Email) error) *enmime.Envelope {
	t.Helper()
	return captureSMTPMessageWithTLS(t, "none" /* tlsMode */, send)
}

func captureSMTPMessageWithTLS(t *testing.T, tlsMode string, send func(*Email) error) *enmime.Envelope {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var tlsConfig *tls.Config
	if tlsMode != "none" {
		certificateServer := httptest.NewTLSServer(http.NotFoundHandler())
		tlsConfig = certificateServer.TLS.Clone()
		certificateServer.Close()
		if tlsMode == "tls" {
			listener = tls.NewListener(listener, tlsConfig)
		}
	}
	t.Cleanup(func() { listener.Close() })
	messages := make(chan []byte, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		protocol := textproto.NewConn(conn)
		protocol.PrintfLine("220 localhost SMTP")
		for {
			line, err := protocol.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				if tlsMode == "starttls" {
					protocol.PrintfLine("250-localhost\r\n250 STARTTLS")
				} else {
					protocol.PrintfLine("250 localhost")
				}
			case line == "STARTTLS" && tlsMode == "starttls":
				protocol.PrintfLine("220 ready for TLS")
				conn = tls.Server(conn, tlsConfig)
				protocol = textproto.NewConn(conn)
			case line == "DATA":
				if tlsMode != "none" {
					if _, secure := conn.(*tls.Conn); !secure {
						protocol.PrintfLine("530 TLS required")
						return
					}
				}
				protocol.PrintfLine("354 send message")
				body, err := protocol.ReadDotBytes()
				if err != nil {
					return
				}
				protocol.PrintfLine("250 accepted")
				messages <- body
			case line == "QUIT":
				protocol.PrintfLine("221 bye")
				return
			default:
				protocol.PrintfLine("250 ok")
			}
		}
	}()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	pools, err := NewSmtpPool([]imodels.SMTPConfig{{
		Host: host, Port: port, TLSType: tlsMode, TLSSkipVerify: tlsMode != "none", MaxConns: 1, MaxMessageRetries: 1,
	}}, nil /* oauth */)
	require.NoError(t, err)
	logger := logf.New(logf.Opts{Writer: io.Discard})
	e := &Email{
		primary: "support@example.com", from: "support@example.com",
		smtpPools: pools, lo: &logger, enablePlusAddressing: true,
	}
	t.Cleanup(func() { e.Close() })
	require.NoError(t, send(e))
	select {
	case raw := <-messages:
		header, _, ok := strings.Cut(string(raw), "\n\n")
		require.True(t, ok)
		for line := range strings.SplitSeq(header, "\n") {
			require.LessOrEqual(t, len(line), 998)
			require.NotEmpty(t, strings.TrimSpace(line))
			if strings.Contains(line, "=?UTF-8?") {
				require.LessOrEqual(t, len(line), 76)
			}
		}
		message, err := enmime.ReadEnvelope(strings.NewReader(string(raw)))
		require.NoError(t, err)
		return message
	case <-t.Context().Done():
		t.Fatal("no SMTP message received")
		return nil
	}
}
