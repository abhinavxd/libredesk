package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/user"
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

			e.lo.Debug("scanning emails", "mailbox", cfg.Mailbox, "inbox_id", e.Identifier())
			if err := e.processMailbox(ctx, scanInboxSince, cfg); err != nil && err != context.Canceled {
				e.lo.Error("error scanning emails", "error", err)
			}
			e.lo.Debug("finished scanning emails", "mailbox", cfg.Mailbox, "inbox_id", e.Identifier())
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
		fmt.Println("starttls")
		fmt.Println("skip verify", cfg.TLSSkipVerify)
		fmt.Println(address)
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

	e.lo.Debug("searching emails", "since", since, "mailbox", cfg.Mailbox, "inbox_id", e.Identifier())

	// Search for messages in the mailbox.
	searchResults, err := e.searchMessages(client, since)
	if err != nil {
		return fmt.Errorf("error searching messages: %w", err)
	}

	return e.fetchAndProcessMessages(ctx, client, searchResults, e.Identifier())
}

// searchMessages searches for messages in the specified time range.
func (e *Email) searchMessages(client *imapclient.Client, since time.Time) (*imap.SearchData, error) {
	searchCMD := client.Search(&imap.SearchCriteria{
		Since: since,
	},
		&imap.SearchOptions{
			ReturnMin:   true,
			ReturnMax:   true,
			ReturnAll:   true,
			ReturnCount: true,
		},
	)
	return searchCMD.Wait()
}

// fetchAndProcessMessages fetches and processes messages based on the search results.
func (e *Email) fetchAndProcessMessages(ctx context.Context, client *imapclient.Client, searchResults *imap.SearchData, inboxID int) error {
	seqSet := imap.SeqSet{}
	seqSet.AddRange(searchResults.Min, searchResults.Max)

	// Fetch only envelope, body is fetch later if the message is new.
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
	}

	fetchCmd := client.Fetch(seqSet, fetchOptions)

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
			return nil
		}

		// Process message envelope.
		for {
			// Check for context cancellation before processing the next item.
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Fetch the next item in the message.
			fetchItem := msg.Next()
			if fetchItem == nil {
				// No message items left to process.
				break
			}

			// Process the envelope item.
			if item, ok := fetchItem.(imapclient.FetchItemDataEnvelope); ok {
				if err := e.processEnvelope(ctx, client, item.Envelope, msg.SeqNum, inboxID); err != nil && err != context.Canceled {
					e.lo.Error("error processing envelope", "error", err)
				}
			}
		}
	}
}

// processEnvelope processes an email envelope.
func (e *Email) processEnvelope(ctx context.Context, client *imapclient.Client, env *imap.Envelope, seqNum uint32, inboxID int) error {
	if len(env.From) == 0 {
		e.lo.Warn("no sender received for email", "message_id", env.MessageID)
		return nil
	}

	exists, err := e.messageStore.MessageExists(env.MessageID)
	if err != nil {
		e.lo.Error("error checking if message exists", "message_id", env.MessageID)
		return fmt.Errorf("checking if message exists in DB: %w", err)
	}

	if exists {
		return nil
	}

	e.lo.Debug("message does not exist", "message_id", env.MessageID)

	// Make contact.
	firstName, lastName := getContactName(env.From[0])
	var contact = umodels.User{
		InboxID:         inboxID,
		FirstName:       firstName,
		LastName:        lastName,
		SourceChannel:   null.NewString(e.Channel(), true),
		SourceChannelID: null.NewString(env.From[0].Addr(), true),
		Email:           null.NewString(env.From[0].Addr(), true),
		Type:            user.UserTypeContact,
	}

	// Set CC addresses in meta.
	var ccAddr = make([]string, 0, len(env.Cc))
	for _, cc := range env.Cc {
		if cc.Addr() != "" {
			ccAddr = append(ccAddr, cc.Addr())
		}
	}
	meta, err := json.Marshal(map[string]interface{}{
		"cc": ccAddr,
	})
	if err != nil {
		e.lo.Error("error marshalling meta", "error", err)
		return fmt.Errorf("marshalling meta: %w", err)
	}
	incomingMsg := models.IncomingMessage{
		Message: models.Message{
			Channel:    e.Channel(),
			SenderType: conversation.SenderTypeContact,
			Type:       conversation.MessageIncoming,
			InboxID:    inboxID,
			Status:     conversation.MessageStatusReceived,
			Subject:    env.Subject,
			SourceID:   null.StringFrom(env.MessageID),
			Meta:       string(meta),
		},
		Contact: contact,
		InboxID: inboxID,
	}

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
			e.lo.Debug("fetching full message body", "message_id", env.MessageID)
			return e.processFullMessage(fullItem, incomingMsg)
		}
	}
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
		incomingMsg.Message.ContentType = conversation.ContentTypeHTML
		e.lo.Debug("extracted HTML content from parts", "message_id", incomingMsg.Message.SourceID.String, "content", incomingMsg.Message.Content)
	} else if len(envelope.HTML) > 0 {
		incomingMsg.Message.Content = envelope.HTML
		incomingMsg.Message.ContentType = conversation.ContentTypeHTML
	} else if len(envelope.Text) > 0 {
		incomingMsg.Message.Content = envelope.Text
		incomingMsg.Message.ContentType = conversation.ContentTypeText
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
