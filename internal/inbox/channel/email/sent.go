package email

import (
	"errors"
	"fmt"
	"net/textproto"
	"strings"
	"time"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/emersion/go-imap/v2"
	"github.com/knadh/smtppool/v2"
)

// fallbackSentMailbox is used when the server marks no mailbox as \Sent.
const fallbackSentMailbox = "Sent"

var errNoSentMailbox = errors.New("no sent mailbox found, set one in the inbox IMAP settings")

// appendSentConfig returns the IMAP config that sent copies are stored with, if any has it enabled.
func (e *Email) appendSentConfig() (imodels.IMAPConfig, bool) {
	for _, cfg := range e.imapCfg {
		if cfg.AppendSent {
			return cfg, true
		}
	}
	return imodels.IMAPConfig{}, false
}

// appendSent stores a copy of a sent email in the IMAP sent mailbox, so the mailbox holds the agent replies too.
// Failures are only logged, the email has already been sent.
func (e *Email) appendSent(cfg imodels.IMAPConfig, email smtppool.Email, sentAt time.Time) {
	if err := e.appendToSentMailbox(cfg, email, sentAt); err != nil {
		e.lo.Error("error storing sent email in IMAP mailbox", "inbox_id", e.Identifier(), "message_id", email.Headers.Get(headerMessageID), "error", err)
	}
}

func (e *Email) appendToSentMailbox(cfg imodels.IMAPConfig, email smtppool.Email, sentAt time.Time) error {
	// Bcc is not sent as a header, but the sender's own copy keeps it, as mail clients do.
	email.Headers = cloneHeader(email.Headers)
	if len(email.Bcc) > 0 {
		email.Headers.Set("Bcc", strings.Join(email.Bcc, ", "))
	}
	raw, err := email.Bytes()
	if err != nil {
		return fmt.Errorf("building email: %w", err)
	}

	client, err := e.connectIMAP(cfg)
	if err != nil {
		return err
	}
	defer client.Logout()

	mailbox := cfg.SentMailbox
	if mailbox == "" {
		listOpts := &imap.ListOptions{}
		if client.Caps().Has(imap.CapSpecialUse) {
			listOpts.ReturnSpecialUse = true
		}
		mailboxes, err := client.List("", "*", listOpts).Collect()
		if err != nil {
			return fmt.Errorf("listing mailboxes: %w", err)
		}
		if mailbox = pickSentMailbox(mailboxes); mailbox == "" {
			return errNoSentMailbox
		}
	}

	cmd := client.Append(mailbox, int64(len(raw)), &imap.AppendOptions{Flags: []imap.Flag{imap.FlagSeen}, Time: sentAt})
	if _, err := cmd.Write(raw); err != nil {
		cmd.Close()
		return fmt.Errorf("writing email to mailbox %q: %w", mailbox, err)
	}
	if err := cmd.Close(); err != nil {
		return fmt.Errorf("writing email to mailbox %q: %w", mailbox, err)
	}
	if _, err := cmd.Wait(); err != nil {
		return fmt.Errorf("appending email to mailbox %q: %w", mailbox, err)
	}
	return nil
}

// pickSentMailbox returns the mailbox marked \Sent (RFC 6154), else one named "Sent", else "".
func pickSentMailbox(mailboxes []*imap.ListData) string {
	var named string
	for _, mb := range mailboxes {
		for _, attr := range mb.Attrs {
			if attr == imap.MailboxAttrSent {
				return mb.Mailbox
			}
		}
		if named == "" && strings.EqualFold(mb.Mailbox, fallbackSentMailbox) {
			named = mb.Mailbox
		}
	}
	return named
}

func cloneHeader(h textproto.MIMEHeader) textproto.MIMEHeader {
	c := make(textproto.MIMEHeader, len(h)+1)
	for k, v := range h {
		c[k] = append([]string(nil), v...)
	}
	return c
}

// maxConcurrentSentAppends caps the IMAP sessions one inbox opens to store sent copies, so a burst
// of sends doesn't run into the server's connection limit.
const maxConcurrentSentAppends = 2

// queueSentAppend runs fn in the background, at most maxConcurrentSentAppends at a time per inbox.
func (e *Email) queueSentAppend(fn func()) {
	e.sentAppendSlotsOnce.Do(func() { e.sentAppendSlots = make(chan struct{}, maxConcurrentSentAppends) })
	slots := e.sentAppendSlots
	go func() {
		slots <- struct{}{}
		defer func() { <-slots }()
		fn()
	}()
}
