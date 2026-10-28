package user

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type userRow struct {
	FirstName string
	LastName  string
	Email     string
	ExtID     string
	Type      string
}

func newTestManager(t *testing.T) (*Manager, *sqlx.DB) {
	t.Helper()
	db := testutil.NewDB(t, "user_contact")
	lo := logf.New(logf.Opts{})
	mgr, err := New(testutil.NewI18n(t), Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatalf("creating user manager: %v", err)
	}
	return mgr, db
}

func newContact(email, extID, firstName, lastName string) *models.User {
	return &models.User{
		Email:          null.NewString(email, email != ""),
		ExternalUserID: null.NewString(extID, extID != ""),
		FirstName:      firstName,
		LastName:       lastName,
	}
}

func TestContactMerge(t *testing.T) {
	u, db := newTestManager(t)
	keep := newContact("keep@example.com", "customer-merge", "Keep", "Person")
	remove := newContact("remove@example.com", "", "Other", "Person")
	resolve(t, u, keep, models.ContactReuse)
	resolve(t, u, remove, models.ContactReuse)
	actor := insertAgent(t, db, "merge-agent@example.com", "")
	db.MustExec(`UPDATE users SET custom_attributes = '{"zero":0,"flag":false,"plan":"old"}' WHERE id = $1`, keep.ID)
	db.MustExec(`UPDATE users SET custom_attributes = '{"zero":1,"flag":true,"plan":"new","extra":"value"}', enabled = false WHERE id = $1`, remove.ID)
	var inboxID, conversationID, messageID int
	if err := db.QueryRow(`INSERT INTO inboxes (name, channel) VALUES ('Merge test', 'email') RETURNING id`).Scan(&inboxID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO conversations (inbox_id, contact_id, subject, status_id) VALUES ($1, $2, 'Keep my history', (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id`, inboxID, remove.ID).Scan(&conversationID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO conversation_messages (conversation_id, sender_id, type, status, sender_type, content) VALUES ($1, $2, 'incoming', 'received', 'contact', 'Original message') RETURNING id`, conversationID, remove.ID).Scan(&messageID); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO media (store, filename, content_type, model_type, model_id, uploaded_by) VALUES ('fs', 'original.txt', 'text/plain', 'message', $1, $2)`, messageID, remove.ID)
	db.MustExec(`UPDATE conversations SET last_message_sender_id = $1, last_interaction_sender_id = $1 WHERE id = $2`, remove.ID, conversationID)
	db.MustExec(`INSERT INTO contact_notes (contact_id, user_id, note) VALUES ($1, $2, 'Original note')`, remove.ID, actor)
	db.MustExec(`INSERT INTO contact_channel_identities (contact_id, channel, identifier) VALUES ($1, 'whatsapp', '123456789')`, remove.ID)
	for _, id := range []int{keep.ID, remove.ID} {
		db.MustExec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)`, conversationID, id)
		db.MustExec(`INSERT INTO conversation_last_seen (conversation_id, user_id) VALUES ($1, $2)`, conversationID, id)
	}
	preview, err := u.PreviewContactMerge(keep.ID, remove.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Remove.Conversations != 1 || preview.Remove.Notes != 1 {
		t.Fatalf("wrong history counts: %+v", preview.Remove)
	}
	again, err := u.PreviewContactMerge(keep.ID, remove.ID)
	if err != nil || preview.Version != again.Version {
		t.Fatalf("preview changed without edits: %v", err)
	}
	req := ContactMergeRequest{RemoveID: remove.ID, Version: preview.Version, Choices: map[string]string{}}
	if _, err := u.MergeContacts(keep.ID, req, actor, "merge-agent@example.com", "127.0.0.1"); err == nil {
		t.Fatal("accepted missing conflict choices")
	}
	for _, field := range preview.Fields {
		if field.Conflict {
			req.Choices[field.Key] = "keep"
		}
	}
	req.Choices["email"] = "remove"
	req.Choices["custom.plan"] = "remove"
	if _, err := u.MergeContacts(keep.ID, req, actor, "merge-agent@example.com", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	row := fetchRow(t, db, keep.ID)
	if row.Email != "remove@example.com" || row.ExtID != "customer-merge" {
		t.Fatalf("wrong merged profile: %+v", row)
	}
	var enabled bool
	var attributes string
	if err := db.QueryRow(`SELECT enabled, custom_attributes::text FROM users WHERE id = $1`, keep.ID).Scan(&enabled, &attributes); err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("merge unblocked contact")
	}
	var attrs map[string]any
	if err := json.Unmarshal([]byte(attributes), &attrs); err != nil {
		t.Fatal(err)
	}
	if attrs["zero"] != float64(0) || attrs["flag"] != false || attrs["plan"] != "new" || attrs["extra"] != "value" {
		t.Fatalf("lost attributes: %v", attrs)
	}
	for _, query := range []string{
		`SELECT contact_id FROM conversations WHERE id = $1`,
		`SELECT sender_id FROM conversation_messages WHERE conversation_id = $1`,
		`SELECT last_message_sender_id FROM conversations WHERE id = $1`,
		`SELECT last_interaction_sender_id FROM conversations WHERE id = $1`,
		`SELECT user_id FROM conversation_participants WHERE conversation_id = $1`,
		`SELECT user_id FROM conversation_last_seen WHERE conversation_id = $1`,
	} {
		var id int
		if err := db.QueryRow(query, conversationID).Scan(&id); err != nil || id != keep.ID {
			t.Fatalf("history lost in %s: %d, %v", query, id, err)
		}
	}
	var noteCount, mediaOwner int
	if err := db.Get(&noteCount, `SELECT COUNT(*) FROM contact_notes WHERE contact_id = $1 AND note = 'Original note'`, keep.ID); err != nil || noteCount != 1 {
		t.Fatalf("note lost after merge: %d %v", noteCount, err)
	}
	if err := db.Get(&mediaOwner, `SELECT uploaded_by FROM media WHERE model_type = 'message' AND model_id = $1 AND filename = 'original.txt'`, messageID); err != nil || mediaOwner != keep.ID {
		t.Fatalf("attachment lost after merge: %d %v", mediaOwner, err)
	}
	identityOwner, err := u.GetContactIDByChannelIdentity("whatsapp", "123456789")
	if err != nil || identityOwner != keep.ID {
		t.Fatalf("channel identity lost: %d %v", identityOwner, err)
	}
	for _, email := range []string{"keep@example.com", "remove@example.com"} {
		found, err := u.GetContactByEmail(email)
		if err != nil || found.ID != keep.ID {
			t.Fatalf("email %s no longer resolves: %+v, %v", email, found, err)
		}
		blocked, err := u.IsEmailBlocked(email)
		if err != nil || !blocked {
			t.Fatalf("email %s bypasses block: %v", email, err)
		}
	}
	resolved, err := u.ResolveMergedContactID(remove.ID)
	if err != nil || resolved != keep.ID {
		t.Fatalf("old session no longer resolves: %d %v", resolved, err)
	}
	if _, err := u.CreateNote(remove.ID, actor, "After merge"); err != nil {
		t.Fatal(err)
	}
	if _, err := u.LinkChannelIdentity(remove.ID, "email", "late-alias@example.com"); err != nil {
		t.Fatal(err)
	}
	var noteOwner int
	if err := db.Get(&noteOwner, `SELECT contact_id FROM contact_notes WHERE note = 'After merge'`); err != nil || noteOwner != keep.ID {
		t.Fatalf("late note owner %d: %v", noteOwner, err)
	}
	if err := db.Get(&identityOwner, `SELECT contact_id FROM contact_channel_identities WHERE channel = 'email' AND identifier = 'late-alias@example.com'`); err != nil || identityOwner != keep.ID {
		t.Fatalf("late identity owner %d: %v", identityOwner, err)
	}
	var logs int
	db.Get(&logs, `SELECT COUNT(*) FROM activity_logs WHERE activity_type = 'contact_merged' AND target_model_id = $1`, keep.ID)
	if logs != 1 {
		t.Fatalf("expected one merge audit, got %d", logs)
	}
	if _, err := u.MergeContacts(keep.ID, req, actor, "merge-agent@example.com", "127.0.0.1"); err == nil {
		t.Fatal("repeated merge accepted")
	}
}

func TestContactMergeRejectsUnsafeChanges(t *testing.T) {
	u, db := newTestManager(t)
	keep := newContact("merge-a@example.com", "", "A", "")
	remove := newContact("merge-b@example.com", "", "B", "")
	resolve(t, u, keep, models.ContactReuse)
	resolve(t, u, remove, models.ContactReuse)
	actor := insertAgent(t, db, "merge-actor@example.com", "")
	for _, ids := range [][2]int{{0, remove.ID}, {-1, remove.ID}, {keep.ID, 0}, {keep.ID, -1}, {keep.ID, keep.ID}, {keep.ID, actor}} {
		if _, err := u.PreviewContactMerge(ids[0], ids[1]); err == nil {
			t.Fatalf("accepted invalid contacts: %v", ids)
		}
	}
	preview, err := u.PreviewContactMerge(keep.ID, remove.ID)
	if err != nil {
		t.Fatal(err)
	}
	req := ContactMergeRequest{RemoveID: remove.ID, Version: preview.Version, Choices: map[string]string{"email": "keep", "first_name": "keep"}}
	db.MustExec(`UPDATE users SET first_name = 'Changed' WHERE id = $1`, remove.ID)
	if _, err := u.MergeContacts(keep.ID, req, actor, "actor@example.com", "127.0.0.1"); err == nil {
		t.Fatal("accepted changed preview")
	}
	preview, err = u.PreviewContactMerge(keep.ID, remove.ID)
	if err != nil {
		t.Fatal(err)
	}
	req.Version = preview.Version
	req.Choices["unknown"] = "keep"
	if _, err := u.MergeContacts(keep.ID, req, actor, "actor@example.com", "127.0.0.1"); err == nil {
		t.Fatal("accepted unknown choice")
	}
	delete(req.Choices, "unknown")
	if _, err := u.MergeContacts(keep.ID, req, -1, "actor@example.com", "127.0.0.1"); err == nil {
		t.Fatal("expected audit failure")
	}
	if row := fetchRow(t, db, keep.ID); row.Email != "merge-a@example.com" {
		t.Fatalf("failed merge changed contact: %+v", row)
	}
	if _, err := u.GetContactOrVisitor(remove.ID, ""); err != nil {
		t.Fatal("failed merge removed source", err)
	}
	db.MustExec(`UPDATE users SET external_user_id = 'customer-a' WHERE id = $1`, keep.ID)
	db.MustExec(`UPDATE users SET external_user_id = 'customer-b' WHERE id = $1`, remove.ID)
	if _, err := u.PreviewContactMerge(keep.ID, remove.ID); err == nil {
		t.Fatal("accepted different customer identities")
	}
}

func TestContactMergeVisitorAndAliasChain(t *testing.T) {
	u, db := newTestManager(t)
	actor := insertAgent(t, db, "chain-actor@example.com", "")
	first := newContact("chain-first@example.com", "chain-customer", "Person", "")
	second := newContact("chain-second@example.com", "", "Person", "")
	resolve(t, u, first, models.ContactReuse)
	resolve(t, u, second, models.ContactReuse)
	visitorID := insertVisitor(t, db, "chain-visitor@example.com")
	for _, query := range []string{"chain-first", "chain-customer", "Visitor"} {
		contacts, err := u.SearchMergeContacts(query)
		if err != nil || len(contacts) == 0 {
			t.Fatalf("merge search failed for %s: %v", query, err)
		}
	}
	for _, pair := range [][2]int{{second.ID, first.ID}, {visitorID, second.ID}} {
		preview, err := u.PreviewContactMerge(pair[0], pair[1])
		if err != nil {
			t.Fatal(err)
		}
		req := ContactMergeRequest{RemoveID: pair[1], Version: preview.Version, Choices: map[string]string{}}
		for _, field := range preview.Fields {
			if field.Conflict {
				req.Choices[field.Key] = "keep"
			}
		}
		if _, err := u.MergeContacts(pair[0], req, actor, "chain-actor@example.com", "127.0.0.1"); err != nil {
			t.Fatal(err)
		}
	}
	identified := newContact("", "chain-customer", "Incoming", "")
	resolve(t, u, identified, models.ContactReuse)
	if identified.ID != visitorID {
		t.Fatalf("external identity lost after merge: %d", identified.ID)
	}
	row := fetchRow(t, db, visitorID)
	if row.Type != models.UserTypeContact || row.ExtID != "chain-customer" {
		t.Fatalf("lost authenticated identity: %+v", row)
	}
	for _, id := range []int{first.ID, second.ID} {
		resolved, err := u.ResolveMergedContactID(id)
		if err != nil || resolved != visitorID {
			t.Fatalf("alias chain lost: %d -> %d, %v", id, resolved, err)
		}
	}
	for _, email := range []string{"chain-first@example.com", "chain-second@example.com", "chain-visitor@example.com"} {
		contact, err := u.GetContactByEmail(email)
		if err != nil || contact.ID != visitorID {
			t.Fatalf("email alias lost: %s %v", email, err)
		}
	}
	db.MustExec(`UPDATE users SET email = NULL WHERE id = $1`, visitorID)
	var reassignedID int
	if err := db.QueryRow(`INSERT INTO users (type, first_name, email, external_user_id) VALUES ('contact', 'New owner', 'chain-first@example.com', 'new-customer') RETURNING id`).Scan(&reassignedID); err != nil {
		t.Fatal(err)
	}
	owner, err := u.GetContactByEmail("chain-first@example.com")
	if err != nil || owner.ID != reassignedID {
		t.Fatalf("old alias took precedence over current email owner: %d %v", owner.ID, err)
	}
	if err := u.DeleteContact(visitorID); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := db.Get(&remaining, `SELECT COUNT(*) FROM users WHERE id = ANY($1)`, pq.Array([]int{visitorID, first.ID, second.ID})); err != nil || remaining != 0 {
		t.Fatalf("contact deletion retained merged personal data: %d %v", remaining, err)
	}
}

func TestContactMergeConcurrentMessage(t *testing.T) {
	u, db := newTestManager(t)
	actor := insertAgent(t, db, "concurrent-actor@example.com", "")
	keep := newContact("concurrent-keep@example.com", "", "Person", "")
	remove := newContact("concurrent-remove@example.com", "", "Person", "")
	resolve(t, u, keep, models.ContactReuse)
	resolve(t, u, remove, models.ContactReuse)
	var conversationID, inboxID int
	if err := db.QueryRow(`INSERT INTO inboxes (name, channel) VALUES ('Concurrent merge', 'email') RETURNING id`).Scan(&inboxID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO conversations (inbox_id, contact_id, status_id) VALUES ($2, $1, (SELECT id FROM conversation_statuses LIMIT 1)) RETURNING id`, remove.ID, inboxID).Scan(&conversationID); err != nil {
		t.Fatal(err)
	}
	preview, err := u.PreviewContactMerge(keep.ID, remove.ID)
	if err != nil {
		t.Fatal(err)
	}
	req := ContactMergeRequest{RemoveID: remove.ID, Version: preview.Version, Choices: map[string]string{"email": "keep"}}
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() {
		<-start
		_, err := u.MergeContacts(keep.ID, req, actor, "concurrent-actor@example.com", "127.0.0.1")
		results <- err
	}()
	go func() {
		<-start
		tx, err := db.Beginx()
		if err != nil {
			results <- err
			return
		}
		defer tx.Rollback()
		senderID, err := u.ResolveMergedContactIDTx(tx, remove.ID)
		if err == nil {
			_, err = tx.Exec(`INSERT INTO conversation_messages (conversation_id,sender_id,type,status,sender_type,content) VALUES ($1,$2,'incoming','received','contact','During merge')`, conversationID, senderID)
		}
		if err == nil {
			err = tx.Commit()
		}
		results <- err
	}()
	close(start)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var sender int
	if err := db.Get(&sender, `SELECT sender_id FROM conversation_messages WHERE conversation_id=$1`, conversationID); err != nil || sender != keep.ID {
		t.Fatalf("concurrent message lost: sender %d %v", sender, err)
	}
}

func TestContactMergeSyncAliases(t *testing.T) {
	u, db := newTestManager(t)
	keep := newContact("sync-merge-primary@example.com", "", "Selected", "Name")
	remove := newContact("sync-merge-alias@example.com", "", "Duplicate", "Name")
	resolve(t, u, keep, models.ContactReuse)
	resolve(t, u, remove, models.ContactReuse)
	actor := insertAgent(t, db, "sync-merge-actor@example.com", "")
	preview, err := u.PreviewContactMerge(keep.ID, remove.ID)
	if err != nil {
		t.Fatal(err)
	}
	req := ContactMergeRequest{RemoveID: remove.ID, Version: preview.Version, Choices: map[string]string{"email": "keep", "first_name": "keep"}}
	if _, err := u.MergeContacts(keep.ID, req, actor, "sync-merge-actor@example.com", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	for _, externalID := range []string{"", "sync-merged-customer"} {
		incoming := newContact("sync-merge-alias@example.com", externalID, "Incoming", "Name")
		resolve(t, u, incoming, models.ContactSync)
		if incoming.ID != keep.ID {
			t.Fatalf("sync created contact %d instead of reusing %d", incoming.ID, keep.ID)
		}
		row := fetchRow(t, db, keep.ID)
		if row.Email != "sync-merge-primary@example.com" || row.FirstName != "Selected" {
			t.Fatalf("alias intake replaced chosen profile details: %+v", row)
		}
		if externalID != "" && row.ExtID != externalID {
			t.Fatalf("alias intake did not enrich customer ID: %+v", row)
		}
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM users WHERE type = 'contact' AND deleted_at IS NULL`); err != nil || count != 1 {
		t.Fatalf("sync left duplicate contacts: %d %v", count, err)
	}
}

func resolve(t *testing.T, u *Manager, c *models.User, policy models.ContactPolicy) {
	t.Helper()
	if err := u.ResolveContact(c, policy); err != nil {
		t.Fatalf("ResolveContact: %v", err)
	}
	if c.ID <= 0 {
		t.Fatalf("ResolveContact returned no ID")
	}
}

func fetchRow(t *testing.T, db *sqlx.DB, id int) userRow {
	t.Helper()
	var r userRow
	err := db.QueryRow(`SELECT first_name, COALESCE(last_name, ''), COALESCE(email, ''), COALESCE(external_user_id, ''), type FROM users WHERE id = $1`,
		id).Scan(&r.FirstName, &r.LastName, &r.Email, &r.ExtID, &r.Type)
	if err != nil {
		t.Fatalf("fetching user %d: %v", id, err)
	}
	return r
}

func countByEmail(t *testing.T, db *sqlx.DB, email string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = $1 AND deleted_at IS NULL`, email).Scan(&n); err != nil {
		t.Fatalf("counting users with email %s: %v", email, err)
	}
	return n
}

func insertAgent(t *testing.T, db *sqlx.DB, email, extID string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO users (type, email, first_name, "password", external_user_id) VALUES ('agent', $1, 'Agent', 'x', $2) RETURNING id`,
		email, null.NewString(extID, extID != "")).Scan(&id)
	if err != nil {
		t.Fatalf("inserting agent: %v", err)
	}
	return id
}

func insertVisitor(t *testing.T, db *sqlx.DB, email string) int {
	t.Helper()
	var id int
	err := db.QueryRow(`INSERT INTO users (type, email, first_name) VALUES ('visitor', $1, 'Visitor') RETURNING id`, email).Scan(&id)
	if err != nil {
		t.Fatalf("inserting visitor: %v", err)
	}
	return id
}

func TestResolveContactReuse(t *testing.T) {
	u, db := newTestManager(t)

	// Creates a contact when none exists.
	alice := newContact("alice@example.com", "", "Alice", "Original")
	resolve(t, u, alice, models.ContactReuse)
	if r := fetchRow(t, db, alice.ID); r.Type != "contact" || r.FirstName != "Alice" {
		t.Fatalf("unexpected created row: %+v", r)
	}

	// A later reuse with a different name resolves to the same contact untouched.
	again := newContact("alice@example.com", "", "Changed", "Name")
	resolve(t, u, again, models.ContactReuse)
	if again.ID != alice.ID {
		t.Fatalf("expected reuse of contact %d, got %d", alice.ID, again.ID)
	}
	if r := fetchRow(t, db, alice.ID); r.FirstName != "Alice" || r.LastName != "Original" {
		t.Fatalf("reuse policy modified the contact: %+v", r)
	}

	// Email matching is case-insensitive.
	upper := newContact("ALICE@Example.COM", "", "X", "Y")
	resolve(t, u, upper, models.ContactReuse)
	if upper.ID != alice.ID {
		t.Fatalf("expected case-insensitive match of contact %d, got %d", alice.ID, upper.ID)
	}

	// Creates a contact with external ID when none matches.
	bob := newContact("bob@example.com", "ext-bob", "Bob", "")
	resolve(t, u, bob, models.ContactReuse)
	if r := fetchRow(t, db, bob.ID); r.ExtID != "ext-bob" {
		t.Fatalf("expected ext_id ext-bob, got %+v", r)
	}

	// External ID match wins over the passed email, and the stored email is returned for use as recipient.
	byExt := newContact("different@example.com", "ext-bob", "B", "")
	resolve(t, u, byExt, models.ContactReuse)
	if byExt.ID != bob.ID {
		t.Fatalf("expected ext_id match of contact %d, got %d", bob.ID, byExt.ID)
	}
	if byExt.Email.String != "bob@example.com" {
		t.Fatalf("expected stored email bob@example.com back, got %q", byExt.Email.String)
	}
	if r := fetchRow(t, db, bob.ID); r.Email != "bob@example.com" {
		t.Fatalf("reuse policy modified stored email: %+v", r)
	}

	// An unknown ext_id with an existing contact's email falls back to the email match instead of inserting a duplicate.
	staleExt := newContact("alice@example.com", "ext-unknown", "A", "")
	resolve(t, u, staleExt, models.ContactReuse)
	if staleExt.ID != alice.ID {
		t.Fatalf("expected email fallback to contact %d, got %d", alice.ID, staleExt.ID)
	}
	if n := countByEmail(t, db, "alice@example.com"); n != 1 {
		t.Fatalf("unknown ext_id created a duplicate contact, got %d rows", n)
	}

	// Plain email reuse also matches a contact that has an ext_id.
	byEmail := newContact("bob@example.com", "", "B", "")
	resolve(t, u, byEmail, models.ContactReuse)
	if byEmail.ID != bob.ID {
		t.Fatalf("expected email match of contact %d, got %d", bob.ID, byEmail.ID)
	}

	// An agent carrying the same external ID must never be resolved as the contact.
	agentID := insertAgent(t, db, "agent@example.com", "ext-agent")
	carol := newContact("carol@example.com", "ext-agent", "Carol", "")
	resolve(t, u, carol, models.ContactReuse)
	if carol.ID == agentID {
		t.Fatalf("resolved an agent (%d) as a contact", agentID)
	}
	if r := fetchRow(t, db, carol.ID); r.Type != "contact" {
		t.Fatalf("expected new contact, got %+v", r)
	}

	// An agent's email must never be resolved as the contact either.
	agentEmail := newContact("agent@example.com", "", "NotAgent", "")
	resolve(t, u, agentEmail, models.ContactReuse)
	if agentEmail.ID == agentID {
		t.Fatalf("resolved an agent (%d) as a contact by email", agentID)
	}

	// Soft-deleted contacts are not matched; a fresh contact is created.
	dave := newContact("dave@example.com", "", "Dave", "")
	resolve(t, u, dave, models.ContactReuse)
	db.MustExec(`UPDATE users SET deleted_at = now() WHERE id = $1`, dave.ID)
	dave2 := newContact("dave@example.com", "", "Dave", "")
	resolve(t, u, dave2, models.ContactReuse)
	if dave2.ID == dave.ID {
		t.Fatalf("resolved a soft-deleted contact %d", dave.ID)
	}

	stale2 := newContact("bob@example.com", "ext-unknown-2", "B", "")
	resolve(t, u, stale2, models.ContactReuse)
	if stale2.ID != bob.ID {
		t.Fatalf("expected email fallback to contact %d, got %d", bob.ID, stale2.ID)
	}
	if r := fetchRow(t, db, bob.ID); r.ExtID != "ext-bob" {
		t.Fatalf("email fallback modified the contact's ext_id: %+v", r)
	}
	if r := fetchRow(t, db, alice.ID); r.ExtID != "" {
		t.Fatalf("earlier fallback wrote the unknown ext_id onto the contact: %+v", r)
	}

	shared := newContact("shared@example.com", "", "Plain", "")
	resolve(t, u, shared, models.ContactSync)
	sharedExt := newContact("shared2@example.com", "ext-shared", "Ext", "")
	resolve(t, u, sharedExt, models.ContactSync)
	moved := newContact("shared@example.com", "ext-shared", "Ext", "")
	resolve(t, u, moved, models.ContactSync)
	if moved.ID != sharedExt.ID || moved.ID == shared.ID {
		t.Fatalf("setup: expected the ext contact to take the shared email, got %d", moved.ID)
	}
	for range 3 {
		pick := newContact("shared@example.com", "", "P", "")
		resolve(t, u, pick, models.ContactReuse)
		if pick.ID != sharedExt.ID {
			t.Fatalf("email match on a shared email is not deterministic: expected %d, got %d", sharedExt.ID, pick.ID)
		}
	}

	gone := newContact("gone@example.com", "ext-gone", "Gone", "")
	resolve(t, u, gone, models.ContactReuse)
	db.MustExec(`UPDATE users SET deleted_at = now() WHERE id = $1`, gone.ID)
	afterGone := newContact("alice@example.com", "ext-gone", "A", "")
	resolve(t, u, afterGone, models.ContactReuse)
	if afterGone.ID != alice.ID {
		t.Fatalf("expected email fallback past the deleted ext_id owner to %d, got %d", alice.ID, afterGone.ID)
	}

	// A deleted contact's ext_id is reusable without a unique violation.
	reborn := newContact("reborn@example.com", "ext-gone", "Reborn", "")
	resolve(t, u, reborn, models.ContactReuse)
	if reborn.ID == gone.ID {
		t.Fatalf("resurrected the soft-deleted contact %d", gone.ID)
	}
	if r := fetchRow(t, db, reborn.ID); r.ExtID != "ext-gone" {
		t.Fatalf("expected the freed ext_id on the new contact, got %+v", r)
	}

	anon := newContact("", "ext-anon", "Anon", "")
	resolve(t, u, anon, models.ContactReuse)
	anon2 := newContact("", "ext-anon", "Anon", "")
	resolve(t, u, anon2, models.ContactReuse)
	if anon2.ID != anon.ID {
		t.Fatalf("email-less ext_id resolve duplicated the contact: %d vs %d", anon.ID, anon2.ID)
	}

	visitorID := insertVisitor(t, db, "visitor@example.com")
	viaEmail := newContact("visitor@example.com", "", "V", "")
	resolve(t, u, viaEmail, models.ContactReuse)
	if viaEmail.ID == visitorID {
		t.Fatalf("resolved a visitor (%d) as a contact", visitorID)
	}
	if r := fetchRow(t, db, viaEmail.ID); r.Type != "contact" {
		t.Fatalf("expected a contact row, got %+v", r)
	}
}

func TestResolveContactSync(t *testing.T) {
	u, db := newTestManager(t)

	// Creates a contact, normalizing the email to lowercase.
	eve := newContact("EVE@Example.com", "", "Eve", "One")
	resolve(t, u, eve, models.ContactSync)
	if r := fetchRow(t, db, eve.ID); r.Email != "eve@example.com" {
		t.Fatalf("expected normalized email, got %+v", r)
	}

	// Sync by email updates the name on the matched contact.
	renamed := newContact("eve@example.com", "", "Eva", "Two")
	resolve(t, u, renamed, models.ContactSync)
	if renamed.ID != eve.ID {
		t.Fatalf("expected match of contact %d, got %d", eve.ID, renamed.ID)
	}
	if r := fetchRow(t, db, eve.ID); r.FirstName != "Eva" || r.LastName != "Two" {
		t.Fatalf("expected updated name, got %+v", r)
	}

	// Blank names do not clobber stored ones.
	blank := newContact("eve@example.com", "", "", "")
	resolve(t, u, blank, models.ContactSync)
	if r := fetchRow(t, db, eve.ID); r.FirstName != "Eva" || r.LastName != "Two" {
		t.Fatalf("blank names clobbered stored ones: %+v", r)
	}

	// Sync with an ext_id enriches the matched no-ext contact.
	enrich := newContact("eve@example.com", "ext-eve", "Eva", "")
	resolve(t, u, enrich, models.ContactSync)
	if enrich.ID != eve.ID {
		t.Fatalf("expected enrichment of contact %d, got %d", eve.ID, enrich.ID)
	}
	if r := fetchRow(t, db, eve.ID); r.ExtID != "ext-eve" {
		t.Fatalf("expected enriched ext_id, got %+v", r)
	}

	// Sync by ext_id updates email and name.
	updated := newContact("eve2@example.com", "ext-eve", "Evelyn", "Three")
	resolve(t, u, updated, models.ContactSync)
	if updated.ID != eve.ID {
		t.Fatalf("expected ext_id match of contact %d, got %d", eve.ID, updated.ID)
	}
	if r := fetchRow(t, db, eve.ID); r.Email != "eve2@example.com" || r.FirstName != "Evelyn" {
		t.Fatalf("expected updated email/name, got %+v", r)
	}

	// An empty email on an ext_id sync keeps the stored email.
	noEmail := newContact("", "ext-eve", "Evelyn", "")
	resolve(t, u, noEmail, models.ContactSync)
	if r := fetchRow(t, db, eve.ID); r.Email != "eve2@example.com" {
		t.Fatalf("empty email clobbered stored one: %+v", r)
	}

	// Sync without ext_id on an email owned by an ext_id contact reuses it without updating the name.
	plain := newContact("eve2@example.com", "", "Someone", "Else")
	resolve(t, u, plain, models.ContactSync)
	if plain.ID != eve.ID {
		t.Fatalf("expected reuse of ext_id contact %d, got %d", eve.ID, plain.ID)
	}
	if r := fetchRow(t, db, eve.ID); r.FirstName != "Evelyn" {
		t.Fatalf("no-ext sync modified an ext_id contact: %+v", r)
	}

	// The same email with a different ext_id creates a second contact - one email can map to multiple external IDs.
	other := newContact("eve2@example.com", "ext-other", "Other", "")
	resolve(t, u, other, models.ContactSync)
	if other.ID == eve.ID {
		t.Fatalf("expected a second contact for a different ext_id, got the same one")
	}
	if n := countByEmail(t, db, "eve2@example.com"); n != 2 {
		t.Fatalf("expected 2 contacts with the shared email, got %d", n)
	}

	// An ext_id already owned by another contact wins over an email-matched contact.
	frank := newContact("frank@example.com", "", "Frank", "")
	resolve(t, u, frank, models.ContactSync)
	taken := newContact("frank@example.com", "ext-eve", "Franklin", "")
	resolve(t, u, taken, models.ContactSync)
	if taken.ID != eve.ID {
		t.Fatalf("expected ext_id owner %d to win, got %d", eve.ID, taken.ID)
	}
	if r := fetchRow(t, db, frank.ID); r.ExtID != "" || r.FirstName != "Frank" {
		t.Fatalf("email-matched contact was modified: %+v", r)
	}

	// An agent's email must never be matched.
	agentID := insertAgent(t, db, "agent2@example.com", "")
	notAgent := newContact("agent2@example.com", "", "NotAgent", "")
	resolve(t, u, notAgent, models.ContactSync)
	if notAgent.ID == agentID {
		t.Fatalf("resolved an agent (%d) as a contact", agentID)
	}
	if r := fetchRow(t, db, agentID); r.FirstName != "Agent" {
		t.Fatalf("sync modified an agent row: %+v", r)
	}

	dead := newContact("dead@example.com", "", "Dead", "")
	resolve(t, u, dead, models.ContactSync)
	db.MustExec(`UPDATE users SET deleted_at = now() WHERE id = $1`, dead.ID)
	fresh := newContact("dead@example.com", "", "Fresh", "")
	resolve(t, u, fresh, models.ContactSync)
	if fresh.ID == dead.ID {
		t.Fatalf("sync resurrected the soft-deleted contact %d", dead.ID)
	}

	deadExt := newContact("deadext@example.com", "ext-dead", "Dead", "")
	resolve(t, u, deadExt, models.ContactSync)
	db.MustExec(`UPDATE users SET deleted_at = now() WHERE id = $1`, deadExt.ID)
	freshExt := newContact("deadext2@example.com", "ext-dead", "Fresh", "")
	resolve(t, u, freshExt, models.ContactSync)
	if freshExt.ID == deadExt.ID {
		t.Fatalf("sync resurrected the soft-deleted contact %d", deadExt.ID)
	}
	if r := fetchRow(t, db, freshExt.ID); r.ExtID != "ext-dead" || r.Email != "deadext2@example.com" {
		t.Fatalf("expected the freed ext_id on a fresh contact, got %+v", r)
	}

	visitorID := insertVisitor(t, db, "visitor2@example.com")
	viaEmail := newContact("visitor2@example.com", "ext-visitor", "V", "")
	resolve(t, u, viaEmail, models.ContactSync)
	if viaEmail.ID == visitorID {
		t.Fatalf("resolved a visitor (%d) as a contact", visitorID)
	}
	if r := fetchRow(t, db, visitorID); r.ExtID != "" {
		t.Fatalf("sync enriched a visitor row: %+v", r)
	}
}

func TestResolveContactConcurrent(t *testing.T) {
	u, db := newTestManager(t)

	race := func(policy models.ContactPolicy, email, extID string) {
		t.Helper()
		var (
			wg  sync.WaitGroup
			mu  sync.Mutex
			ids = map[int]struct{}{}
		)
		for range 10 {
			wg.Go(func() {
				c := newContact(email, extID, "Race", "")
				if err := u.ResolveContact(c, policy); err != nil {
					t.Errorf("ResolveContact: %v", err)
					return
				}
				mu.Lock()
				ids[c.ID] = struct{}{}
				mu.Unlock()
			})
		}
		wg.Wait()
		if len(ids) != 1 {
			t.Fatalf("concurrent resolves produced %d distinct contacts: %v", len(ids), ids)
		}
		if n := countByEmail(t, db, email); n != 1 {
			t.Fatalf("expected 1 contact with email %s, got %d", email, n)
		}
	}

	race(models.ContactReuse, "race-reuse@example.com", "")
	race(models.ContactReuse, "race-reuse-ext@example.com", "ext-race-reuse")
	race(models.ContactSync, "race-sync@example.com", "")
	race(models.ContactSync, "race-sync-ext@example.com", "ext-race-sync")
}
