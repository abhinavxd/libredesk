package email

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	conversationmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/stretchr/testify/require"
)

func TestXOAUTH2Strings(t *testing.T) {
	want := []byte("user=shared@contoso.com\x01auth=Bearer app-token\x01\x01")

	mech, resp, err := (&xoauth2IMAPClient{username: "shared@contoso.com", token: "app-token"}).Start()
	require.NoError(t, err)
	require.Equal(t, "XOAUTH2", mech)
	require.Equal(t, want, resp)

	mech, resp, err = (&XOAuth2SMTPAuth{Username: "shared@contoso.com", Token: "app-token"}).Start(nil)
	require.NoError(t, err)
	require.Equal(t, "XOAUTH2", mech)
	require.Equal(t, want, resp)
}

// startXOAUTH2IMAPServer accepts one IMAP connection, reports the AUTHENTICATE command and rejects it.
func startXOAUTH2IMAPServer(t *testing.T) (string, int, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })
	commands := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		fmt.Fprint(conn, "* OK [CAPABILITY IMAP4rev1 SASL-IR AUTH=XOAUTH2] ready\r\n")
		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			tag, command, _ := strings.Cut(strings.TrimRight(line, "\r\n"), " ")
			if strings.HasPrefix(command, "AUTHENTICATE") {
				commands <- command
				fmt.Fprintf(conn, "%s NO AUTHENTICATE failed\r\n", tag)
				continue
			}
			fmt.Fprintf(conn, "%s OK done\r\n", tag)
		}
	}()
	host, portText, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	return host, port, commands
}

func TestAppOnlyIMAPAuthenticatesAsMailbox(t *testing.T) {
	tokens := &stubAppOnlyTokens{}
	tokens.install(t)
	tokens.set("imap-token", time.Hour, nil)
	status := &authStatusRecorder{}
	e := newAppOnlyEmail(t, nil, status)

	host, port, commands := startXOAUTH2IMAPServer(t)
	err := e.processMailbox(context.Background(), time.Hour, imodels.IMAPConfig{
		Host: host, Port: port, TLSType: "none", Username: "shared@contoso.com", Mailbox: "INBOX",
	})
	require.ErrorContains(t, err, "OAuth")
	require.Equal(t, "AUTHENTICATE XOAUTH2 "+base64.StdEncoding.EncodeToString([]byte("user=shared@contoso.com\x01auth=Bearer imap-token\x01\x01")), <-commands)
	require.False(t, status.last(), "rejected credentials are flagged")
}

func TestAppOnlyFirstSendBuildsPool(t *testing.T) {
	tokens := &stubAppOnlyTokens{}
	tokens.install(t)
	tokens.set("", 0, errors.New("network unreachable"))
	status := &authStatusRecorder{}
	host, port, auths := startAuthSMTPServer(t)
	e := newAppOnlyEmail(t, []imodels.SMTPConfig{{Host: host, Port: port, TLSType: "none", Username: "shared@contoso.com", MaxConns: 1}}, status)
	require.Empty(t, e.smtpPools)

	message := conversationmodels.OutboundMessage{From: "shared@contoso.com", To: []string{"customer@example.net"}, ContentType: "plain", Content: "Reply\n"}
	require.Error(t, e.Send(message), "no token, no send")
	require.Empty(t, e.smtpPools)

	tokens.set("smtp-token", time.Hour, nil)
	require.NoError(t, e.Send(message))
	require.Len(t, e.smtpPools, 1)
	require.Equal(t, xoauth2AuthLine("shared@contoso.com", "smtp-token"), <-auths)
	require.True(t, status.last())
	select {
	case line := <-auths:
		t.Fatalf("unexpected extra AUTH: %s", line)
	default:
	}
}
