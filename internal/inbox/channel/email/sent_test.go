package email

import (
	"bytes"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
	"github.com/jhillyerd/enmime/v2"
	"github.com/stretchr/testify/require"
	"github.com/zerodha/logf"
)

func TestPickSentMailbox(t *testing.T) {
	tests := []struct {
		name      string
		mailboxes []*imap.ListData
		want      string
	}{
		{
			name: "special-use attribute wins over the name",
			mailboxes: []*imap.ListData{
				{Mailbox: "Sent"},
				{Mailbox: "Gesendete Elemente", Attrs: []imap.MailboxAttr{imap.MailboxAttrSent}},
			},
			want: "Gesendete Elemente",
		},
		{
			name:      "name fallback ignores case",
			mailboxes: []*imap.ListData{{Mailbox: "INBOX"}, {Mailbox: "SENT"}},
			want:      "SENT",
		},
		{
			name:      "nothing found",
			mailboxes: []*imap.ListData{{Mailbox: "INBOX"}, {Mailbox: "Sent Items"}},
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, pickSentMailbox(tt.mailboxes))
		})
	}
}

func TestSendAppendsCopyToSentMailbox(t *testing.T) {
	imapCfg, client := startIMAPWithMailboxes(t, "INBOX", "Sent")
	imapCfg.AppendSent = true
	e, sent := newSentTestEmail(t, imapCfg, io.Discard)

	require.NoError(t, e.Send(models.OutboundMessage{
		From: "support@example.com", To: []string{"customer@example.net"}, BCC: []string{"audit@example.com"},
		Subject: "Re: Order", SourceID: "reply-1@example.com", ContentType: "plain", Content: "Reply body\n",
	}))
	sentRaw := <-sent

	stored := waitForMessages(t, client, "Sent", 1)[0]
	require.Contains(t, stored.Flags, imap.FlagSeen)

	sentMsg, err := enmime.ReadEnvelope(bytes.NewReader(sentRaw))
	require.NoError(t, err)
	storedMsg, err := enmime.ReadEnvelope(bytes.NewReader(stored.raw))
	require.NoError(t, err)
	require.Equal(t, "<reply-1@example.com>", storedMsg.GetHeader("Message-Id"))
	require.Equal(t, sentMsg.GetHeader("Message-Id"), storedMsg.GetHeader("Message-Id"))
	require.Equal(t, sentMsg.GetHeader("Date"), storedMsg.GetHeader("Date"))
	require.Equal(t, sentMsg.GetHeader("Subject"), storedMsg.GetHeader("Subject"))
	// The SMTP sink reads dot-encoded lines with LF endings, the stored copy keeps CRLF.
	require.Equal(t, sentMsg.Text, strings.ReplaceAll(storedMsg.Text, "\r\n", "\n"))
	date, err := storedMsg.Date()
	require.NoError(t, err)
	require.WithinDuration(t, date, stored.InternalDate, time.Second)
	// Bcc stays off the wire but is kept in the sender's own copy.
	require.Empty(t, sentMsg.GetHeader("Bcc"))
	require.Equal(t, "audit@example.com", storedMsg.GetHeader("Bcc"))
}

func TestSendAppendsWithGeneratedMessageID(t *testing.T) {
	imapCfg, client := startIMAPWithMailboxes(t, "INBOX", "Outbox")
	imapCfg.AppendSent = true
	imapCfg.SentMailbox = "Outbox"
	e, sent := newSentTestEmail(t, imapCfg, io.Discard)

	require.NoError(t, e.Send(models.OutboundMessage{
		From: "support@example.com", To: []string{"customer@example.net"}, ContentType: "plain", Content: "Hi\n",
	}))
	sentMsg, err := enmime.ReadEnvelope(bytes.NewReader(<-sent))
	require.NoError(t, err)
	storedMsg, err := enmime.ReadEnvelope(bytes.NewReader(waitForMessages(t, client, "Outbox", 1)[0].raw))
	require.NoError(t, err)
	require.NotEmpty(t, sentMsg.GetHeader("Message-Id"))
	require.Equal(t, sentMsg.GetHeader("Message-Id"), storedMsg.GetHeader("Message-Id"))
}

func TestSendDoesNotAppendWhenDisabled(t *testing.T) {
	imapCfg, client := startIMAPWithMailboxes(t, "INBOX", "Sent")
	e, sent := newSentTestEmail(t, imapCfg, io.Discard)

	require.NoError(t, e.Send(models.OutboundMessage{
		From: "support@example.com", To: []string{"customer@example.net"}, ContentType: "plain", Content: "Hi\n",
	}))
	<-sent
	time.Sleep(300 * time.Millisecond)
	require.Empty(t, fetchMessages(t, client, "Sent"))
}

func TestSendSucceedsWhenAppendFails(t *testing.T) {
	// No mailbox is named or marked as sent, so the copy cannot be stored.
	imapCfg, _ := startIMAPWithMailboxes(t, "INBOX")
	imapCfg.AppendSent = true
	logs := &syncBuffer{}
	e, sent := newSentTestEmail(t, imapCfg, logs)

	require.NoError(t, e.Send(models.OutboundMessage{
		From: "support@example.com", To: []string{"customer@example.net"}, ContentType: "plain", Content: "Hi\n",
	}))
	<-sent
	require.Eventually(t, func() bool {
		return strings.Contains(logs.String(), "error storing sent email in IMAP mailbox")
	}, 5*time.Second, 20*time.Millisecond)
	require.Contains(t, logs.String(), errNoSentMailbox.Error())
}

func TestStoredSentCopyIsNotIngested(t *testing.T) {
	imapCfg, client := startIMAPWithMailboxes(t, "INBOX", "Sent")
	imapCfg.AppendSent = true
	e, sent := newSentTestEmail(t, imapCfg, io.Discard)
	require.NoError(t, e.Send(models.OutboundMessage{
		From: "support@example.com", To: []string{"customer@example.net"}, ContentType: "plain", Content: "Hi\n",
	}))
	<-sent
	waitForMessages(t, client, "Sent", 1)

	// Pointing the inbox at the sent mailbox must skip our own copies: the inbox has no message store,
	// so any attempt to import one would panic.
	imapCfg.Mailbox = "Sent"
	require.NoError(t, e.processMailbox(t.Context(), time.Hour, imapCfg))
}

type storedMessage struct {
	*imapclient.FetchMessageBuffer
	raw []byte
}

// startIMAPWithMailboxes starts an in-memory IMAP server and returns its inbox config and a logged-in client.
func startIMAPWithMailboxes(t *testing.T, mailboxes ...string) (imodels.IMAPConfig, *imapclient.Client) {
	t.Helper()
	user := imapmemserver.NewUser("test", "secret")
	for _, name := range mailboxes {
		require.NoError(t, user.Create(name, nil /* options */))
	}
	mem := imapmemserver.New()
	mem.AddUser(user)
	server := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapUIDPlus: {}},
		InsecureAuth: true,
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { server.Close() })
	go server.Serve(listener)

	client, err := imapclient.DialInsecure(listener.Addr().String(), nil /* options */)
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })
	require.NoError(t, client.Login("test", "secret").Wait())

	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	portNum, err := strconv.Atoi(port)
	require.NoError(t, err)
	return imodels.IMAPConfig{Host: host, Port: portNum, Username: "test", Password: "secret", Mailbox: "INBOX", TLSType: "none"}, client
}

// newSentTestEmail returns an email inbox that sends through a local SMTP sink, which yields each received message.
func newSentTestEmail(t *testing.T, imapCfg imodels.IMAPConfig, logs io.Writer) (*Email, <-chan []byte) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
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
			switch line {
			case "DATA":
				protocol.PrintfLine("354 send message")
				body, err := protocol.ReadDotBytes()
				if err != nil {
					return
				}
				protocol.PrintfLine("250 accepted")
				messages <- body
			case "QUIT":
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
	pools, err := NewSmtpPool([]imodels.SMTPConfig{{Host: host, Port: port, TLSType: "none", MaxConns: 1, MaxMessageRetries: 1}}, nil /* oauth */)
	require.NoError(t, err)
	logger := logf.New(logf.Opts{Writer: logs})
	e := &Email{
		uuid: "inbox-uuid", primary: "support@example.com", from: "support@example.com",
		smtpPools: pools, imapCfg: []imodels.IMAPConfig{imapCfg}, lo: &logger,
	}
	t.Cleanup(func() { e.Close() })
	return e, messages
}

func waitForMessages(t *testing.T, client *imapclient.Client, mailbox string, n int) []storedMessage {
	t.Helper()
	var msgs []storedMessage
	require.Eventually(t, func() bool {
		msgs = fetchMessages(t, client, mailbox)
		return len(msgs) == n
	}, 5*time.Second, 20*time.Millisecond)
	return msgs
}

func fetchMessages(t *testing.T, client *imapclient.Client, mailbox string) []storedMessage {
	t.Helper()
	data, err := client.Select(mailbox, nil /* options */).Wait()
	require.NoError(t, err)
	if data.NumMessages == 0 {
		return nil
	}
	var seq imap.SeqSet
	seq.AddRange(1, data.NumMessages)
	section := &imap.FetchItemBodySection{Peek: true}
	buffers, err := client.Fetch(seq, &imap.FetchOptions{
		Flags: true, InternalDate: true, BodySection: []*imap.FetchItemBodySection{section},
	}).Collect()
	require.NoError(t, err)
	msgs := make([]storedMessage, 0, len(buffers))
	for _, buf := range buffers {
		msg := storedMessage{FetchMessageBuffer: buf}
		for _, raw := range buf.BodySection {
			msg.raw = raw
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

// syncBuffer is a bytes.Buffer that is safe to write from the send goroutine while the test reads it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
