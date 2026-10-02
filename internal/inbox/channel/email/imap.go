package email

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jakedolan443/fernmail/internal/inbox"
	"io"
	"mime"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jakedolan443/fernmail/internal/attachment"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"
	"github.com/jakedolan443/fernmail/internal/stringutil"
	"github.com/jhillyerd/enmime/v2"
	"github.com/volatiletech/null/v9"
)

const (
	defaultReadInterval = time.Duration(5 * time.Minute)
)

// Charset autodetection is disabled: it overrides the declared charset and misreads mostly-ASCII UTF-8 bodies as ISO-8859-1.
var mimeParser = enmime.NewParser(enmime.DisableCharacterDetection(true))

var errIMAPBacklog = errors.New("mailbox has more UIDs to synchronize")

// ReadIncomingMessages reads and processes incoming messages from an IMAP server based on the provided configuration.
func (e *Email) ReadIncomingMessages(ctx context.Context, cfg imodels.IMAPConfig) error {
	readInterval, err := time.ParseDuration(cfg.ReadInterval)
	if err != nil || readInterval <= 0 {
		e.lo.Warn("could not parse IMAP read interval, using the default read interval of 5 minutes", "interval", cfg.ReadInterval, "inbox_id", e.Identifier(), "error", err)
		readInterval = defaultReadInterval
	}

	for {
		if ctx.Err() != nil {
			return nil
		}
		err := e.processMailbox(ctx, 0, cfg)
		delay := readInterval
		switch {
		case errors.Is(err, errIMAPBacklog):
			delay = time.Second
		case errors.Is(err, inbox.ErrIncomingQueueFull):
			delay = 5 * time.Second
		case err != nil && ctx.Err() == nil:
			e.lo.Error("synchronizing mailbox", "error", err)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

// processMailbox processes emails in the specified mailbox.
func (e *Email) processMailbox(ctx context.Context, _ time.Duration, cfg imodels.IMAPConfig) error {
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

	defer client.Close()
	cancelled := make(chan struct{})
	defer close(cancelled)
	go func() {
		select {
		case <-ctx.Done():
			client.Close()
		case <-cancelled:
		}
	}()

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

	selected, err := client.Select(cfg.Mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return fmt.Errorf("selecting mailbox: %w", err)
	}
	if selected.UIDValidity == 0 {
		return fmt.Errorf("mailbox omitted UIDVALIDITY")
	}
	// A fresh account or UIDVALIDITY reset performs a full backfill. The legacy
	// scan window never bounds recovery: already-durable UIDs are cheap to skip.
	key := fmt.Sprintf("%s:%d/%s/%s", cfg.Host, cfg.Port, cfg.Username, cfg.Mailbox)
	cursor, failed, err := e.messageStore.IMAPState(e.id, key, selected.UIDValidity)
	if err != nil {
		return err
	}
	pending := imap.UIDSet{}
	pending.AddRange(imap.UID(cursor)+1, 0)
	result, err := client.UIDSearch(&imap.SearchCriteria{UID: []imap.UIDSet{pending}}, nil).Wait()
	if err != nil {
		return err
	}
	uids := map[uint32]bool{}
	newUIDs := result.AllUIDs()
	sort.Slice(newUIDs, func(i, j int) bool { return newUIDs[i] < newUIDs[j] })
	for _, uid := range newUIDs {
		if uint32(uid) > cursor {
			uids[uint32(uid)] = true
			if len(uids) >= 100 {
				break
			}
		}
	}
	for _, uid := range failed {
		uids[uid] = true
	}
	ordered := make([]uint32, 0, len(uids))
	for uid := range uids {
		ordered = append(ordered, uid)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	for _, uid := range ordered {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		failure := e.fetchUID(ctx, client, key, selected.UIDValidity, uid)
		// Backpressure is not a poison message: leave this UID unacknowledged
		// so the next poll starts here when workers free staging capacity.
		if errors.Is(failure, inbox.ErrIncomingQueueFull) {
			return inbox.ErrIncomingQueueFull
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := e.messageStore.RecordIMAPResult(e.id, key, selected.UIDValidity, uid, failure); err != nil {
			return err
		}
		if failure != nil {
			e.lo.Error("incoming email retained for retry", "uid", uid, "error", failure)
		}
	}
	var newestProcessed uint32
	for uid := range uids {
		if uid > newestProcessed {
			newestProcessed = uid
		}
	}
	for _, uid := range newUIDs {
		if uint32(uid) > cursor && uint32(uid) > newestProcessed {
			return errIMAPBacklog
		}
	}
	return nil
}

func (e *Email) fetchUID(ctx context.Context, client *imapclient.Client, key string, validity, uid uint32) error {
	set := imap.UIDSet{}
	set.AddNum(imap.UID(uid))
	metadata, err := client.Fetch(set, &imap.FetchOptions{UID: true, Envelope: true, RFC822Size: true}).Collect()
	if err != nil {
		return err
	}
	if len(metadata) == 0 {
		return nil
	} // Expunged from the provider before retrieval.
	item := metadata[0]
	if shouldSkipMessage(item.RFC822Size, e.incomingMessageSizeLimit()) {
		return fmt.Errorf("message size %d exceeds configured limit", item.RFC822Size)
	}
	if item.Envelope == nil {
		return fmt.Errorf("missing message envelope")
	}
	incoming, err := incomingFromEnvelope(item.Envelope, e.id, key, validity, uid)
	if err != nil {
		return err
	}
	fetch := client.Fetch(set, &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{{Peek: true}}})
	defer fetch.Close()
	received := false
	for message := fetch.Next(); message != nil; message = fetch.Next() {
		for part := message.Next(); part != nil; part = message.Next() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if body, ok := part.(imapclient.FetchItemDataBodySection); ok {
				received = true
				if err = e.processFullMessage(body, incoming); err != nil {
					return err
				}
			}
		}
	}
	if err := fetch.Close(); err != nil {
		return err
	}
	if !received {
		return fmt.Errorf("message body unavailable")
	}
	return nil
}

func incomingFromEnvelope(env *imap.Envelope, inboxID int, key string, validity, uid uint32) (models.IncomingMessage, error) {
	if len(env.From) == 0 {
		return models.IncomingMessage{}, fmt.Errorf("email has no sender")
	}
	addresses := func(items []imap.Address) []string {
		out := []string{}
		for _, item := range items {
			if item.Addr() != "" {
				out = append(out, strings.ToLower(item.Addr()))
			}
		}
		return out
	}
	meta, err := json.Marshal(map[string]any{"from": addresses(env.From), "to": addresses(env.To), "cc": addresses(env.Cc), "bcc": addresses(env.Bcc), "reply_to": addresses(env.ReplyTo), "subject": env.Subject, "date": env.Date})
	if err != nil {
		return models.IncomingMessage{}, err
	}
	first, last := getContactName(env.From[0])
	sourceID := env.MessageID
	if sourceID == "" {
		sourceID = fmt.Sprintf("imap-%x@local.invalid", sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%d:%d", inboxID, key, validity, uid))))
	}
	return models.IncomingMessage{Channel: ChannelEmail, InboxID: inboxID, MailboxKey: key, UIDValidity: validity, UID: uid, Contact: models.IncomingContact{FirstName: first, LastName: last, Email: null.StringFrom(strings.ToLower(env.From[0].Addr()))}, Subject: env.Subject, SourceID: null.StringFrom(sourceID), Meta: meta}, nil
}

// processFullMessage processes the full message and enqueues it for inserting into the database.
func (e *Email) processFullMessage(item imapclient.FetchItemDataBodySection, incomingMsg models.IncomingMessage) error {
	var body io.Reader = item.Literal
	var limitedBody *io.LimitedReader
	maxMessageSize := e.incomingMessageSizeLimit()
	if maxMessageSize > 0 {
		// RFC822.SIZE is the normal guard. This second ceiling protects us from
		// servers that omit or misreport it while still draining the literal so
		// the IMAP connection remains usable.
		limitedBody = &io.LimitedReader{R: item.Literal, N: maxMessageSize + 1}
		body = limitedBody
	}
	envelope, err := mimeParser.ReadEnvelope(body)
	tooLarge := false
	if limitedBody != nil {
		// Ensure the parser cannot return successfully after only reading a
		// prefix. The one-byte-over-limit probe is discarded, never retained.
		_, _ = io.Copy(io.Discard, limitedBody)
		tooLarge = limitedBody.N == 0
		_, _ = io.Copy(io.Discard, item.Literal)
	}
	if tooLarge {
		e.lo.Warn("incoming email exceeded configured size limit",
			"message_id", incomingMsg.SourceID.String,
			"max_size_bytes", maxMessageSize)
		return fmt.Errorf("incoming email exceeds configured size limit of %d bytes", maxMessageSize)
	}
	if err != nil {
		if limitedBody != nil {
			e.lo.Warn("unable to parse incoming email within configured size limit",
				"message_id", incomingMsg.SourceID.String,
				"max_size_bytes", maxMessageSize)
		}
		e.lo.Error("error parsing email envelope", "error", err, "message_id", incomingMsg.SourceID.String)
		return fmt.Errorf("parsing email envelope: %w", err)
	}

	if id := extractMessageIDFromHeaders(envelope); id != "" {
		incomingMsg.SourceID = null.StringFrom(id)
	}
	var meta map[string]any
	if json.Unmarshal(incomingMsg.Meta, &meta) != nil {
		meta = map[string]any{}
	}
	meta["auto_submitted"] = envelope.GetHeader("Auto-Submitted")
	meta["auto_reply"] = isAutoReply(envelope)
	incomingMsg.Meta, _ = json.Marshal(meta)

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
	incomingMsg.EmailAlias = resolveRecipientAlias(e.emailAliases, map[string]string{
		"Delivered-To":  envelope.GetHeader("Delivered-To"),
		"X-Original-To": envelope.GetHeader("X-Original-To"),
		"To":            envelope.GetHeader("To"),
		"Cc":            envelope.GetHeader("Cc"),
	})
	if incomingMsg.EmailAlias != "" {
		var meta map[string]any
		if err := json.Unmarshal(incomingMsg.Meta, &meta); err != nil {
			meta = map[string]any{}
		}
		meta["email_alias"] = incomingMsg.EmailAlias
		if incomingMsg.Meta, err = json.Marshal(meta); err != nil {
			return fmt.Errorf("marshalling email alias metadata: %w", err)
		}
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

func shouldSkipMessage(size, maxSize int64) bool {
	return maxSize > 0 && size > maxSize
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
