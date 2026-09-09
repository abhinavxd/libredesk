package email

import (
	"context"
	"fmt"
	"sort"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// PurgeMessages removes the mails carrying the given Message-IDs from the inbox's IMAP mailboxes and
// returns the Message-IDs it could not remove.
//
// The desk deduplicates incoming mail against the messages it still holds, so a mail left on the
// server after its conversation is deleted is re-imported as a brand new conversation on the next
// scan. Mails are flagged \Deleted and expunged rather than moved to a Trash folder: the inbox
// configuration names only the mailbox to scan, and RFC 6154 special-use discovery is not something
// every server offers, whereas \Deleted plus EXPUNGE is plain IMAP4rev1 that works everywhere.
func (e *Email) PurgeMessages(ctx context.Context, messageIDs []string) ([]string, error) {
	if len(messageIDs) == 0 {
		return nil, nil
	}
	if len(e.imapCfg) == 0 {
		return messageIDs, fmt.Errorf("inbox %d has no IMAP configuration to purge from", e.Identifier())
	}

	// Mails still to be found. A Message-ID is dropped as soon as one mailbox yields it.
	pending := make(map[string]struct{}, len(messageIDs))
	for _, messageID := range messageIDs {
		if messageID != "" {
			pending[messageID] = struct{}{}
		}
	}

	var lastErr error
	for _, cfg := range e.imapCfg {
		if len(pending) == 0 {
			break
		}
		if err := ctx.Err(); err != nil {
			return sortedKeys(pending), err
		}
		if err := e.purgeFromMailbox(ctx, cfg, pending); err != nil {
			e.lo.Error("error purging mails from mailbox", "mailbox", cfg.Mailbox, "inbox_id", e.Identifier(), "error", err)
			lastErr = err
		}
	}
	return sortedKeys(pending), lastErr
}

// purgeFromMailbox deletes every pending Message-ID it finds in the given mailbox, removing the ones
// it deleted from pending. Individual failures are logged and skipped so one unreachable mail does
// not strand the rest.
func (e *Email) purgeFromMailbox(ctx context.Context, cfg imodels.IMAPConfig, pending map[string]struct{}) error {
	client, err := e.dialIMAP(cfg)
	if err != nil {
		return err
	}
	defer client.Logout()

	// Read-write, unlike the scan, which selects the mailbox read-only.
	if _, err := client.Select(cfg.Mailbox, nil).Wait(); err != nil {
		return fmt.Errorf("error selecting mailbox %q: %w", cfg.Mailbox, err)
	}

	// UID EXPUNGE removes only the mails this purge flagged. A plain EXPUNGE also drops anything
	// another client left flagged \Deleted in the same mailbox, so it is the last resort.
	canUIDExpunge := client.Caps().Has(imap.CapUIDPlus) || client.Caps().Has(imap.CapIMAP4rev2)

	for messageID := range pending {
		if err := ctx.Err(); err != nil {
			return err
		}

		uids, err := searchByMessageID(client, messageID)
		if err != nil {
			e.lo.Error("error searching mailbox for Message-ID", "message_id", messageID, "mailbox", cfg.Mailbox, "inbox_id", e.Identifier(), "error", err)
			continue
		}
		if len(uids) == 0 {
			e.lo.Info("Message-ID not found in mailbox", "message_id", messageID, "mailbox", cfg.Mailbox, "inbox_id", e.Identifier())
			continue
		}

		var uidSet imap.UIDSet
		uidSet.AddNum(uids...)

		store := &imap.StoreFlags{
			Op:     imap.StoreFlagsAdd,
			Silent: true,
			Flags:  []imap.Flag{imap.FlagDeleted},
		}
		if err := client.Store(uidSet, store, nil).Close(); err != nil {
			e.lo.Error("error flagging mail as deleted", "message_id", messageID, "mailbox", cfg.Mailbox, "inbox_id", e.Identifier(), "error", err)
			continue
		}

		if canUIDExpunge {
			err = client.UIDExpunge(uidSet).Close()
		} else {
			err = client.Expunge().Close()
		}
		if err != nil {
			e.lo.Error("error expunging mail", "message_id", messageID, "mailbox", cfg.Mailbox, "inbox_id", e.Identifier(), "error", err)
			continue
		}

		e.lo.Info("purged mail from mailbox", "message_id", messageID, "uids", len(uids), "mailbox", cfg.Mailbox, "inbox_id", e.Identifier())
		delete(pending, messageID)
	}
	return nil
}

// searchByMessageID returns the UIDs of the mails whose Message-ID header matches.
// Options are left nil so the search works on servers without ESEARCH; a Message-ID matches at most
// a handful of mails, so there is no result set worth narrowing.
func searchByMessageID(client *imapclient.Client, messageID string) ([]imap.UID, error) {
	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{{Key: headerMessageID, Value: messageID}},
	}
	data, err := client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, err
	}
	return data.AllUIDs(), nil
}

// sortedKeys returns the map keys in a stable order so the API response does not shuffle.
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
