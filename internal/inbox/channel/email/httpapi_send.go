package email

import (
	"context"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/email/httpapi"
	"github.com/abhinavxd/libredesk/internal/stringutil"
)

// httpAPISendTimeout bounds a single call to the configured HTTP API provider.
const httpAPISendTimeout = 30 * time.Second

// sendViaHTTPAPI sends an email through the configured HTTP API provider (e.g. Resend),
// mirroring the header/threading semantics of sendViaSMTP.
func (e *Email) sendViaHTTPAPI(m models.OutboundMessage) error {
	emailAddress, err := stringutil.ExtractEmail(m.From)
	if err != nil {
		e.lo.Error("failed to extract email address from the 'from' header", "error", err)
		return fmt.Errorf("failed to extract email address from 'From' header: %w", err)
	}

	headers := make(map[string]string, len(e.headers)+4)
	for key, value := range e.headers {
		headers[key] = value
	}

	// Set libredesk loop prevention header to from address.
	headers[headerLibredeskLoopPrevention] = emailAddress

	replyTo := resolveReplyTo(m.ReplyTo, e.replyTo, emailAddress, m.ConversationUUID, e.enablePlusAddressing)
	if replyTo != "" {
		e.lo.Debug("reply-to header set", "reply_to", replyTo)
	}

	if m.InReplyTo != "" {
		headers[headerInReplyTo] = "<" + m.InReplyTo + ">"
		e.lo.Debug("in-reply-to header set", "message_id", m.InReplyTo)
	}

	if m.SourceID != "" {
		headers[headerMessageID] = fmt.Sprintf("<%s>", m.SourceID)
		e.lo.Debug("message-id header set", "message_id", m.SourceID)
	}

	if len(m.References) > 0 {
		var references string
		for _, ref := range m.References {
			references += "<" + ref + "> "
		}
		headers[headerReferences] = references
		e.lo.Debug("references header set", "references", references)
	}

	if m.ConversationUUID != "" {
		headers[headerLibredeskConversationID] = m.ConversationUUID
		e.lo.Debug("conversation uuid header set", "conversation_uuid", m.ConversationUUID)
	}

	var attachments []httpapi.Attachment
	if m.Attachments != nil {
		attachments = make([]httpapi.Attachment, 0, len(m.Attachments))
		for _, file := range m.Attachments {
			attachments = append(attachments, httpapi.Attachment{
				Filename:    file.Name,
				ContentType: file.ContentType,
				Content:     append([]byte(nil), file.Content...),
			})
		}
	}

	outbound := httpapi.OutboundEmail{
		From:        m.From,
		To:          m.To,
		CC:          m.CC,
		BCC:         m.BCC,
		ReplyTo:     replyTo,
		Subject:     m.Subject,
		Headers:     headers,
		Attachments: attachments,
	}

	switch m.ContentType {
	case "plain":
		outbound.Text = m.Content
	default:
		outbound.HTML = m.Content
		if len(m.AltContent) > 0 {
			outbound.Text = m.AltContent
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpAPISendTimeout)
	defer cancel()

	_, err = e.httpProvider.Send(ctx, outbound)
	if err != nil {
		return fmt.Errorf("sending email via %s: %w", e.httpProvider.Name(), err)
	}
	return nil
}
