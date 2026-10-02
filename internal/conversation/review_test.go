package conversation

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/envelope"
	imodels "github.com/jakedolan443/fernmail/internal/inbox/models"
	mediamanager "github.com/jakedolan443/fernmail/internal/media"
	mmodels "github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jakedolan443/fernmail/internal/template"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/jakedolan443/fernmail/internal/user"
	umodels "github.com/jakedolan443/fernmail/internal/user/models"
	"github.com/jakedolan443/fernmail/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// reviewInboxStore reads transport records straight from the test database.
type reviewInboxStore struct {
	inboxStore
	db *sqlx.DB
}

func (s reviewInboxStore) GetDBRecord(id any) (imodels.Inbox, error) {
	var record imodels.Inbox
	err := s.db.Get(&record, `SELECT id, uuid, created_at, updated_at, name, channel, enabled, csat_enabled, "from", from_name_template, config, secret, linked_email_inbox_id FROM inboxes WHERE id=$1`, id)
	return record, err
}

type reviewFixture struct {
	m            *Manager
	db           *sqlx.DB
	files        *mediamanager.Manager
	users        *user.Manager
	addressID    int
	conversation models.Conversation
	contributor  umodels.User
	reviewer     umodels.User
	outsider     umodels.User
	sender       string
}

func newReviewFixture(t *testing.T, name string) reviewFixture {
	t.Helper()
	db := testutil.NewDB(t, name)
	lo := logf.New(logf.Opts{})
	i18n := testutil.NewI18n(t)
	users, err := user.New(i18n, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	files, err := mediamanager.New(mediamanager.Opts{DB: db, Lo: &lo, I18n: i18n, Store: draftBlobStore{}, RootURL: func() string { return "https://desk.test" }, SigningKey: "test"})
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(ws.NewHub(&lo, nil), i18n, reviewInboxStore{db: db}, users, files, stubSettingsStore{}, &template.Manager{}, receiptWebhookStore{}, Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	f := reviewFixture{m: m, db: db, files: files, users: users, sender: "sender@example.test"}
	var inboxID int
	if err := db.Get(&inboxID, `INSERT INTO inboxes(name,channel,"from",enabled) VALUES('Mail','email','mail@example.test',true) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&f.addressID, `INSERT INTO email_addresses(inbox_id,address,kind) VALUES($1,'mail@example.test','mailbox') RETURNING id`, inboxID); err != nil {
		t.Fatal(err)
	}
	agent := func(email, role string, granted bool) umodels.User {
		var id int
		if err := db.Get(&id, `INSERT INTO users(type,email,first_name,last_name) VALUES('agent',$1,$2,'Tester') RETURNING id`, email, strings.Split(email, "@")[0]); err != nil {
			t.Fatal(err)
		}
		db.MustExec(`INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name=$2`, id, role)
		if granted {
			db.MustExec(`INSERT INTO email_address_users(address_id,user_id) VALUES($1,$2)`, f.addressID, id)
		}
		loaded, err := users.GetAgentCachedOrLoad(id)
		if err != nil {
			t.Fatal(err)
		}
		return loaded
	}
	f.contributor = agent("contributor@example.test", "Contributor", true)
	f.reviewer = agent("reviewer@example.test", "Agent", true)
	f.outsider = agent("outsider@example.test", "Agent", false)
	incoming, err := m.ProcessIncomingMessage(models.IncomingMessage{InboxID: inboxID, Channel: "email", EmailAlias: "mail@example.test",
		Contact: models.IncomingContact{Email: null.StringFrom(f.sender), FirstName: "Sender"}, Subject: "Question", Content: "Can you help?",
		ContentType: models.ContentTypeText, SourceID: null.StringFrom("question@example.test")})
	if err != nil {
		t.Fatal(err)
	}
	if f.conversation, err = m.GetConversation(incoming.ConversationID, "", ""); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f reviewFixture) upload(t *testing.T, owner int) mmodels.Media {
	t.Helper()
	file, err := f.files.Insert(null.StringFrom("attachment"), "notes.txt", "text/plain", "", null.StringFrom(mmodels.ModelMessages), uuid.NewString(), null.Int{}, 7, []byte(`{}`), true, owner)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func (f reviewFixture) reply(t *testing.T, content string, media ...mmodels.Media) (models.Review, error) {
	t.Helper()
	return f.m.SubmitReview(f.contributor.ID, models.ReviewInput{AddressID: f.addressID, ConversationID: f.conversation.ID,
		Content: content, To: []string{f.sender}, Media: media})
}

func errorType(err error) string {
	var envErr envelope.Error
	if errors.As(err, &envErr) {
		return string(envErr.ErrorType)
	}
	return ""
}

func countRows(t *testing.T, db *sqlx.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.Get(&n, query, args...); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDeniedReplyReturnsToDraftAndApprovalSendsOnce(t *testing.T) {
	f := newReviewFixture(t, "review_reply")
	file := f.upload(t, f.contributor.ID)
	submitted, err := f.reply(t, "<p>Happy to help</p>", file)
	if err != nil {
		t.Fatal(err)
	}
	if submitted.Status != models.ReviewStatusPending || submitted.Kind != models.ReviewKindReply || len(submitted.Attachments) != 1 {
		t.Fatalf("submission = %+v", submitted)
	}
	if _, err := f.reply(t, "<p>Second try</p>"); errorType(err) != envelope.ConflictError {
		t.Fatalf("second pending reply error = %v", err)
	}
	// Nothing leaves Fernmail until approval.
	if n := countRows(t, f.db, `SELECT count(*) FROM conversation_messages WHERE type='outgoing'`); n != 0 {
		t.Fatalf("submission queued %d messages", n)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM media WHERE id=$1 AND NOT EXISTS (SELECT 1 FROM outbound_review_media rm WHERE rm.media_id=media.id)`, file.ID); n != 0 {
		t.Fatal("submission attachment is not protected from cleanup")
	}

	queue, err := f.m.ListReviews(f.reviewer, true)
	if err != nil || len(queue) != 1 {
		t.Fatalf("reviewer queue = %d err=%v", len(queue), err)
	}
	if queue, _ := f.m.ListReviews(f.outsider, true); len(queue) != 0 {
		t.Fatal("reviewer without address access sees the submission")
	}
	if counts, _ := f.m.CountReviews(f.reviewer, true); counts.Pending != 1 {
		t.Fatalf("reviewer counts = %+v", counts)
	}

	denied, err := f.m.ReturnReview(submitted.UUID, &f.reviewer, "Please add a greeting")
	if err != nil || denied.Status != models.ReviewStatusDenied || denied.DecisionNote != "Please add a greeting" {
		t.Fatalf("deny = %+v err=%v", denied, err)
	}
	var draft struct {
		ID      int             `db:"id"`
		Content string          `db:"content"`
		Meta    json.RawMessage `db:"meta"`
	}
	if err := f.db.Get(&draft, `SELECT id, content, meta FROM conversation_drafts WHERE conversation_id=$1 AND user_id=$2 AND type='reply'`, f.conversation.ID, f.contributor.ID); err != nil {
		t.Fatalf("denied reply not returned to draft: %v", err)
	}
	var meta struct {
		Attachments []struct {
			ID int `json:"id"`
		} `json:"attachments"`
		Recipients map[string]string `json:"recipients"`
	}
	if err := json.Unmarshal(draft.Meta, &meta); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(draft.Content, "Happy to help") || len(meta.Attachments) != 1 || meta.Attachments[0].ID != file.ID || meta.Recipients["to"] != f.sender {
		t.Fatalf("restored draft = %s %s", draft.Content, draft.Meta)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM conversation_draft_media WHERE draft_id=$1 AND media_id=$2`, draft.ID, file.ID); n != 1 {
		t.Fatal("returned attachment lost its retention reference")
	}
	if _, err := f.m.ReturnReview(submitted.UUID, &f.reviewer, ""); errorType(err) != envelope.ConflictError {
		t.Fatalf("second decision error = %v", err)
	}
	notices, err := f.m.ConversationReviews(f.conversation.ID, f.contributor, false)
	if err != nil || len(notices) != 1 || notices[0].Status != models.ReviewStatusDenied {
		t.Fatalf("author notice = %+v err=%v", notices, err)
	}

	// Resubmit, then approve twice at once: exactly one message is queued.
	resubmitted, err := f.reply(t, "<p>Hi! Happy to help</p>", file)
	if err != nil {
		t.Fatal(err)
	}
	if notices, _ := f.m.ConversationReviews(f.conversation.ID, f.contributor, false); len(notices) != 1 || notices[0].UUID != resubmitted.UUID {
		t.Fatalf("old denial still shown after resubmitting: %+v", notices)
	}
	var wg sync.WaitGroup
	results := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, results[i] = f.m.ApproveReview(resubmitted.UUID, f.reviewer)
		}()
	}
	wg.Wait()
	succeeded := 0
	for _, err := range results {
		if err == nil {
			succeeded++
		} else if errorType(err) != envelope.ConflictError {
			t.Fatalf("losing approval error = %v", err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("approvals succeeded = %d", succeeded)
	}
	var sent struct {
		ID       int    `db:"id"`
		SenderID int    `db:"sender_id"`
		Status   string `db:"status"`
	}
	if err := f.db.Get(&sent, `SELECT id, sender_id, status FROM conversation_messages WHERE type='outgoing'`); err != nil {
		t.Fatalf("expected exactly one queued message: %v", err)
	}
	if sent.SenderID != f.contributor.ID || sent.Status != models.MessageStatusPending {
		t.Fatalf("approved message = %+v", sent)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM media WHERE id=$1 AND model_id=$2`, file.ID, sent.ID); n != 1 {
		t.Fatal("approved attachment not linked to the message")
	}
	approved, err := f.m.GetReview(resubmitted.UUID)
	if err != nil || approved.Status != models.ReviewStatusApproved || !approved.MessageUUID.Valid || approved.ReviewerName.String == "" {
		t.Fatalf("approved review = %+v err=%v", approved, err)
	}

	// Withdrawing puts the reply back in the author's composer too.
	pending, err := f.reply(t, "<p>Follow-up</p>")
	if err != nil {
		t.Fatal(err)
	}
	withdrawn, err := f.m.ReturnReview(pending.UUID, nil, "")
	if err != nil || withdrawn.Status != models.ReviewStatusWithdrawn || withdrawn.ReviewerID.Valid {
		t.Fatalf("withdraw = %+v err=%v", withdrawn, err)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM conversation_drafts WHERE user_id=$1 AND content LIKE '%Follow-up%'`, f.contributor.ID); n != 1 {
		t.Fatal("withdrawn reply not returned to draft")
	}
}

func TestNewEmailReviewCreatesConversationOnlyWhenApproved(t *testing.T) {
	f := newReviewFixture(t, "review_new")
	input := models.ReviewInput{AddressID: f.addressID, Subject: "Introduction", Content: "<p>Hello there</p>", To: []string{"new@example.test"}, CC: []string{"cc@example.test"}}
	if _, err := f.m.SubmitReview(f.contributor.ID, models.ReviewInput{AddressID: f.addressID, Content: "<p>No subject</p>", To: []string{"new@example.test"}}); errorType(err) != envelope.InputError {
		t.Fatalf("missing subject error = %v", err)
	}
	if _, err := f.m.SubmitReview(f.contributor.ID, models.ReviewInput{AddressID: f.addressID, Subject: "x", Content: "<p>x</p>", To: []string{"not an email"}}); errorType(err) != envelope.InputError {
		t.Fatalf("invalid recipient error = %v", err)
	}
	conversations := countRows(t, f.db, `SELECT count(*) FROM conversations`)
	submitted, err := f.m.SubmitReview(f.contributor.ID, input)
	if err != nil || submitted.Kind != models.ReviewKindNew || submitted.ConversationUUID.Valid {
		t.Fatalf("submission = %+v err=%v", submitted, err)
	}
	if countRows(t, f.db, `SELECT count(*) FROM conversations`) != conversations {
		t.Fatal("a conversation exists before approval")
	}

	if _, err := f.m.ReturnReview(submitted.UUID, &f.reviewer, "Wrong address"); err != nil {
		t.Fatal(err)
	}
	mine, err := f.m.ListReviews(f.contributor, false)
	if err != nil || len(mine) != 1 || mine[0].Status != models.ReviewStatusDenied {
		t.Fatalf("returned new email not listed for its author: %+v err=%v", mine, err)
	}
	if counts, _ := f.m.CountReviews(f.contributor, false); counts.Returned != 1 || counts.Pending != 0 {
		t.Fatalf("contributor counts = %+v", counts)
	}
	input.Subject = "Introduction, revised"
	if _, err := f.m.ResubmitReview(submitted.UUID, f.outsider.ID, input); errorType(err) != envelope.ConflictError {
		t.Fatalf("another user resubmitted: %v", err)
	}
	resubmitted, err := f.m.ResubmitReview(submitted.UUID, f.contributor.ID, input)
	if err != nil || resubmitted.Status != models.ReviewStatusPending || resubmitted.DecisionNote != "" {
		t.Fatalf("resubmit = %+v err=%v", resubmitted, err)
	}

	approved, _, err := f.m.ApproveReview(submitted.UUID, f.reviewer)
	if err != nil || !approved.ConversationUUID.Valid {
		t.Fatalf("approve = %+v err=%v", approved, err)
	}
	created, err := f.m.GetConversation(0, approved.ConversationUUID.String, "")
	if err != nil || created.Subject.String != "Introduction, revised" || created.AddressID.Int != f.addressID {
		t.Fatalf("created conversation = %+v err=%v", created, err)
	}
	var meta struct {
		To []string `json:"to"`
		CC []string `json:"cc"`
	}
	var raw json.RawMessage
	if err := f.db.Get(&raw, `SELECT meta FROM conversation_messages WHERE conversation_id=$1 AND type='outgoing' AND sender_id=$2`, created.ID, f.contributor.ID); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &meta); err != nil || len(meta.To) != 1 || meta.To[0] != "new@example.test" || len(meta.CC) != 1 {
		t.Fatalf("queued recipients = %s err=%v", raw, err)
	}
	if err := f.m.DiscardReview(submitted.UUID, f.contributor.ID); errorType(err) != envelope.ConflictError {
		t.Fatalf("discarding an approved email error = %v", err)
	}
}

func TestReviewsOfAuthorsWhoLostAccessCannotBeApproved(t *testing.T) {
	f := newReviewFixture(t, "review_author_access")
	submitted, err := f.reply(t, "<p>Thanks</p>")
	if err != nil {
		t.Fatal(err)
	}
	f.db.MustExec(`UPDATE users SET enabled=false WHERE id=$1`, f.contributor.ID)
	if queue, _ := f.m.ListReviews(f.reviewer, true); len(queue) != 0 {
		t.Fatal("disabled author's email still in the queue")
	}
	if _, _, err := f.m.ApproveReview(submitted.UUID, f.reviewer); errorType(err) != envelope.InputError {
		t.Fatalf("approving for a disabled author error = %v", err)
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM conversation_messages WHERE type='outgoing'`); n != 0 {
		t.Fatal("email queued for an author who lost access")
	}
}

func TestAttachmentsOfOtherUsersCannotBeSubmitted(t *testing.T) {
	f := newReviewFixture(t, "review_media_owner")
	theirs := f.upload(t, f.reviewer.ID)
	if _, err := f.reply(t, "<p>Borrowed file</p>", theirs); err == nil {
		t.Fatal("another user's upload was accepted")
	}
	if n := countRows(t, f.db, `SELECT count(*) FROM outbound_reviews`); n != 0 {
		t.Fatal("rejected submission partially persisted")
	}
}
