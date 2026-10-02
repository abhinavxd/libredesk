package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jakedolan443/fernmail/internal/dbutil"
	mediamanager "github.com/jakedolan443/fernmail/internal/media"
	mmodels "github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type draftBlobStore struct{ mediamanager.Store }

func (draftBlobStore) Name() string { return "fs" }

func TestDraftMediaOwnershipRetentionAndCanonicalMetadata(t *testing.T) {
	db := testutil.NewDB(t, "draft_ownership")
	lo := logf.New(logf.Opts{})
	i18n := testutil.NewI18n(t)
	files, err := mediamanager.New(mediamanager.Opts{DB: db, Lo: &lo, I18n: i18n, Store: draftBlobStore{}, RootURL: func() string { return "https://desk.test" }, SigningKey: "test"})
	if err != nil {
		t.Fatal(err)
	}
	manager := newTestManager()
	manager.db, manager.lo, manager.i18n, manager.mediaStore = db, &lo, i18n, files
	if err := dbutil.ScanSQLFile("queries.sql", &manager.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var owner, other, conversation int
	for _, r := range []struct {
		out *int
		q   string
	}{{&owner, `INSERT INTO users(type,email,first_name) VALUES('agent','owner@test','Owner') RETURNING id`}, {&other, `INSERT INTO users(type,email,first_name) VALUES('agent','other@test','Other') RETURNING id`}} {
		if err := db.Get(r.out, r.q); err != nil {
			t.Fatal(err)
		}
	}
	db.MustExec(`INSERT INTO inboxes(name,channel,"from") VALUES('Mail','email','mail@test'); INSERT INTO email_addresses(inbox_id,address,kind) SELECT id,'mail@test','mailbox' FROM inboxes;
		INSERT INTO email_address_users(address_id,user_id) SELECT a.id,u.id FROM email_addresses a CROSS JOIN users u WHERE u.type='agent'`)
	if err := db.Get(&conversation, `INSERT INTO conversations(contact_id,inbox_id,address_id,status_id) SELECT $1,a.inbox_id,a.id,s.id FROM email_addresses a CROSS JOIN conversation_statuses s WHERE s.name='Open' RETURNING id`, owner); err != nil {
		t.Fatal(err)
	}
	owned, err := files.Insert(null.StringFrom("attachment"), "real.txt", "text/plain", "", null.StringFrom("messages"), uuid.NewString(), null.Int{}, 7, []byte(`{}`), true, owner)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		source  string
		claimed bool
	}{
		{"/uploads/" + owned.UUID, true},
		{"cid:ldsk-" + owned.UUID, true},
		{"https://legacy-bucket.s3.amazonaws.com/" + owned.UUID + "?X-Amz-Signature=old", true},
		{"https://customer.test/images/" + uuid.NewString() + ".png", false},
	} {
		body := `<img src="` + tc.source + `">`
		normalized, claims := manager.normalizeInlineUploads(body, conversation, owner)
		if tc.claimed {
			if len(claims) != 1 || claims[0] != owned.UUID || !strings.Contains(normalized, "cid:ldsk-"+owned.UUID) {
				t.Fatalf("upload reference not claimed: %s -> %s, %v", tc.source, normalized, claims)
			}
		} else if normalized != body || len(claims) != 0 {
			t.Fatalf("external image mistaken for attachment: %s", normalized)
		}
	}
	meta := json.RawMessage(fmt.Sprintf(`{"attachments":[{"id":%d,"uuid":"forged","filename":"forged","url":"https://evil.test"}]}`, owned.ID))
	if _, err := manager.UpsertConversationDraft(conversation, other, "reply", "content", meta); err == nil {
		t.Fatal("cross-user attachment accepted")
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM conversation_drafts`)
	if count != 0 {
		t.Fatal("rejected draft partially persisted")
	}
	inline := fmt.Sprintf(`<img src="/uploads/%s">`, owned.UUID)
	if _, err := manager.UpsertConversationDraft(conversation, other, "reply", inline, json.RawMessage(`{}`)); err == nil {
		t.Fatal("cross-user inline image accepted")
	}
	draft, err := manager.UpsertConversationDraft(conversation, owner, "reply", inline, meta)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(draft.Content, "sig=") {
		t.Fatalf("owned inline image not resolved: %s", draft.Content)
	}
	var returned struct {
		Attachments []mmodels.Media `json:"attachments"`
	}
	if err := json.Unmarshal(draft.Meta, &returned); err != nil {
		t.Fatal(err)
	}
	if len(returned.Attachments) != 1 || returned.Attachments[0].Filename != "real.txt" || !strings.Contains(returned.Attachments[0].URL, "sig=") {
		t.Fatalf("browser metadata trusted: %s", draft.Meta)
	}
	if err := db.Get(&count, `SELECT count(*) FROM conversation_draft_media WHERE draft_id=$1 AND media_id=$2`, draft.ID, owned.ID); err != nil || count != 1 {
		t.Fatalf("missing retention reference: count=%d err=%v", count, err)
	}
	// Editing a draft renews its retention period; creation date must not expire active work.
	db.MustExec(`UPDATE conversation_drafts SET created_at=NOW()-INTERVAL '30 days',updated_at=NOW()-INTERVAL '20 days' WHERE id=$1`, draft.ID)
	if _, err := manager.UpsertConversationDraft(conversation, owner, "reply", inline, meta); err != nil {
		t.Fatal(err)
	}
	if err := manager.DeleteStaleDrafts(context.Background(), 15*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	got, err := manager.GetAllUserDrafts(owner)
	if err != nil || len(got) != 1 {
		t.Fatalf("active draft expired: %v %v", got, err)
	}
	// Removing the attachment also removes the retention reference atomically.
	if _, err := manager.UpsertConversationDraft(conversation, owner, "reply", "plain", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	db.Get(&count, `SELECT count(*) FROM conversation_draft_media`)
	if count != 0 {
		t.Fatal("removed attachment still retained")
	}
	// A linked historic message attachment can be quoted by another reader of this conversation.
	var message int
	if err := db.Get(&message, `INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status) VALUES($1,$2,'agent','outgoing','sent') RETURNING id`, conversation, owner); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`UPDATE media SET model_id=$1 WHERE id=$2`, message, owned.ID)
	if _, err := manager.UpsertConversationDraft(conversation, other, "reply", inline, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("historic same-conversation quote rejected: %v", err)
	}
	// Legacy unattributed pending files are never assigned an owner from draft metadata.
	db.MustExec(`UPDATE media SET model_id=NULL,uploaded_by=NULL WHERE id=$1`, owned.ID)
	if _, err := manager.UpsertConversationDraft(conversation, owner, "reply", inline, meta); err == nil {
		t.Fatal("legacy unknown uploader granted by metadata")
	}
}

func TestDraftMediaReferenceLimit(t *testing.T) {
	entries := make([]map[string]int, mediamanager.MaxMessageMedia+1)
	for i := range entries {
		entries[i] = map[string]int{"id": i + 1}
	}
	raw, _ := json.Marshal(map[string]any{"attachments": entries})
	if _, _, err := draftMediaReferences("", raw); err == nil {
		t.Fatal("oversized attachment list accepted")
	}
	if _, _, err := draftMediaReferences("", json.RawMessage(`{"attachments":[{"id":-1}]}`)); err == nil {
		t.Fatal("invalid media ID accepted")
	}
}
