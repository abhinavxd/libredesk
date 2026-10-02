package conversation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/testutil"
	"github.com/jakedolan443/fernmail/internal/user"
	"github.com/jakedolan443/fernmail/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func mailTestManager(t *testing.T) (*Manager, *sqlx.DB, int, int, int) {
	t.Helper()
	db := testutil.NewDB(t, "mail_reliability")
	lo := logf.New(logf.Opts{})
	i18n := testutil.NewI18n(t)
	users, err := user.New(i18n, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := New(ws.NewHub(&lo, nil), i18n, nil, nil, users, receiptMediaStore{}, stubSettingsStore{}, nil, receiptWebhookStore{}, Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	var inboxID, a, b int
	if err = db.Get(&inboxID, `INSERT INTO inboxes(name,channel,"from") VALUES('Mail','email','a@example.test') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err = db.Get(&a, `INSERT INTO email_addresses(inbox_id,address,kind) VALUES($1,'a@example.test','mailbox') RETURNING id`, inboxID); err != nil {
		t.Fatal(err)
	}
	if err = db.Get(&b, `INSERT INTO email_addresses(inbox_id,address,kind) VALUES($1,'b@example.test','alias') RETURNING id`, inboxID); err != nil {
		t.Fatal(err)
	}
	return manager, db, inboxID, a, b
}
func testIncoming(inboxID int, address, source, parent string) models.IncomingMessage {
	return models.IncomingMessage{InboxID: inboxID, Channel: "email", EmailAlias: address, Contact: models.IncomingContact{Email: null.StringFrom("sender@example.test"), FirstName: "Sender"}, Subject: "Mail", Content: "Body", ContentType: models.ContentTypeText, SourceID: null.StringFrom(source), InReplyTo: parent}
}
func TestIncomingThreadingAndDedupRespectAddress(t *testing.T) {
	m, db, inboxID, a, b := mailTestManager(t)
	first, err := m.ProcessIncomingMessage(testIncoming(inboxID, "a@example.test", "original@test", ""))
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"references", "plus"} {
		incoming := testIncoming(inboxID, "b@example.test", route+"@test", "original@test")
		if route == "plus" {
			incoming.InReplyTo = ""
			incoming.ConversationUUIDFromReplyTo = first.ConversationUUID
		}
		msg, err := m.ProcessIncomingMessage(incoming)
		if err != nil {
			t.Fatal(err)
		}
		var actual int
		if err = db.Get(&actual, `SELECT address_id FROM conversations WHERE id=$1`, msg.ConversationID); err != nil {
			t.Fatal(err)
		}
		if actual != b || msg.ConversationID == first.ConversationID {
			t.Fatalf("%s crossed addresses: %d != %d", route, actual, b)
		}
	}
	// Same RFC Message-ID delivered to a different authorized address survives.
	second, err := m.ProcessIncomingMessage(testIncoming(inboxID, "b@example.test", "original@test", ""))
	if err != nil || second.ID == 0 {
		t.Fatalf("second delivery discarded: %+v %v", second, err)
	}
	reply, err := m.ProcessIncomingMessage(testIncoming(inboxID, "a@example.test", "reply@test", "original@test"))
	if err != nil || reply.ConversationID != first.ConversationID {
		t.Fatalf("same-address threading failed: %+v %v", reply, err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM conversations WHERE address_id=$1`, a)
	if count != 1 {
		t.Fatalf("a conversations=%d", count)
	}
}
func TestConcurrentIncomingCreatesOneMessageAndConversation(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	db.SetMaxOpenConns(2)
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := m.ProcessIncomingMessage(testIncoming(inboxID, "a@example.test", "same@test", ""))
			errs <- err
		}()
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent ingestion deadlocked with two DB connections")
	}
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE source_id='same@test'`)
	if count != 1 {
		t.Fatalf("messages=%d", count)
	}
	db.Get(&count, `SELECT count(*) FROM conversations`)
	if count != 1 {
		t.Fatalf("conversations=%d", count)
	}
	// Database-level guard also rejects callers bypassing the normal manager.
	_, err := db.Exec(`INSERT INTO conversation_messages(type,status,conversation_id,sender_id,sender_type,source_id) SELECT type,status,conversation_id,sender_id,sender_type,source_id FROM conversation_messages WHERE source_id='same@test'`)
	if err == nil {
		t.Fatal("database accepted duplicate address/source identity")
	}
}
func TestDurableIncomingSurvivesRestartAndRetries(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	incoming := testIncoming(inboxID, "missing@example.test", "queued@test", "")
	incoming.UID = 4
	incoming.UIDValidity = 2
	incoming.MailboxKey = "account/INBOX"
	// Queue payload persists even with no memory worker or queue capacity.
	if err := m.EnqueueIncoming(incoming); err != nil {
		t.Fatal(err)
	}
	if err := m.EnqueueIncoming(incoming); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`DELETE FROM email_addresses`)
	if !m.processDurableIncoming(context.Background()) {
		t.Fatal("nothing claimed")
	}
	var failure string
	var attempts int
	if err := db.QueryRow(`SELECT last_error,attempts FROM incoming_mail_queue`).Scan(&failure, &attempts); err != nil || failure == "" || attempts != 1 {
		t.Fatalf("retry not retained: %q %d %v", failure, attempts, err)
	}
	db.MustExec(`INSERT INTO email_addresses(inbox_id,address,kind) VALUES($1,'a@example.test','mailbox')`, inboxID)
	db.MustExec(`UPDATE incoming_mail_queue SET next_attempt_at=now()`)
	// Fresh Manager using the same DB represents a process restart.
	restarted, err := New(m.wsHub, m.i18n, m.statusStore, m.inboxStore, m.userStore, m.mediaStore, m.settingsStore, m.template, m.webhookStore, Opts{DB: db, Lo: m.lo})
	if err != nil {
		t.Fatal(err)
	}
	if !restarted.processDurableIncoming(context.Background()) {
		t.Fatal("retry missing after restart")
	}
	var complete bool
	var payload sql.NullString
	if err := db.QueryRow(`SELECT completed_at IS NOT NULL,payload::text FROM incoming_mail_queue`).Scan(&complete, &payload); err != nil || !complete || payload.Valid {
		t.Fatalf("durable completion/payload cleanup: %v %v %v", complete, payload, err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE source_id='queued@test'`)
	if count != 1 {
		t.Fatalf("delivered count=%d", count)
	}
}
func TestIMAPCheckpointKeepsFailuresAndSeparatesUIDValidity(t *testing.T) {
	m, _, inboxID, _, _ := mailTestManager(t)
	if err := m.RecordIMAPResult(inboxID, "server/user/INBOX", 7, 3, errors.New("temporary parse failure")); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordIMAPResult(inboxID, "server/user/INBOX", 7, 8, nil); err != nil {
		t.Fatal(err)
	}
	cursor, failed, err := m.IMAPState(inboxID, "server/user/INBOX", 7)
	if err != nil || cursor != 8 || len(failed) != 1 || failed[0] != 3 {
		t.Fatalf("state=%d %v %v", cursor, failed, err)
	}
	cursor, failed, err = m.IMAPState(inboxID, "server/user/INBOX", 8)
	if err != nil || cursor != 0 || len(failed) != 0 {
		t.Fatalf("reset=%d %v %v", cursor, failed, err)
	}
	if err = m.RecordIMAPResult(inboxID, "server/user/INBOX", 7, 3, nil); err != nil {
		t.Fatal(err)
	}
	cursor, failed, err = m.IMAPState(inboxID, "server/user/INBOX", 7)
	if err != nil || cursor != 8 || len(failed) != 0 {
		t.Fatalf("recovery=%d %v %v", cursor, failed, err)
	}
}
func pendingTestMessage(t *testing.T, m *Manager, inboxID int) models.Message {
	t.Helper()
	incoming, err := m.ProcessIncomingMessage(testIncoming(inboxID, "a@example.test", "parent@test", ""))
	if err != nil {
		t.Fatal(err)
	}
	var msg models.Message
	if err = m.db.Get(&msg, `INSERT INTO conversation_messages(type,status,conversation_id,sender_id,sender_type,source_id,reply_to_source_id,content,text_content,content_type,meta) VALUES('outgoing','pending',$1,$2,'agent','out@test','parent@test','Reply','Reply','text','{"to":["sender@example.test"]}') RETURNING id,uuid`, incoming.ConversationID, incoming.SenderID); err != nil {
		t.Fatal(err)
	}
	msg.ConversationUUID = incoming.ConversationUUID
	msg.ConversationID = incoming.ConversationID
	return msg
}
func TestDeliveryClaimsAreExclusiveAndAmbiguityRequiresRetry(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	msg := pendingTestMessage(t, m, inboxID)
	var wg sync.WaitGroup
	tokens := make(chan string, 10)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, err := m.claimDelivery(msg.ID)
			if err == nil {
				tokens <- token
			}
		}()
	}
	wg.Wait()
	close(tokens)
	if len(tokens) != 1 {
		t.Fatalf("claims=%d", len(tokens))
	}
	token := <-tokens
	if err := m.finishDelivery(msg, token, "unknown", errors.New("connection lost awaiting SMTP response")); err != nil {
		t.Fatal(err)
	}
	if _, err := m.claimDelivery(msg.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unknown was automatically retried: %v", err)
	}
	var uncertain bool
	db.Get(&uncertain, `SELECT (meta->>'delivery_uncertain')::boolean FROM conversation_messages WHERE id=$1`, msg.ID)
	if !uncertain {
		t.Fatal("uncertain delivery not visible")
	}
	if err := m.MarkMessageAsPending(msg.UUID); err != nil {
		t.Fatal(err)
	}
	token, err := m.claimDelivery(msg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.finishDelivery(msg, token, "sent", nil); err != nil {
		t.Fatal(err)
	}
	if _, err = m.claimDelivery(msg.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("sent retried: %v", err)
	}
}
func TestExpiredDeliveryLeaseNeverBlindlyResends(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	msg := pendingTestMessage(t, m, inboxID)
	token, err := m.claimDelivery(msg.ID)
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`UPDATE mail_delivery_attempts SET lease_until=now()-interval '1 second' WHERE token=$1`, token)
	if err = m.recoverExpiredDeliveries(); err != nil {
		t.Fatal(err)
	}
	if _, err = m.claimDelivery(msg.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expired SMTP attempt retried: %v", err)
	}
	var status string
	db.Get(&status, `SELECT status FROM conversation_messages WHERE id=$1`, msg.ID)
	if status != "failed" {
		t.Fatalf("status=%s", status)
	}
}
func TestThreadParentExcludesConcurrentAndFailedReplies(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	msg := pendingTestMessage(t, m, inboxID)
	for i, status := range []string{"pending", "failed", "sent"} {
		db.MustExec(`INSERT INTO conversation_messages(type,status,conversation_id,sender_id,sender_type,source_id) SELECT 'outgoing',$2,conversation_id,sender_id,'agent',$3 FROM conversation_messages WHERE id=$1`, msg.ID, status, fmt.Sprintf("later%d@test", i))
	}
	refs, parent := m.BuildEmailThreadingHeaders(msg.ConversationID, "out@test")
	if parent != "parent@test" || len(refs) != 1 || refs[0] != parent {
		t.Fatalf("headers=%v %s", refs, parent)
	}
}

func TestIncomingQueueAdmissionIsBoundedAndAtomic(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	m.incomingMessageQueue = make(chan models.IncomingMessage, 1)
	var wg sync.WaitGroup
	success := make(chan bool, 8)
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := m.EnqueueIncoming(testIncoming(inboxID, "a@example.test", fmt.Sprintf("queued-%d@test", i), ""))
			success <- err == nil
		}(i)
	}
	wg.Wait()
	close(success)
	var accepted int
	for ok := range success {
		if ok {
			accepted++
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted %d messages for one staging slot", accepted)
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM incoming_mail_queue WHERE completed_at IS NULL`); err != nil || count != 1 {
		t.Fatalf("queue=%d %v", count, err)
	}
	if !m.processDurableIncoming(context.Background()) {
		t.Fatal("could not consume staged message")
	}
	if err := m.EnqueueIncoming(testIncoming(inboxID, "a@example.test", "after-free@test", "")); err != nil {
		t.Fatalf("completed payload did not free capacity: %v", err)
	}
}

func TestSentStatusPersistenceFailureDoesNotReclaimSMTPAttempt(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	msg := pendingTestMessage(t, m, inboxID)
	token, err := m.claimDelivery(msg.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a database failure precisely after SMTP accepted the message.
	db.MustExec(`CREATE FUNCTION fail_delivery_completion() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF NEW.status='sent' THEN RAISE EXCEPTION 'simulated completion failure'; END IF; RETURN NEW; END$$`)
	db.MustExec(`CREATE TRIGGER fail_delivery_completion BEFORE UPDATE ON conversation_messages FOR EACH ROW EXECUTE FUNCTION fail_delivery_completion()`)
	if err = m.finishDelivery(msg, token, "sent", nil); err == nil {
		t.Fatal("completion failure was not surfaced")
	}
	if _, err = m.claimDelivery(msg.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("accepted SMTP message reclaimed after failed DB write: %v", err)
	}
	db.MustExec(`DROP TRIGGER fail_delivery_completion ON conversation_messages`)
	db.MustExec(`UPDATE mail_delivery_attempts SET lease_until=now()-interval '1 second' WHERE token=$1`, token)
	if err = m.recoverExpiredDeliveries(); err != nil {
		t.Fatal(err)
	}
	var uncertain bool
	if err = db.Get(&uncertain, `SELECT (meta->>'delivery_uncertain')::boolean FROM conversation_messages WHERE id=$1`, msg.ID); err != nil || !uncertain {
		t.Fatalf("interrupted completion must require operator reconciliation: %v %v", uncertain, err)
	}
}

func TestCloseCancelsBlockedOutgoingProducer(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	msg := pendingTestMessage(t, m, inboxID)
	m.outgoingMessageQueue = make(chan models.Message)
	defer m.Close()
	go m.Run(context.Background(), 0, 0, 10*time.Millisecond)
	deadline := time.After(3 * time.Second)
	for {
		if _, ok := m.outgoingProcessingMessages.Load(msg.ID); ok {
			break
		}
		select {
		case <-deadline:
			t.Fatal("scanner never queued pending message")
		case <-time.After(time.Millisecond):
		}
	}
	stopped := make(chan struct{})
	go func() { m.Close(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(3 * time.Second):
		t.Fatal("Close deadlocked on blocked producer")
	}
	var status string
	if err := db.Get(&status, `SELECT status FROM conversation_messages WHERE id=$1`, msg.ID); err != nil || status != "pending" {
		t.Fatalf("shutdown lost durable pending message: %s %v", status, err)
	}
	// Repeated Close and Run-after-Close are safe.
	m.Close()
	m.Run(context.Background(), 1, 1, time.Millisecond)
}

func TestRetiredRestrictedAddressKeepsIncomingAndBlocksDispatch(t *testing.T) {
	m, db, inboxID, _, b := mailTestManager(t)
	db.MustExec(`UPDATE email_addresses SET enabled=false WHERE id=$1`, b)
	received, err := m.ProcessIncomingMessage(testIncoming(inboxID, "b@example.test", "retired@test", ""))
	if err != nil {
		t.Fatal(err)
	}
	var addressID int
	if err = db.Get(&addressID, `SELECT address_id FROM conversations WHERE id=$1`, received.ConversationID); err != nil || addressID != b {
		t.Fatalf("retired mail changed ACL boundary: %d %v", addressID, err)
	}
	if err = m.ensureDeliveryAllowed(received.ID); err == nil {
		t.Fatal("retired address permitted dispatch")
	}
}

func TestReplyWaitsForFailedStagedParentRatherThanSplittingThread(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	parent := testIncoming(inboxID, "a@example.test", "parent-retry@test", "")
	reply := testIncoming(inboxID, "a@example.test", "dependent-reply@test", "parent-retry@test")
	if err := m.EnqueueIncoming(parent); err != nil {
		t.Fatal(err)
	}
	if err := m.EnqueueIncoming(reply); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`CREATE FUNCTION fail_parent_insert() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF NEW.source_id='parent-retry@test' THEN RAISE EXCEPTION 'temporary ingestion failure'; END IF; RETURN NEW; END$$`)
	db.MustExec(`CREATE TRIGGER fail_parent_insert BEFORE INSERT ON conversation_messages FOR EACH ROW EXECUTE FUNCTION fail_parent_insert()`)
	if !m.processDurableIncoming(context.Background()) {
		t.Fatal("parent not claimed")
	}
	if !m.processDurableIncoming(context.Background()) {
		t.Fatal("reply not claimed")
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM conversations`); err != nil || count != 0 {
		t.Fatalf("dependent reply created its own thread: %d %v", count, err)
	}
	db.MustExec(`DROP TRIGGER fail_parent_insert ON conversation_messages`)
	db.MustExec(`UPDATE incoming_mail_queue SET next_attempt_at=now()`)
	if !m.processDurableIncoming(context.Background()) || !m.processDurableIncoming(context.Background()) {
		t.Fatal("parent/reply did not recover")
	}
	if err := db.Get(&count, `SELECT count(DISTINCT conversation_id) FROM conversation_messages WHERE source_id IN ('parent-retry@test','dependent-reply@test')`); err != nil || count != 1 {
		t.Fatalf("recovered mail split into %d threads: %v", count, err)
	}
	if err := db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE source_id IN ('parent-retry@test','dependent-reply@test')`); err != nil || count != 2 {
		t.Fatalf("recovered mail count %d: %v", count, err)
	}
}

func TestMaintenanceFailureIsThrottledWithoutBlockingFastDispatch(t *testing.T) {
	m, db, inboxID, _, _ := mailTestManager(t)
	first := pendingTestMessage(t, m, inboxID)
	// Sequence increments survive the injected DELETE failure, letting us count
	// real maintenance attempts without relying on logs or source inspection.
	db.MustExec(`CREATE SEQUENCE mail_maintenance_attempts`)
	db.MustExec(`CREATE FUNCTION fail_mail_maintenance() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN PERFORM nextval('mail_maintenance_attempts'); RAISE EXCEPTION 'temporary maintenance failure'; END$$`)
	db.MustExec(`CREATE TRIGGER fail_mail_maintenance BEFORE DELETE ON incoming_mail_queue FOR EACH STATEMENT EXECUTE FUNCTION fail_mail_maintenance()`)
	m.outgoingMessageQueue = make(chan models.Message, 2)
	defer m.Close()
	go m.Run(context.Background(), 0, 0, 5*time.Millisecond)
	deadline := time.After(3 * time.Second)
	for {
		var called bool
		if err := db.Get(&called, `SELECT is_called FROM mail_maintenance_attempts`); err != nil {
			t.Fatal(err)
		}
		if called {
			break
		}
		select {
		case <-deadline:
			t.Fatal("startup maintenance did not run")
		case <-time.After(time.Millisecond):
		}
	}
	receive := func() models.Message {
		t.Helper()
		select {
		case msg := <-m.outgoingMessageQueue:
			return msg
		case <-time.After(3 * time.Second):
			t.Fatal("maintenance failure blocked fast dispatch")
			return models.Message{}
		}
	}
	if got := receive(); got.ID != first.ID {
		t.Fatalf("first dispatch=%d want %d", got.ID, first.ID)
	}
	var secondID int
	if err := db.Get(&secondID, `INSERT INTO conversation_messages(type,status,conversation_id,sender_id,sender_type,source_id,content,text_content,content_type,meta) SELECT type,status,conversation_id,sender_id,sender_type,'second-dispatch@test',content,text_content,content_type,meta FROM conversation_messages WHERE id=$1 RETURNING id`, first.ID); err != nil {
		t.Fatal(err)
	}
	if got := receive(); got.ID != secondID {
		t.Fatalf("second dispatch=%d want %d", got.ID, secondID)
	}
	// Permit many 5ms dispatch polls. Reconciliation remains on its minute clock
	// even after failure and does not block the independently running producer.
	time.Sleep(100 * time.Millisecond)
	m.Close()
	var attempts int64
	var called bool
	if err := db.QueryRow(`SELECT last_value,is_called FROM mail_maintenance_attempts`).Scan(&attempts, &called); err != nil || !called || attempts != 1 {
		t.Fatalf("maintenance retry loop: calls=%d called=%v error=%v", attempts, called, err)
	}
}
