package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"mime"
	"net/mail"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jhillyerd/enmime/v2"
	"github.com/volatiletech/null/v9"
)

const (
	defaultReadInterval   = time.Duration(5 * time.Minute)
	defaultScanInboxSince = time.Duration(48 * time.Hour)
)

// Charset autodetection is disabled: it overrides the declared charset and misreads mostly-ASCII UTF-8 bodies as ISO-8859-1.
var mimeParser = enmime.NewParser(enmime.DisableCharacterDetection(true))

// ReadIncomingMessages reads and processes incoming messages from an IMAP server based on the provided configuration.
func (e *Email) ReadIncomingMessages(ctx context.Context, cfg imodels.IMAPConfig) error {
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
func (e *Email) processMailbox(ctx context.Context, scanInboxSince time.Duration, cfg imodels.IMAPConfig) error {
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

	// Authenticate based on auth type
	if e.authType == imodels.AuthTypeOAuth2 && e.oauth != nil {
		// Refresh OAuth token if needed
		oauthConfig, _, err := e.refreshOAuthIfNeeded()
		if err != nil {
			return err
		}

		// Use XOAUTH2 authentication
		saslClient := &xoauth2IMAPClient{
			username: cfg.Username,
			token:    oauthConfig.AccessToken,
		}
		if err := client.Authenticate(saslClient); err != nil {
			return fmt.Errorf("error authenticating with OAuth to IMAP server: %w", err)
		}
	} else {
		if err := client.Login(cfg.Username, cfg.Password).Wait(); err != nil {
			return fmt.Errorf("error logging in to the IMAP server: %w", err)
		}
	}

	if _, err := client.Select(cfg.Mailbox, &imap.SelectOptions{ReadOnly: false}).Wait(); err != nil {
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
		UID:      true,
		BodySection: []*imap.FetchItemBodySection{
			{
				Specifier: imap.PartSpecifierHeader,
				HeaderFields: []string{
					headerAutoSubmitted,
					headerAutoreply,
					headerLibredeskLoopPrevention,
					headerAliasVerification,
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
		verificationToken  string
		uid                imap.UID
		extractedMessageID string
	}
	var messages []msgData

	fetchCmd := client.Fetch(seqSet, fetchOptions)

	if e.PrimaryAddress() == "" {
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
			verificationToken  string
			uid                imap.UID
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
				envelope, err := mimeParser.ReadEnvelope(bs.Literal)
				if err != nil {
					e.lo.Error("error reading envelope", "error", err)
					continue
				}
				if isAutoReply(envelope) {
					autoReply = true
				}
				if e.isLoopMessage(envelope) {
					isLoop = true
				}
				verificationToken = strings.TrimSpace(envelope.GetHeader(headerAliasVerification))

				// Extract Message-Id from raw headers as fallback for problematic Message IDs
				extractedMessageID = extractMessageIDFromHeaders(envelope)
			}

			// Envelope.
			if ed, ok := item.(imapclient.FetchItemDataEnvelope); ok {
				env = ed.Envelope
			}
			if uidData, ok := item.(imapclient.FetchItemDataUID); ok {
				uid = uidData.UID
			}
		}

		// Skip if we couldn't get the envelope.
		if env == nil {
			e.lo.Warn("skipping message without envelope", "seq_num", msg.SeqNum, "inbox_id", e.Identifier())
			continue
		}

		messages = append(messages, msgData{env: env, seqNum: msg.SeqNum, uid: uid, autoReply: autoReply, isLoop: isLoop, verificationToken: verificationToken, extractedMessageID: extractedMessageID})
	}

	// Now process each collected message.
	var verificationSeqNums []uint32
	var verificationUIDs []imap.UID
	for _, msgData := range messages {
		// Check for context cancellation before processing each message.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Verification messages must be handled before loop prevention because
		// outgoing verification mail carries the normal LibreDesk loop header.
		if msgData.verificationToken != "" {
			if err := e.processAliasVerification(ctx, client, msgData.seqNum, msgData.verificationToken); err != nil {
				e.lo.Error("error processing alias verification", "error", err, "inbox_id", inboxID)
			}
			verificationSeqNums = append(verificationSeqNums, msgData.seqNum)
			if msgData.uid != 0 {
				verificationUIDs = append(verificationUIDs, msgData.uid)
			}
			continue
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
	if len(verificationSeqNums) > 0 {
		var deleteSet imap.NumSet
		if len(verificationUIDs) == len(verificationSeqNums) {
			uidSet := imap.UIDSet{}
			uidSet.AddNum(verificationUIDs...)
			deleteSet = uidSet
		} else {
			seqSet := imap.SeqSet{}
			seqSet.AddNum(verificationSeqNums...)
			deleteSet = seqSet
		}
		if err := client.Store(deleteSet, &imap.StoreFlags{Op: imap.StoreFlagsAdd, Silent: true, Flags: []imap.Flag{imap.FlagDeleted}}, nil).Close(); err != nil {
			e.lo.Error("error deleting alias verification messages", "error", err, "inbox_id", inboxID)
		} else if uidSet, ok := deleteSet.(imap.UIDSet); ok && client.Caps().Has(imap.CapUIDPlus) {
			if err := client.UIDExpunge(uidSet).Close(); err != nil {
				e.lo.Error("error expunging alias verification messages", "error", err, "inbox_id", inboxID)
			}
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

	// Check if any contact with this email is blocked, if so, ignore the message.
	if blocked, err := e.userStore.IsEmailBlocked(fromAddress); err != nil {
		e.lo.Error("error checking if email is blocked", "email", fromAddress, "error", err)
		return fmt.Errorf("checking if email is blocked: %w", err)
	} else if blocked {
		e.lo.Info("contact email is blocked dropping incoming email", "email", fromAddress)
		return nil
	}

	e.lo.Debug("processing new incoming message", "message_id", messageID, "subject", env.Subject, "from", fromAddress, "inbox_id", inboxID)

	// Make contact.
	firstName, lastName := getContactName(env.From[0])
	contact := models.IncomingContact{
		FirstName: firstName,
		LastName:  lastName,
		Email:     null.StringFrom(fromAddress),
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

	meta, err := json.Marshal(map[string]any{
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
		Channel:  ChannelEmail,
		InboxID:  inboxID,
		Contact:  contact,
		Subject:  env.Subject,
		SourceID: null.StringFrom(messageID),
		Meta:     meta,
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
	envelope, err := mimeParser.ReadEnvelope(item.Literal)
	if err != nil {
		e.lo.Error("error parsing email envelope", "error", err, "message_id", incomingMsg.SourceID.String)
		return fmt.Errorf("parsing email envelope: %w", err)
	}

	// Log any envelope errors.
	for _, err := range envelope.Errors {
		e.lo.Error("error parsing email envelope", "error", err.Error(), "message_id", incomingMsg.SourceID.String)
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
		incomingMsg.Content = allHTML.String()
		incomingMsg.ContentType = models.ContentTypeHTML
		e.lo.Debug("extracted HTML content from parts", "message_id", incomingMsg.SourceID.String, "content", incomingMsg.Content)
	} else if len(envelope.HTML) > 0 {
		incomingMsg.Content = envelope.HTML
		incomingMsg.ContentType = models.ContentTypeHTML
	} else if len(envelope.Text) > 0 {
		incomingMsg.Content = envelope.Text
		incomingMsg.ContentType = models.ContentTypeText
	}

	e.lo.Debug("envelope HTML content", "message_id", incomingMsg.SourceID.String, "content", incomingMsg.Content)
	e.lo.Debug("envelope text content", "message_id", incomingMsg.SourceID.String, "content", envelope.Text)

	// Clean headers
	inReplyTo := strings.ReplaceAll(strings.ReplaceAll(envelope.GetHeader("In-Reply-To"), "<", ""), ">", "")
	references := strings.Fields(envelope.GetHeader("References"))
	for i, ref := range references {
		references[i] = strings.Trim(strings.TrimSpace(ref), " <>")
	}

	incomingMsg.InReplyTo = inReplyTo
	incomingMsg.References = references
	if err := setInboxAddressMeta(&incomingMsg, e.resolveInboxAddress(envelope)); err != nil {
		return err
	}

	// Extract conversation UUID from plus-addressed recipient (e.g., inbox+conv-{uuid}@domain)
	incomingMsg.ConversationUUIDFromReplyTo = extractConversationUUIDFromRecipient(envelope)
	if incomingMsg.ConversationUUIDFromReplyTo != "" {
		e.lo.Debug("extracted conversation UUID from plus-addressed recipient",
			"conversation_uuid", incomingMsg.ConversationUUIDFromReplyTo,
			"message_id", incomingMsg.SourceID.String)
	}

	incomingMsg.Attachments = collectAttachments(envelope)

	incomingMsg.Content = stringutil.SanitizeUTF8(incomingMsg.Content)
	incomingMsg.Subject = stringutil.SanitizeUTF8(incomingMsg.Subject)
	incomingMsg.Contact.FirstName = stringutil.SanitizeUTF8(incomingMsg.Contact.FirstName)
	incomingMsg.Contact.LastName = stringutil.SanitizeUTF8(incomingMsg.Contact.LastName)

	e.lo.Debug("enqueuing incoming email message", "message_id", incomingMsg.SourceID.String,
		"collected_attachments", len(incomingMsg.Attachments),
		"attachments", len(envelope.Attachments), "inline_attachments", len(envelope.Inlines),
		"other_parts", len(envelope.OtherParts))

	return e.messageStore.EnqueueIncoming(incomingMsg)
}

// collectAttachments builds the attachment list from an envelope's attachment, inline, and unclassified parts.
func collectAttachments(envelope *enmime.Envelope) []attachment.Attachment {
	attachments := make([]attachment.Attachment, 0, len(envelope.Attachments)+len(envelope.Inlines)+len(envelope.OtherParts))

	for _, att := range envelope.Attachments {
		attachments = append(attachments, partToAttachment(att, attachment.DispositionAttachment))
	}

	for _, part := range envelope.Inlines {
		attachments = append(attachments, partToAttachment(part, dispositionForPart(part)))
	}

	// OtherParts with neither a ContentID nor a filename are transport noise (DSN reports, signature blobs).
	for _, part := range envelope.OtherParts {
		if part.ContentID == "" && part.FileName == "" {
			continue
		}
		attachments = append(attachments, partToAttachment(part, dispositionForPart(part)))
	}

	return attachments
}

// dispositionForPart returns inline for parts that have a ContentID, attachment otherwise.
func dispositionForPart(part *enmime.Part) string {
	if part.ContentID == "" {
		return attachment.DispositionAttachment
	}
	return attachment.DispositionInline
}

// partToAttachment converts an enmime part and makes up a filename if the part has none.
func partToAttachment(part *enmime.Part, disposition string) attachment.Attachment {
	name := part.FileName
	if name == "" {
		name = "attachment"
		if exts, _ := mime.ExtensionsByType(part.ContentType); len(exts) > 0 {
			name += exts[0]
		}
	}
	return attachment.Attachment{
		Name:        name,
		Content:     part.Content,
		ContentType: part.ContentType,
		ContentID:   part.ContentID,
		Size:        len(part.Content),
		Disposition: disposition,
	}
}

// getContactName extracts the contact's first and last name from the IMAP address.
func getContactName(imapAddr imap.Address) (string, string) {
	first, last := stringutil.SplitName(imapAddr.Name)
	if first == "" {
		return imapAddr.Mailbox, ""
	}
	return first, last
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

// isLoopMessage accepts the UUID identity and legacy owned-address identities during upgrades.
func (e *Email) isLoopMessage(envelope *enmime.Envelope) bool {
	loopHeader := strings.TrimSpace(envelope.GetHeader(headerLibredeskLoopPrevention))
	if loopHeader == "" {
		return false
	}
	return strings.EqualFold(loopHeader, e.uuid) || e.OwnsAddress(loopHeader)
}

func (e *Email) resolveInboxAddress(envelope *enmime.Envelope) string {
	for _, header := range []string{"To", "Cc", "Bcc"} {
		for _, address := range parseHeaderAddresses(envelope.GetHeader(header)) {
			if e.ReceivesAddress(address) {
				normalized, _ := inbox.NormalizeEmailAddress(address)
				return normalized
			}
		}
	}
	for _, header := range []string{"X-Original-To", "Original-Recipient", "Envelope-To", "X-Envelope-To", "Delivered-To"} {
		for _, address := range parseHeaderAddresses(envelope.GetHeader(header)) {
			if e.OwnsAddress(address) {
				normalized, _ := inbox.NormalizeEmailAddress(address)
				return normalized
			}
		}
	}
	return e.PrimaryAddress()
}

func (e *Email) processAliasVerification(ctx context.Context, client *imapclient.Client, seqNum uint32, token string) error {
	if e.aliasVerificationCallback == nil {
		return nil
	}
	failVerification := func(err error) error {
		if callbackErr := e.aliasVerificationCallback(ctx, token, ""); callbackErr != nil {
			return callbackErr
		}
		return err
	}
	seqSet := imap.SeqSet{}
	seqSet.AddNum(seqNum)
	cmd := client.Fetch(seqSet, &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{{}}})
	message := cmd.Next()
	if message == nil {
		return failVerification(fmt.Errorf("verification message not found"))
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		item := message.Next()
		if item == nil {
			return failVerification(fmt.Errorf("verification message body not found"))
		}
		section, ok := item.(imapclient.FetchItemDataBodySection)
		if !ok {
			continue
		}
		envelope, err := mimeParser.ReadEnvelope(section.Literal)
		if err != nil {
			return failVerification(err)
		}
		from := ""
		addresses := parseHeaderAddresses(envelope.GetHeader("From"))
		if len(addresses) > 0 {
			from = addresses[0]
		}
		return e.aliasVerificationCallback(ctx, token, from)
	}
}

func parseHeaderAddresses(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if i := strings.LastIndex(value, ";"); i >= 0 {
		value = strings.TrimSpace(value[i+1:])
	}
	parsed, err := mail.ParseAddressList(value)
	if err != nil {
		if address, singleErr := mail.ParseAddress(value); singleErr == nil {
			parsed = []*mail.Address{address}
		} else {
			return nil
		}
	}
	addresses := make([]string, 0, len(parsed))
	for _, address := range parsed {
		addresses = append(addresses, address.Address)
	}
	return addresses
}

func setInboxAddressMeta(message *models.IncomingMessage, address string) error {
	meta := make(map[string]any)
	if len(message.Meta) > 0 {
		if err := json.Unmarshal(message.Meta, &meta); err != nil {
			return fmt.Errorf("unmarshalling incoming message meta: %w", err)
		}
	}
	meta["inbox_address"] = address
	encoded, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshalling incoming message meta: %w", err)
	}
	message.Meta = encoded
	return nil
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

// extractUUIDFromReplyAddress extracts a UUID from the reply address if present.
// The UUID is expected to be in the format "username+<UUID>@domain" within the email address.
// Returns an empty string if the UUID is not found or invalid.
func (e *Email) extractUUIDFromReplyAddress(address string) string {
	// Remove angle brackets if present
	address = strings.Trim(address, "<>")

	// Check if it contains +
	if !strings.Contains(address, "+") {
		return ""
	}

	// Extract the part between + and @
	parts := strings.Split(address, "@")
	if len(parts) != 2 {
		return ""
	}

	// Get the UUID
	uuid := strings.SplitN(parts[0], "+", 2)[1]
	if uuid == "" {
		return ""
	}

	// Validate UUID format (36 chars with hyphens at specific positions)
	if len(uuid) == 36 &&
		uuid[8] == '-' &&
		uuid[13] == '-' &&
		uuid[18] == '-' &&
		uuid[23] == '-' {
		return uuid
	}

	return ""
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

// extractConversationUUIDFromRecipient extracts conversation UUID from plus-addressed recipient.
// Checks Delivered-To, X-Original-To, and To headers for plus-addressing pattern.
// e.g., support+conv-abc123-def456@company.com → abc123-def456
func extractConversationUUIDFromRecipient(envelope *enmime.Envelope) string {
	headers := []string{"Delivered-To", "X-Original-To", "To"}
	for _, h := range headers {
		addr := envelope.GetHeader(h)
		if uuid := stringutil.ExtractConvUUID(addr); uuid != "" {
			return uuid
		}
	}
	return ""
}
