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

	aimodels "github.com/abhinavxd/libredesk/internal/ai/models"
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
	maxMessagesPerPage = 100
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

	if inb.Channel() == inbox.ChannelEmail {
		// Set from address of the inbox
		message.From = inb.FromAddress()

		// Set "In-Reply-To" and "References" headers, logging any errors but continuing to send the message.
		// Include only the last 20 messages as references to avoid exceeding header size limits.
		message.References, err = m.GetMessageSourceIDs(message.ConversationID, 20)
		if err != nil {
			m.lo.Error("Error fetching conversation source IDs", "error", err)
		}

		// References is sorted in DESC i.e newest message first, so reverse it to keep the references in order.
		stringutil.ReverseSlice(message.References)

		// Remove the current message ID from the references.
		message.References = stringutil.RemoveItemByValue(message.References, message.SourceID.String)

		if len(message.References) > 0 {
			message.InReplyTo = message.References[len(message.References)-1]
		}
	}

	// Send message
	err = inb.Send(message)
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
		conversation, err := m.GetConversation(message.ConversationID, "")
		if err != nil {
			m.lo.Error("error fetching conversation", "conversation_id", message.ConversationID, "error", err)
			return
		}

		now := time.Now()
		if conversation.FirstReplyAt.IsZero() {
			m.UpdateConversationFirstReplyAt(message.ConversationUUID, message.ConversationID, now)
		}
		m.UpdateConversationLastReplyAt(message.ConversationUUID, message.ConversationID, now)

		// Clear waiting since timestamp as agent has replied to the conversation.
		m.UpdateConversationWaitingSince(message.ConversationUUID, nil)

		// Mark latest SLA event for next response as met.
		metAt, err := m.slaStore.SetLatestSLAEventMetAt(conversation.AppliedSLAID.Int, sla.MetricNextResponse)
		if err != nil && !errors.Is(err, sla.ErrLatestSLAEventNotFound) {
			m.lo.Error("error setting next response SLA event `met_at`", "conversation_id", conversation.ID, "metric", sla.MetricNextResponse, "applied_sla_id", conversation.AppliedSLAID.Int, "error", err)
		} else if !metAt.IsZero() {
			m.BroadcastConversationUpdate(message.ConversationUUID, "next_response_met_at", metAt.Format(time.RFC3339))
		}

		// Evaluate automation rules for outgoing message.
		m.automation.EvaluateConversationUpdateRulesByID(message.ConversationID, "", amodels.EventConversationMessageOutgoing)
	}
}

// RenderMessageInTemplate renders message content in template.
func (m *Manager) RenderMessageInTemplate(channel string, message *models.Message) error {
	switch channel {
	case inbox.ChannelEmail:
		conversation, err := m.GetConversation(0, message.ConversationUUID)
		if err != nil {
			m.lo.Error("error fetching conversation", "uuid", message.ConversationUUID, "error", err)
			return fmt.Errorf("fetching conversation: %w", err)
		}

		sender, err := m.userStore.Get(message.SenderID, "", "")
		if err != nil {
			m.lo.Error("error fetching message sender user", "sender_id", message.SenderID, "error", err)
			return fmt.Errorf("fetching message sender user: %w", err)
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
		if sender.IsSystemUser() || sender.IsAiAssistant() {
			data["Author"] = map[string]any{
				"FirstName": "",
				"LastName":  "",
				"FullName":  "",
				"Email":     "",
			}
		}

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
func (m *Manager) GetConversationMessages(conversationUUID string, types []string, privateMsgs *bool, page, pageSize int) ([]models.Message, int, error) {
	var (
		messages = make([]models.Message, 0)
		qArgs    []any
	)

	qArgs = append(qArgs, conversationUUID)
	if len(types) > 0 {
		qArgs = append(qArgs, pq.Array(types))
	} else {
		qArgs = append(qArgs, pq.Array(nil))
	}
	qArgs = append(qArgs, privateMsgs)
	query, pageSize, qArgs, err := m.generateMessagesQuery(m.q.GetMessages, qArgs, page, pageSize)
	if err != nil {
		m.lo.Error("error generating messages query", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.message}"), nil)
	}

	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	defer tx.Rollback()
	if err != nil {
		m.lo.Error("error preparing get messages query", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.message}"), nil)
	}

	if err := tx.Select(&messages, query, qArgs...); err != nil {
		m.lo.Error("error fetching conversations", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.message}"), nil)
	}

	return messages, pageSize, nil
}

// GetMessage retrieves a message by UUID.
func (m *Manager) GetMessage(uuid string) (models.Message, error) {
	var message models.Message
	if err := m.q.GetMessage.Get(&message, uuid); err != nil {
		m.lo.Error("error fetching message", "uuid", uuid, "error", err)
		return message, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.message}"), nil)
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
	m.BroadcastMessageUpdate(conversationUUID, messageUUID, "status" /*property*/, status)

	// Trigger webhook for message update.
	if message, err := m.GetMessage(messageUUID); err != nil {
		m.lo.Error("error fetching message for webhook event", "uuid", messageUUID, "error", err)
	} else {
		m.webhookStore.TriggerEvent(wmodels.EventMessageUpdated, message)
	}

	return nil
}

// MarkMessageAsPending updates message status to `Pending`, so if it's a outgoing message it can be picked up again by a worker.
func (m *Manager) MarkMessageAsPending(uuid string) error {
	if err := m.UpdateMessageStatus(uuid, models.MessageStatusPending); err != nil {
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorSending", "name", "{globals.terms.message}"), nil)
	}
	return nil
}

// SendPrivateNote inserts a private message in a conversation.
func (m *Manager) SendPrivateNote(media []mmodels.Media, senderID int, conversationUUID, content string) (models.Message, error) {
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
	return message, nil
}

// SendAutoReply sends a reply with automatically computed recipients based on the conversation's last message.
// For incoming messages: replies to the sender (from), CC'ing other participants.
// For outgoing messages: continues the existing recipient list.
// Used for AI responses, automation rules, and CSAT replies where manual recipient selection isn't needed.
func (m *Manager) SendAutoReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, metaMap map[string]any) (models.Message, error) {
	conv, err := m.GetConversation(0, conversationUUID)
	if err != nil {
		return models.Message{}, fmt.Errorf("fetching conversation for auto reply: %w", err)
	}

	// Compute recipients
	to, cc, bcc, err := m.makeRecipients(conv.ID, conv.Contact.Email.String, conv.InboxMail)
	if err != nil {
		return models.Message{}, fmt.Errorf("computing recipients for auto reply: %w", err)
	}

	// Send reply with computed recipients
	return m.SendReply(media, inboxID, senderID, contactID, conversationUUID, content, to, cc, bcc, metaMap)
}

// SendReply inserts a reply message for a conversation.
func (m *Manager) SendReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, to, cc, bcc []string, metaMap map[string]any) (models.Message, error) {
	inboxRecord, err := m.inboxStore.GetDBRecord(inboxID)
	if err != nil {
		return models.Message{}, err
	}

	if !inboxRecord.Enabled {
		return models.Message{}, envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.disabled", "name", "{globals.terms.inbox}"), nil)
	}

	var sourceID = ""
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
			return models.Message{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.empty", "name", "`to`"), nil)
		}
		sourceID, err = stringutil.GenerateEmailMessageID(conversationUUID, inboxRecord.From)
		if err != nil {
			m.lo.Error("error generating source message id", "error", err)
			return models.Message{}, envelope.NewError(envelope.GeneralError, m.i18n.T("conversation.errorGeneratingMessageID"), nil)
		}
	case inbox.ChannelLiveChat:
		sourceID, err = stringutil.RandomAlphanumeric(35)
		if err != nil {
			m.lo.Error("error generating random source id", "error", err)
			return models.Message{}, envelope.NewError(envelope.GeneralError, m.i18n.T("conversation.errorGeneratingMessageID"), nil)
		}
		sourceID = "livechat-" + sourceID
	}

	// Marshal meta.
	metaJSON, err := json.Marshal(metaMap)
	if err != nil {
		m.lo.Error("error marshalling message meta map to JSON", "error", err)
		return models.Message{}, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorInserting", "name", "{globals.terms.message}"), nil)
	}

	// Insert the message into the database
	message := models.Message{
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

	// Save message as text.
	message.TextContent = stringutil.HTML2Text(message.Content)

	// Insert Message.
	if err := m.q.InsertMessage.Get(message, message.Type, message.Status, message.ConversationID, message.ConversationUUID, message.Content, message.TextContent, message.SenderID, message.SenderType,
		message.Private, message.ContentType, message.SourceID, message.Meta); err != nil {
		m.lo.Error("error inserting message in db", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorInserting", "name", "{globals.terms.message}"), nil)
	}

	// Attach just inserted message to the media.
	for _, media := range message.Media {
		m.mediaStore.Attach(media.ID, mmodels.ModelMessages, message.ID)
	}

	// Add this user as a participant if not already present.
	m.addConversationParticipant(message.SenderID, message.ConversationUUID)

	// Hide CSAT message content as it contains a public link to the survey.
	lastMessage := message.TextContent
	if message.HasCSAT() {
		lastMessage = "Please rate your experience with us"
	}

	// Get sender user and store last message in conversation.
	var (
		sender            umodels.User
		conversationMeta  = map[string]any{}
		lastInteractionAt = null.Time{}
		err               error
	)

	sender, err = m.userStore.Get(message.SenderID, "", "")
	if err != nil {
		m.lo.Error("error fetching message sender user", "sender_id", message.SenderID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorInserting", "name", "{globals.terms.message}"), nil)
	}

	// Censor CSAT content before saving last message details.
	message.CensorCSATContent()

	if slices.Contains([]string{models.MessageIncoming, models.MessageOutgoing}, message.Type) && !message.Private {
		conversationMeta["last_chat_message"] = map[string]any{
			"uuid":         message.UUID,
			"created_at":   message.CreatedAt,
			"text_content": message.TextContent,
			"sender": map[string]any{
				"id":         sender.ID,
				"first_name": sender.FirstName,
				"last_name":  sender.LastName,
				"type":       sender.Type,
			},
		}
		lastInteractionAt = null.TimeFrom(message.CreatedAt)
	}
	conversationMeta["last_message"] = map[string]any{
		"uuid":         message.UUID,
		"created_at":   message.CreatedAt,
		"text_content": message.TextContent,
		"sender": map[string]any{
			"id":         sender.ID,
			"first_name": sender.FirstName,
			"last_name":  sender.LastName,
			"type":       sender.Type,
		},
	}
	conversationMetaB, err := json.Marshal(conversationMeta)
	if err != nil {
		m.lo.Error("error marshalling conversation meta to JSON", "error", err)
		conversationMetaB = []byte("{}")
	}

	m.UpdateConversationLastMessage(message.ConversationID, message.ConversationUUID, lastMessage, message.SenderType, message.CreatedAt, lastInteractionAt, conversationMetaB)

	// Broadcast new message.
	m.BroadcastNewMessage(message)

	// Refetch message if this message has media attachments, as media gets linked after inserting the message.
	if len(message.Media) > 0 {
		refetchedMessage, err := m.GetMessage(message.UUID)
		if err != nil {
			m.lo.Error("error fetching message after insert", "error", err)
		} else {
			// Replace the message in the struct with the refetched message.
			*message = refetchedMessage
		}
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
	assignee, err := m.userStore.Get(assigneeID, "", "")
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
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorGenerating", "name", "{globals.terms.activityMessage}"), nil)
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
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorInserting", "name", "{globals.terms.activityMessage}"), nil)
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
	var (
		isNewConversation = false
		conversationID    int
		err               error
	)

	// Do channel specific processing.
	switch in.Channel {
	case inbox.ChannelEmail:
		// Find or create contact and set sender ID in message.
		if err := m.userStore.CreateContact(&in.Contact); err != nil {
			m.lo.Error("error upserting contact", "error", err)
			return models.Message{}, err
		}
		in.Message.SenderID = in.Contact.ID

		// Conversations exists for this message?
		conversationID, err = m.findConversationID([]string{in.Message.SourceID.String})
		if err != nil && err != errConversationNotFound {
			return models.Message{}, err
		}
		if conversationID > 0 {
			return models.Message{}, nil
		}

		// Find or create new conversation.
		isNewConversation, err = m.findOrCreateConversation(&in.Message, in.InboxID, in.Contact.ID)
		if err != nil {
			return models.Message{}, err
		}
	case inbox.ChannelLiveChat:
		// For live chat, a conversation is created before the message is processed. So nothing to do here.
	}

	// Upload message attachments.
	if err := m.uploadMessageAttachments(&in.Message); err != nil {
		// Log error but continue processing.
		m.lo.Error("error uploading message attachments", "message_source_id", in.Message.SourceID, "error", err)
	}

	// Insert message.
	if err = m.InsertMessage(&in.Message); err != nil {
		return models.Message{}, err
	}

	// Process post-message hooks (automation rules, webhooks, SLA, etc.).
	if err := m.ProcessIncomingMessageHooks(in.Message.ConversationUUID, isNewConversation); err != nil {
		m.lo.Error("error processing incoming message hooks", "conversation_uuid", in.Message.ConversationUUID, "error", err)
		return models.Message{}, fmt.Errorf("processing incoming message hooks: %w", err)
	}
	return in.Message, nil
}

// MessageExists checks if a message with the given messageID exists.
func (m *Manager) MessageExists(messageID string) (bool, error) {
	_, err := m.findConversationID([]string{messageID})
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
			return conversation, envelope.NewError(envelope.NotFoundError, m.i18n.Ts("globals.messages.notFound", "name", "{globals.terms.conversation}"), nil)
		}
		m.lo.Error("error fetching message from DB", "error", err)
		return conversation, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.conversation}"), nil)
	}
	return conversation, nil
}

// generateMessagesQuery generates the SQL query for fetching messages in a conversation.
func (c *Manager) generateMessagesQuery(baseQuery string, qArgs []interface{}, page, pageSize int) (string, int, []interface{}, error) {
	if page <= 0 {
		return "", 0, nil, errors.New("page must be greater than 0")
	}
	if pageSize > maxMessagesPerPage {
		pageSize = maxMessagesPerPage
	}
	if pageSize <= 0 {
		return "", 0, nil, errors.New("page size must be greater than 0")
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
func (m *Manager) uploadMessageAttachments(message *models.Message) []error {
	if len(message.Attachments) == 0 {
		return nil
	}

	var uploadErr []error
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
			uploadErr = append(uploadErr, err)
			m.lo.Error("failed to upload attachment", "name", attachment.Name, "error", err)
		}

		// If the attachment is an image, generate and upload thumbnail.
		attachmentExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(attachment.Name)), ".")
		if slices.Contains(image.Exts, attachmentExt) {
			if err := m.uploadThumbnailForMedia(media, attachment.Content); err != nil {
				uploadErr = append(uploadErr, err)
				m.lo.Error("error uploading thumbnail", "error", err)
			}
		}
		message.Media = append(message.Media, media)
	}
	return uploadErr
}

// findOrCreateConversation finds or creates a conversation for the given message.
func (m *Manager) findOrCreateConversation(in *models.Message, inboxID, contactID int) (bool, error) {
	var (
		new              bool
		err              error
		conversationID   int
		conversationUUID string
	)

	// Search for existing conversation using the in-reply-to and references.
	sourceIDs := append([]string{in.InReplyTo}, in.References...)
	conversationID, err = m.findConversationID(sourceIDs)
	if err != nil && err != errConversationNotFound {
		return new, err
	}

	// Conversation not found, create one.
	if conversationID == 0 {
		new = true
		lastMessage := stringutil.HTML2Text(in.Content)
		lastMessageAt := time.Now()
		conversationID, conversationUUID, err = m.CreateConversation(contactID, inboxID, lastMessage, lastMessageAt, in.Subject, false /**append reference number to subject**/)
		if err != nil || conversationID == 0 {
			return new, err
		}
		in.ConversationID = conversationID
		in.ConversationUUID = conversationUUID
		return new, nil
	}
	// Get UUID.
	if conversationUUID == "" {
		conversationUUID, err = m.GetConversationUUID(conversationID)
		if err != nil {
			return new, err
		}
	}
	in.ConversationID = conversationID
	in.ConversationUUID = conversationUUID
	return new, nil
}

// findConversationID finds the conversation ID from the message source ID.
func (m *Manager) findConversationID(messageSourceIDs []string) (int, error) {
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

// attachAttachmentsToMessage attaches attachment blobs to message.
func (m *Manager) attachAttachmentsToMessage(message *models.Message) error {
	var attachments attachment.Attachments

	// Get all media for this message.
	medias, err := m.mediaStore.GetByModel(message.ID, mmodels.ModelMessages)
	if err != nil {
		m.lo.Error("error fetching message attachments", "error", err)
		return err
	}

	// Fetch blobs.
	for _, media := range medias {
		blob, err := m.mediaStore.GetBlob(media.UUID)
		if err != nil {
			m.lo.Error("error fetching media blob", "error", err)
			return err
		}
		attachment := attachment.Attachment{
			Name:        media.Filename,
			UUID:        media.UUID,
			ContentType: media.ContentType,
			Content:     blob,
			Size:        media.Size,
			Header:      attachment.MakeHeader(media.ContentType, media.UUID, media.Filename, "base64", media.Disposition.String),
			URL:         m.mediaStore.GetURL(media.UUID),
		}
		attachments = append(attachments, attachment)
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
	if _, err := m.mediaStore.Upload(thumbName, media.ContentType, thumbFile); err != nil {
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
	// Handle new conversation events.
	if isNewConversation {
		conversation, err := m.GetConversation(0, conversationUUID)
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

	// Set waiting since timestamp, this gets cleared when agent replies to the conversation.
	now := time.Now()
	m.UpdateConversationWaitingSince(conversationUUID, &now)

	// Create SLA event for next response if a SLA is applied and has next response time set, subsequent agent replies will mark this event as met.
	// This cycle continues for next response time SLA metric.
	conversation, err := m.GetConversation(0, conversationUUID)
	if err != nil {
		m.lo.Error("error fetching conversation", "conversation_uuid", conversationUUID, "error", err)
	} else {
		// Enqueue conversation for AI completion if assigned to an AI assistant.
		m.enqueueMessageForAICompletion(conversation)

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
			m.BroadcastConversationUpdate(conversationUUID, "next_response_deadline_at", deadline.Format(time.RFC3339))
			// Clear next response met at timestamp as this event was just created.
			m.BroadcastConversationUpdate(conversationUUID, "next_response_met_at", nil)
		}
	}

	return nil
}

// enqueueMessageForAICompletion enqueues message for AI completion if the conversation is assigned to an AI assistant and if the inbox has help center attached.
func (m *Manager) enqueueMessageForAICompletion(conversation models.Conversation) {
	m.lo.Debug("checking conversation for AI completion", "conversation_id", conversation.ID)

	if m.aiStore == nil {
		m.lo.Warn("AI store not configured, skipping AI completion request")
		return
	}

	// Get the latest message to check if it's from a contact
	latestMsg, err := m.getLatestMessage(conversation.ID,
		[]string{models.MessageIncoming, models.MessageOutgoing},
		[]string{models.MessageStatusSent, models.MessageStatusReceived},
		true)
	if err != nil {
		m.lo.Error("error fetching latest message for AI completion", "conversation_id", conversation.ID, "error", err)
		return
	}

	// Only process incoming messages from contacts.
	if latestMsg.Type != models.MessageIncoming || latestMsg.SenderType != models.SenderTypeContact {
		m.lo.Debug("latest message is not from a contact, skipping AI completion", "conversation_id", conversation.ID, "type", latestMsg.Type, "sender_type", latestMsg.SenderType)
		return
	}

	// Get the attached help center to the inbox.
	inbox, err := m.inboxStore.GetDBRecord(conversation.InboxID)
	if err != nil {
		m.lo.Error("error fetching inbox for AI completion", "inbox_id", conversation.InboxID, "error", err)
		return
	}

	// Make sure the conversation has an assigned user.
	if !conversation.AssignedUserID.Valid {
		m.lo.Debug("conversation is not assigned to a user, skipping AI completion", "conversation_id", conversation.ID)
		return
	}

	// Make sure there's an helpcenter linked to the inbox as AI completions use help center articles for context.
	if !inbox.HelpCenterID.Valid {
		m.lo.Debug("inbox is not linked to a help center, skipping AI completion", "inbox_id", conversation.InboxID)
		return
	}

	// Check if assignee is an AI assistant and is enabled.
	assigneUser, err := m.userStore.Get(conversation.AssignedUserID.Int, "", "")
	if err != nil {
		m.lo.Error("error fetching assignee user for AI completion", "assignee_user_id", conversation.AssignedUserID.Int, "error", err)
		return
	}

	if !assigneUser.IsAiAssistant() {
		m.lo.Debug("conversation is not assigned to an AI assistant, skipping AI completion", "conversation_id", conversation.ID)
		return
	}

	if !assigneUser.Enabled {
		m.lo.Debug("AI assistant is not enabled, skipping AI completion", "conversation_id", conversation.ID)
		return
	}

	messages, _, err := m.GetConversationMessages(conversation.UUID, []string{models.MessageIncoming, models.MessageOutgoing}, nil, 1, 20)
	if err != nil {
		m.lo.Error("error fetching conversation message history for AI completion", "conversation_uuid", conversation.UUID, "error", err)
		return
	}

	req := aimodels.ConversationCompletionRequest{
		Messages:         messages,
		InboxID:          conversation.InboxID,
		ContactID:        conversation.ContactID,
		ConversationUUID: conversation.UUID,
		AIAssistant:      assigneUser,
		HelpCenterID:     inbox.HelpCenterID,
	}

	if err := m.aiStore.EnqueueConversationCompletion(req); err != nil {
		m.lo.Error("error enqueuing AI completion request", "error", err)
		return
	}

	m.lo.Info("AI completion request enqueued", "conversation_uuid", conversation.UUID)
}
