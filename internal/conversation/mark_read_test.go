package conversation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jakedolan443/fernmail/internal/conversation/models"
	mmodels "github.com/jakedolan443/fernmail/internal/media/models"
	"github.com/jmoiron/sqlx"

	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/zerodha/logf"
)

func TestMarkAddressReadScopesUserAddressAndSnapshot(t *testing.T) {
	db := testutil.NewDB(t, "mark_address_read")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t)}
	var reader, teammate, sender, inbox, addressA, addressB int
	for _, item := range []struct {
		dest  *int
		query string
	}{
		{&reader, `INSERT INTO users(type,email,first_name) VALUES('agent','reader@example.test','Reader') RETURNING id`},
		{&teammate, `INSERT INTO users(type,email,first_name) VALUES('agent','teammate@example.test','Teammate') RETURNING id`},
		{&sender, `INSERT INTO users(type,email,first_name) VALUES('contact','sender@example.test','Sender') RETURNING id`},
		{&inbox, `INSERT INTO inboxes(name,channel) VALUES('Mail','email') RETURNING id`},
	} {
		if err := db.Get(item.dest, item.query); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Get(&addressA, `INSERT INTO email_addresses(inbox_id,address) VALUES($1,'a@example.test') RETURNING id`, inbox); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&addressB, `INSERT INTO email_addresses(inbox_id,address) VALUES($1,'b@example.test') RETURNING id`, inbox); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO email_address_users(address_id,user_id) SELECT a.id,u.id FROM email_addresses a CROSS JOIN users u WHERE u.type='agent'`)
	var conversationA int
	for _, status := range []string{"Open", "Closed", "Snoozed"} {
		var id int
		if err := db.Get(&id, `INSERT INTO conversations(contact_id,inbox_id,address_id,status_id)
			VALUES($1,$2,$3,(SELECT id FROM conversation_statuses WHERE name=$4)) RETURNING id`, sender, inbox, addressA, status); err != nil {
			t.Fatal(err)
		}
		conversationA = id
		// More messages than a page, including resolved and snoozed conversations.
		db.MustExec(`INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,created_at)
			SELECT $1,$2,'contact','incoming','received',NOW()-interval '1 minute' FROM generate_series(1,205)`, id, sender)
	}
	var conversationB int
	if err := db.Get(&conversationB, `INSERT INTO conversations(contact_id,inbox_id,address_id,status_id)
		VALUES($1,$2,$3,(SELECT id FROM conversation_statuses WHERE name='Open')) RETURNING id`, sender, inbox, addressB); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status) VALUES($1,$2,'contact','incoming','received')`, conversationB, sender)
	permissions := []string{"conversations:read", "conversations:read_all"}
	assertCounts := func(user, wantA, wantB int) {
		t.Helper()
		counts, err := m.GetSidebarCounts(user, permissions, nil)
		if err != nil {
			t.Fatal(err)
		}
		if counts.Addresses[addressA] != wantA || counts.Addresses[addressB] != wantB {
			t.Fatalf("user %d counts=%v, want A=%d B=%d", user, counts, wantA, wantB)
		}
	}
	assertCounts(reader, 615, 1)
	markedAt, err := m.MarkAddressRead(context.Background(), reader, addressA, permissions, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertCounts(reader, 0, 1)
	assertCounts(teammate, 615, 1)
	// Repeating the operation is safe and cannot affect another user's state.
	if _, err := m.MarkAddressRead(context.Background(), reader, addressA, permissions, nil); err != nil {
		t.Fatal(err)
	}
	assertCounts(reader, 0, 1)
	var unchangedStatuses int
	db.Get(&unchangedStatuses, `SELECT count(DISTINCT status_id) FROM conversations WHERE address_id=$1`, addressA)
	if unchangedStatuses != 3 {
		t.Fatal("mark read changed conversation statuses")
	}
	// A newly arriving message must remain unread after the snapshot is marked.
	db.MustExec(`INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,created_at)
		VALUES($1,$2,'contact','incoming','received',$3::timestamptz+interval '1 second')`, conversationA, sender, markedAt)
	assertCounts(reader, 1, 1)
	// Revocation is enforced in the mutation, not merely by the sidebar.
	db.MustExec(`DELETE FROM email_address_users WHERE address_id=$1 AND user_id=$2`, addressA, reader)
	if _, err := m.MarkAddressRead(context.Background(), reader, addressA, permissions, nil); err == nil {
		t.Fatal("revoked address was marked read")
	}
	if _, err := m.MarkAddressRead(context.Background(), reader, addressB, []string{"conversations:read"}, nil); err == nil {
		t.Fatal("missing conversation scope was accepted")
	}
	if _, err := m.MarkAddressRead(context.Background(), reader, addressB, []string{"conversations:read_all"}, nil); err == nil {
		t.Fatal("missing read permission was accepted")
	}
}

// Hold a real insertion transaction open after the message is written, as a
// slow attachment link would. The reader must wait for that writer to commit.
type readOrderingMediaStore struct {
	receiptMediaStore
	inserted chan struct{}
	release  chan struct{}
}

func (s readOrderingMediaStore) LinkMessageMediaTx(*sqlx.Tx, int, []mmodels.Media, []string, int) error {
	close(s.inserted)
	<-s.release
	return nil
}

func TestMarkAddressReadOrdersConcurrentMessageCommits(t *testing.T) {
	m, db, inboxID, address, _ := mailTestManager(t)
	first, err := m.ProcessIncomingMessage(testIncoming(inboxID, "a@example.test", "first@read.test", ""))
	if err != nil {
		t.Fatal(err)
	}
	var reader int
	if err := db.Get(&reader, `INSERT INTO users(type,email,first_name) VALUES('agent','reader@read.test','Reader') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO email_address_users(address_id,user_id) VALUES($1,$2)`, address, reader)
	permissions := []string{"conversations:read", "conversations:read_all"}
	media := readOrderingMediaStore{inserted: make(chan struct{}), release: make(chan struct{})}
	m.mediaStore = media
	done := make(chan error, 1)
	message := models.Message{Type: models.MessageIncoming, Status: models.MessageStatusReceived, ConversationID: first.ConversationID, SenderID: first.SenderID, SenderType: models.SenderTypeContact, Content: "In flight"}
	go func() { done <- m.InsertMessage(&message) }()
	select {
	case <-media.inserted:
	case <-time.After(5 * time.Second):
		t.Fatal("writer never reached attachment linkage")
	}
	var once sync.Once
	release := func() { once.Do(func() { close(media.release) }) }
	defer release()
	readDone := make(chan error, 1)
	go func() {
		_, err := m.MarkAddressRead(context.Background(), reader, address, permissions, nil)
		readDone <- err
	}()
	// Observe PostgreSQL's lock wait, rather than depending on goroutine timing.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if err := db.Get(&waiting, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE '%ORDER BY conversations.id FOR NO KEY UPDATE%')`); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case err := <-readDone:
			t.Fatalf("mark-read bypassed uncommitted writer: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("mark-read did not wait on the in-flight writer")
		}
		time.Sleep(time.Millisecond)
	}
	release()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	counts, err := m.GetSidebarCounts(reader, permissions, nil)
	if err != nil || counts.Addresses[address] != 0 {
		t.Fatalf("committed snapshot was not marked read: %+v %v", counts, err)
	}
	m.mediaStore = receiptMediaStore{}
	// Start the next writer before the read watermark is saved. Its timestamp
	// must be assigned after the serialization lock, not at transaction start.
	readTx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	if _, err := readTx.Exec(`SELECT id FROM conversations WHERE id=$1 FOR NO KEY UPDATE`, first.ConversationID); err != nil {
		t.Fatal(err)
	}
	message.ID = 0
	message.UUID = ""
	message.Content = "Arrived after read"
	go func() { done <- m.InsertMessage(&message) }()
	deadline = time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if err := db.Get(&waiting, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE 'SELECT id,uuid FROM conversations%')`); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("writer did not wait for the read transaction")
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := readTx.Exec(`UPDATE conversation_last_seen SET last_seen_at=clock_timestamp() WHERE user_id=$1 AND conversation_id=$2`, reader, first.ConversationID); err != nil {
		t.Fatal(err)
	}
	if err := readTx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	counts, err = m.GetSidebarCounts(reader, permissions, nil)
	if err != nil || counts.Addresses[address] != 1 {
		t.Fatalf("later message was hidden: %+v %v", counts, err)
	}
}
