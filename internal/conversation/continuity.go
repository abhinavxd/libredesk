package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"slices"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/volatiletech/null/v9"
)

// RunContinuity starts a goroutine that sends continuity emails containing unread outgoing messages to contacts who have been offline for a configured duration.
func (m *Manager) RunContinuity(ctx context.Context) {
	m.lo.Info("starting conversation continuity processor", "check_interval", m.continuityConfig.BatchCheckInterval)

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

// processContinuityEmails finds offline livechat conversations and sends batched unread messages emails to contacts
func (m *Manager) processContinuityEmails() error {
	var (
		offlineThresholdMinutes = int(m.continuityConfig.OfflineThreshold.Minutes())
		minEmailIntervalMinutes = int(m.continuityConfig.MinEmailInterval.Minutes())
		maxMessagesPerEmail     = m.continuityConfig.MaxMessagesPerEmail
		conversations           []models.ContinuityConversation
	)

	m.lo.Debug("fetching offline conversations for continuity emails", "offline_threshold_minutes", offlineThresholdMinutes, "min_email_interval_minutes", minEmailIntervalMinutes)

	if err := m.q.GetOfflineLiveChatConversations.Select(&conversations, offlineThresholdMinutes, minEmailIntervalMinutes); err != nil {
		return fmt.Errorf("error fetching offline conversations: %w", err)
	}

	m.lo.Debug("fetched offline conversations for continuity emails", "count", len(conversations))

	for _, conv := range conversations {
		m.lo.Info("sending continuity email for conversation", "conversation_uuid", conv.UUID, "contact_email", conv.ContactEmail)
		if err := m.sendContinuityEmail(conv, maxMessagesPerEmail); err != nil {
			m.lo.Error("error sending continuity email", "conversation_uuid", conv.UUID, "error", err)
			continue
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
		m.lo.Debug("no contact email for conversation, skipping continuity email", "conversation_uuid", conv.UUID)
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
		return fmt.Errorf("error fetching unread messages: %w", err)
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
		return fmt.Errorf("error fetching linked email inbox: %w", err)
	}

	// Build email content with all unread messages
	emailContent := m.buildContinuityEmailContent(unreadMessages)

	// Collect attachments from all unread messages
	attachments, err := m.collectAttachmentsFromMessages(unreadMessages)
	if err != nil {
		m.lo.Error("error collecting attachments from messages", "conversation_uuid", conv.UUID, "error", err)
		return fmt.Errorf("error collecting attachments for continuity email: %w", err)
	}

	// Generate email subject with site name, this subject is translated
	siteName := "Support"
	if siteNameJSON, err := m.settingsStore.Get("app.site_name"); err == nil {
		siteName = strings.Trim(strings.TrimSpace(string(siteNameJSON)), "\"")
	}
	emailSubject := m.i18n.Ts("admin.inbox.livechat.continuityEmailSubject", "site_name", siteName)

	// Generate unique Message-ID for threading
	sourceID, err := stringutil.GenerateEmailMessageID(conv.UUID, linkedEmailInbox.FromAddress())
	if err != nil {
		return fmt.Errorf("error generating message ID: %w", err)
	}

	// Get system user for sending the email
	systemUser, err := m.userStore.GetSystemUser()
	if err != nil {
		return fmt.Errorf("error fetching system user: %w", err)
	}

	metaJSON, err := json.Marshal(map[string]any{
		"continuity_email": true,
	})
	if err != nil {
		m.lo.Error("error marshalling continuity email meta", "error", err, "conversation_uuid", conv.UUID)
		return fmt.Errorf("error marshalling continuity email meta: %w", err)
	}

	// Create message for sending
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
		Attachments:       attachments,
	}

	// Set Reply-To header for conversation continuity
	emailAddress, err := stringutil.ExtractEmail(linkedEmailInbox.FromAddress())
	if err == nil {
		emailUserPart := strings.Split(emailAddress, "@")
		if len(emailUserPart) == 2 {
			message.Headers = map[string][]string{
				"Reply-To": {fmt.Sprintf("%s+%s@%s", emailUserPart[0], conv.UUID, emailUserPart[1])},
			}
		}
	}

	// Insert message into database
	if err := m.InsertMessage(&message); err != nil {
		return fmt.Errorf("error inserting continuity message: %w", err)
	}

	// Get all message source IDs for References header and threading
	references, err := m.GetMessageSourceIDs(conv.ID, 100)
	if err != nil {
		m.lo.Error("error fetching conversation source IDs for continuity email", "error", err)
		references = []string{}
	}

	// References is sorted in DESC i.e newest message first, so reverse it to keep the references in order.
	slices.Reverse(references)

	// Filter out livechat references (ones without @) and keep only the last 20
	var filteredReferences []string
	for _, ref := range references {
		if strings.Contains(ref, "@") {
			filteredReferences = append(filteredReferences, ref)
			// Keep only the last 20 references, remove the first one if exceeding
			if len(filteredReferences) > 20 {
				filteredReferences = filteredReferences[1:]
			}
		}
	}
	message.References = filteredReferences

	// Set In-Reply-To if we have references
	if len(filteredReferences) > 0 {
		message.InReplyTo = filteredReferences[len(filteredReferences)-1]
	}

	// Render message template
	if err := m.RenderMessageInTemplate(linkedEmailInbox.Channel(), &message); err != nil {
		// Clean up the inserted message on failure
		cleanUp = true
		m.lo.Error("error rendering email template for continuity email", "error", err, "message_id", message.ID, "message_uuid", message.UUID, "conversation_uuid", conv.UUID)
		return fmt.Errorf("error rendering email template: %w", err)
	}

	// Send the email
	if err := linkedEmailInbox.Send(message); err != nil {
		// Clean up the inserted message on failure
		cleanUp = true
		m.lo.Error("error sending continuity email", "error", err, "message_id", message.ID, "message_uuid", message.UUID, "conversation_uuid", conv.UUID)
		return fmt.Errorf("error sending continuity email: %w", err)
	}

	// Mark in DB that continuity email was sent now
	if _, err := m.q.UpdateContinuityEmailTracking.Exec(conv.ID); err != nil {
		m.lo.Error("error updating continuity email tracking", "conversation_uuid", conv.UUID, "error", err)
		return fmt.Errorf("error updating continuity email tracking: %w", err)
	}

	m.lo.Info("sent conversation continuity email",
		"conversation_uuid", conv.UUID,
		"contact_email", conv.ContactEmail,
		"message_count", len(unreadMessages),
		"linked_email_inbox_id", conv.LinkedEmailInboxID.Int)

	return nil
}

// buildContinuityEmailContent creates email content with conversation summary and unread messages
func (m *Manager) buildContinuityEmailContent(unreadMessages []models.ContinuityUnreadMessage) string {
	var content strings.Builder

	for _, msg := range unreadMessages {
		// Get sender display name
		senderName := "Agent"
		if msg.SenderFirstName.Valid || msg.SenderLastName.Valid {
			firstName := strings.TrimSpace(msg.SenderFirstName.String)
			lastName := strings.TrimSpace(msg.SenderLastName.String)
			fullName := strings.TrimSpace(firstName + " " + lastName)
			if fullName != "" {
				senderName = fullName
			}
		}

		// Format timestamp
		timestamp := msg.CreatedAt.Format("Mon, Jan 2, 2006 at 3:04 PM")

		// Add message header with agent name and timestamp
		content.WriteString(fmt.Sprintf("<p><strong>%s</strong> <em>%s</em></p>\n",
			html.EscapeString(senderName),
			html.EscapeString(timestamp)))

		// Add message content
		content.WriteString(msg.Content)
		content.WriteString("\n<br/>\n")
	}

	// Add footer with reply instructions, footer is translated
	content.WriteString("<hr/>\n")
	content.WriteString(fmt.Sprintf("<p><em>%s</em></p>\n", html.EscapeString(m.i18n.T("admin.inbox.livechat.continuityEmailFooter"))))

	return content.String()
}

// collectAttachmentsFromMessages collects all attachments from unread messages for the continuity email
func (m *Manager) collectAttachmentsFromMessages(unreadMessages []models.ContinuityUnreadMessage) (attachment.Attachments, error) {
	var allAttachments attachment.Attachments

	for _, msg := range unreadMessages {
		msgAttachments, err := m.fetchMessageAttachments(msg.ID)
		if err != nil {
			m.lo.Error("error fetching attachments for message", "error", err, "message_id", msg.ID)
			continue
		}
		allAttachments = append(allAttachments, msgAttachments...)
	}

	return allAttachments, nil
}
