package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jhillyerd/enmime"
	"github.com/volatiletech/null/v9"
)

const (
	defaultReadInterval   = time.Duration(5 * time.Minute)
	defaultScanInboxSince = time.Duration(48 * time.Hour)
)

// ReadIncomingMessages reads and processes incoming messages from an IMAP server based on the provided configuration.
func (e *Email) ReadIncomingMessages(ctx context.Context, cfg IMAPConfig) error {
	readInterval, err := time.ParseDuration(cfg.ReadInterval)
	if err != nil {
		e.lo.Warn("could not parse IMAP read interval, using the default read interval of 5 minutes", "interval", cfg.ReadInterval, "inbox_id", e.Identifier(), "error", err)
		readInterval = defaultReadInterval
	}

	scanInboxSince, err := time.ParseDuration(cfg.ScanInboxSince)
	if err != nil {
		e.lo.Warn("could not parse IMAP scan inbox since duration, using the default value of 48 hours", "interval", cfg.ScanInboxSince, "inbox_id", e.Identifier(), "error", err)
		scanInboxSince = defaultScanInboxSince
	}

	readTicker := time.NewTicker(readInterval)
	defer readTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-readTicker.C:
			// If the ticker interval is too short, it may trigger while the previous `processMailbox` call is still running,
			// leading to overlapping executions or delays in handling context cancellation, check if the context is already done.
			if ctx.Err() != nil {
				return nil
			}

			if err := e.processMailbox(ctx, scanInboxSince, cfg); err != nil && err != context.Canceled {
				e.lo.Error("error searching emails", "error", err)
			}
			e.lo.Info("email search complete", "mailbox", cfg.Mailbox, "inbox_id", e.Identifier())
		}
	}
}

// processMailbox processes emails in the specified mailbox.
func (e *Email) processMailbox(ctx context.Context, scanInboxSince time.Duration, cfg IMAPConfig) error {
	var (
		client *imapclient.Client
		err    error
	)

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	imapOptions := &imapclient.Options{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		},
	}
	switch cfg.TLSType {
	case "none":
		client, err = imapclient.DialInsecure(address, imapOptions)
	case "starttls":
		client, err = imapclient.DialStartTLS(address, imapOptions)
	case "tls":
		client, err = imapclient.DialTLS(address, imapOptions)
	default:
		return fmt.Errorf("unknown IMAP TLS type: %q", cfg.TLSType)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}

	defer client.Logout()
	if err := client.Login(cfg.Username, cfg.Password).Wait(); err != nil {
		return fmt.Errorf("error logging in to the IMAP server: %w", err)
	}

	if _, err := client.Select(cfg.Mailbox, &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return fmt.Errorf("error selecting mailbox: %w", err)
	}

	// Scan emails since the specified duration.
	since := time.Now().Add(-scanInboxSince)

	e.lo.Info("searching emails", "since", since, "mailbox", cfg.Mailbox, "inbox_id", e.Identifier())

	// Search for messages in the mailbox.
	searchResults, err := e.searchMessages(client, since)
	if err != nil {
		return fmt.Errorf("error searching messages: %w", err)
	}

	return e.fetchAndProcessMessages(ctx, client, searchResults, e.Identifier())
}

// searchMessages searches for messages in the specified time range.
// Uses ESEARCH if supported by the server, otherwise falls back to standard SEARCH.
func (e *Email) searchMessages(client *imapclient.Client, since time.Time) (*imap.SearchData, error) {
	criteria := &imap.SearchCriteria{
		Since: since,
	}

	// Attempt ESEARCH if server supports it
	if client.Caps().Has(imap.CapESearch) {
		opts := &imap.SearchOptions{
			ReturnMin:   true,
			ReturnMax:   true,
			ReturnAll:   true,
			ReturnCount: true,
		}

		result, err := client.Search(criteria, opts).Wait()
		if err == nil {
			return result, nil
		}

		e.lo.Warn("ESEARCH failed, falling back to standard SEARCH", "error", err, "inbox_id", e.Identifier())
	}

	return client.Search(criteria, nil).Wait()
}

// fetchAndProcessMessages fetches and processes messages based on the search results.
func (e *Email) fetchAndProcessMessages(ctx context.Context, client *imapclient.Client, searchResults *imap.SearchData, inboxID int) error {
	seqSet := imap.SeqSet{}
	if searchResults.Min > 0 && searchResults.Max > 0 {
		e.lo.Debug("using ESEARCH range", "min", searchResults.Min, "max", searchResults.Max, "inbox_id", inboxID)
		seqSet.AddRange(searchResults.Min, searchResults.Max)
	} else if seqNums := searchResults.AllSeqNums(); len(seqNums) > 0 {
		e.lo.Debug("using SEARCH fallback (no ESEARCH support)", "count", len(seqNums), "inbox_id", inboxID)
		seqSet.AddNum(seqNums...)
	} else {
		// No results found
		e.lo.Debug("no messages found in search results", "inbox_id", inboxID)
		return nil
	}

	// Fetch envelope and headers needed for auto-reply detection.
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
		BodySection: []*imap.FetchItemBodySection{
			{
				Specifier: imap.PartSpecifierHeader,
				HeaderFields: []string{
					headerAutoSubmitted,
					headerAutoreply,
					headerLibredeskLoopPrevention,
					headerMessageID,
				},
			},
		},
	}

	// Collect messages to process later.
	type msgData struct {
		env                *imap.Envelope
		seqNum             uint32
		autoReply          bool
		isLoop             bool
		extractedMessageID string
	}
	var messages []msgData

	fetchCmd := client.Fetch(seqSet, fetchOptions)

	// Extract the inbox email address.
	inboxEmail, err := stringutil.ExtractEmail(e.FromAddress())
	if err != nil {
		e.lo.Error("failed to extract email address from the 'From' header", "error", err)
		return fmt.Errorf("failed to extract email address from 'From' header: %w", err)
	}
	if inboxEmail == "" {
		e.lo.Error("inbox email address is empty, cannot process messages", "inbox_id", e.Identifier())
		return fmt.Errorf("inbox (%d) email address is empty, cannot process messages", e.Identifier())
	}
	for {
		// Check for context cancellation before fetching the next message.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Fetch the next message.
		msg := fetchCmd.Next()
		if msg == nil {
			// No more messages to process.
			break
		}

		var (
			env                *imap.Envelope
			autoReply          bool
			isLoop             bool
			extractedMessageID string
		)
		// Process all fetch items for the current message.
		for {
			// Check for context cancellation before processing the next item.
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Fetch the next item in the message.
			item := msg.Next()
			if item == nil {
				// No message items left to process.
				break
			}

			// Body section.
			if bs, ok := item.(imapclient.FetchItemDataBodySection); ok && bs.Literal != nil {
				envelope, err := enmime.ReadEnvelope(bs.Literal)
				if err != nil {
					e.lo.Error("error reading envelope", "error", err)
					continue
				}
				if isAutoReply(envelope) {
					autoReply = true
				}
				if isLoopMessage(envelope, inboxEmail) {
					isLoop = true
				}

				// Extract Message-Id from raw headers as fallback for problematic Message IDs
				extractedMessageID = extractMessageIDFromHeaders(envelope)
			}

			// Envelope.
			if ed, ok := item.(imapclient.FetchItemDataEnvelope); ok {
				env = ed.Envelope
			}
		}

		// Skip if we couldn't get the envelope.
		if env == nil {
			e.lo.Warn("skipping message without envelope", "seq_num", msg.SeqNum, "inbox_id", e.Identifier())
			continue
		}

		messages = append(messages, msgData{env: env, seqNum: msg.SeqNum, autoReply: autoReply, isLoop: isLoop, extractedMessageID: extractedMessageID})
	}

	// Now process each collected message.
	for _, msgData := range messages {
		// Check for context cancellation before processing each message.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip if this is an auto-reply message.
		if msgData.autoReply {
			e.lo.Info("skipping auto-reply message", "subject", msgData.env.Subject, "message_id", msgData.env.MessageID)
			continue
		}

		// Skip if this message is a loop prevention message.
		if msgData.isLoop {
			e.lo.Info("skipping message with loop prevention header", "subject", msgData.env.Subject, "message_id", msgData.env.MessageID)
			continue
		}

		// Process the envelope.
		if err := e.processEnvelope(ctx, client, msgData.env, msgData.seqNum, inboxID, msgData.extractedMessageID); err != nil && err != context.Canceled {
			e.lo.Error("error processing envelope", "error", err)
		}
	}

	return nil
}

// processEnvelope processes a single email envelope.
func (e *Email) processEnvelope(ctx context.Context, client *imapclient.Client, env *imap.Envelope, seqNum uint32, inboxID int, extractedMessageID string) error {
	if len(env.From) == 0 {
		e.lo.Warn("no sender received for email", "message_id", env.MessageID)
		return nil
	}
	var fromAddress = strings.ToLower(env.From[0].Addr())

	// Determine final Message ID - prefer IMAP-parsed, fallback to raw header extraction
	messageID := env.MessageID
	if messageID == "" {
		messageID = extractedMessageID
		if messageID != "" {
			e.lo.Debug("using raw header Message-ID as fallback for malformed ID", "message_id", messageID, "subject", env.Subject, "from", fromAddress)
		}
	}

	// Drop message if we still don't have a valid Message ID
	if messageID == "" {
		e.lo.Error("dropping message: no valid Message-ID found in IMAP parsing or raw headers", "subject", env.Subject, "from", fromAddress)
		return nil
	}

	// Check if the message already exists in the database; if it does, ignore it.
	exists, err := e.messageStore.MessageExists(messageID)
	if err != nil {
		e.lo.Error("error checking if message exists", "message_id", messageID)
		return fmt.Errorf("checking if message exists in DB: %w", err)
	}
	if exists {
		return nil
	}

	// Check if contact with this email is blocked / disabed, if so, ignore the message.
	if contact, err := e.userStore.GetContact(0, fromAddress); err != nil {
		envErr, ok := err.(envelope.Error)
		if !ok || envErr.ErrorType != envelope.NotFoundError {
			e.lo.Error("error checking if user is blocked", "email", fromAddress, "error", err)
			return fmt.Errorf("checking if user is blocked: %w", err)
		}
	} else if !contact.Enabled {
		e.lo.Debug("contact is blocked, ignoring message", "email", fromAddress)
		return nil
	}

	e.lo.Debug("processing new incoming message", "message_id", messageID, "subject", env.Subject, "from", fromAddress, "inbox_id", inboxID)

	// Make contact.
	firstName, lastName := getContactName(env.From[0])
	var contact = umodels.User{
		InboxID:         inboxID,
		FirstName:       firstName,
		LastName:        lastName,
		SourceChannel:   null.NewString(e.Channel(), true),
		SourceChannelID: null.NewString(fromAddress, true),
		Email:           null.NewString(fromAddress, true),
		Type:            umodels.UserTypeContact,
	}

	// Lowercase and set the `to`, `cc`, `from` and `bcc` addresses in message meta.
	var ccAddr = make([]string, 0, len(env.Cc))
	var toAddr = make([]string, 0, len(env.To))
	var bccAddr = make([]string, 0, len(env.Bcc))
	var fromAddr = make([]string, 0, len(env.From))
	for _, cc := range env.Cc {
		if cc.Addr() != "" {
			ccAddr = append(ccAddr, strings.ToLower(cc.Addr()))
		}
	}
	for _, to := range env.To {
		if to.Addr() != "" {
			toAddr = append(toAddr, strings.ToLower(to.Addr()))
		}
	}
	for _, bcc := range env.Bcc {
		if bcc.Addr() != "" {
			bccAddr = append(bccAddr, strings.ToLower(bcc.Addr()))
		}
	}
	for _, from := range env.From {
		if from.Addr() != "" {
			fromAddr = append(fromAddr, strings.ToLower(from.Addr()))
		}
	}

	meta, err := json.Marshal(map[string]interface{}{
		"from":    fromAddr,
		"cc":      ccAddr,
		"bcc":     bccAddr,
		"to":      toAddr,
		"subject": env.Subject,
	})
	if err != nil {
		e.lo.Error("error marshalling meta", "error", err)
		return fmt.Errorf("marshalling meta: %w", err)
	}
	incomingMsg := models.IncomingMessage{
		Message: models.Message{
			Channel:    e.Channel(),
			SenderType: models.SenderTypeContact,
			Type:       models.MessageIncoming,
			InboxID:    inboxID,
			Status:     models.MessageStatusReceived,
			Subject:    env.Subject,
			SourceID:   null.StringFrom(messageID),
			Meta:       meta,
		},
		Contact: contact,
		InboxID: inboxID,
	}

	// Fetch full message body.
	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{{}},
	}
	seqSet := imap.SeqSet{}
	seqSet.AddNum(seqNum)

	fullFetchCmd := client.Fetch(seqSet, fetchOptions)
	fullMsg := fullFetchCmd.Next()
	if fullMsg == nil {
		return nil
	}

	// Fetch full message.
	for {
		// Check for context cancellation before processing the next item.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fullFetchItem := fullMsg.Next()
		if fullFetchItem == nil {
			return nil
		}

		if fullItem, ok := fullFetchItem.(imapclient.FetchItemDataBodySection); ok {
			e.lo.Debug("fetching full message body", "message_id", messageID)
			return e.processFullMessage(fullItem, incomingMsg)
		}
	}
}

// processFullMessage processes the full message and enqueues it for inserting into the database.
func (e *Email) processFullMessage(item imapclient.FetchItemDataBodySection, incomingMsg models.IncomingMessage) error {
	envelope, err := enmime.ReadEnvelope(item.Literal)
	if err != nil {
		e.lo.Error("error parsing email envelope", "error", err, "message_id", incomingMsg.Message.SourceID.String)
		for _, err := range envelope.Errors {
			e.lo.Error("error parsing email envelope. envelope_error: ", "error", err.Error(), "message_id", incomingMsg.Message.SourceID.String)
		}
		return fmt.Errorf("parsing email envelope: %w", err)
	}

	// Log any envelope errors.
	for _, err := range envelope.Errors {
		e.lo.Error("error parsing email envelope", "error", err.Error(), "message_id", incomingMsg.Message.SourceID.String)
	}

	// Extract all HTML content by traversing the tree
	var allHTML strings.Builder
	if envelope.Root != nil {
		htmlParts := extractAllHTMLParts(envelope.Root)
		if len(htmlParts) > 0 {
			allHTML.WriteString("<div>")
			for _, part := range htmlParts {
				allHTML.WriteString(part)
			}
			allHTML.WriteString("</div>")
		}
	}

	// Set message content - prioritize combined HTML
	if allHTML.Len() > 0 {
		incomingMsg.Message.Content = allHTML.String()
		incomingMsg.Message.ContentType = models.ContentTypeHTML
		e.lo.Debug("extracted HTML content from parts", "message_id", incomingMsg.Message.SourceID.String, "content", incomingMsg.Message.Content)
	} else if len(envelope.HTML) > 0 {
		incomingMsg.Message.Content = envelope.HTML
		incomingMsg.Message.ContentType = models.ContentTypeHTML
	} else if len(envelope.Text) > 0 {
		incomingMsg.Message.Content = envelope.Text
		incomingMsg.Message.ContentType = models.ContentTypeText
	}

	e.lo.Debug("envelope HTML content", "message_id", incomingMsg.Message.SourceID.String, "content", incomingMsg.Message.Content)
	e.lo.Debug("envelope text content", "message_id", incomingMsg.Message.SourceID.String, "content", envelope.Text)

	// Clean headers
	inReplyTo := strings.ReplaceAll(strings.ReplaceAll(envelope.GetHeader("In-Reply-To"), "<", ""), ">", "")
	references := strings.Fields(envelope.GetHeader("References"))
	for i, ref := range references {
		references[i] = strings.Trim(strings.TrimSpace(ref), " <>")
	}

	incomingMsg.Message.InReplyTo = inReplyTo
	incomingMsg.Message.References = references

	// Process attachments
	for _, att := range envelope.Attachments {
		incomingMsg.Message.Attachments = append(incomingMsg.Message.Attachments, attachment.Attachment{
			Name:        att.FileName,
			Content:     att.Content,
			ContentType: att.ContentType,
			ContentID:   att.ContentID,
			Size:        len(att.Content),
			Disposition: attachment.DispositionAttachment,
		})
	}

	// Process inlines - treat ones without ContentID as regular attachments
	for _, inline := range envelope.Inlines {
		disposition := attachment.DispositionInline
		if inline.ContentID == "" {
			disposition = attachment.DispositionAttachment
		}

		incomingMsg.Message.Attachments = append(incomingMsg.Message.Attachments, attachment.Attachment{
			Name:        inline.FileName,
			Content:     inline.Content,
			ContentType: inline.ContentType,
			ContentID:   inline.ContentID,
			Size:        len(inline.Content),
			Disposition: disposition,
		})
	}

	e.lo.Debug("enqueuing incoming email message", "message_id", incomingMsg.Message.SourceID.String,
		"attachments", len(envelope.Attachments), "inline_attachments", len(envelope.Inlines))

	if err := e.messageStore.EnqueueIncoming(incomingMsg); err != nil {
		return err
	}
	return nil
}

// getContactName extracts the contact's first and last name from the IMAP address.
func getContactName(imapAddr imap.Address) (string, string) {
	from := strings.TrimSpace(imapAddr.Name)
	names := strings.Fields(from)
	if len(names) == 0 {
		return imapAddr.Host, ""
	}
	if len(names) == 1 {
		return names[0], ""
	}
	return names[0], names[1]
}

// isAutoReply checks if a given email envelope indicates an auto-reply message.
func isAutoReply(envelope *enmime.Envelope) bool {
	if as := strings.ToLower(strings.TrimSpace(envelope.GetHeader("Auto-Submitted"))); as != "" && as != "no" {
		return true
	}
	if strings.TrimSpace(envelope.GetHeader("X-Autoreply")) != "" {
		return true
	}
	return false
}

// isLoopMessage returns true if the email is a loop prevention message. i.e., it has the `X-Libredesk-Loop-Prevention` header with the inbox email address.
func isLoopMessage(envelope *enmime.Envelope, inboxEmailaddress string) bool {
	loopHeader := envelope.GetHeader(headerLibredeskLoopPrevention)
	if loopHeader == "" {
		return false
	}
	return strings.EqualFold(loopHeader, inboxEmailaddress)
}

// extractAllHTMLParts extracts all HTML parts from the given enmime part by traversing the tree.
func extractAllHTMLParts(part *enmime.Part) []string {
	var htmlParts []string

	// Check current part
	if strings.HasPrefix(part.ContentType, "text/html") && len(part.Content) > 0 {
		htmlParts = append(htmlParts, string(part.Content))
	}

	// Process children recursively
	for child := part.FirstChild; child != nil; child = child.NextSibling {
		childParts := extractAllHTMLParts(child)
		htmlParts = append(htmlParts, childParts...)
	}

	return htmlParts
}

// extractMessageIDFromHeaders extracts and cleans the Message-ID from email headers.
// This function handles problematic Message IDs by extracting them from raw headers
// and cleaning them of angle brackets and whitespace.
func extractMessageIDFromHeaders(envelope *enmime.Envelope) string {
	if rawMessageID := envelope.GetHeader(headerMessageID); rawMessageID != "" {
		return strings.TrimSpace(strings.Trim(rawMessageID, "<>"))
	}
	return ""
}
