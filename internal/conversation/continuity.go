package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

const (
	defaultOfflineThreshold    = 10 * time.Minute
	defaultMinEmailInterval    = 15 * time.Minute
	defaultMaxMessagesPerEmail = 10
)

// RunContinuity starts a goroutine that sends continuity emails containing unread outgoing messages to contacts who have been offline for a configured duration.
func (m *Manager) RunContinuity(ctx context.Context) {
	ticker := time.NewTicker(m.continuityConfig.BatchCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.processContinuityEmails(); err != nil {
				m.lo.Error("error processing continuity emails", "error", err)
			}
		}
	}
}

// processContinuityEmails finds offline livechat conversations and sends batched unread messages emails to contacts.
func (m *Manager) processContinuityEmails() error {
	inboxes, err := m.inboxStore.GetAll()
	if err != nil {
		return fmt.Errorf("fetching inboxes: %w", err)
	}

	for _, inb := range inboxes {
		if inb.Channel != "livechat" || !inb.Enabled || !inb.LinkedEmailInboxID.Valid || inb.LinkedEmailInboxID.Int == 0 {
			continue
		}

		offlineThreshold, minEmailInterval, maxMessages := m.parseContinuityConfig(inb.Config)
		offlineThresholdMinutes := int(offlineThreshold.Minutes())
		minEmailIntervalMinutes := int(minEmailInterval.Minutes())

		m.lo.Debug("fetching offline conversations for continuity emails", "inbox_id", inb.ID, "offline_threshold_minutes", offlineThresholdMinutes, "min_email_interval_minutes", minEmailIntervalMinutes, "max_messages_per_email", maxMessages)

		var conversations []models.ContinuityConversation
		if err := m.q.GetOfflineLiveChatConversations.Select(&conversations, offlineThresholdMinutes, minEmailIntervalMinutes, inb.ID); err != nil {
			m.lo.Error("error fetching offline conversations", "inbox_id", inb.ID, "error", err)
			continue
		}

		m.lo.Debug("fetched offline conversations", "inbox_id", inb.ID, "count", len(conversations))

		for _, conv := range conversations {
			m.lo.Info("sending continuity email for conversation", "conversation_uuid", conv.UUID, "contact_email", conv.ContactEmail.String)
			if err := m.sendContinuityEmail(conv, maxMessages); err != nil {
				m.lo.Error("error sending continuity email", "conversation_uuid", conv.UUID, "error", err)
				continue
			}
		}
	}

	return nil
}

// sendContinuityEmail sends a batched continuity email for a conversation
func (m *Manager) sendContinuityEmail(conv models.ContinuityConversation, maxMessages int) error {
	var (
		message models.Message
		cleanUp = false
	)

	if conv.ContactEmail.String == "" {
		m.lo.Warn("no contact email for conversation, skipping continuity email", "conversation_uuid", conv.UUID)
		return fmt.Errorf("no contact email for conversation")
	}

	// Cleanup inserted message on failure
	defer func() {
		if cleanUp {
			if _, delErr := m.q.DeleteMessage.Exec(message.ID, message.UUID); delErr != nil {
				m.lo.Error("error cleaning up failed continuity message",
					"error", delErr,
					"message_id", message.ID,
					"message_uuid", message.UUID,
					"conversation_uuid", conv.UUID)
			}
		}
	}()

	m.lo.Debug("fetching unread messages for continuity email", "conversation_uuid", conv.UUID, "contact_last_seen_at", conv.ContactLastSeenAt, "max_messages", maxMessages)

	var unreadMessages []models.ContinuityUnreadMessage
	if err := m.q.GetUnreadMessages.Select(&unreadMessages, conv.ID, conv.ContactLastSeenAt, maxMessages); err != nil {
		return fmt.Errorf("fetching unread messages: %w", err)
	}
	m.lo.Debug("fetched unread messages for continuity email", "conversation_uuid", conv.UUID, "unread_count", len(unreadMessages))

	if len(unreadMessages) == 0 {
		m.lo.Debug("no unread messages found for conversation, skipping continuity email", "conversation_uuid", conv.UUID)
		return nil
	}

	// Get linked email inbox
	if !conv.LinkedEmailInboxID.Valid {
		return fmt.Errorf("no linked email inbox configured for livechat inbox")
	}
	linkedEmailInbox, err := m.inboxStore.Get(conv.LinkedEmailInboxID.Int)
	if err != nil {
		return fmt.Errorf("fetching linked email inbox: %w", err)
	}

	// Fetch livechat inbox config for website URL
	var websiteURL string
	if livechatInbox, err := m.inboxStore.GetDBRecord(conv.InboxID); err == nil {
		var lcConfig struct {
			WebsiteURL string `json:"website_url"`
		}
		if err := json.Unmarshal(livechatInbox.Config, &lcConfig); err == nil {
			websiteURL = lcConfig.WebsiteURL
		}
	}

	// Build email content with all unread messages
	emailContent := m.buildContinuityEmailContent(unreadMessages, websiteURL)

	// Reuse saved subject for threading, or build from first message on first email
	emailSubject := conv.ContinuityEmailSubject.String
	if emailSubject == "" {
		emailSubject = m.formatRefMarker(conv.ReferenceNumber)
		if text := strings.TrimSpace(unreadMessages[0].TextContent); text != "" {
			if len(text) > 100 {
				text = text[:100] + "..."
			}
			emailSubject = fmt.Sprintf("%s - %s", text, emailSubject)
		}
	}

	// Generate unique Message-ID for threading
	sourceID, err := stringutil.GenerateEmailMessageID(conv.UUID, linkedEmailInbox.FromAddress())
	if err != nil {
		return fmt.Errorf("generating message ID: %w", err)
	}

	// Get system user for sending the email
	systemUser, err := m.userStore.GetSystemUser()
	if err != nil {
		return fmt.Errorf("fetching system user: %w", err)
	}

	messageIDs := make([]int, len(unreadMessages))
	for i, msg := range unreadMessages {
		messageIDs[i] = msg.ID
	}

	metaJSON, err := json.Marshal(map[string]any{
		"continuity_email": true,
	})
	if err != nil {
		m.lo.Error("error marshalling continuity email meta", "error", err, "conversation_uuid", conv.UUID)
		return fmt.Errorf("marshalling continuity email meta: %w", err)
	}

	message = models.Message{
		InboxID:           conv.LinkedEmailInboxID.Int,
		ConversationID:    conv.ID,
		ConversationUUID:  conv.UUID,
		SenderID:          systemUser.ID,
		Type:              models.MessageOutgoing,
		SenderType:        models.SenderTypeAgent,
		Status:            models.MessageStatusSent,
		Content:           emailContent,
		ContentType:       models.ContentTypeHTML,
		Private:           false,
		SourceID:          null.StringFrom(sourceID),
		MessageReceiverID: conv.ContactID,
		From:              linkedEmailInbox.FromAddress(),
		To:                []string{conv.ContactEmail.String},
		Subject:           emailSubject,
		Meta:              metaJSON,
	}

	// Insert message into database
	if err := m.InsertMessage(&message); err != nil {
		return fmt.Errorf("inserting continuity message: %w", err)
	}

	// Build References and In-Reply-To headers for email threading.
	references, inReplyTo := m.BuildEmailThreadingHeaders(conv.ID, sourceID)

	// Render message template
	if err := m.RenderMessageInTemplate(linkedEmailInbox.Channel(), &message); err != nil {
		// Clean up the inserted message on failure
		cleanUp = true
		m.lo.Error("error rendering email template for continuity email", "error", err, "message_id", message.ID, "message_uuid", message.UUID, "conversation_uuid", conv.UUID)
		return fmt.Errorf("rendering email template: %w", err)
	}

	// Plus-address on the inbox reply_to when set so replies route there, otherwise on From.
	replyToSource := linkedEmailInbox.ReplyToAddress()
	if replyToSource == "" {
		replyToSource = linkedEmailInbox.FromAddress()
	}
	var replyTo string
	if emailAddress, err := stringutil.ExtractEmail(replyToSource); err == nil {
		if parts := strings.SplitN(emailAddress, "@", 2); len(parts) == 2 {
			replyTo = fmt.Sprintf("%s+conv-%s@%s", parts[0], conv.UUID, parts[1])
		}
	}

	// Create OutboundMessage with all transport fields for sending
	outbound := models.OutboundMessage{
		UUID:              message.UUID,
		ConversationUUID:  conv.UUID,
		SenderID:          message.SenderID,
		MessageReceiverID: conv.ContactID,
		Content:           message.Content,
		TextContent:       message.TextContent,
		ContentType:       message.ContentType,
		From:              linkedEmailInbox.FromAddress(),
		To:                []string{conv.ContactEmail.String},
		Subject:           emailSubject,
		SourceID:          sourceID,
		References:        references,
		InReplyTo:         inReplyTo,
		ReplyTo:           replyTo,
		Meta:              message.Meta,
		CreatedAt:         message.CreatedAt,
	}

	// Send the email
	if err := linkedEmailInbox.Send(outbound); err != nil {
		// Clean up the inserted message on failure
		cleanUp = true
		m.lo.Error("error sending continuity email", "error", err, "message_id", message.ID, "message_uuid", message.UUID, "conversation_uuid", conv.UUID)
		return fmt.Errorf("sending continuity email: %w", err)
	}

	// Mark original messages as sent via continuity email.
	if _, err := m.q.MarkMessagesContinuityEmailed.Exec(pq.Array(messageIDs)); err != nil {
		m.lo.Error("error marking messages as continuity emailed", "conversation_uuid", conv.UUID, "error", err)
	}

	// Mark in DB that continuity email was sent now
	if _, err := m.q.UpdateContinuityEmailTracking.Exec(conv.ID, emailSubject); err != nil {
		m.lo.Error("error updating continuity email tracking", "conversation_uuid", conv.UUID, "error", err)
		return fmt.Errorf("updating continuity email tracking: %w", err)
	}

	m.lo.Info("sent conversation continuity email",
		"conversation_uuid", conv.UUID,
		"contact_email", conv.ContactEmail.String,
		"message_count", len(unreadMessages),
		"linked_email_inbox_id", conv.LinkedEmailInboxID.Int)

	return nil
}

// buildContinuityEmailContent creates email content with conversation summary and unread messages
func (m *Manager) buildContinuityEmailContent(unreadMessages []models.ContinuityUnreadMessage, websiteURL string) string {
	var content strings.Builder

	for i, msg := range unreadMessages {
		senderName := m.i18n.T("globals.terms.agent")
		if msg.SenderFirstName.Valid || msg.SenderLastName.Valid {
			firstName := strings.TrimSpace(msg.SenderFirstName.String)
			lastName := strings.TrimSpace(msg.SenderLastName.String)
			fullName := strings.TrimSpace(firstName + " " + lastName)
			if fullName != "" {
				senderName = fullName
			}
		}

		timestamp := msg.CreatedAt.Format("3:04 PM")
		marginTop := ""
		if i == 0 {
			marginTop = "margin-top:8px;"
		}
		fmt.Fprintf(&content, `<div style="border-left:2px solid #e0e0e0;padding-left:12px;margin-bottom:8px;%s">`+
			`<div style="font-size:12px;color:#888;margin-bottom:2px"><strong>%s</strong> · %s</div>`+
			`<div>%s</div>`,
			marginTop,
			html.EscapeString(senderName),
			html.EscapeString(timestamp),
			msg.Content)

		// Show attachment placeholders if the message had attachments. (We don't send actual attachments to continuity emails)
		if msg.AttachmentNames != "" {
			for name := range strings.SplitSeq(msg.AttachmentNames, ",") {
				name = strings.TrimSpace(name)
				if name != "" {
					fmt.Fprintf(&content, `<div style="font-size:12px;color:#888;margin-top:4px">&#128206; %s</div>`, html.EscapeString(name))
				}
			}
		}

		content.WriteString("</div>\n")
	}

	footerText := m.i18n.T("admin.inbox.livechat.continuityEmailFooter")
	if websiteURL != "" {
		footerText = m.i18n.Ts("admin.inbox.livechat.continuityEmailFooterWithLink",
			"link", fmt.Sprintf(`<a href="%s" style="color:#2563eb">`, html.EscapeString(websiteURL)),
			"endlink", "</a>")
	}
	fmt.Fprintf(&content, `<div style="border-top:1px solid #e0e0e0;margin-top:12px;padding-top:8px">`+
		`<div style="font-size:12px;color:#999">%s</div></div>`, footerText)

	return content.String()
}

// parseContinuityConfig reads per-inbox continuity settings, falling back to defaults.
func (m *Manager) parseContinuityConfig(configJSON json.RawMessage) (time.Duration, time.Duration, int) {
	var cfg struct {
		Continuity livechat.ContinuityConfig `json:"continuity"`
	}

	offlineThreshold := defaultOfflineThreshold
	minEmailInterval := defaultMinEmailInterval
	maxMessages := defaultMaxMessagesPerEmail

	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return offlineThreshold, minEmailInterval, maxMessages
	}

	if d, err := time.ParseDuration(cfg.Continuity.OfflineThreshold); err == nil && d > 0 {
		offlineThreshold = d
	}
	if d, err := time.ParseDuration(cfg.Continuity.MinEmailInterval); err == nil && d > 0 {
		minEmailInterval = d
	}
	if cfg.Continuity.MaxMessagesPerEmail > 0 {
		maxMessages = cfg.Continuity.MaxMessagesPerEmail
	}

	return offlineThreshold, minEmailInterval, maxMessages
}
