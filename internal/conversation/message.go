package conversation

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/sla"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

const (
	maxMessagesPerPage = 500
	// Only allow visitor-to-contact upgrade within this window after the last continuity email.
	upgradeWindowTTL = 7 * 24 * time.Hour
)

// Run starts a pool of worker goroutines to handle message dispatching via inbox's channel and processes incoming messages. It scans for
// pending outgoing messages at the specified read interval and pushes them to the outgoing queue to be sent.
func (m *Manager) Run(ctx context.Context, incomingQWorkers, outgoingQWorkers, scanInterval time.Duration) {
	dbScanner := time.NewTicker(scanInterval)
	defer dbScanner.Stop()

	for range outgoingQWorkers {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.MessageSenderWorker(ctx)
		}()
	}
	for range incomingQWorkers {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.IncomingMessageWorker(ctx)
		}()
	}

	// Scan pending outgoing messages and send them.
	for {
		select {
		case <-ctx.Done():
			return
		case <-dbScanner.C:
			var (
				pendingMessages = []models.Message{}
				messageIDs      = m.getOutgoingProcessingMessageIDs()
			)

			// Get pending outgoing messages and skip the currently processing message ids.
			if err := m.q.GetOutgoingPendingMessages.Select(&pendingMessages, pq.Array(messageIDs)); err != nil {
				m.lo.Error("error fetching pending messages from db", "error", err)
				continue
			}

			// Prepare and push the message to the outgoing queue.
			for _, message := range pendingMessages {
				// Put the message ID in the processing map.
				m.outgoingProcessingMessages.Store(message.ID, message.ID)

				// Push the message to the outgoing message queue.
				m.outgoingMessageQueue <- message
			}
		}
	}
}

// Close signals the Manager to stop processing messages, closes channels,
// and waits for all worker goroutines to finish processing.
func (m *Manager) Close() {
	m.closedMu.Lock()
	defer m.closedMu.Unlock()
	m.closed = true
	close(m.outgoingMessageQueue)
	close(m.incomingMessageQueue)
	m.wg.Wait()
}

// IncomingMessageWorker processes incoming messages from the incoming message queue.
func (m *Manager) IncomingMessageWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-m.incomingMessageQueue:
			if !ok {
				return
			}
			if _, err := m.ProcessIncomingMessage(msg); err != nil {
				m.lo.Error("error processing incoming msg", "error", err)
			}
		}
	}
}

// MessageSenderWorker sends outgoing pending messages.
func (m *Manager) MessageSenderWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-m.outgoingMessageQueue:
			if !ok {
				return
			}
			m.sendOutgoingMessage(message)
		}
	}
}

// sendOutgoingMessage sends an outgoing message.
func (m *Manager) sendOutgoingMessage(message models.Message) {
	defer m.outgoingProcessingMessages.Delete(message.ID)

	// Helper function to handle errors
	handleError := func(err error, errorMsg string) bool {
		if err != nil {
			m.lo.Error(errorMsg, "error", err, "message_id", message.ID)
			m.UpdateMessageStatus(message.UUID, models.MessageStatusFailed)
			return true
		}
		return false
	}

	// Get inbox
	inb, err := m.inboxStore.Get(message.InboxID)
	if handleError(err, "error fetching inbox") {
		return
	}

	// Render content in template
	if err := m.RenderMessageInTemplate(inb.Channel(), &message); err != nil {
		handleError(err, "error rendering content in template")
		return
	}

	// Attach attachments to the message
	if err := m.attachAttachmentsToMessage(&message); err != nil {
		handleError(err, "error attaching attachments to message")
		return
	}

	// Convert to OutboundMessage for transport
	outbound := message.ToOutbound()

	// Detect forward via meta. Forwards re-route from a conversation to an
	// external recipient, and the recipient should see them as a fresh
	// thread, not as a reply to the originating customer's message.
	// Unmarshal failure is non-fatal — we just treat the message as a
	// regular reply.
	isForward := false
	if len(message.Meta) > 0 {
		var meta map[string]any
		if err := json.Unmarshal(message.Meta, &meta); err != nil {
			m.lo.Debug("send: meta unmarshal failed, treating as non-forward", "error", err, "message_id", message.ID)
		} else if _, ok := meta["forwarded_to"]; ok {
			isForward = true
		}
	}

	if inb.Channel() == inbox.ChannelEmail {
		// Set from address of the inbox
		outbound.From = inb.FromAddress()

		if isForward {
			// Strip threading headers so the forward starts a new email
			// thread for the recipient rather than chaining onto the
			// original customer conversation.
			outbound.References = nil
			outbound.InReplyTo = ""
			// Conventional "Fwd: " subject prefix (idempotent).
			if outbound.Subject != "" && !strings.HasPrefix(strings.ToLower(outbound.Subject), "fwd:") {
				outbound.Subject = "Fwd: " + outbound.Subject
			}
		} else {
			// Set "In-Reply-To" and "References" headers for email threading.
			outbound.References, outbound.InReplyTo = m.BuildEmailThreadingHeaders(message.ConversationID, outbound.SourceID)
		}
	}

	// Send message
	err = inb.Send(outbound)
	if err != nil && err != livechat.ErrClientNotConnected {
		handleError(err, "error sending message")
		return
	}

	// Update status as sent.
	m.UpdateMessageStatus(message.UUID, models.MessageStatusSent)

	// Skip system user replies since we only update timestamps and SLA for human replies.
	systemUser, err := m.userStore.GetSystemUser()
	if err != nil {
		m.lo.Error("error fetching system user", "error", err)
		return
	}
	if message.SenderID != systemUser.ID {
		conversation, err := m.GetConversation(message.ConversationID, "", "")
		if err != nil {
			m.lo.Error("error fetching conversation", "conversation_id", message.ConversationID, "error", err)
			return
		}

		now := time.Now()
		nowStr := now.Format(time.RFC3339)
		wsData := map[string]any{"last_reply_at": nowStr, "waiting_since": nil}

		var isFirstReply bool
		if err := m.q.UpdateConversationReplyTimestamps.QueryRow(message.ConversationID, now).Scan(&isFirstReply); err != nil {
			m.lo.Error("error updating conversation reply timestamps", "error", err)
		} else if isFirstReply {
			wsData["first_reply_at"] = nowStr
		}

		// Mark latest SLA event for next response as met.
		metAt, err := m.slaStore.SetLatestSLAEventMetAt(conversation.AppliedSLAID.Int, sla.MetricNextResponse)
		if err != nil && !errors.Is(err, sla.ErrLatestSLAEventNotFound) {
			m.lo.Error("error setting next response SLA event `met_at`", "conversation_id", conversation.ID, "metric", sla.MetricNextResponse, "applied_sla_id", conversation.AppliedSLAID.Int, "error", err)
		} else if !metAt.IsZero() {
			wsData["next_response_met_at"] = metAt.Format(time.RFC3339)
		}

		m.BroadcastConversationUpdate(message.ConversationUUID, wsData)

		// Evaluate automation rules for outgoing message.
		m.automation.EvaluateConversationUpdateRulesByID(message.ConversationID, "", amodels.EventConversationMessageOutgoing)
	}
}

// BuildTemplateData builds the common template data map for rendering message content variables.
func (m *Manager) BuildTemplateData(conversationUUID string, senderID int) (map[string]any, error) {
	conversation, err := m.GetConversation(0, conversationUUID, "")
	if err != nil {
		return nil, fmt.Errorf("fetching conversation: %w", err)
	}

	sender, err := m.userStore.GetAgent(senderID, "")
	if err != nil {
		return nil, fmt.Errorf("fetching message sender user: %w", err)
	}

	data := map[string]any{
		"Conversation": map[string]any{
			"ReferenceNumber": conversation.ReferenceNumber,
			"Subject":         conversation.Subject.String,
			"Priority":        conversation.Priority.String,
			"UUID":            conversation.UUID,
		},
		"Contact": map[string]any{
			"FirstName": conversation.Contact.FirstName,
			"LastName":  conversation.Contact.LastName,
			"FullName":  conversation.Contact.FullName(),
			"Email":     conversation.Contact.Email.String,
		},
		"Recipient": map[string]any{
			"FirstName": conversation.Contact.FirstName,
			"LastName":  conversation.Contact.LastName,
			"FullName":  conversation.Contact.FullName(),
			"Email":     conversation.Contact.Email.String,
		},
		"Author": map[string]any{
			"FirstName": sender.FirstName,
			"LastName":  sender.LastName,
			"FullName":  sender.FullName(),
			"Email":     sender.Email.String,
		},
	}

	// For automated replies set author fields to empty strings as the recipients will see name as System.
	if sender.IsSystemUser() {
		data["Author"] = map[string]any{
			"FirstName": "",
			"LastName":  "",
			"FullName":  "",
			"Email":     "",
		}
	}

	return data, nil
}

// RenderMessageInTemplate renders message content in the email base template for sending.
func (m *Manager) RenderMessageInTemplate(channel string, message *models.Message) error {
	switch channel {
	case inbox.ChannelEmail:
		data, err := m.BuildTemplateData(message.ConversationUUID, message.SenderID)
		if err != nil {
			return err
		}

		// Expose message meta flags to the template.
		var (
			isContinuity bool
			meta         map[string]any
		)
		if len(message.Meta) > 0 && json.Unmarshal(message.Meta, &meta) == nil {
			isContinuity, _ = meta["continuity_email"].(bool)
		}
		data["IsContinuityEmail"] = isContinuity

		message.Content, err = m.template.RenderEmailWithTemplate(data, message.Content)
		if err != nil {
			m.lo.Error("could not render email content using template", "id", message.ID, "error", err)
			return fmt.Errorf("could not render email content using template: %w", err)
		}
	case inbox.ChannelLiveChat:
		// Live chat doesn't use templates for rendering messages.
		return nil
	default:
		m.lo.Warn("unknown message channel", "channel", channel)
		return fmt.Errorf("unknown message channel: %s", channel)
	}
	return nil
}

// GetConversationMessages retrieves messages for a specific conversation.
func (m *Manager) GetConversationMessages(conversationUUID string, page, pageSize int, private *bool, msgTypes []string) ([]models.Message, int, error) {
	var (
		messages = make([]models.Message, 0)
		qArgs    []any
	)

	// Convert msgTypes slice to pq.StringArray for PostgreSQL
	var typesArg any
	if len(msgTypes) > 0 {
		typesArg = pq.StringArray(msgTypes)
	}

	qArgs = append(qArgs, conversationUUID, private, typesArg)
	query, pageSize, qArgs, err := m.generateMessagesQuery(m.q.GetMessages, qArgs, page, pageSize)
	if err != nil {
		m.lo.Error("error generating messages query", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	defer tx.Rollback()
	if err != nil {
		m.lo.Error("error preparing get messages query", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if err := tx.Select(&messages, query, qArgs...); err != nil {
		m.lo.Error("error fetching conversations", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	return messages, pageSize, nil
}

// GetMessage retrieves a message by UUID.
func (m *Manager) GetMessage(uuid string) (models.Message, error) {
	var message models.Message
	if err := m.q.GetMessage.Get(&message, uuid); err != nil {
		m.lo.Error("error fetching message", "uuid", uuid, "error", err)
		return message, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Generate signed URLs for attachments.
	for i := range message.Attachments {
		message.Attachments[i].URL = m.mediaStore.GetSignedURL(message.Attachments[i].UUID)
	}

	return message, nil
}

// UpdateMessageStatus updates the status of a message.
func (m *Manager) UpdateMessageStatus(messageUUID string, status string) error {
	if _, err := m.q.UpdateMessageStatus.Exec(status, messageUUID); err != nil {
		m.lo.Error("error updating message status", "message_uuid", messageUUID, "error", err)
		return err
	}

	// Broadcast message status update to all conversation subscribers.
	conversationUUID, _ := m.getConversationUUIDFromMessageUUID(messageUUID)
	m.BroadcastMessageUpdate(conversationUUID, messageUUID, map[string]any{"status": status})

	// Trigger webhook for message update.
	if message, err := m.GetMessage(messageUUID); err != nil {
		m.lo.Error("error fetching message for webhook event", "uuid", messageUUID, "error", err)
	} else {
		m.webhookStore.TriggerEvent(wmodels.EventMessageUpdated, message)
	}

	return nil
}

// MarkMessageAsPending updates message status to `Pending`, enqueuing it for sending.
func (m *Manager) MarkMessageAsPending(uuid string) error {
	if err := m.UpdateMessageStatus(uuid, models.MessageStatusPending); err != nil {
		m.lo.Error("error marking message as pending", "uuid", uuid, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.errorSendingMessage"), nil)
	}
	return nil
}

// SendPrivateNote inserts a private message in a conversation.
func (m *Manager) SendPrivateNote(media []mmodels.Media, senderID int, conversationUUID, content string, mentions []models.MentionInput) (models.Message, error) {
	// Best-effort render template variables before saving.
	if data, err := m.BuildTemplateData(conversationUUID, senderID); err == nil {
		content = m.template.RenderString(data, content)
	}

	message := models.Message{
		ConversationUUID: conversationUUID,
		SenderID:         senderID,
		Type:             models.MessageOutgoing,
		SenderType:       models.SenderTypeAgent,
		Status:           models.MessageStatusSent,
		Content:          content,
		ContentType:      models.ContentTypeHTML,
		Private:          true,
		Media:            media,
	}
	if err := m.InsertMessage(&message); err != nil {
		return models.Message{}, err
	}

	// Insert mentions if any.
	if len(mentions) > 0 {
		if err := m.InsertMentions(message.ConversationID, message.ID, senderID, mentions); err != nil {
			m.lo.Error("error inserting mentions", "error", err)
		}
		go m.NotifyMention(conversationUUID, message, mentions, senderID)
	}

	return message, nil
}

// CreateContactMessage creates a contact message in a conversation.
func (m *Manager) CreateContactMessage(media []mmodels.Media, contactID int, conversationUUID, content, contentType string, isNewConversation bool) (models.Message, error) {
	message := models.Message{
		ConversationUUID: conversationUUID,
		SenderID:         contactID,
		Type:             models.MessageIncoming,
		SenderType:       models.SenderTypeContact,
		Status:           models.MessageStatusReceived,
		Content:          content,
		ContentType:      contentType,
		Private:          false,
		Media:            media,
	}
	if err := m.InsertMessage(&message); err != nil {
		return models.Message{}, err
	}

	// Process post-message hooks (reopen, waiting since, automation, SLA).
	if err := m.ProcessIncomingMessageHooks(conversationUUID, isNewConversation); err != nil {
		m.lo.Error("error processing incoming message hooks", "conversation_uuid", conversationUUID, "error", err)
	}

	return message, nil
}

// QueueReply queues a reply message in a conversation.
func (m *Manager) QueueReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, to, cc, bcc []string, metaMap map[string]interface{}) (models.Message, error) {
	var (
		message = models.Message{}
	)

	inboxRecord, err := m.inboxStore.GetDBRecord(inboxID)
	if err != nil {
		m.lo.Error("error fetching inbox record", "inbox_id", inboxID, "error", err)
		return models.Message{}, err
	}

	if !inboxRecord.Enabled {
		return models.Message{}, envelope.NewError(envelope.InputError, m.i18n.T("status.disabledInbox"), nil)
	}

	var sourceID string
	switch inboxRecord.Channel {
	case inbox.ChannelEmail:
		// Add `to`, `cc`, and `bcc` recipients to meta map.
		to = stringutil.RemoveEmpty(to)
		cc = stringutil.RemoveEmpty(cc)
		bcc = stringutil.RemoveEmpty(bcc)
		if len(to) > 0 {
			metaMap["to"] = to
		}
		if len(cc) > 0 {
			metaMap["cc"] = cc
		}
		if len(bcc) > 0 {
			metaMap["bcc"] = bcc
		}
		if len(to) == 0 {
			return message, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.empty", "name", "`to`"), nil)
		}
		sourceID, err = stringutil.GenerateEmailMessageID(conversationUUID, inboxRecord.From)
		if err != nil {
			m.lo.Error("error generating source message id", "error", err)
			return models.Message{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
	}

	// Marshal meta.
	metaJSON, err := json.Marshal(metaMap)
	if err != nil {
		m.lo.Error("error marshalling message meta map to JSON", "error", err)
		return models.Message{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Best-effort render template variables before saving so agents see rendered content immediately.
	if data, err := m.BuildTemplateData(conversationUUID, senderID); err == nil {
		content = m.template.RenderString(data, content)
	}

	// Insert the message into the database
	message = models.Message{
		ConversationUUID:  conversationUUID,
		SenderID:          senderID,
		Type:              models.MessageOutgoing,
		SenderType:        models.SenderTypeAgent,
		Status:            models.MessageStatusPending,
		Content:           content,
		ContentType:       models.ContentTypeHTML,
		Private:           false,
		Media:             media,
		SourceID:          null.StringFrom(sourceID),
		MessageReceiverID: contactID,
		Meta:              metaJSON,
	}
	if err := m.InsertMessage(&message); err != nil {
		return models.Message{}, err
	}
	return message, nil
}

// InsertMessage inserts a message and attaches the media to the message.
func (m *Manager) InsertMessage(message *models.Message) error {
	if message.Private {
		message.Status = models.MessageStatusSent
	}
	if len(message.Meta) == 0 || string(message.Meta) == "null" {
		message.Meta = json.RawMessage(`{}`)
	}

	// Handle empty content type enum, default to text.
	if message.ContentType == "" {
		message.ContentType = models.ContentTypeText
	}

	// Convert content to plain text for search.
	if message.ContentType == models.ContentTypeText {
		message.TextContent = message.Content
	} else {
		message.TextContent = stringutil.HTML2Text(message.Content)
	}

	// Insert Message.
	if err := m.q.InsertMessage.Get(message, message.Type, message.Status, message.ConversationID, message.ConversationUUID, message.Content, message.TextContent, message.SenderID, message.SenderType,
		message.Private, message.ContentType, message.SourceID, message.Meta); err != nil {
		m.lo.Error("error inserting message in db", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Attach just inserted message to the media.
	for _, media := range message.Media {
		m.mediaStore.Attach(media.ID, mmodels.ModelMessages, message.ID)
	}

	// Add this user as a participant if not already present.
	m.addConversationParticipant(message.SenderID, message.ConversationUUID)

	// Skip updating last_message and broadcasting for continuity emails.
	if !message.IsContinuityMessage() {
		// Hide CSAT message content as it contains a public link to the survey.
		lastMessage := message.TextContent
		if message.HasCSAT() {
			lastMessage = "Please rate your experience with us"
		}

		// If no text content but has media, set last message preview based on media type.
		if strings.TrimSpace(lastMessage) == "" && len(message.Media) > 0 {
			lastMessage = m.getMediaPreview(message.Media[0])
		}

		// Update conversation last message details (also conditionally updates last_interaction if not activity/private).
		m.UpdateConversationLastMessage(message.ConversationID, message.ConversationUUID, lastMessage, message.SenderType, message.Type, message.Private, message.CreatedAt, message.SenderID)

		// Broadcast new message with computed preview.
		m.BroadcastNewMessage(message, lastMessage)
	}

	// Refetch the message to get all fields populated (e.g., author, media URLs).
	refetchedMessage, err := m.GetMessage(message.UUID)
	if err != nil {
		m.lo.Error("error fetching message after insert", "error", err)
	} else {
		*message = refetchedMessage
	}

	// Trigger webhook for new message created.
	m.webhookStore.TriggerEvent(wmodels.EventMessageCreated, message)

	return nil
}

// RecordAssigneeUserChange records an activity for a user assignee change.
func (m *Manager) RecordAssigneeUserChange(conversationUUID string, assigneeID int, actor umodels.User) error {
	// Self assignment.
	if assigneeID == actor.ID {
		return m.InsertConversationActivity(models.ActivitySelfAssign, conversationUUID, actor.FullName(), actor)
	}

	// Assignment to another user.
	assignee, err := m.userStore.GetAgent(assigneeID, "")
	if err != nil {
		return err
	}
	return m.InsertConversationActivity(models.ActivityAssignedUserChange, conversationUUID, assignee.FullName(), actor)
}

// RecordAssigneeTeamChange records an activity for a team assignee change.
func (m *Manager) RecordAssigneeTeamChange(conversationUUID string, teamID int, actor umodels.User) error {
	team, err := m.teamStore.Get(teamID)
	if err != nil {
		return err
	}
	return m.InsertConversationActivity(models.ActivityAssignedTeamChange, conversationUUID, team.Name, actor)
}

// RecordPriorityChange records an activity for a priority change.
func (m *Manager) RecordPriorityChange(priority, conversationUUID string, actor umodels.User) error {
	return m.InsertConversationActivity(models.ActivityPriorityChange, conversationUUID, priority, actor)
}

// RecordStatusChange records an activity for a status change.
func (m *Manager) RecordStatusChange(status, conversationUUID string, actor umodels.User) error {
	return m.InsertConversationActivity(models.ActivityStatusChange, conversationUUID, status, actor)
}

// RecordSLASet records an activity for an SLA set.
func (m *Manager) RecordSLASet(conversationUUID string, slaName string, actor umodels.User) error {
	return m.InsertConversationActivity(models.ActivitySLASet, conversationUUID, slaName, actor)
}

// RecordTagAddition records an activity for a tag addition.
func (m *Manager) RecordTagAddition(conversationUUID string, tag string, actor umodels.User) error {
	return m.InsertConversationActivity(models.ActivityTagAdded, conversationUUID, tag, actor)
}

// RecordTagRemoval records an activity for a tag removal.
func (m *Manager) RecordTagRemoval(conversationUUID string, tag string, actor umodels.User) error {
	return m.InsertConversationActivity(models.ActivityTagRemoved, conversationUUID, tag, actor)
}

// InsertConversationActivity inserts an activity message.
func (m *Manager) InsertConversationActivity(activityType, conversationUUID, newValue string, actor umodels.User) error {
	content, err := m.getMessageActivityContent(activityType, newValue, actor.FullName())
	if err != nil {
		m.lo.Error("error could not generate activity content", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	message := models.Message{
		Type:             models.MessageActivity,
		Status:           models.MessageStatusSent,
		Content:          content,
		ContentType:      models.ContentTypeText,
		ConversationUUID: conversationUUID,
		Private:          true,
		SenderID:         actor.ID,
		SenderType:       models.SenderTypeAgent,
	}

	if err := m.InsertMessage(&message); err != nil {
		m.lo.Error("error inserting activity message", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// getConversationUUIDFromMessageUUID returns conversation UUID from message UUID.
func (m *Manager) getConversationUUIDFromMessageUUID(uuid string) (string, error) {
	var conversationUUID string
	if err := m.q.GetConversationUUIDFromMessageUUID.Get(&conversationUUID, uuid); err != nil {
		m.lo.Error("error fetching conversation uuid from message uuid", "uuid", uuid, "error", err)
		return conversationUUID, err
	}
	return conversationUUID, nil
}

// getMessageActivityContent generates activity content based on the activity type.
func (m *Manager) getMessageActivityContent(activityType, newValue, actorName string) (string, error) {
	var content = ""
	switch activityType {
	case models.ActivityAssignedUserChange:
		content = fmt.Sprintf("Assigned to %s by %s", newValue, actorName)
	case models.ActivityAssignedTeamChange:
		content = fmt.Sprintf("Assigned to %s team by %s", newValue, actorName)
	case models.ActivitySelfAssign:
		content = fmt.Sprintf("%s self-assigned this conversation", actorName)
	case models.ActivityPriorityChange:
		content = fmt.Sprintf("%s set priority to %s", actorName, newValue)
	case models.ActivityStatusChange:
		content = fmt.Sprintf("%s marked the conversation as %s", actorName, newValue)
	case models.ActivityTagAdded:
		content = fmt.Sprintf("%s added tag %s", actorName, newValue)
	case models.ActivityTagRemoved:
		content = fmt.Sprintf("%s removed tag %s", actorName, newValue)
	case models.ActivitySLASet:
		content = fmt.Sprintf("%s set %s SLA policy", actorName, newValue)
	case models.ActivityParticipantAdded:
		content = fmt.Sprintf("%s joined the conversation", newValue)
	case models.ActivityMessageForwarded:
		content = fmt.Sprintf("%s forwarded the conversation to %s", actorName, newValue)
	default:
		return "", fmt.Errorf("invalid activity type %s", activityType)
	}
	return content, nil
}

// ProcessIncomingMessage handles the insertion of an incoming message and
// associated contact. It finds or creates the contact, checks for existing
// conversations, and creates a new conversation if necessary. It also
// inserts the message, uploads any attachments, and queues the conversation evaluation of automation rules.
func (m *Manager) ProcessIncomingMessage(in models.IncomingMessage) (models.Message, error) {
	// Return early if this message already exists (same source ID).
	dupConvID, err := m.messageExistsBySourceID([]string{in.SourceID.String})
	if err != nil && err != errConversationNotFound {
		return models.Message{}, err
	}
	if dupConvID > 0 {
		return models.Message{}, nil
	}

	// Resolve sender and conversation from plus addressing.
	senderID, conversationID, conversationUUID, err := m.resolveSender(&in)
	if err != nil {
		return models.Message{}, err
	}

	// Find or create contact.
	if senderID == 0 {
		user := umodels.User{
			FirstName: in.Contact.FirstName,
			LastName:  in.Contact.LastName,
			Email:     in.Contact.Email,
			Type:      umodels.UserTypeContact,
		}
		if err := m.userStore.CreateContact(&user); err != nil {
			return models.Message{}, fmt.Errorf("creating contact: %w", err)
		}
		senderID = user.ID
		in.Contact.ID = senderID
	}

	// Match conversation if not already matched by plus-addressing.
	var isNewConversation bool
	if conversationID == 0 {
		conversationID, conversationUUID, isNewConversation, err = m.findOrCreateConversation(in)
		if err != nil {
			return models.Message{}, err
		}
	}

	// For existing conversations, override sender with the conversation's contact when emails match.
	if !isNewConversation && conversationID > 0 {
		conversation, convErr := m.GetConversation(conversationID, "", "")
		if convErr == nil && strings.EqualFold(conversation.Contact.Email.String, in.Contact.Email.String) {
			senderID = conversation.ContactID
			in.Contact.ID = senderID
		}
	}

	// Convert to Message for attachment upload and insertion.
	msg := in.ToMessage(senderID, conversationID, conversationUUID)

	// Upload message attachments. On failure, delete the conversation if it was just created for this message.
	if upErr := m.uploadMessageAttachments(&msg); upErr != nil {
		m.lo.Error("error uploading message attachments", "message_source_id", in.SourceID, "error", upErr)
		if isNewConversation && conversationUUID != "" {
			m.lo.Info("deleting conversation as message attachment upload failed", "conversation_uuid", conversationUUID, "message_source_id", in.SourceID)
			if err := m.DeleteConversation(conversationUUID); err != nil {
				return models.Message{}, fmt.Errorf("deleting conversation after message attachment upload failure: %w", err)
			}
		}
		return models.Message{}, fmt.Errorf("uploading message attachments: %w", upErr)
	}

	// If this incoming message is a reply to a previously-forwarded email
	// (or chained off such a reply), tag it `meta.from_forward = true`. The
	// reply box's auto-quoted-thread builder filters tagged messages out, so
	// internal back-and-forth with a forwarded recipient never leaks into a
	// customer-facing email. Build the source-id set first so we can skip
	// the DB roundtrip when there's nothing to check.
	sourceIDs := make([]string, 0, len(in.References)+1)
	if in.InReplyTo != "" {
		sourceIDs = append(sourceIDs, in.InReplyTo)
	}
	for _, r := range in.References {
		if r != "" {
			sourceIDs = append(sourceIDs, r)
		}
	}
	if len(sourceIDs) > 0 {
		var isFromFwd bool
		if err := m.q.IsSourceIDFromForward.Get(&isFromFwd, pq.Array(sourceIDs)); err != nil {
			m.lo.Warn("failed to check from-forward provenance, leaving message untagged", "error", err)
		} else if isFromFwd {
			var meta map[string]any
			if len(msg.Meta) > 0 {
				if jerr := json.Unmarshal(msg.Meta, &meta); jerr != nil {
					m.lo.Debug("from_forward: existing meta unmarshal failed, starting fresh", "error", jerr)
				}
			}
			if meta == nil {
				meta = map[string]any{}
			}
			meta["from_forward"] = true
			if metaBytes, jerr := json.Marshal(meta); jerr == nil {
				msg.Meta = metaBytes
			} else {
				m.lo.Warn("from_forward: meta marshal failed, leaving untagged", "error", jerr)
			}
		}
	}

	// Insert message.
	if err = m.InsertMessage(&msg); err != nil {
		return models.Message{}, err
	}

	// When a customer replies to a continuity emailsync the message to their live chat widget via WebSocket.
	// No-op if the conversation's inbox isn't livechat.
	m.broadcastMessageToWidgetClients(&msg)

	// Process post-message hooks (automation rules, webhooks, SLA, etc.).
	if err := m.ProcessIncomingMessageHooks(msg.ConversationUUID, isNewConversation); err != nil {
		m.lo.Error("error processing incoming message hooks", "conversation_uuid", msg.ConversationUUID, "error", err)
		return models.Message{}, fmt.Errorf("processing incoming message hooks: %w", err)
	}
	return msg, nil
}

// resolveSender resolves the sender for an incoming message via plus-addressing.
// Returns senderID, and optionally conversationID/UUID if matched.
// If sender is not resolved here, ProcessIncomingMessage handles it with conversation context.
func (m *Manager) resolveSender(in *models.IncomingMessage) (senderID, conversationID int, conversationUUID string, err error) {
	if in.ConversationUUIDFromReplyTo != "" {
		senderID, conversationID, conversationUUID, err = m.resolveByPlusAddress(in)
		if err != nil {
			return 0, 0, "", err
		}
		if senderID > 0 {
			in.Contact.ID = senderID
		}
	}
	return senderID, conversationID, conversationUUID, nil
}

// resolveByPlusAddress attempts to match a conversation via plus-addressed Reply-To
// (e.g., inbox+conv-{uuid}@domain). If the conversation contact is a visitor, it upgrades
// them to a contact (proving email ownership). Returns senderID > 0 if resolved.
func (m *Manager) resolveByPlusAddress(in *models.IncomingMessage) (senderID, conversationID int, conversationUUID string, err error) {
	conversation, err := m.GetConversation(0, in.ConversationUUIDFromReplyTo, "")
	if err != nil {
		// Not found return with no error.
		if envErr, ok := err.(envelope.Error); ok && envErr.ErrorType == envelope.NotFoundError {
			return 0, 0, "", nil
		}
		// Other errors.
		return 0, 0, "", fmt.Errorf("fetching conversation: %w", err)
	}

	m.lo.Debug("matched conversation by plus-addressed Reply-To", "conversation_uuid", conversation.UUID, "contact_email", in.Contact.Email.String)

	conversationID = conversation.ID
	conversationUUID = conversation.UUID
	senderID = conversation.Contact.ID

	// Already a contact - if same email, return as sender. If different email, let CreateContact resolve actual sender.
	if conversation.Contact.Type == umodels.UserTypeContact {
		if !strings.EqualFold(conversation.Contact.Email.String, in.Contact.Email.String) {
			return 0, conversationID, conversationUUID, nil
		}
		return senderID, conversationID, conversationUUID, nil
	}

	// Visitor upgrade requires email match - proves email ownership.
	// If a different email replied, thread the message but don't upgrade.
	if !strings.EqualFold(conversation.Contact.Email.String, in.Contact.Email.String) {
		return 0, conversationID, conversationUUID, nil
	}

	user, contactErr := m.userStore.Get(0, in.Contact.Email.String, []string{umodels.UserTypeContact})
	if contactErr == nil {
		m.lo.Debug("a contact already exists with the same email as visitor; not upgrading visitor", "conversation_uuid", conversation.UUID, "contact_email", in.Contact.Email.String, "contact_user_id", user.ID)
		// A contact with this email already exists; don't upgrade visitor.
		// Let CreateContact resolve the correct sender ID.
		return 0, conversationID, conversationUUID, nil
	}

	if envErr, ok := contactErr.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
		return 0, 0, "", fmt.Errorf("fetching contact by email: %w", contactErr)
	}

	// Block upgrade if continuity email TTL expired.
	if !m.isVisitorUpgradeSafe(conversation) {
		return 0, conversationID, conversationUUID, nil
	}

	// Upgrade visitor as no contact exist with this email.
	if err := m.userStore.UpgradeVisitorToContact(conversation.Contact.ID); err != nil {
		return 0, 0, "", fmt.Errorf("upgrading visitor to contact: %w", err)
	}

	m.lo.Debug("upgraded visitor to contact", "conversation_uuid", conversation.UUID, "contact_id", conversation.Contact.ID)

	// Notify conversation subscribers that the contact type has changed.
	m.BroadcastContactUpdate(conversation.ContactID, map[string]any{"type": umodels.UserTypeContact})

	return senderID, conversationID, conversationUUID, nil
}

// isVisitorUpgradeSafe checks whether a visitor-to-contact upgrade should proceed.
// Blocks upgrade if the continuity email TTL has expired.
func (m *Manager) isVisitorUpgradeSafe(conversation models.Conversation) bool {
	if conversation.LastContinuityEmailSentAt.Valid {
		if time.Since(conversation.LastContinuityEmailSentAt.Time) > upgradeWindowTTL {
			m.lo.Info("visitor upgrade blocked: continuity email TTL expired",
				"conversation_uuid", conversation.UUID,
				"last_sent", conversation.LastContinuityEmailSentAt.Time,
				"age", time.Since(conversation.LastContinuityEmailSentAt.Time).String(),
				"max_ttl", upgradeWindowTTL.String())
			return false
		}
	}
	return true
}

// ProcessIncomingLiveChatMessage handles incoming live chat messages.
func (m *Manager) ProcessIncomingLiveChatMessage(msg models.Message) (models.Message, error) {
	// Upload message attachments.
	if err := m.uploadMessageAttachments(&msg); err != nil {
		return models.Message{}, fmt.Errorf("uploading message attachments: %w", err)
	}

	// Insert message.
	if err := m.InsertMessage(&msg); err != nil {
		return models.Message{}, err
	}

	// Advance contact_last_seen_at.
	if err := m.UpdateConversationContactLastSeen(msg.ConversationUUID); err != nil {
		m.lo.Error("error updating contact last seen after livechat message", "conversation_uuid", msg.ConversationUUID, "error", err)
	}

	// Process post-message hooks (automation rules, webhooks, SLA, etc.).
	// isNewConversation = false since conversation always exists for live chat.
	if err := m.ProcessIncomingMessageHooks(msg.ConversationUUID, false); err != nil {
		m.lo.Error("error processing incoming message hooks", "conversation_uuid", msg.ConversationUUID, "error", err)
	}

	return msg, nil
}

// MessageExists checks if a message with the given messageID exists.
func (m *Manager) MessageExists(messageID string) (bool, error) {
	_, err := m.messageExistsBySourceID([]string{messageID})
	if err != nil {
		if errors.Is(err, errConversationNotFound) {
			return false, nil
		}
		m.lo.Error("error fetching message from db", "error", err)
		return false, err
	}
	return true, nil
}

// EnqueueIncoming enqueues an incoming message for inserting in db.
func (m *Manager) EnqueueIncoming(message models.IncomingMessage) error {
	m.closedMu.Lock()
	defer m.closedMu.Unlock()
	if m.closed {
		return errors.New("incoming message queue is closed")
	}

	select {
	case m.incomingMessageQueue <- message:
		return nil
	default:
		m.lo.Warn("WARNING: incoming message queue is full")
		return errors.New("incoming message queue is full")
	}
}

// GetConversationByMessageID returns conversation by message id.
func (m *Manager) GetConversationByMessageID(id int) (models.Conversation, error) {
	var conversation = models.Conversation{}
	if err := m.q.GetConversationByMessageID.Get(&conversation, id); err != nil {
		if err == sql.ErrNoRows {
			return conversation, envelope.NewError(envelope.NotFoundError, m.i18n.T("validation.notFoundConversation"), nil)
		}
		m.lo.Error("error fetching message from DB", "error", err)
		return conversation, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return conversation, nil
}

// generateMessagesQuery generates the SQL query for fetching messages in a conversation.
func (c *Manager) generateMessagesQuery(baseQuery string, qArgs []interface{}, page, pageSize int) (string, int, []interface{}, error) {
	if pageSize > maxMessagesPerPage {
		pageSize = maxMessagesPerPage
	}

	// Calculate the offset
	offset := (page - 1) * pageSize

	// Append LIMIT and OFFSET to query arguments
	qArgs = append(qArgs, pageSize, offset)

	// Include LIMIT and OFFSET in the SQL query
	sqlQuery := fmt.Sprintf(baseQuery, fmt.Sprintf("LIMIT $%d OFFSET $%d", len(qArgs)-1, len(qArgs)))
	return sqlQuery, pageSize, qArgs, nil
}

// uploadMessageAttachments uploads all attachments for a message.
func (m *Manager) uploadMessageAttachments(message *models.Message) error {
	if len(message.Attachments) == 0 {
		return nil
	}

	for _, attachment := range message.Attachments {
		// Check if this attachment already exists by the content ID, as inline images can be repeated across conversations.
		contentID := attachment.ContentID
		if contentID != "" {
			// Make content ID MORE unique by prefixing it with the conversation UUID, as content id is not globally unique practically,
			// different messages can have the same content ID, I do not have the message ID at this point, so I am using sticking with the conversation UUID
			// to make it more unique.
			contentID = message.ConversationUUID + "_" + contentID

			exists, uuid, err := m.mediaStore.ContentIDExists(contentID)
			if err != nil {
				m.lo.Error("error checking media existence by content ID", "content_id", contentID, "error", err)
			}

			// This attachment already exists, replace the cid:content_id with the media relative url, not using absolute path as the root path can change.
			if exists {
				m.lo.Debug("attachment with content ID already exists replacing content ID with media relative URL", "content_id", contentID, "media_uuid", uuid)
				message.Content = strings.ReplaceAll(message.Content, fmt.Sprintf("cid:%s", attachment.ContentID), "/uploads/"+uuid)
				continue
			}

			// Attachment does not exist, replace the content ID with the new more unique content ID.
			message.Content = strings.ReplaceAll(message.Content, fmt.Sprintf("cid:%s", attachment.ContentID), fmt.Sprintf("cid:%s", contentID))
		}

		// Sanitize filename.
		attachment.Name = stringutil.SanitizeFilename(attachment.Name)

		m.lo.Debug("uploading message attachment", "name", attachment.Name, "content_id", contentID, "size", attachment.Size, "content_type", attachment.ContentType,
			"content_id", contentID, "disposition", attachment.Disposition)

		// Upload and insert entry in media table.
		attachReader := bytes.NewReader(attachment.Content)
		media, err := m.mediaStore.UploadAndInsert(
			attachment.Name,
			attachment.ContentType,
			contentID,
			/** Linking media to message happens later **/
			null.String{}, /** modelType */
			null.Int{},    /** modelID **/
			attachReader,
			attachment.Size,
			null.StringFrom(attachment.Disposition),
			[]byte("{}"), /** meta **/
		)
		if err != nil {
			m.lo.Error("failed to upload attachment", "name", attachment.Name, "error", err)
			return fmt.Errorf("failed to upload media %s: %w", attachment.Name, err)
		}

		// If the attachment is an image, generate and upload a thumbnail. Log any errors and continue, as thumbnail generation failure should not block message processing.
		attachmentExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(attachment.Name)), ".")
		if slices.Contains(image.Exts, attachmentExt) {
			if err := m.uploadThumbnailForMedia(media, attachment.Content); err != nil {
				m.lo.Error("error uploading thumbnail", "error", err)
			}
		}
		message.Media = append(message.Media, media)
	}
	return nil
}

// findOrCreateConversation finds or creates a conversation for the given incoming message.
func (m *Manager) findOrCreateConversation(in models.IncomingMessage) (int, string, bool, error) {
	var (
		conversationID   int
		conversationUUID string
		err              error
	)

	// Search for existing conversation using the in-reply-to and references.
	m.lo.Debug("searching conversation using in-reply-to and references", "in_reply_to", in.InReplyTo, "references", in.References)

	sourceIDs := append([]string{in.InReplyTo}, in.References...)
	conversationID, err = m.messageExistsBySourceID(sourceIDs)
	if err != nil && err != errConversationNotFound {
		return 0, "", false, err
	}

	// Conversation not found, create one.
	if conversationID == 0 {
		m.lo.Debug("no conversation found with in-reply-to and references, creating new conversation", "in_reply_to", in.InReplyTo, "references", in.References)
		lastMessage := stringutil.HTML2Text(in.Content)
		lastMessageAt := time.Now()
		conversationID, conversationUUID, err = m.CreateConversation(in.Contact.ID,
			in.InboxID,
			lastMessage,
			lastMessageAt,
			in.Subject,
			false, /**append reference number to subject**/
			nil,   /** meta **/
			nil,   /** customer attributes **/
			0,     /** max conversation **/
			0,     /** rate limit window **/
		)
		if err != nil || conversationID == 0 {
			return 0, "", false, err
		}
		return conversationID, conversationUUID, true, nil
	}

	// Get UUID for the found conversation ID.
	conversationUUID, err = m.GetConversationUUID(conversationID)
	if err != nil {
		return 0, "", false, err
	}
	return conversationID, conversationUUID, false, nil
}

// messageExistsBySourceID returns conversation ID if a message with any of the given source IDs exists.
func (m *Manager) messageExistsBySourceID(messageSourceIDs []string) (int, error) {
	messageSourceIDs = stringutil.RemoveEmpty(messageSourceIDs)
	if len(messageSourceIDs) == 0 {
		return 0, errConversationNotFound
	}
	var conversationID int
	if err := m.q.MessageExistsBySourceID.QueryRow(pq.Array(messageSourceIDs)).Scan(&conversationID); err != nil {
		if err == sql.ErrNoRows {
			return conversationID, errConversationNotFound
		}
		m.lo.Error("error fetching msg from DB", "error", err)
		return conversationID, err
	}
	return conversationID, nil
}

// fetchMessageAttachments fetches attachments for a single message ID - extracted for reuse
func (m *Manager) fetchMessageAttachments(messageID int) (attachment.Attachments, error) {
	var attachments attachment.Attachments

	// Get all media for this message
	medias, err := m.mediaStore.GetByModel(messageID, mmodels.ModelMessages)
	if err != nil {
		return attachments, fmt.Errorf("error fetching message attachments: %w", err)
	}

	// Fetch blobs for each media item
	for _, media := range medias {
		blob, err := m.mediaStore.GetBlob(media.UUID)
		if err != nil {
			return attachments, fmt.Errorf("error fetching media blob: %w", err)
		}

		attachment := attachment.Attachment{
			Name:        media.Filename,
			UUID:        media.UUID,
			ContentType: media.ContentType,
			Content:     blob,
			Size:        media.Size,
			Header:      attachment.MakeHeader(media.ContentType, media.UUID, media.Filename, "base64", media.Disposition.String),
			URL:         m.mediaStore.GetSignedURL(media.UUID),
		}
		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// attachAttachmentsToMessage attaches attachment blobs to message.
func (m *Manager) attachAttachmentsToMessage(message *models.Message) error {
	attachments, err := m.fetchMessageAttachments(message.ID)
	if err != nil {
		m.lo.Error("error fetching message attachments", "error", err)
		return err
	}

	// Attach attachments.
	message.Attachments = attachments

	return nil
}

// getOutgoingProcessingMessageIDs returns the IDs of outgoing messages currently being processed.
func (m *Manager) getOutgoingProcessingMessageIDs() []int {
	var out = make([]int, 0)
	m.outgoingProcessingMessages.Range(func(key, _ any) bool {
		if k, ok := key.(int); ok {
			out = append(out, k)
		}
		return true
	})
	return out
}

// uploadThumbnailForMedia prepares and uploads a thumbnail for an image attachment.
func (m *Manager) uploadThumbnailForMedia(media mmodels.Media, content []byte) error {
	// Create a reader from the content
	file := bytes.NewReader(content)

	// Seek to the beginning of the file
	file.Seek(0, 0)

	// Create the thumbnail
	thumbFile, err := image.CreateThumb(image.DefThumbSize, file)
	if err != nil {
		return fmt.Errorf("error creating thumbnail: %w", err)
	}

	// Generate thumbnail name
	thumbName := fmt.Sprintf("thumb_%s", media.UUID)

	// Upload the thumbnail
	if _, _, err := m.mediaStore.Upload(thumbName, media.ContentType, thumbFile); err != nil {
		m.lo.Error("error uploading thumbnail", "error", err)
		return fmt.Errorf("error uploading thumbnail: %w", err)
	}
	return nil
}

// getLatestMessage returns the latest message in a conversation.
func (m *Manager) getLatestMessage(conversationID int, typ []string, status []string, excludePrivate bool) (models.Message, error) {
	var message models.Message
	if err := m.q.GetLatestMessage.Get(&message, conversationID, pq.Array(typ), pq.Array(status), excludePrivate); err != nil {
		if err == sql.ErrNoRows {
			return message, sql.ErrNoRows
		}
		m.lo.Error("error fetching latest message from DB", "error", err)
		return message, fmt.Errorf("fetching latest message: %w", err)
	}
	return message, nil
}

// ProcessIncomingMessageHooks handles automation rules, webhooks, SLA events, and other post-processing
// for incoming messages. This allows other channels to insert messages first and then call this
// function to trigger the necessary hooks.
func (m *Manager) ProcessIncomingMessageHooks(conversationUUID string, isNewConversation bool) error {
	// Start waiting since clock, cleared when agent replies.
	now := time.Now()
	m.UpdateConversationWaitingSince(conversationUUID, &now)

	// Handle new conversation events.
	if isNewConversation {
		conversation, err := m.GetConversation(0, conversationUUID, "")
		if err == nil {
			m.webhookStore.TriggerEvent(wmodels.EventConversationCreated, conversation)
			m.automation.EvaluateNewConversationRules(conversation)
		}
		return nil
	}

	// Reopen conversation if it's not Open.
	systemUser, err := m.userStore.GetSystemUser()
	if err != nil {
		m.lo.Error("error fetching system user", "error", err)
	} else {
		if err := m.ReOpenConversation(conversationUUID, systemUser); err != nil {
			m.lo.Error("error reopening conversation", "error", err)
		}
	}

	// Create SLA event for next response if a SLA is applied and has next response time set, subsequent agent replies will mark this event as met.
	// This cycle continues for next response time SLA metric.
	conversation, err := m.GetConversation(0, conversationUUID, "")
	if err != nil {
		m.lo.Error("error fetching conversation for incoming message hooks", "conversation_uuid", conversationUUID, "error", err)
	} else {
		// Trigger automations on incoming message event.
		m.automation.EvaluateConversationUpdateRules(conversation, amodels.EventConversationMessageIncoming)

		if conversation.SLAPolicyID.Int == 0 {
			m.lo.Info("no SLA policy applied to conversation, skipping next response SLA event creation")
			return nil
		}
		if deadline, err := m.slaStore.CreateNextResponseSLAEvent(conversation.ID, conversation.AppliedSLAID.Int, conversation.SLAPolicyID.Int, conversation.AssignedTeamID.Int); err != nil && !errors.Is(err, sla.ErrUnmetSLAEventAlreadyExists) {
			m.lo.Error("error creating next response SLA event", "conversation_id", conversation.ID, "error", err)
		} else if !deadline.IsZero() {
			m.lo.Info("next response SLA event created for conversation", "conversation_id", conversation.ID, "deadline", deadline, "sla_policy_id", conversation.SLAPolicyID.Int)
			m.BroadcastConversationUpdate(conversationUUID, map[string]any{
				"next_response_deadline_at": deadline.Format(time.RFC3339),
				"next_response_met_at":      nil,
			})
		}
	}
	return nil
}

// broadcastMessageToWidgetClients sends a message to widget clients if the conversation belongs to a livechat inbox.
func (m *Manager) broadcastMessageToWidgetClients(message *models.Message) {
	conversation, err := m.GetConversation(0, message.ConversationUUID, "")
	if err != nil {
		return
	}

	inboxInstance, err := m.inboxStore.Get(conversation.InboxID)
	if err != nil {
		return
	}

	liveChatInbox, ok := inboxInstance.(*livechat.LiveChat)
	if !ok {
		return
	}

	m.SignAvatarURL(&message.Author.AvatarURL)
	liveChatInbox.BroadcastMessageToClients(message.ConversationUUID, conversation.ContactID, models.ChatMessage{
		UUID:             message.UUID,
		Status:           message.Status,
		ConversationUUID: message.ConversationUUID,
		CreatedAt:        message.CreatedAt,
		Content:          message.Content,
		TextContent:      message.TextContent,
		Author:           message.Author,
		Attachments:      message.Attachments,
		Meta:             message.Meta,
	})
}

// getMediaPreview returns a localized preview string based on attachment type.
func (m *Manager) getMediaPreview(media mmodels.Media) string {
	contentType := media.ContentType
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return m.i18n.T("globals.terms.image")
	case strings.HasPrefix(contentType, "video/"):
		return m.i18n.T("globals.terms.video")
	case strings.HasPrefix(contentType, "audio/"):
		return m.i18n.T("globals.terms.audio")
	default:
		return m.i18n.T("globals.terms.file")
	}
}
