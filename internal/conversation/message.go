package conversation

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/jakedolan443/fernmail/internal/attachment"

	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	"github.com/jakedolan443/fernmail/internal/image"
	"github.com/jakedolan443/fernmail/internal/inbox"

	mmodels "github.com/jakedolan443/fernmail/internal/media/models"

	"github.com/google/uuid"
	"github.com/jakedolan443/fernmail/internal/stringutil"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	wmodels "github.com/jakedolan443/fernmail/internal/webhook/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

const (
	maxMessagesPerPage = 500
	// Only allow visitor-to-contact upgrade within this window after the last continuity email.
	upgradeWindowTTL = 7 * 24 * time.Hour
)

// Matches <img ... src="URL"> and captures the URL for downstream parsing.
var imgSrcPattern = regexp.MustCompile(`(?i)<img\b[^>]*?\bsrc=["']([^"']*)["']`)

// fromNameVars is the template context for an inbox's from-name template.
type fromNameVars struct {
	Agent fromNameAgent
	Inbox fromNameInbox
}

type fromNameAgent struct{ FirstName, LastName, FullName string }

type fromNameInbox struct{ Name string }

// Run starts a pool of worker goroutines to handle message dispatching via inbox's channel and processes incoming messages. It scans for
// pending outgoing messages at the specified read interval and pushes them to the outgoing queue to be sent.
func (m *Manager) Run(ctx context.Context, incomingQWorkers, outgoingQWorkers, scanInterval time.Duration) {
	m.closedMu.Lock()
	if m.closed {
		m.closedMu.Unlock()
		return
	}
	if m.stopCh == nil {
		m.stopCh = make(chan struct{})
	}
	m.wg.Add(1)
	defer m.wg.Done()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-m.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

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

	m.wg.Add(1)
	go func() { defer m.wg.Done(); m.runMailMaintenance(ctx) }()
	m.closedMu.Unlock()

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
				select {
				case m.outgoingMessageQueue <- message:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// Close signals the Manager to stop processing messages, closes channels,
// and waits for all worker goroutines to finish processing.
func (m *Manager) Close() {
	m.closedMu.Lock()
	if !m.closed {
		m.closed = true
		if m.stopCh != nil {
			close(m.stopCh)
		}
	}
	m.closedMu.Unlock()
	// Producers and consumers share cancellation. Never close a queue beneath a
	// producer; queued delivery remains pending in Postgres for the next run.
	m.wg.Wait()
}

// IncomingMessageWorker processes incoming messages from the incoming message queue.
func (m *Manager) IncomingMessageWorker(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if m.processDurableIncoming(ctx) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
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
	token, err := m.claimDelivery(message.ID)
	if err != nil {
		if err != sql.ErrNoRows {
			m.lo.Error("claiming outgoing message", "error", err)
		}
		return
	}
	leaseCtx, cancelLease := context.WithCancel(context.Background())
	defer cancelLease()
	go m.heartbeatDelivery(leaseCtx, token)

	// Helper function to handle errors
	handleError := func(err error, errorMsg string) bool {
		if err != nil {
			m.lo.Error(errorMsg, "error", err, "message_id", message.ID)
			if saveErr := m.finishDelivery(message, token, "failed", err); saveErr != nil {
				m.lo.Error("recording failed delivery", "error", saveErr)
			}
			return true
		}
		return false
	}

	if err := m.ensureDeliveryAllowed(message.ID); handleError(err, "outgoing address or sender is no longer authorized") {
		return
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

	if inb.Channel() == inbox.ChannelEmail {
		outbound.From = m.emailFromAddress(inb, message)

		// Set "In-Reply-To" and "References" headers for email threading.
		outbound.References, outbound.InReplyTo = m.BuildEmailThreadingHeaders(message.ConversationID, outbound.SourceID)
	}

	// Send message
	err = inb.Send(outbound)
	if err != nil {
		if saveErr := m.finishDelivery(message, token, "unknown", err); saveErr != nil {
			m.lo.Error("recording uncertain delivery", "error", saveErr)
		}
		return
	}

	// Update status as sent.
	if err := m.finishDelivery(message, token, "sent", nil); err != nil {
		m.lo.Error("SMTP succeeded but completion could not be saved; held for reconciliation", "error", err)
		return
	}

	// Skip system user replies since we only update timestamps and SLA for human replies.
	systemUser, err := m.userStore.GetSystemUser()
	if err != nil {
		m.lo.Error("error fetching system user", "error", err)
		return
	}
	if message.SenderID != systemUser.ID {

		now := time.Now()
		nowStr := now.Format(time.RFC3339)
		wsData := map[string]any{"last_reply_at": nowStr, "waiting_since": nil}

		var isFirstReply bool
		if err := m.q.UpdateConversationReplyTimestamps.QueryRow(message.ConversationID, now).Scan(&isFirstReply); err != nil {
			m.lo.Error("error updating conversation reply timestamps", "error", err)
		} else if isFirstReply {
			wsData["first_reply_at"] = nowStr
			// Stamp the first-response SLA immediately.

		}

		m.BroadcastConversationUpdate(message.ConversationUUID, wsData)

	}
}

// BuildTemplateData builds the common template data map for rendering message content variables.
func (m *Manager) BuildTemplateData(conversationUUID string, senderID int) (map[string]any, error) {
	conversation, err := m.GetConversation(0, conversationUUID, "")
	if err != nil {
		return nil, fmt.Errorf("fetching conversation: %w", err)
	}

	sender, err := m.userStore.GetAgentCachedOrLoad(senderID)
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
		// Lets templates disclose AI-composed replies, e.g. {{ if .IsAIComposed }}Composed by AI{{ end }}.
		"IsAIComposed": sender.Type == umodels.UserTypeAIAssistant,
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
	if err != nil {
		m.lo.Error("error preparing get messages query", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	defer tx.Rollback()
	if err := tx.Select(&messages, query, qArgs...); err != nil {
		m.lo.Error("error fetching conversations", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	return messages, pageSize, nil
}

// GetAllConversationMessages returns the newest messages in a conversation in chronological order, capped at limit; a non-positive limit returns them all.
func (m *Manager) GetAllConversationMessages(conversationUUID string, private *bool, msgTypes []string, limit int) ([]models.Message, error) {
	var all []models.Message
	pageSize := maxMessagesPerPage
	if limit > 0 && limit < pageSize {
		pageSize = limit
	}
	for page := 1; ; page++ {
		messages, _, err := m.GetConversationMessages(conversationUUID, page, pageSize, private, msgTypes)
		if err != nil {
			return nil, err
		}
		all = append(all, messages...)
		if len(messages) == 0 || len(all) >= messages[0].Total || (limit > 0 && len(all) >= limit) {
			break
		}
	}
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	slices.Reverse(all)
	return all, nil
}

// GetMessage retrieves a message by UUID.
func (m *Manager) GetMessage(uuid string) (models.Message, error) {
	var message models.Message
	if err := m.q.GetMessage.Get(&message, uuid); err != nil {
		m.lo.Error("error fetching message", "uuid", uuid, "error", err)
		return message, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Generate signed URLs for attachments.
	m.SignAttachmentURLs(message.Attachments)

	return message, nil
}

// SignAttachmentURLs adds access URLs for the original image and its thumbnail.
func (m *Manager) SignAttachmentURLs(attachments attachment.Attachments) {
	for i := range attachments {
		if attachments[i].Unavailable {
			attachments[i].URL = ""
			attachments[i].ThumbnailURL = ""
			continue
		}
		attachments[i].URL = m.mediaStore.GetURL(attachments[i].UUID, attachments[i].ContentType, attachments[i].Name)
		if strings.HasPrefix(attachments[i].ContentType, "image/") {
			attachments[i].ThumbnailURL = m.mediaStore.GetThumbnailURL(attachments[i].UUID)
		}
	}
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
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int
	if err = tx.Get(&id, `SELECT id FROM conversation_messages WHERE uuid=$1 AND status='failed' FOR UPDATE`, uuid); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE mail_delivery_attempts SET state='retry_authorized' WHERE message_id=$1 AND state='unknown'`, id); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE conversation_messages SET status='pending',meta=COALESCE(meta,'{}')-'delivery_uncertain'-'delivery_error',updated_at=now() WHERE id=$1`, id); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	conversationUUID, _ := m.getConversationUUIDFromMessageUUID(uuid)
	m.BroadcastMessageUpdate(conversationUUID, uuid, map[string]any{"status": models.MessageStatusPending, "meta": m.deliveryMeta(uuid)})
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

	}

	return message, nil
}

// CreateContactMessage creates a contact message in a conversation.
// sourceID is the bare RFC 5322 Message-ID of the inbound message; it is normalized and stored on the message so replies thread on it, mirroring the IMAP ingestion path. Empty leaves the column NULL.
func (m *Manager) CreateContactMessage(media []mmodels.Media, contactID int, conversationUUID, content, contentType string, isNewConversation bool, sourceID string, uploadUserIDs ...int) (models.Message, error) {
	sourceID = stringutil.NormalizeMessageID(sourceID)
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
		SourceID:         null.NewString(sourceID, sourceID != ""),
	}
	if len(uploadUserIDs) > 0 {
		message.UploadUserID = uploadUserIDs[0]
	}
	if err := m.InsertMessage(&message); err != nil {
		return models.Message{}, err
	}

	// Process post-message hooks (reopen, waiting since, automation, SLA).
	if err := m.ProcessIncomingMessageHooks(message, isNewConversation); err != nil {
		m.lo.Error("error processing incoming message hooks", "conversation_uuid", conversationUUID, "error", err)
	}

	return message, nil
}

// QueueReply queues a reply message in a conversation.
func (m *Manager) QueueReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, to, cc, bcc []string, metaMap map[string]interface{}) (models.Message, error) {
	return m.queueReply(media, inboxID, senderID, contactID, conversationUUID, content, to, cc, bcc, metaMap, nil)
}

// queueReply queues an outgoing reply. beforeCommit runs inside the insert
// transaction, so a caller's own bookkeeping commits or rolls back with the message.
func (m *Manager) queueReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, to, cc, bcc []string, metaMap map[string]interface{}, beforeCommit func(*sqlx.Tx, *models.Message) error) (models.Message, error) {
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
		sourceFrom := inboxRecord.From
		if alias := m.emailAddressForConversationUUID(conversationUUID); alias != "" {
			metaMap["email_alias"] = alias
			sourceFrom = alias
		}
		sourceID, err = stringutil.GenerateEmailMessageID(conversationUUID, sourceFrom)
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

	var parent string
	_ = m.db.Get(&parent, `SELECT COALESCE(source_id,'') FROM conversation_messages WHERE conversation_id=(SELECT id FROM conversations WHERE uuid=$1) AND NOT private AND type IN ('incoming','outgoing') AND status IN ('received','sent') AND source_id>'' ORDER BY id DESC LIMIT 1`, conversationUUID)
	// Insert the message into the database
	message = models.Message{
		ReplyToSourceID:   parent,
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
	if err := m.insertMessage(context.Background(), &message, beforeCommit); err != nil {
		return models.Message{}, err
	}
	return message, nil
}

// InsertMessage inserts a message and attaches the media to the message.
func (m *Manager) InsertMessage(message *models.Message) error {
	return m.insertMessage(context.Background(), message, nil)
}

func (m *Manager) insertMessage(ctx context.Context, message *models.Message, beforeCommit func(*sqlx.Tx, *models.Message) error) error {
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

	var inlineUUIDs []string
	// Server-ingested MIME attachments carry explicit media IDs. Remote inbound
	// images never claim uploads, even if their URLs contain UUID-shaped paths.
	if message.Type != models.MessageIncoming || message.UploadUserID > 0 {
		conversationID := message.ConversationID
		if conversationID == 0 && message.ConversationUUID != "" {
			if err := m.db.Get(&conversationID, `SELECT id FROM conversations WHERE uuid=$1`, message.ConversationUUID); err != nil {
				return err
			}
		}
		ownerID := message.UploadUserID
		if message.SenderType == models.SenderTypeAgent {
			ownerID = message.SenderID
		}
		message.Content, inlineUUIDs = m.normalizeInlineUploads(message.Content, conversationID, ownerID)
	}

	// Convert content to plain text for search.
	if message.ContentType == models.ContentTypeText {
		message.TextContent = message.Content
	} else {
		message.TextContent = stringutil.HTML2Text(message.Content)
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error beginning message insert transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()
	// Serialize receipt creation with mailbox mark-read snapshots. The insert
	// clock is taken after this lock, never at the start of a long transaction.
	if err := tx.QueryRow(`SELECT id,uuid FROM conversations WHERE ($1>0 AND id=$1) OR ($1=0 AND uuid=NULLIF($2,'')::uuid) FOR NO KEY UPDATE`, message.ConversationID, message.ConversationUUID).Scan(&message.ConversationID, &message.ConversationUUID); err != nil {
		return err
	}

	if err := tx.Stmtx(m.q.InsertMessage).Get(message, message.Type, message.Status, message.ConversationID, message.ConversationUUID, message.Content, message.TextContent, message.SenderID, message.SenderType,
		message.Private, message.ContentType, message.SourceID, message.Meta, message.ReplyToSourceID); err != nil {
		m.lo.Error("error inserting message in db", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	uploadUserID := message.UploadUserID
	if message.SenderType == models.SenderTypeAgent {
		uploadUserID = message.SenderID
	}
	if err := m.mediaStore.LinkMessageMediaTx(tx, message.ID, message.Media, inlineUUIDs, uploadUserID); err != nil {
		var inputError envelope.Error
		if errors.As(err, &inputError) {
			return err
		}
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if beforeCommit != nil {
		if err := beforeCommit(tx, message); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing message insert transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	if m.cacheIncomingImages != nil && message.Type == models.MessageIncoming && message.ContentType == models.ContentTypeHTML && !message.Private {
		if err := m.cacheIncomingImages(ctx, message.ID, message.Content); err != nil {
			m.lo.Error("error caching incoming message images", "message_id", message.ID, "error", err)
		}
	}

	// Add this user as a participant if not already present.
	m.addConversationParticipant(message.SenderID, message.ConversationUUID)

	// Load the actual message author before broadcasting (a reply can come from
	// someone other than the conversation's original correspondent).
	refetchedMessage, refetchErr := m.GetMessage(message.UUID)
	if refetchErr != nil {
		m.lo.Error("error fetching message after insert", "error", refetchErr)
	} else {
		message.Author = refetchedMessage.Author
	}

	// Skip updating last_message and broadcasting for continuity emails.
	if !message.IsContinuityMessage() {
		// Hide CSAT message content as it contains a public link to the survey.
		lastMessage := message.TextContent
		if message.HasCSAT() {
			lastMessage = "Please rate your experience with us"
		}

		// HTML2Text drops <img> tags, so image-only messages have empty text. Fall back to a media-type preview.
		if strings.TrimSpace(lastMessage) == "" {
			switch {
			case len(message.Media) > 0:
				lastMessage = m.getMediaPreview(message.Media[0])
			case len(inlineUUIDs) > 0:
				lastMessage = m.i18n.T("globals.terms.image")
			}
		}

		// Update conversation last message details (also conditionally updates last_interaction if not activity/private).
		m.UpdateConversationLastMessage(message.ConversationID, message.ConversationUUID, lastMessage, message.SenderType, message.Type, message.Private, message.CreatedAt, message.SenderID)

		var convItem *models.ConversationListItem
		if item, err := m.GetConversationListItem(message.ConversationUUID); err == nil {
			convItem = &item
		} else {
			m.lo.Error("error fetching conversation list item for broadcast", "uuid", message.ConversationUUID, "error", err)
		}
		m.BroadcastNewMessage(message, convItem, lastMessage)
	}

	// Return all populated fields, including media URLs.
	if refetchErr == nil {
		*message = refetchedMessage
	}

	// Trigger webhook for new message created.
	m.webhookStore.TriggerEvent(wmodels.EventMessageCreated, message)

	return nil
}

// RecordStatusChange records an activity for a status change.
func (m *Manager) RecordStatusChange(status, conversationUUID string, actor umodels.User) error {
	return m.InsertConversationActivity(models.ActivityStatusChange, conversationUUID, status, actor)
}

// InsertConversationActivity inserts an activity message.
func (m *Manager) InsertConversationActivity(activityType, conversationUUID, newValue string, actor umodels.User) error {
	content, err := m.getMessageActivityContent(activityType, newValue, actor.FullName())
	if err != nil {
		m.lo.Error("error could not generate activity content", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Store the activity type structurally so callers can filter activities without parsing i18n content.
	meta, _ := json.Marshal(map[string]string{"activity_type": activityType})

	message := models.Message{
		Type:             models.MessageActivity,
		Status:           models.MessageStatusSent,
		Content:          content,
		ContentType:      models.ContentTypeText,
		ConversationUUID: conversationUUID,
		Private:          true,
		SenderID:         actor.ID,
		SenderType:       models.SenderTypeAgent,
		Meta:             meta,
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
	case models.ActivityAssigneeUserRemoved:
		content = fmt.Sprintf("%s removed %s as assignee", actorName, newValue)
	case models.ActivitySelfAssign:
		content = fmt.Sprintf("%s self-assigned this conversation", actorName)
	case models.ActivitySelfUnassign:
		content = fmt.Sprintf("%s unassigned themselves", actorName)
	case models.ActivityPriorityChange:
		content = fmt.Sprintf("%s set priority to %s", actorName, newValue)
	case models.ActivityStatusChange:
		content = fmt.Sprintf("%s marked the conversation as %s", actorName, newValue)
	case models.ActivitySLASet:
		content = fmt.Sprintf("%s set %s SLA policy", actorName, newValue)
	case models.ActivityParticipantAdded:
		content = fmt.Sprintf("%s joined the conversation", newValue)
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
	return m.processIncomingMessage(context.Background(), in)
}

func (m *Manager) processIncomingMessage(ctx context.Context, in models.IncomingMessage) (models.Message, error) {
	addressID, err := m.resolveIncomingAddress(in.InboxID, in.EmailAlias)
	if err != nil {
		return models.Message{}, err
	}
	in.AddressID = addressID
	// Return early if this message already exists at this recipient address.
	dupConvID, err := m.messageExistsBySourceID([]string{in.SourceID.String}, addressID)
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
		if err := m.userStore.ResolveEmailSender(&user); err != nil {
			m.lo.Error("error creating contact for incoming message", "message_source_id", in.SourceID.String, "error", err)
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
			m.lo.Error("error finding or creating conversation for incoming message", "message_source_id", in.SourceID.String, "error", err)
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
		m.lo.Error("error uploading message attachments", "message_source_id", in.SourceID.String, "error", upErr)
		if isNewConversation && conversationUUID != "" {
			m.lo.Info("deleting conversation as message attachment upload failed", "conversation_uuid", conversationUUID, "message_source_id", in.SourceID.String)
			if err := m.DeleteConversation(conversationUUID); err != nil {
				return models.Message{}, fmt.Errorf("deleting conversation after message attachment upload failure: %w", err)
			}
		}
		return models.Message{}, fmt.Errorf("uploading message attachments: %w", upErr)
	}

	// Insert message. On failure, delete the conversation if it was just created for this message.
	if err = m.insertMessage(ctx, &msg, nil); err != nil {
		m.lo.Error("error inserting incoming message", "message_source_id", in.SourceID.String, "conversation_uuid", conversationUUID, "is_new", isNewConversation, "error", err)
		if isNewConversation && conversationUUID != "" {
			if delErr := m.DeleteConversation(conversationUUID); delErr != nil {
				return models.Message{}, fmt.Errorf("deleting conversation after message insert failure: %w", delErr)
			}
		}
		// The unique address/source reservation in the insert transaction wins
		// races between receivers. A losing provisional conversation is gone;
		// acknowledge the already committed copy rather than retrying forever.
		if in.SourceID.String != "" {
			if existing, checkErr := m.messageExistsBySourceID([]string{in.SourceID.String}, in.AddressID); checkErr == nil && existing > 0 {
				return models.Message{}, nil
			}
		}
		return models.Message{}, fmt.Errorf("inserting message: %w", err)
	}

	if isNewConversation {
		if item, err := m.GetConversationListItem(conversationUUID); err == nil {
			m.BroadcastNewConversation(&item)
		}
	}

	// When a customer replies to a continuity emailsync the message to their live chat widget via WebSocket.
	// No-op if the conversation's inbox isn't livechat.

	// Process post-message hooks (automation rules, webhooks, SLA, etc.).
	if err := m.ProcessIncomingMessageHooks(msg, isNewConversation); err != nil {
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

	if conversation.InboxID != in.InboxID || !conversation.AddressID.Valid || conversation.AddressID.Int != in.AddressID {
		return 0, 0, "", nil
	}

	m.lo.Debug("matched conversation by plus-addressed Reply-To", "conversation_uuid", conversation.UUID, "contact_email", in.Contact.Email.String)

	conversationID = conversation.ID
	conversationUUID = conversation.UUID
	senderID = conversation.Contact.ID

	// Already a contact - if same email, return as sender. If different email, let contact resolution find the actual sender.
	if conversation.Contact.Type == umodels.UserTypeContact {
		if !strings.EqualFold(conversation.Contact.Email.String, in.Contact.Email.String) {
			return 0, conversationID, conversationUUID, nil
		}
		return senderID, conversationID, conversationUUID, nil
	}

	return 0, conversationID, conversationUUID, nil
}

// EnqueueIncoming enqueues an incoming message for inserting in db.
func (m *Manager) EnqueueIncoming(message models.IncomingMessage) error {
	m.closedMu.RLock()
	defer m.closedMu.RUnlock()
	if m.closed {
		return errors.New("incoming message queue is closed")
	}
	return m.persistIncoming(message)
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

// extractInlineImageUUIDs returns unique media UUIDs from <img src="..."> URLs in order of first appearance, skipping the cid: form.
func extractInlineImageUUIDs(content string) []string {
	matches := imgSrcPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		url := m[1]
		if strings.HasPrefix(url, "cid:") {
			continue
		}
		u := stringutil.ExtractUUID(url)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

// extractInlineContentIDs returns unique content_ids referenced via <img src="cid:..."> in the body.
func extractInlineContentIDs(content string) []string {
	matches := imgSrcPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		url := m[1]
		if !strings.HasPrefix(url, "cid:") {
			continue
		}
		cid := strings.TrimPrefix(url, "cid:")
		if cid == "" || seen[cid] {
			continue
		}
		seen[cid] = true
		out = append(out, cid)
	}
	return out
}

// rewriteInlineImagesToCID rewrites every <img src="...<uuid>..."> to <img src="cid:ldsk-<uuid>">. Already-cid form is left alone.
func rewriteInlineImagesToCID(content string) string {
	return imgSrcPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := imgSrcPattern.FindStringSubmatch(match)
		url := sub[1]
		if strings.HasPrefix(url, "cid:") {
			return match
		}
		u := stringutil.ExtractUUID(url)
		if u == "" {
			return match
		}
		return strings.Replace(match, url, "cid:"+inlineContentID(u), 1)
	})
}

// uploadMessageAttachments uploads all attachments for a message.
func (m *Manager) uploadMessageAttachments(message *models.Message) error {
	if len(message.Attachments) == 0 {
		return nil
	}

	for _, attachment := range message.Attachments {
		contentID := attachment.ContentID
		if contentID != "" {
			storedCID, exists, mediaUUID := m.findExistingMedia(contentID, message.ConversationUUID)

			// Make body's cid match the stored content_id so the read path can find it.
			if storedCID != contentID {
				message.Content = strings.ReplaceAll(message.Content, fmt.Sprintf("cid:%s", contentID), fmt.Sprintf("cid:%s", storedCID))
			}

			if exists {
				m.lo.Debug("inline attachment exists, reusing", "content_id", storedCID, "media_uuid", mediaUUID)
				continue
			}
			contentID = storedCID
		}

		attachment.Name = stringutil.SanitizeFilename(attachment.Name)

		if len(attachment.Content) == 0 {
			m.lo.Warn("skipping empty attachment", "name", attachment.Name, "content_id", contentID, "content_type", attachment.ContentType, "disposition", attachment.Disposition, "message_source_id", message.SourceID.String, "conversation_uuid", message.ConversationUUID)
			continue
		}

		m.lo.Debug("uploading message attachment", "name", attachment.Name, "content_id", contentID, "size", attachment.Size, "content_type", attachment.ContentType, "disposition", attachment.Disposition)

		// Upload and insert entry in media table.
		attachment.ContentID = contentID
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
			true,         /** private **/
		)
		if err != nil {
			var storageError envelope.Error
			if message.Type == models.MessageIncoming && message.Channel == inbox.ChannelEmail && errors.As(err, &storageError) && storageError.ErrorType == envelope.StorageFullError {
				// Keep the email and any successfully stored attachments. Persist
				// only a descriptor for the missing file, never its in-memory bytes.
				if err := recordUnavailableAttachment(message, attachment); err != nil {
					return err
				}
				continue
			}
			m.lo.Error("failed to upload attachment", "name", attachment.Name, "content_type", attachment.ContentType, "size", attachment.Size, "content_id", contentID, "disposition", attachment.Disposition, "conversation_uuid", message.ConversationUUID, "message_source_id", message.SourceID.String, "error", err)
			return fmt.Errorf("failed to upload media %s: %w", attachment.Name, err)
		}

		// If the attachment is an image, generate and upload a thumbnail. Log any errors and continue.
		attachmentExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(attachment.Name)), ".")
		if slices.Contains(image.Exts, attachmentExt) && image.IsImageByContent(bytes.NewReader(attachment.Content)) {
			if err := m.uploadThumbnailForMedia(media, attachment.Content); err != nil {
				m.lo.Error("error uploading thumbnail", "error", err)
			}
		}

		message.Media = append(message.Media, media)
	}
	return nil
}

// recordUnavailableAttachment preserves missing attachment metadata with the
// message itself, so it survives reloads without consuming durable-media quota.
func recordUnavailableAttachment(message *models.Message, original attachment.Attachment) error {
	meta := map[string]json.RawMessage{}
	if len(message.Meta) > 0 && string(message.Meta) != "null" {
		if err := json.Unmarshal(message.Meta, &meta); err != nil {
			return fmt.Errorf("reading message metadata: %w", err)
		}
	}
	var missing attachment.Attachments
	if raw := meta["unavailable_attachments"]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &missing); err != nil {
			return err
		}
	}
	missing = append(missing, attachment.Attachment{
		UUID: uuid.NewString(), Name: original.Name, Size: original.Size,
		ContentID: original.ContentID, ContentType: original.ContentType,
		Disposition: original.Disposition, Unavailable: true,
		UnavailableReason: "storage_full",
	})
	raw, err := json.Marshal(missing)
	if err != nil {
		return err
	}
	meta["unavailable_attachments"] = raw
	message.Meta, err = json.Marshal(meta)
	return err
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
	conversationID, err = m.messageExistsBySourceID(sourceIDs, in.AddressID)
	if err != nil && err != errConversationNotFound {
		return 0, "", false, err
	}

	// A failed earlier delivery may still be staged. Wait for that parent
	// rather than creating a second thread while its transient failure retries.
	if conversationID == 0 {
		refs := stringutil.RemoveItemByValue(stringutil.RemoveEmpty(sourceIDs), in.SourceID.String)
		if len(refs) > 0 {
			var pending bool
			if err := m.db.Get(&pending, `SELECT EXISTS(SELECT 1 FROM incoming_mail_queue q JOIN email_addresses a ON a.inbox_id=q.inbox_id AND lower(a.address)=lower(COALESCE(NULLIF(q.payload->>'EmailAlias',''),(SELECT address FROM email_addresses WHERE inbox_id=q.inbox_id AND kind='mailbox' LIMIT 1))) WHERE a.id=$1 AND q.completed_at IS NULL AND q.payload->>'SourceID'=ANY($2::text[]))`, in.AddressID, pq.Array(refs)); err != nil {
				return 0, "", false, err
			}
			if pending {
				return 0, "", false, fmt.Errorf("referenced parent email is awaiting ingestion")
			}
		}
	}
	// Conversation not found, create one.
	if conversationID == 0 {
		m.lo.Debug("no conversation found with in-reply-to and references, creating new conversation", "in_reply_to", in.InReplyTo, "references", in.References)
		lastMessage := stringutil.HTML2Text(in.Content)
		lastMessageAt := time.Now()
		conversationMeta := map[string]any{}
		if in.EmailAlias != "" {
			conversationMeta["email_alias"] = in.EmailAlias
		}
		addressID := in.AddressID
		conversationID, conversationUUID, err = m.CreateConversation(in.Contact.ID,
			in.InboxID,
			addressID,
			lastMessage,
			lastMessageAt,
			in.Subject,
			false,            /**append reference number to subject**/
			conversationMeta, /** meta **/
			nil,              /** customer attributes **/
			0,                /** max conversation **/
			0,                /** rate limit window **/
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

// resolveIncomingAddress converts the transport receiver's recipient match into
// a first-class address. The primary mailbox is the safe fallback for mail
// providers which do not retain the envelope recipient in a header.
func (m *Manager) resolveIncomingAddress(inboxID int, recipient string) (int, error) {
	var id int
	if recipient != "" {
		err := m.db.Get(&id, `SELECT id FROM email_addresses
			WHERE inbox_id=$1 AND lower(address)=lower($2)`, inboxID, recipient)
		if err == nil {
			return id, nil
		}
		if err != sql.ErrNoRows {
			return 0, err
		}
	}
	if err := m.db.Get(&id, `SELECT id FROM email_addresses
		WHERE inbox_id=$1 AND kind='mailbox' ORDER BY id LIMIT 1`, inboxID); err != nil {
		return 0, fmt.Errorf("resolving incoming address for inbox %d: %w", inboxID, err)
	}
	return id, nil
}

// messageExistsBySourceID returns conversation ID if a message with any of the given source IDs exists.
func (m *Manager) messageExistsBySourceID(messageSourceIDs []string, addressID int) (int, error) {
	messageSourceIDs = stringutil.RemoveEmpty(messageSourceIDs)
	if len(messageSourceIDs) == 0 {
		return 0, errConversationNotFound
	}
	var conversationID int
	if err := m.q.MessageExistsBySourceID.QueryRow(pq.Array(messageSourceIDs), addressID).Scan(&conversationID); err != nil {
		if err == sql.ErrNoRows {
			return conversationID, errConversationNotFound
		}
		m.lo.Error("error fetching msg from DB", "error", err)
		return conversationID, err
	}
	return conversationID, nil
}

// GetInlineMediaRefs returns media referenced via cid: in the body but linked to other messages (quoted history).
func (m *Manager) GetInlineMediaRefs(message *models.Message) ([]mmodels.Media, error) {
	cids := extractInlineContentIDs(message.Content)
	if len(cids) == 0 {
		return nil, nil
	}
	existing := make(map[string]bool, len(message.Attachments))
	for _, a := range message.Attachments {
		if a.ContentID != "" {
			existing[a.ContentID] = true
		}
	}
	missing := make([]string, 0, len(cids))
	for _, cid := range cids {
		if !existing[cid] {
			missing = append(missing, cid)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	return m.mediaStore.GetByContentIDs(missing, message.ConversationUUID)
}

// fetchMessageAttachments fetches attachments (also inline images) for a single message ID.
func (m *Manager) fetchMessageAttachments(messageID int) (attachment.Attachments, error) {
	var attachments attachment.Attachments

	// Get all media for this message.
	medias, err := m.mediaStore.GetByModel(messageID, mmodels.ModelMessages)
	if err != nil {
		return attachments, fmt.Errorf("error fetching message attachments: %w", err)
	}

	// Fetch blobs for each media item.
	for _, media := range medias {
		blob, err := m.mediaStore.GetBlob(media.UUID)
		if err != nil {
			return attachments, fmt.Errorf("error fetching media blob: %w", err)
		}

		contentID := media.ContentID
		if contentID == "" {
			contentID = media.UUID
		}

		attachment := attachment.Attachment{
			Name:        media.Filename,
			UUID:        media.UUID,
			ContentType: media.ContentType,
			ContentID:   contentID,
			Content:     blob,
			Size:        media.Size,
			Header:      attachment.MakeHeader(media.ContentType, contentID, media.Filename, "base64", media.Disposition.String),
			URL:         m.mediaStore.GetURL(media.UUID, media.ContentType, media.Filename),
		}
		if strings.HasPrefix(media.ContentType, "image/") {
			attachment.ThumbnailURL = m.mediaStore.GetThumbnailURL(media.UUID)
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

// ProcessIncomingMessageHooks handles automation rules, webhooks, SLA events, and other post-processing
// for incoming messages. This allows other channels to insert messages first and then call this
// function to trigger the necessary hooks.
func (m *Manager) ProcessIncomingMessageHooks(message models.Message, isNewConversation bool) error {
	conversationUUID := message.ConversationUUID

	// Start waiting since clock, cleared when agent replies.
	m.StartConversationWaitingSince(conversationUUID, time.Now())

	// Handle new conversation events.
	if isNewConversation {
		conversation, err := m.GetConversation(0, conversationUUID, "")
		if err == nil {
			m.webhookStore.TriggerEvent(wmodels.EventConversationCreated, conversation)
		}
		return nil
	}

	// Reopen conversation if it's not Open.

	systemUser, err := m.userStore.GetSystemUser()
	if err != nil {
		m.lo.Error("error fetching system user", "error", err)
	} else {
		var err error
		if _, err = m.ReOpenConversation(conversationUUID, systemUser); err != nil {
			m.lo.Error("error reopening conversation", "error", err)
		}
	}

	return nil
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

// inlineContentID lowercases the uuid to match the content_id the DB stamps from uuid::TEXT.
func inlineContentID(uuid string) string {
	return "ldsk-" + strings.ToLower(uuid)
}

// findExistingMedia resolves an inbound cid to its stored form: ldsk-* is left as-is, others are namespaced by conversation to avoid cross-conversation collisions.
func (m *Manager) findExistingMedia(rawContentID, conversationUUID string) (string, bool, string) {
	storedCID := rawContentID
	if !strings.HasPrefix(rawContentID, "ldsk-") {
		storedCID = conversationUUID + "_" + rawContentID
	}
	exists, mediaUUID, err := m.mediaStore.ContentIDExists(storedCID, conversationUUID)
	if err != nil {
		m.lo.Error("error checking media existence by content ID", "content_id", storedCID, "error", err)
	}
	return storedCID, exists, mediaUUID
}

// emailFromAddress returns the From header, applying the inbox from-name template for agent senders
// Falls back to the inbox's default from address if the template is empty, the sender is not an agent, or any errors occur.
func (m *Manager) emailFromAddress(inb inbox.Inbox, message models.Message) string {
	from := inb.FromAddress()
	alias := emailAliasFromMessageMeta(message.Meta)
	if alias == "" {
		alias = m.emailAddressForConversationID(message.ConversationID)
	}
	if alias != "" {
		from = alias
	}

	tpl := inb.FromNameTemplate()
	if tpl == "" || message.SenderType != models.SenderTypeAgent {
		return from
	}

	agent, err := m.userStore.GetAgentCachedOrLoad(message.SenderID)
	if err != nil {
		m.lo.Error("error fetching agent for from name template", "error", err, "sender_id", message.SenderID)
		return from
	}
	if agent.IsSystemUser() {
		return from
	}

	addr, err := mail.ParseAddress(from)
	if err != nil {
		m.lo.Error("error parsing inbox from address for name template", "error", err, "from", from)
		return from
	}

	firstName := strings.TrimSpace(agent.FirstName)
	lastName := strings.TrimSpace(agent.LastName)
	data := fromNameVars{
		Agent: fromNameAgent{
			FirstName: firstName,
			LastName:  lastName,
			FullName:  strings.TrimSpace(firstName + " " + lastName),
		},
		Inbox: fromNameInbox{Name: inb.Name()},
	}

	t, err := template.New("from").Parse(tpl)
	if err != nil {
		m.lo.Error("error parsing from name template", "error", err, "template", tpl)
		return from
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		m.lo.Error("error executing from name template", "error", err, "template", tpl)
		return from
	}

	name := strings.TrimSpace(buf.String())
	if name == "" {
		return from
	}
	addr.Name = name
	return addr.String()
}

func emailAliasFromMessageMeta(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		return ""
	}
	alias, _ := meta["email_alias"].(string)
	return strings.TrimSpace(alias)
}

func (m *Manager) emailAddressForConversationID(conversationID int) string {
	if conversationID <= 0 {
		return ""
	}
	var alias string
	if err := m.db.Get(&alias, `SELECT COALESCE(a.address, c.meta->>'email_alias', '')
		FROM conversations c LEFT JOIN email_addresses a ON a.id=c.address_id WHERE c.id=$1`, conversationID); err != nil {
		m.lo.Error("error fetching conversation address", "conversation_id", conversationID, "error", err)
		return ""
	}
	return alias
}

func (m *Manager) emailAddressForConversationUUID(conversationUUID string) string {
	if conversationUUID == "" {
		return ""
	}
	var alias string
	if err := m.db.Get(&alias, `SELECT COALESCE(a.address, c.meta->>'email_alias', '')
		FROM conversations c LEFT JOIN email_addresses a ON a.id=c.address_id WHERE c.uuid=$1`, conversationUUID); err != nil {
		m.lo.Error("error fetching conversation address", "conversation_uuid", conversationUUID, "error", err)
		return ""
	}
	return alias
}
