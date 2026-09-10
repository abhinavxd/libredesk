package email

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strings"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/volatiletech/null/v9"
)

// ReceiveWebhook verifies and processes an inbound-email webhook call from the configured
// HTTP API provider, enqueuing the parsed message the same way IMAP polling does for the
// smtp_imap transport.
func (e *Email) ReceiveWebhook(headers http.Header, body []byte) error {
	if e.transport != imodels.TransportHTTPAPI || e.httpProvider == nil {
		return fmt.Errorf("inbox %d does not accept inbound webhooks", e.id)
	}

	inbound, err := e.httpProvider.VerifyAndParseWebhook(headers, body)
	if err != nil {
		return fmt.Errorf("verifying/parsing inbound webhook: %w", err)
	}

	if inbound.MessageID == "" {
		e.lo.Error("dropping inbound webhook message: no message id", "inbox_id", e.id)
		return nil
	}

	exists, err := e.messageStore.MessageExists(inbound.MessageID)
	if err != nil {
		return fmt.Errorf("checking if message exists in DB: %w", err)
	}
	if exists {
		return nil
	}

	fromAddr, err := stringutil.ExtractEmail(inbound.From)
	if err != nil || fromAddr == "" {
		e.lo.Error("dropping inbound webhook message: could not parse from address", "from", inbound.From, "inbox_id", e.id)
		return nil
	}
	fromAddr = strings.ToLower(fromAddr)

	if blocked, err := e.userStore.IsEmailBlocked(fromAddr); err != nil {
		return fmt.Errorf("checking if email is blocked: %w", err)
	} else if blocked {
		e.lo.Info("contact email is blocked, dropping incoming webhook message", "email", fromAddr)
		return nil
	}

	firstName, lastName := stringutil.SplitName(extractDisplayName(inbound.From))
	if firstName == "" {
		firstName = fromAddr
	}

	toLower := lowerAll(inbound.To)
	ccLower := lowerAll(inbound.CC)
	bccLower := lowerAll(inbound.BCC)

	var conversationUUID string
	for _, to := range append(append([]string{}, toLower...), ccLower...) {
		if uuid := stringutil.ExtractConvUUID(to); uuid != "" {
			conversationUUID = uuid
			break
		}
	}

	meta, err := json.Marshal(map[string]any{
		"from":    []string{fromAddr},
		"cc":      ccLower,
		"bcc":     bccLower,
		"to":      toLower,
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

	e.lo.Debug("enqueuing inbound webhook email message",
		"message_id", incomingMsg.SourceID.String, "inbox_id", e.id, "attachments", len(incomingMsg.Attachments))

	return e.messageStore.EnqueueIncoming(incomingMsg)
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
