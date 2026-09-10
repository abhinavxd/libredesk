package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/httpapi"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/volatiletech/null/v9"
)

// ReceiveWebhook verifies an inbound-email webhook from the configured HTTP API provider and
// enqueues the message the same way IMAP polling does for the smtp_imap transport.
func (e *Email) ReceiveWebhook(ctx context.Context, headers http.Header, body []byte) error {
	if e.transport != imodels.TransportHTTPAPI || e.httpProvider == nil {
		return fmt.Errorf("inbox %d does not accept inbound webhooks", e.id)
	}

	inbound, err := e.httpProvider.VerifyAndParseWebhook(ctx, headers, body)
	if err != nil {
		return fmt.Errorf("verifying/parsing inbound webhook: %w", err)
	}

	if len(inbound.RawMessage) > 0 {
		return e.enqueueInboundFromRawMessage(inbound)
	}
	return e.enqueueInboundFromStructured(inbound)
}

// enqueueInboundFromRawMessage parses a full RFC822 message (fetched from the provider's API)
// and enqueues it, deriving the contact, subject, message id and threading from the message
// itself - the same enmime pipeline the IMAP path uses.
func (e *Email) enqueueInboundFromRawMessage(inbound httpapi.InboundEmail) error {
	envelope, err := mimeParser.ReadEnvelope(bytes.NewReader(inbound.RawMessage))
	if err != nil {
		return fmt.Errorf("parsing inbound message: %w", err)
	}
	for _, perr := range envelope.Errors {
		e.lo.Error("inbound message envelope error", "error", perr.Error(), "inbox_id", e.id)
	}

	fromRaw := envelope.GetHeader("From")
	fromAddr, err := stringutil.ExtractEmail(fromRaw)
	if err != nil || fromAddr == "" {
		e.lo.Error("dropping inbound webhook message: no parseable From", "from", fromRaw, "inbox_id", e.id)
		return nil
	}
	fromAddr = strings.ToLower(fromAddr)

	messageID := strings.Trim(strings.TrimSpace(envelope.GetHeader(headerMessageID)), "<>")
	if messageID == "" {
		e.lo.Error("dropping inbound webhook message: no Message-ID", "inbox_id", e.id)
		return nil
	}

	if skip, err := e.skipInbound(messageID, fromAddr); err != nil || skip {
		return err
	}

	firstName, lastName := stringutil.SplitName(extractDisplayName(fromRaw))
	if firstName == "" {
		firstName = localPart(fromAddr)
	}

	subject := envelope.GetHeader("Subject")
	meta, err := json.Marshal(map[string]any{
		"from":    []string{fromAddr},
		"to":      headerAddrs(envelope.GetHeader("To")),
		"cc":      headerAddrs(envelope.GetHeader("Cc")),
		"bcc":     headerAddrs(envelope.GetHeader("Bcc")),
		"subject": subject,
	})
	if err != nil {
		return fmt.Errorf("marshalling meta: %w", err)
	}

	incomingMsg := models.IncomingMessage{
		Channel: ChannelEmail,
		InboxID: e.id,
		Contact: models.IncomingContact{
			FirstName: firstName,
			LastName:  lastName,
			Email:     null.StringFrom(fromAddr),
		},
		Subject:  subject,
		SourceID: null.StringFrom(messageID),
		Meta:     meta,
	}

	populateIncomingFromEnvelope(envelope, &incomingMsg)

	// The MIME To: header can be rewritten by forwarders; fall back to the envelope recipients
	// (RCPT TO) to recover the plus-addressed conversation UUID.
	if incomingMsg.ConversationUUIDFromReplyTo == "" {
		incomingMsg.ConversationUUIDFromReplyTo = convUUIDFromRecipients(inbound.Recipients)
	}

	e.lo.Debug("enqueuing inbound webhook email message",
		"message_id", messageID, "inbox_id", e.id, "attachments", len(incomingMsg.Attachments))
	return e.messageStore.EnqueueIncoming(incomingMsg)
}

// enqueueInboundFromStructured builds an IncomingMessage from a provider that delivers parsed
// content in the webhook body rather than a raw message.
func (e *Email) enqueueInboundFromStructured(inbound httpapi.InboundEmail) error {
	fromAddr, err := stringutil.ExtractEmail(inbound.From)
	if err != nil || fromAddr == "" {
		e.lo.Error("dropping inbound webhook message: could not parse from address", "from", inbound.From, "inbox_id", e.id)
		return nil
	}
	fromAddr = strings.ToLower(fromAddr)

	if inbound.MessageID == "" {
		e.lo.Error("dropping inbound webhook message: no message id", "inbox_id", e.id)
		return nil
	}

	if skip, err := e.skipInbound(inbound.MessageID, fromAddr); err != nil || skip {
		return err
	}

	firstName, lastName := stringutil.SplitName(extractDisplayName(inbound.From))
	if firstName == "" {
		firstName = localPart(fromAddr)
	}

	toLower := lowerAll(inbound.To)
	ccLower := lowerAll(inbound.CC)
	bccLower := lowerAll(inbound.BCC)

	conversationUUID := convUUIDFromRecipients(inbound.Recipients)
	if conversationUUID == "" {
		conversationUUID = convUUIDFromRecipients(append(append([]string{}, toLower...), ccLower...))
	}

	meta, err := json.Marshal(map[string]any{
		"from":    []string{fromAddr},
		"to":      toLower,
		"cc":      ccLower,
		"bcc":     bccLower,
		"subject": inbound.Subject,
	})
	if err != nil {
		return fmt.Errorf("marshalling meta: %w", err)
	}

	incomingMsg := models.IncomingMessage{
		Channel: ChannelEmail,
		InboxID: e.id,
		Contact: models.IncomingContact{
			FirstName: stringutil.SanitizeUTF8(firstName),
			LastName:  stringutil.SanitizeUTF8(lastName),
			Email:     null.StringFrom(fromAddr),
		},
		Subject:                     stringutil.SanitizeUTF8(inbound.Subject),
		SourceID:                    null.StringFrom(inbound.MessageID),
		Meta:                        meta,
		InReplyTo:                   inbound.InReplyTo,
		References:                  inbound.References,
		ConversationUUIDFromReplyTo: conversationUUID,
	}

	if inbound.HTML != "" {
		incomingMsg.Content = inbound.HTML
		incomingMsg.ContentType = models.ContentTypeHTML
	} else {
		incomingMsg.Content = inbound.Text
		incomingMsg.ContentType = models.ContentTypeText
	}
	incomingMsg.Content = stringutil.SanitizeUTF8(incomingMsg.Content)

	for _, att := range inbound.Attachments {
		disposition := attachment.DispositionAttachment
		if att.ContentID != "" {
			disposition = attachment.DispositionInline
		}
		incomingMsg.Attachments = append(incomingMsg.Attachments, attachment.Attachment{
			Name:        att.Filename,
			Content:     att.Content,
			ContentType: att.ContentType,
			ContentID:   att.ContentID,
			Size:        len(att.Content),
			Disposition: disposition,
		})
	}

	return e.messageStore.EnqueueIncoming(incomingMsg)
}

// skipInbound reports whether an inbound message should be dropped: it's a duplicate delivery
// (webhook retry) or the sender is blocked.
func (e *Email) skipInbound(messageID, fromAddr string) (bool, error) {
	exists, err := e.messageStore.MessageExists(messageID)
	if err != nil {
		return false, fmt.Errorf("checking if message exists in DB: %w", err)
	}
	if exists {
		return true, nil
	}
	blocked, err := e.userStore.IsEmailBlocked(fromAddr)
	if err != nil {
		return false, fmt.Errorf("checking if email is blocked: %w", err)
	}
	if blocked {
		e.lo.Info("contact email is blocked, dropping inbound webhook message", "email", fromAddr)
		return true, nil
	}
	return false, nil
}

// extractDisplayName returns the display name portion of a "Name <email>" address, or the
// address itself if there's no name.
func extractDisplayName(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil || addr.Name == "" {
		return from
	}
	return addr.Name
}

// localPart returns the part of an email address before the '@'.
func localPart(email string) string {
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	return email
}

// headerAddrs parses an address-list header value into a lowercased slice of bare addresses.
func headerAddrs(header string) []string {
	list, err := mail.ParseAddressList(header)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, a := range list {
		if a.Address != "" {
			out = append(out, strings.ToLower(a.Address))
		}
	}
	return out
}

// convUUIDFromRecipients returns the first plus-addressed conversation UUID found in a list
// of recipient addresses, or "" if none carry one.
func convUUIDFromRecipients(addrs []string) string {
	for _, a := range addrs {
		if uuid := stringutil.ExtractConvUUID(a); uuid != "" {
			return uuid
		}
	}
	return ""
}

// lowerAll returns a lowercased copy of addrs, skipping empty entries.
func lowerAll(addrs []string) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if a == "" {
			continue
		}
		out = append(out, strings.ToLower(a))
	}
	return out
}
