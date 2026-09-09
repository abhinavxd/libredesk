package email

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/textproto"
	"slices"
	"sort"
	"strings"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// commonTrashMailboxes are the Trash folder names to try on servers that do not advertise RFC 6154
// SPECIAL-USE, in preference order. Only a name the server itself lists is ever used.
var commonTrashMailboxes = []string{"Trash", "INBOX.Trash", "INBOX/Trash", "Deleted Items", "Deleted Messages"}

// Reasons reported with a not_purged or failed outcome.
const (
	reasonNoTrashNoUIDPlus = "the mailbox has no Trash folder and the server lacks UIDPLUS"
	reasonNoMoveNoUIDPlus  = "the server supports neither MOVE nor UIDPLUS"
)

// purgeClient is the slice of an IMAP client the purge needs. The folder discovery, the Message-ID
// matching and the move-or-delete decision are written against it so they can be tested without a
// server; imapPurgeClient is the only production implementation.
type purgeClient interface {
	HasCap(imap.Cap) bool
	List(ref, pattern string, options *imap.ListOptions) ([]*imap.ListData, error)
	SearchMessageID(messageID string) ([]imap.UID, error)
	FetchMessageIDs(uids imap.UIDSet) (map[imap.UID]string, error)
	Move(uids imap.UIDSet, mailbox string) error
	Copy(uids imap.UIDSet, mailbox string) error
	MarkDeleted(uids imap.UIDSet) error
	UIDExpunge(uids imap.UIDSet) error
}

// PurgeMessages takes the mails carrying the given Message-IDs out of the inbox's IMAP mailboxes and
// reports what happened to each of them.
//
// The desk deduplicates incoming mail against the messages it still holds, so a mail left in the
// scanned mailbox after its conversation is deleted is imported again as a brand new conversation on
// the next scan. The mails are therefore moved to the mailbox's Trash folder: the scan reads only
// the configured mailbox, so a mail in Trash never comes back, and a deletion made by mistake can
// still be recovered from any mail client. Mails are expunged outright only when the account has no
// Trash folder at all.
func (e *Email) PurgeMessages(ctx context.Context, messageIDs []string) (imodels.MailPurgeResult, error) {
	var result imodels.MailPurgeResult
	if len(messageIDs) == 0 {
		return result, nil
	}

	// Mails still to be found. A Message-ID is dropped as soon as one mailbox yields it.
	pending := make(map[string]struct{}, len(messageIDs))
	for _, messageID := range messageIDs {
		if messageID == "" {
			continue
		}
		pending[messageID] = struct{}{}
		result.Record(messageID, imodels.MailNotFound, "")
	}
	if len(pending) == 0 {
		return result, nil
	}

	if len(e.imapCfg) == 0 {
		err := fmt.Errorf("inbox %d has no IMAP configuration to purge from", e.Identifier())
		recordAll(&result, pending, imodels.MailPurgeFailed, err.Error())
		return result, err
	}

	var lastErr error
	for _, cfg := range e.imapCfg {
		if len(pending) == 0 {
			break
		}
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := e.purgeFromMailbox(ctx, cfg, pending, &result); err != nil {
			e.lo.Error("error purging mails from mailbox", "mailbox", cfg.Mailbox, "inbox_id", e.Identifier(), "error", err)
			recordAll(&result, pending, imodels.MailPurgeFailed, err.Error())
			lastErr = err
		}
	}
	return result, lastErr
}

// purgeFromMailbox opens the mailbox and hands it to purgeMailbox.
func (e *Email) purgeFromMailbox(ctx context.Context, cfg imodels.IMAPConfig, pending map[string]struct{}, result *imodels.MailPurgeResult) error {
	client, err := e.dialIMAP(cfg)
	if err != nil {
		return err
	}
	defer client.Logout()

	// Read-write, unlike the scan, which selects the mailbox read-only.
	if _, err := client.Select(cfg.Mailbox, nil).Wait(); err != nil {
		return fmt.Errorf("error selecting mailbox %q: %w", cfg.Mailbox, err)
	}
	return e.purgeMailbox(ctx, imapPurgeClient{client: client}, cfg.Mailbox, pending, result)
}

// purgeMailbox moves every pending Message-ID it finds in the selected mailbox to the account's
// Trash folder, removing the ones it handled from pending. Individual failures are recorded and
// skipped so one unreachable mail does not strand the rest.
func (e *Email) purgeMailbox(ctx context.Context, client purgeClient, mailbox string, pending map[string]struct{}, result *imodels.MailPurgeResult) error {
	// A purge that cannot tell whether the account has a Trash folder does nothing: falling through
	// to the expunge would delete mails outright that were meant to be recoverable.
	trash, err := discoverTrashMailbox(client)
	if err != nil {
		return err
	}
	// A mail moved into the mailbox that is being scanned would simply be imported again.
	if trash != "" && strings.EqualFold(trash, mailbox) {
		trash = ""
	}
	if trash == "" {
		e.lo.Warn("no Trash folder found, mails will be expunged", "mailbox", mailbox, "inbox_id", e.Identifier())
	}

	var (
		canMove = client.HasCap(imap.CapMove)
		// UID EXPUNGE removes only the mails this purge flagged. A plain EXPUNGE also drops
		// anything another client left flagged \Deleted in the same mailbox, so it is never used.
		canUIDExpunge = client.HasCap(imap.CapUIDPlus)
	)

	for _, messageID := range sortedKeys(pending) {
		if err := ctx.Err(); err != nil {
			return err
		}

		uids, err := matchingUIDs(client, messageID)
		if err != nil {
			e.lo.Error("error looking up Message-ID in mailbox", "message_id", messageID, "mailbox", mailbox, "inbox_id", e.Identifier(), "error", err)
			result.Record(messageID, imodels.MailPurgeFailed, err.Error())
			continue
		}
		if len(uids) == 0 {
			e.lo.Info("Message-ID not found in mailbox", "message_id", messageID, "mailbox", mailbox, "inbox_id", e.Identifier())
			continue
		}

		var uidSet imap.UIDSet
		uidSet.AddNum(uids...)

		outcome, reason := purgeUIDs(client, uidSet, trash, canMove, canUIDExpunge)
		if reason != "" {
			e.lo.Error("could not purge mail from mailbox", "message_id", messageID, "mailbox", mailbox, "inbox_id", e.Identifier(), "outcome", outcome, "reason", reason)
		}
		result.Record(messageID, outcome, reason)
		if outcome == imodels.MailMovedToTrash || outcome == imodels.MailExpunged {
			if outcome == imodels.MailMovedToTrash {
				result.TrashMailbox = trash
			}
			e.lo.Info("purged mail from mailbox", "message_id", messageID, "uids", len(uids), "outcome", outcome, "mailbox", mailbox, "trash", trash, "inbox_id", e.Identifier())
			delete(pending, messageID)
		}
	}
	return nil
}

// purgeUIDs takes the given UIDs out of the selected mailbox and reports the outcome.
//
// A mail is only ever removed from the mailbox once it is safely somewhere else, or once it can be
// removed without touching anything the purge did not flag itself. That leaves one case with no safe
// move: a server with no Trash folder and no UIDPLUS, where the only way out would be a plain
// EXPUNGE, which would also destroy mails another client flagged \Deleted. Those mails are left
// alone and reported.
func purgeUIDs(client purgeClient, uids imap.UIDSet, trash string, canMove, canUIDExpunge bool) (string, string) {
	switch {
	case trash != "" && canMove:
		if err := client.Move(uids, trash); err != nil {
			return imodels.MailPurgeFailed, err.Error()
		}
		return imodels.MailMovedToTrash, ""
	case trash != "" && canUIDExpunge:
		// The library's MOVE has a COPY fallback of its own, but it pipelines the STORE and the
		// EXPUNGE behind the COPY without waiting for it, so a rejected COPY still deletes the
		// originals. Copy first, and only remove the originals once the copy is confirmed.
		if err := client.Copy(uids, trash); err != nil {
			return imodels.MailPurgeFailed, err.Error()
		}
		if err := client.MarkDeleted(uids); err != nil {
			return imodels.MailPurgeFailed, err.Error()
		}
		if err := client.UIDExpunge(uids); err != nil {
			return imodels.MailPurgeFailed, err.Error()
		}
		return imodels.MailMovedToTrash, ""
	case trash != "":
		return imodels.MailNotPurged, reasonNoMoveNoUIDPlus
	case canUIDExpunge:
		if err := client.MarkDeleted(uids); err != nil {
			return imodels.MailPurgeFailed, err.Error()
		}
		if err := client.UIDExpunge(uids); err != nil {
			return imodels.MailPurgeFailed, err.Error()
		}
		return imodels.MailExpunged, ""
	default:
		return imodels.MailNotPurged, reasonNoTrashNoUIDPlus
	}
}

// discoverTrashMailbox returns the name of the account's Trash folder, or an empty string when the
// server has none.
//
// A server that advertises SPECIAL-USE (RFC 6154, and part of LIST in IMAP4rev2) names its Trash
// folder itself with the \Trash attribute, whatever the folder is called in the user's language.
// RETURN SPECIAL-USE lists every mailbox and tags the special ones, so the same LIST also serves the
// name fallback for servers without the extension.
func discoverTrashMailbox(client purgeClient) (string, error) {
	var options *imap.ListOptions
	if client.HasCap(imap.CapSpecialUse) || client.HasCap(imap.CapIMAP4rev2) {
		options = &imap.ListOptions{ReturnSpecialUse: true}
	}
	mailboxes, err := client.List("", "*", options)
	if err != nil {
		return "", fmt.Errorf("error listing mailboxes: %w", err)
	}

	for _, mailbox := range mailboxes {
		if selectableMailbox(mailbox) && slices.Contains(mailbox.Attrs, imap.MailboxAttrTrash) {
			return mailbox.Mailbox, nil
		}
	}
	for _, name := range commonTrashMailboxes {
		for _, mailbox := range mailboxes {
			if selectableMailbox(mailbox) && strings.EqualFold(mailbox.Mailbox, name) {
				return mailbox.Mailbox, nil
			}
		}
	}
	return "", nil
}

// selectableMailbox reports whether mails can be stored in the mailbox at all. \Noselect names a
// pure container, such as the "INBOX." parent on a server with a hierarchy under INBOX.
func selectableMailbox(mailbox *imap.ListData) bool {
	return mailbox != nil && mailbox.Mailbox != "" &&
		!slices.Contains(mailbox.Attrs, imap.MailboxAttrNoSelect) &&
		!slices.Contains(mailbox.Attrs, imap.MailboxAttrNonExistent)
}

// matchingUIDs returns the UIDs in the selected mailbox whose Message-ID is exactly messageID.
//
// An IMAP HEADER search is a case-insensitive substring match and the desk stores Message-IDs
// without their angle brackets, so searching for "abc@example.com" also returns a mail whose
// Message-ID is "<xabc@example.com.br>". Every candidate is therefore fetched and its Message-ID
// compared as a whole value; only exact matches are moved or deleted.
func matchingUIDs(client purgeClient, messageID string) ([]imap.UID, error) {
	candidates, err := client.SearchMessageID(messageID)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	var uidSet imap.UIDSet
	uidSet.AddNum(candidates...)
	headers, err := client.FetchMessageIDs(uidSet)
	if err != nil {
		return nil, err
	}
	return exactMessageIDMatches(candidates, headers, messageID), nil
}

// exactMessageIDMatches keeps the candidates whose fetched Message-ID is the wanted one. A candidate
// whose header could not be read is dropped: an unread header is not a match.
func exactMessageIDMatches(candidates []imap.UID, headers map[imap.UID]string, want string) []imap.UID {
	wanted := normalizeMessageID(want)
	if wanted == "" {
		return nil
	}
	var uids []imap.UID
	for _, uid := range candidates {
		header, ok := headers[uid]
		if ok && normalizeMessageID(header) == wanted {
			uids = append(uids, uid)
		}
	}
	return uids
}

// normalizeMessageID strips the angle brackets, the surrounding space and the case from a
// Message-ID so the value the desk stored and the value in the header compare as one.
func normalizeMessageID(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.Trim(strings.TrimSpace(value), "<>")))
}

// parseMessageIDHeader reads the Message-ID out of a fetched header block.
func parseMessageIDHeader(raw []byte) string {
	// The block ends with the header terminator only when the mail had a body; add one so a
	// truncated block still parses.
	reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(append(raw, "\r\n"...))))
	header, err := reader.ReadMIMEHeader()
	if err != nil && len(header) == 0 {
		return ""
	}
	return header.Get(headerMessageID)
}

// recordAll reports the same outcome for every Message-ID still pending.
func recordAll(result *imodels.MailPurgeResult, pending map[string]struct{}, outcome, reason string) {
	for _, messageID := range sortedKeys(pending) {
		result.Record(messageID, outcome, reason)
	}
}

// sortedKeys returns the map keys in a stable order so the purge and the API response do not shuffle.
func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// imapPurgeClient adapts the go-imap client to purgeClient.
type imapPurgeClient struct {
	client *imapclient.Client
}

func (c imapPurgeClient) HasCap(capability imap.Cap) bool {
	return c.client.Caps().Has(capability)
}

func (c imapPurgeClient) List(ref, pattern string, options *imap.ListOptions) ([]*imap.ListData, error) {
	return c.client.List(ref, pattern, options).Collect()
}

// SearchMessageID returns the UIDs the server considers a Message-ID match. Options are left nil so
// the search works on servers without ESEARCH.
func (c imapPurgeClient) SearchMessageID(messageID string) ([]imap.UID, error) {
	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{{Key: headerMessageID, Value: messageID}},
	}
	data, err := c.client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, err
	}
	return data.AllUIDs(), nil
}

// FetchMessageIDs reads the Message-ID header of the given UIDs. BODY.PEEK keeps the mails unread,
// which matters because a purge that leaves mails behind must not have touched their state either.
func (c imapPurgeClient) FetchMessageIDs(uids imap.UIDSet) (map[imap.UID]string, error) {
	section := &imap.FetchItemBodySection{
		Specifier:    imap.PartSpecifierHeader,
		HeaderFields: []string{headerMessageID},
		Peek:         true,
	}
	buffers, err := c.client.Fetch(uids, &imap.FetchOptions{
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{section},
	}).Collect()
	if err != nil {
		return nil, err
	}

	headers := make(map[imap.UID]string, len(buffers))
	for _, buf := range buffers {
		// Exactly one section was requested, and the response carries its own section pointer.
		for _, raw := range buf.BodySection {
			if messageID := parseMessageIDHeader(raw); messageID != "" {
				headers[buf.UID] = messageID
				break
			}
		}
	}
	return headers, nil
}

func (c imapPurgeClient) Move(uids imap.UIDSet, mailbox string) error {
	_, err := c.client.Move(uids, mailbox).Wait()
	return err
}

func (c imapPurgeClient) Copy(uids imap.UIDSet, mailbox string) error {
	_, err := c.client.Copy(uids, mailbox).Wait()
	return err
}

func (c imapPurgeClient) MarkDeleted(uids imap.UIDSet) error {
	return c.client.Store(uids, &imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagDeleted},
	}, nil).Close()
}

func (c imapPurgeClient) UIDExpunge(uids imap.UIDSet) error {
	return c.client.UIDExpunge(uids).Close()
}
