package email

import (
	"context"
	"errors"
	"io"
	"testing"

	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/emersion/go-imap/v2"
	"github.com/zerodha/logf"
)

// fakePurgeClient stands in for an IMAP server so the folder discovery, the Message-ID matching and
// the move-or-delete decision can be tested without one.
type fakePurgeClient struct {
	caps      map[imap.Cap]bool
	mailboxes []*imap.ListData
	listErr   error
	search    map[string][]imap.UID
	searchErr error
	headers   map[imap.UID]string
	fetchErr  error
	moveErr   error
	copyErr   error

	listOptions   *imap.ListOptions
	listCalls     int
	moved         map[string][]imap.UID
	copied        map[string][]imap.UID
	markedDeleted []imap.UID
	expunged      []imap.UID
}

func (c *fakePurgeClient) HasCap(capability imap.Cap) bool { return c.caps[capability] }

func (c *fakePurgeClient) List(ref, pattern string, options *imap.ListOptions) ([]*imap.ListData, error) {
	c.listCalls++
	c.listOptions = options
	return c.mailboxes, c.listErr
}

func (c *fakePurgeClient) SearchMessageID(messageID string) ([]imap.UID, error) {
	if c.searchErr != nil {
		return nil, c.searchErr
	}
	return c.search[messageID], nil
}

func (c *fakePurgeClient) FetchMessageIDs(uids imap.UIDSet) (map[imap.UID]string, error) {
	if c.fetchErr != nil {
		return nil, c.fetchErr
	}
	found := map[imap.UID]string{}
	for uid, header := range c.headers {
		if uids.Contains(uid) {
			found[uid] = header
		}
	}
	return found, nil
}

func (c *fakePurgeClient) Move(uids imap.UIDSet, mailbox string) error {
	if c.moveErr != nil {
		return c.moveErr
	}
	if c.moved == nil {
		c.moved = map[string][]imap.UID{}
	}
	c.moved[mailbox] = append(c.moved[mailbox], uidNums(uids)...)
	return nil
}

func (c *fakePurgeClient) Copy(uids imap.UIDSet, mailbox string) error {
	if c.copyErr != nil {
		return c.copyErr
	}
	if c.copied == nil {
		c.copied = map[string][]imap.UID{}
	}
	c.copied[mailbox] = append(c.copied[mailbox], uidNums(uids)...)
	return nil
}

func (c *fakePurgeClient) MarkDeleted(uids imap.UIDSet) error {
	c.markedDeleted = append(c.markedDeleted, uidNums(uids)...)
	return nil
}

func (c *fakePurgeClient) UIDExpunge(uids imap.UIDSet) error {
	c.expunged = append(c.expunged, uidNums(uids)...)
	return nil
}

func uidNums(uids imap.UIDSet) []imap.UID {
	nums, _ := uids.Nums()
	return nums
}

func mailbox(name string, attrs ...imap.MailboxAttr) *imap.ListData {
	return &imap.ListData{Mailbox: name, Delim: '/', Attrs: attrs}
}

func testEmail(t *testing.T) *Email {
	t.Helper()
	lo := logf.New(logf.Opts{Writer: io.Discard})
	return &Email{id: 1, lo: &lo}
}

func TestDiscoverTrashMailbox(t *testing.T) {
	for _, tc := range []struct {
		name             string
		caps             map[imap.Cap]bool
		mailboxes        []*imap.ListData
		want             string
		wantSpecialUseIn bool
	}{
		{
			name: "special-use \\Trash wins over the folder name",
			caps: map[imap.Cap]bool{imap.CapSpecialUse: true},
			mailboxes: []*imap.ListData{
				mailbox("INBOX"),
				mailbox("Trash"),
				mailbox("Papierkorb", imap.MailboxAttrTrash),
			},
			want:             "Papierkorb",
			wantSpecialUseIn: true,
		},
		{
			name: "special-use asked for on an IMAP4rev2 server",
			caps: map[imap.Cap]bool{imap.CapIMAP4rev2: true},
			mailboxes: []*imap.ListData{
				mailbox("INBOX"),
				mailbox("Bin", imap.MailboxAttrTrash),
			},
			want:             "Bin",
			wantSpecialUseIn: true,
		},
		{
			name: "common name when the server has no special-use",
			mailboxes: []*imap.ListData{
				mailbox("INBOX"),
				mailbox("INBOX.Sent"),
				mailbox("INBOX.Trash"),
			},
			want: "INBOX.Trash",
		},
		{
			name: "common names are tried in order, not in list order",
			mailboxes: []*imap.ListData{
				mailbox("Deleted Items"),
				mailbox("Trash"),
			},
			want: "Trash",
		},
		{
			name:      "folder name matching ignores case",
			mailboxes: []*imap.ListData{mailbox("INBOX"), mailbox("TRASH")},
			want:      "TRASH",
		},
		{
			name: "a container that cannot hold mail is skipped",
			mailboxes: []*imap.ListData{
				mailbox("Trash", imap.MailboxAttrNoSelect),
				mailbox("Deleted Messages"),
			},
			want: "Deleted Messages",
		},
		{
			name:      "nothing to move to",
			mailboxes: []*imap.ListData{mailbox("INBOX"), mailbox("Archive"), mailbox("Junk")},
			want:      "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakePurgeClient{caps: tc.caps, mailboxes: tc.mailboxes}
			got, err := discoverTrashMailbox(client)
			if err != nil {
				t.Fatalf("discoverTrashMailbox() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("discoverTrashMailbox() = %q, want %q", got, tc.want)
			}
			if client.listCalls != 1 {
				t.Errorf("LIST issued %d times, want once per connection", client.listCalls)
			}
			gotSpecialUse := client.listOptions != nil && client.listOptions.ReturnSpecialUse
			if gotSpecialUse != tc.wantSpecialUseIn {
				t.Errorf("LIST RETURN SPECIAL-USE = %v, want %v", gotSpecialUse, tc.wantSpecialUseIn)
			}
		})
	}
}

func TestDiscoverTrashMailboxListError(t *testing.T) {
	client := &fakePurgeClient{listErr: errors.New("LIST rejected")}
	got, err := discoverTrashMailbox(client)
	if err == nil {
		t.Fatal("discoverTrashMailbox() error = nil, want the LIST failure")
	}
	if got != "" {
		t.Errorf("discoverTrashMailbox() = %q, want no mailbox", got)
	}
}

func TestMatchingUIDsKeepsOnlyExactMessageIDs(t *testing.T) {
	// An IMAP HEADER search is a case-insensitive substring match, so the server hands back mails
	// whose Message-ID merely contains the wanted one.
	client := &fakePurgeClient{
		search: map[string][]imap.UID{"abc@x.test": {1, 2, 3, 4, 5}},
		headers: map[imap.UID]string{
			1: "<abc@x.test>",
			2: "<xabc@x.test.br>",
			3: "<abc@x.test.evil>",
			4: "ABC@X.TEST",
			// 5 has no readable Message-ID header.
		},
	}

	got, err := matchingUIDs(client, "abc@x.test")
	if err != nil {
		t.Fatalf("matchingUIDs() error = %v", err)
	}
	want := []imap.UID{1, 4}
	if len(got) != len(want) {
		t.Fatalf("matchingUIDs() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("matchingUIDs() = %v, want %v", got, want)
		}
	}
}

func TestMatchingUIDsNoCandidates(t *testing.T) {
	client := &fakePurgeClient{}
	got, err := matchingUIDs(client, "gone@x.test")
	if err != nil {
		t.Fatalf("matchingUIDs() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("matchingUIDs() = %v, want none", got)
	}
}

func TestParseMessageIDHeader(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{"terminated header block", "Message-ID: <abc@x.test>\r\n\r\n", "<abc@x.test>"},
		{"unterminated header block", "Message-ID: <abc@x.test>", "<abc@x.test>"},
		{"folded value", "Message-ID:\r\n <abc@x.test>\r\n\r\n", "<abc@x.test>"},
		{"header absent", "Subject: hello\r\n\r\n", ""},
		{"empty block", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseMessageIDHeader([]byte(tc.raw)); got != tc.want {
				t.Errorf("parseMessageIDHeader(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestPurgeUIDs(t *testing.T) {
	var uids imap.UIDSet
	uids.AddNum(7)

	for _, tc := range []struct {
		name          string
		trash         string
		caps          map[imap.Cap]bool
		wantOutcome   string
		wantReason    string
		wantMoved     bool
		wantCopied    bool
		wantExpunged  bool
		wantUntouched bool
	}{
		{
			name:        "MOVE to the Trash folder",
			trash:       "Trash",
			caps:        map[imap.Cap]bool{imap.CapMove: true, imap.CapUIDPlus: true},
			wantOutcome: imodels.MailMovedToTrash,
			wantMoved:   true,
		},
		{
			name:         "COPY to the Trash folder when the server has no MOVE",
			trash:        "Trash",
			caps:         map[imap.Cap]bool{imap.CapUIDPlus: true},
			wantOutcome:  imodels.MailMovedToTrash,
			wantCopied:   true,
			wantExpunged: true,
		},
		{
			name:          "left alone when the mail cannot be removed after the copy",
			trash:         "Trash",
			caps:          map[imap.Cap]bool{},
			wantOutcome:   imodels.MailNotPurged,
			wantReason:    reasonNoMoveNoUIDPlus,
			wantUntouched: true,
		},
		{
			name:         "expunged when the account has no Trash folder",
			caps:         map[imap.Cap]bool{imap.CapUIDPlus: true},
			wantOutcome:  imodels.MailExpunged,
			wantExpunged: true,
		},
		{
			name:          "left alone rather than risking a plain EXPUNGE",
			caps:          map[imap.Cap]bool{imap.CapMove: true},
			wantOutcome:   imodels.MailNotPurged,
			wantReason:    reasonNoTrashNoUIDPlus,
			wantUntouched: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &fakePurgeClient{caps: tc.caps}
			outcome, reason := purgeUIDs(client, uids, tc.trash, client.HasCap(imap.CapMove), client.HasCap(imap.CapUIDPlus))
			if outcome != tc.wantOutcome {
				t.Errorf("outcome = %q, want %q", outcome, tc.wantOutcome)
			}
			if reason != tc.wantReason {
				t.Errorf("reason = %q, want %q", reason, tc.wantReason)
			}
			if got := len(client.moved[tc.trash]) > 0; got != tc.wantMoved {
				t.Errorf("moved = %v, want %v", got, tc.wantMoved)
			}
			if got := len(client.copied[tc.trash]) > 0; got != tc.wantCopied {
				t.Errorf("copied = %v, want %v", got, tc.wantCopied)
			}
			if got := len(client.expunged) > 0; got != tc.wantExpunged {
				t.Errorf("expunged = %v, want %v", got, tc.wantExpunged)
			}
			if tc.wantUntouched && len(client.markedDeleted) > 0 {
				t.Errorf("mail was flagged \\Deleted with no way to expunge it safely")
			}
		})
	}
}

func TestPurgeUIDsReportsAFailedMove(t *testing.T) {
	var uids imap.UIDSet
	uids.AddNum(7)
	client := &fakePurgeClient{caps: map[imap.Cap]bool{imap.CapMove: true}, moveErr: errors.New("[TRYCREATE] no such mailbox")}

	outcome, reason := purgeUIDs(client, uids, "Trash", true, false)
	if outcome != imodels.MailPurgeFailed {
		t.Errorf("outcome = %q, want %q", outcome, imodels.MailPurgeFailed)
	}
	if reason == "" {
		t.Error("reason is empty, want the server's rejection")
	}
}

func TestPurgeUIDsDoesNotDeleteOriginalsWhenTheCopyFails(t *testing.T) {
	var uids imap.UIDSet
	uids.AddNum(7)
	client := &fakePurgeClient{caps: map[imap.Cap]bool{imap.CapUIDPlus: true}, copyErr: errors.New("over quota")}

	outcome, _ := purgeUIDs(client, uids, "Trash", false, true)
	if outcome != imodels.MailPurgeFailed {
		t.Errorf("outcome = %q, want %q", outcome, imodels.MailPurgeFailed)
	}
	if len(client.markedDeleted) > 0 || len(client.expunged) > 0 {
		t.Error("originals were deleted although the copy to Trash failed")
	}
}

func TestPurgeMailboxMovesFoundMailsAndReportsTheRest(t *testing.T) {
	client := &fakePurgeClient{
		caps:      map[imap.Cap]bool{imap.CapSpecialUse: true, imap.CapMove: true, imap.CapUIDPlus: true},
		mailboxes: []*imap.ListData{mailbox("INBOX"), mailbox("Trash", imap.MailboxAttrTrash)},
		search: map[string][]imap.UID{
			"found@x.test":     {11},
			"lookalike@x.test": {12},
		},
		headers: map[imap.UID]string{
			11: "<found@x.test>",
			12: "<not-lookalike@x.test.br>",
		},
	}
	pending := map[string]struct{}{
		"found@x.test":     {},
		"lookalike@x.test": {},
		"missing@x.test":   {},
	}
	var result imodels.MailPurgeResult
	for messageID := range pending {
		result.Record(messageID, imodels.MailNotFound, "")
	}

	if err := testEmail(t).purgeMailbox(context.Background(), client, "INBOX", pending, &result); err != nil {
		t.Fatalf("purgeMailbox() error = %v", err)
	}

	if got := client.moved["Trash"]; len(got) != 1 || got[0] != 11 {
		t.Errorf("moved to Trash = %v, want [11]", got)
	}
	if result.TrashMailbox != "Trash" {
		t.Errorf("TrashMailbox = %q, want %q", result.TrashMailbox, "Trash")
	}
	if got := result.Count(imodels.MailMovedToTrash); got != 1 {
		t.Errorf("moved_to_trash = %d, want 1", got)
	}
	if got := result.Count(imodels.MailNotFound); got != 2 {
		t.Errorf("not_found = %d, want 2", got)
	}
	if _, still := pending["found@x.test"]; still {
		t.Error("the moved mail is still pending for the next mailbox")
	}
	if len(pending) != 2 {
		t.Errorf("pending = %v, want the two unfound mails", pending)
	}
	if len(result.Unpurged()) != 2 {
		t.Errorf("Unpurged() = %v, want the two unfound mails", result.Unpurged())
	}
}

func TestPurgeMailboxNeverMovesIntoTheScannedMailbox(t *testing.T) {
	// An inbox configured to scan Trash itself would keep importing whatever was moved there.
	client := &fakePurgeClient{
		caps:      map[imap.Cap]bool{imap.CapSpecialUse: true, imap.CapMove: true, imap.CapUIDPlus: true},
		mailboxes: []*imap.ListData{mailbox("INBOX"), mailbox("Trash", imap.MailboxAttrTrash)},
		search:    map[string][]imap.UID{"found@x.test": {11}},
		headers:   map[imap.UID]string{11: "<found@x.test>"},
	}
	pending := map[string]struct{}{"found@x.test": {}}
	var result imodels.MailPurgeResult

	if err := testEmail(t).purgeMailbox(context.Background(), client, "trash", pending, &result); err != nil {
		t.Fatalf("purgeMailbox() error = %v", err)
	}

	if len(client.moved) > 0 {
		t.Errorf("mail was moved to %v, want it expunged from the scanned mailbox", client.moved)
	}
	if got := result.Count(imodels.MailExpunged); got != 1 {
		t.Errorf("expunged = %d, want 1", got)
	}
	if result.TrashMailbox != "" {
		t.Errorf("TrashMailbox = %q, want none", result.TrashMailbox)
	}
}

func TestPurgeMessagesWithoutIMAPConfigReportsEveryMail(t *testing.T) {
	result, err := testEmail(t).PurgeMessages(context.Background(), []string{"a@x.test", "b@x.test"})
	if err == nil {
		t.Fatal("PurgeMessages() error = nil, want the missing configuration")
	}
	if got := result.Count(imodels.MailPurgeFailed); got != 2 {
		t.Errorf("failed = %d, want 2", got)
	}
	if got := len(result.Unpurged()); got != 2 {
		t.Errorf("Unpurged() = %d, want 2", got)
	}
}

func TestPurgeMessagesWithoutMessageIDs(t *testing.T) {
	result, err := testEmail(t).PurgeMessages(context.Background(), nil)
	if err != nil {
		t.Fatalf("PurgeMessages() error = %v", err)
	}
	if len(result.Mails) != 0 {
		t.Errorf("Mails = %v, want none", result.Mails)
	}
}

func TestPurgeMailboxStopsWhenTheTrashFolderCannotBeLookedUp(t *testing.T) {
	// Expunging here would destroy mails that were meant to be recoverable from Trash.
	client := &fakePurgeClient{
		caps:    map[imap.Cap]bool{imap.CapMove: true, imap.CapUIDPlus: true},
		listErr: errors.New("LIST rejected"),
		search:  map[string][]imap.UID{"found@x.test": {11}},
		headers: map[imap.UID]string{11: "<found@x.test>"},
	}
	pending := map[string]struct{}{"found@x.test": {}}
	var result imodels.MailPurgeResult

	if err := testEmail(t).purgeMailbox(context.Background(), client, "INBOX", pending, &result); err == nil {
		t.Fatal("purgeMailbox() error = nil, want the LIST failure")
	}
	if len(client.expunged) > 0 || len(client.markedDeleted) > 0 || len(client.moved) > 0 {
		t.Error("mails were touched although the Trash folder could not be looked up")
	}
	if _, still := pending["found@x.test"]; !still {
		t.Error("the mail was dropped from pending although nothing was done with it")
	}
}
