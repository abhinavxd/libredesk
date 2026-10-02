package media

import (
	"bytes"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/volatiletech/null/v9"
)

func ownershipFixture(t *testing.T, name string) (*Manager, *quotaTestStore, int, int, int, int) {
	t.Helper()
	m, store := quotaFixture(t, name, 0)
	var owner, other, conversation, otherConversation int
	for _, row := range []struct {
		out *int
		q   string
	}{
		{&owner, `INSERT INTO users(type,email,first_name) VALUES('agent','owner@test','Owner') RETURNING id`},
		{&other, `INSERT INTO users(type,email,first_name) VALUES('agent','other@test','Other') RETURNING id`},
	} {
		if err := m.db.Get(row.out, row.q); err != nil {
			t.Fatal(err)
		}
	}
	m.db.MustExec(`INSERT INTO inboxes(name,channel) VALUES('Test','email')`)
	for _, out := range []*int{&conversation, &otherConversation} {
		if err := m.db.Get(out, `INSERT INTO conversations(contact_id,inbox_id,status_id) SELECT $1,i.id,s.id FROM inboxes i CROSS JOIN conversation_statuses s WHERE s.name='Open' RETURNING id`, owner); err != nil {
			t.Fatal(err)
		}
	}
	return m, store, owner, other, conversation, otherConversation
}
func uploadOwned(t *testing.T, m *Manager, userID int) models.Media {
	t.Helper()
	item, err := m.UploadForUser("secret.txt", "text/plain", null.StringFrom(models.ModelMessages), bytes.NewReader([]byte("private draft")), 13, null.StringFrom("attachment"), []byte(`{}`), userID)
	if err != nil {
		t.Fatal(err)
	}
	return item
}
func messageFor(t *testing.T, m *Manager, user, conversation int) int {
	t.Helper()
	var id int
	if err := m.db.Get(&id, `INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status) VALUES($1,$2,'agent','outgoing','sent') RETURNING id`, conversation, user); err != nil {
		t.Fatal(err)
	}
	return id
}
func TestUploadOwnershipAndAtomicMessageClaim(t *testing.T) {
	m, _, owner, other, conversation, otherConversation := ownershipFixture(t, "upload_ownership")
	victim := uploadOwned(t, m, owner)
	if _, err := m.GetPendingForUser([]int{victim.ID}, other); err == nil {
		t.Fatal("another user resolved a pending attachment ID")
	}
	if _, err := m.GetDraftInlineMedia(victim.UUID, otherConversation, other); err == nil {
		t.Fatal("another user resolved pending inline UUID")
	}
	if _, err := m.GetPendingForUser([]int{victim.ID}, owner); err != nil {
		t.Fatal(err)
	}
	if victim.UploadedBy.Int != owner {
		t.Fatal("owner missing from upload reservation")
	}
	for _, inline := range []bool{false, true} {
		tx := m.db.MustBegin()
		id := messageFor(t, m, other, otherConversation)
		files := []models.Media{victim}
		uuids := []string{}
		if inline {
			files = nil
			uuids = []string{victim.UUID}
		}
		if err := m.LinkMessageMediaTx(tx, id, files, uuids, other); err == nil {
			t.Fatalf("cross-user claim allowed (inline=%v)", inline)
		}
		tx.Rollback()
	}
	first, second := messageFor(t, m, owner, conversation), messageFor(t, m, owner, conversation)
	var wins atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, id := range []int{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			tx, err := m.db.Beginx()
			if err != nil {
				t.Error(err)
				return
			}
			defer tx.Rollback()
			if err = m.LinkMessageMediaTx(tx, id, []models.Media{victim}, nil, owner); err == nil {
				if err = tx.Commit(); err != nil {
					t.Error(err)
				} else {
					wins.Add(1)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("claimed same file %d times", wins.Load())
	}
	saved, err := m.Get(victim.ID, "")
	if err != nil || saved.ModelID.Int == 0 {
		t.Fatalf("successful claim not persisted: %+v %v", saved, err)
	}
	// A colleague may quote a historic attachment in the same mailbox conversation,
	// but it remains attached to its original message and cannot cross conversations.
	quoted := messageFor(t, m, other, conversation)
	tx := m.db.MustBegin()
	if err := m.LinkMessageMediaTx(tx, quoted, nil, []string{victim.UUID}, other); err != nil {
		t.Fatalf("same-conversation quote: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx = m.db.MustBegin()
	if err := m.LinkMessageMediaTx(tx, messageFor(t, m, other, otherConversation), nil, []string{victim.UUID}, other); err == nil {
		t.Fatal("cross-conversation quote allowed")
	}
	tx.Rollback()
	// UUID aliases preserve quoted MIME attachments whose original CID is not
	// the application's ldsk format. Both aliases resolve inside this thread.
	m.db.MustExec(`UPDATE media SET content_id='original@sender.test' WHERE id=$1`, victim.ID)
	var conversationUUID, otherUUID string
	m.db.Get(&conversationUUID, `SELECT uuid::text FROM conversations WHERE id=$1`, conversation)
	m.db.Get(&otherUUID, `SELECT uuid::text FROM conversations WHERE id=$1`, otherConversation)
	refs, err := m.GetByContentIDs([]string{"original@sender.test", "ldsk-" + victim.UUID}, conversationUUID)
	if err != nil || len(refs) != 2 {
		t.Fatalf("historic quote aliases: %+v %v", refs, err)
	}
	refs, err = m.GetByContentIDs([]string{"original@sender.test", "ldsk-" + victim.UUID}, otherUUID)
	if err != nil || len(refs) != 0 {
		t.Fatalf("historic quote crossed conversations: %+v %v", refs, err)
	}
	original, _ := m.Get(victim.ID, "")
	if original.ModelID.Int != saved.ModelID.Int {
		t.Fatal("quote stole original attachment")
	}
	if _, err := m.GetDraftInlineMedia(victim.UUID, conversation, other); err != nil {
		t.Fatalf("historic attachment no longer readable: %v", err)
	}
}
func TestIncomingMediaClaimsDoNotBypassBrowserOwnership(t *testing.T) {
	m, _, owner, _, conversation, _ := ownershipFixture(t, "incoming_media_ownership")
	incoming, err := m.UploadAndInsert("mail.txt", "text/plain", "", null.String{}, null.Int{}, bytes.NewReader([]byte("mail")), 4, null.StringFrom("attachment"), []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	id := messageFor(t, m, owner, conversation)
	tx := m.db.MustBegin()
	if err := m.LinkMessageMediaTx(tx, id, []models.Media{incoming}, nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	browser := uploadOwned(t, m, owner)
	tx = m.db.MustBegin()
	defer tx.Rollback()
	if err := m.LinkMessageMediaTx(tx, id, []models.Media{browser}, nil, 0); err == nil {
		t.Fatal("ingestion claimed a browser-owned file")
	}
}
func TestDraftReferencesProtectUploadsUntilDraftDeletion(t *testing.T) {
	m, store, owner, _, conversation, _ := ownershipFixture(t, "draft_media_cleanup")
	file := uploadOwned(t, m, owner)
	var draftID int
	meta, _ := json.Marshal(map[string]any{"attachments": []models.Media{file}})
	if err := m.db.Get(&draftID, `INSERT INTO conversation_drafts(conversation_id,user_id,content,meta) VALUES($1,$2,'Draft',$3) RETURNING id`, conversation, owner, meta); err != nil {
		t.Fatal(err)
	}
	m.db.MustExec(`INSERT INTO conversation_draft_media(draft_id,media_id) VALUES($1,$2)`, draftID, file.ID)
	m.db.MustExec(`UPDATE media SET created_at=NOW()-INTERVAL '10 days' WHERE id=$1`, file.ID)
	m.deleteUnlinked()
	if _, err := m.Get(file.ID, ""); err != nil {
		t.Fatalf("live draft upload collected: %v", err)
	}
	if _, ok := store.blobs[file.UUID]; !ok {
		t.Fatal("live draft bytes collected")
	}
	m.db.MustExec(`DELETE FROM conversation_drafts WHERE id=$1`, draftID)
	m.deleteUnlinked()
	if _, err := m.Get(file.ID, ""); err == nil {
		t.Fatal("orphan upload not collected")
	}
	if _, ok := store.blobs[file.UUID]; ok {
		t.Fatal("orphan bytes not collected")
	}
}
