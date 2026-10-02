package conversation

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jakedolan443/fernmail/internal/conversation/models"
	"github.com/jakedolan443/fernmail/internal/inbox"
	wmodels "github.com/jakedolan443/fernmail/internal/webhook/models"
)

// IMAPState returns a durable high-water mark and individually failed UIDs.
// The mailbox key includes host, account and folder; UIDVALIDITY isolates resets.
func (m *Manager) IMAPState(inboxID int, key string, validity uint32) (uint32, []uint32, error) {
	var cursor uint32
	if err := m.db.Get(&cursor, `SELECT COALESCE((SELECT last_uid FROM mail_sync_cursors WHERE inbox_id=$1 AND mailbox_key=$2 AND uid_validity=$3),0)`, inboxID, key, validity); err != nil {
		return 0, nil, err
	}
	failures := []uint32{}
	err := m.db.Select(&failures, `SELECT uid FROM mail_sync_failures WHERE inbox_id=$1 AND mailbox_key=$2 AND uid_validity=$3 ORDER BY updated_at,uid LIMIT 25`, inboxID, key, validity)
	return cursor, failures, err
}

func (m *Manager) RecordIMAPResult(inboxID int, key string, validity, uid uint32, failure error) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if failure != nil {
		_, err = tx.Exec(`INSERT INTO mail_sync_failures(inbox_id,mailbox_key,uid_validity,uid,error) VALUES($1,$2,$3,$4,$5) ON CONFLICT(inbox_id,mailbox_key,uid_validity,uid) DO UPDATE SET error=EXCLUDED.error,updated_at=now()`, inboxID, key, validity, uid, failure.Error())
	} else {
		_, err = tx.Exec(`DELETE FROM mail_sync_failures WHERE inbox_id=$1 AND mailbox_key=$2 AND uid_validity=$3 AND uid=$4`, inboxID, key, validity, uid)
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO mail_sync_cursors(inbox_id,mailbox_key,uid_validity,last_uid) VALUES($1,$2,$3,$4) ON CONFLICT(inbox_id,mailbox_key,uid_validity) DO UPDATE SET last_uid=GREATEST(mail_sync_cursors.last_uid,EXCLUDED.last_uid)`, inboxID, key, validity, uid)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (m *Manager) persistIncoming(message models.IncomingMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	identity := fmt.Sprintf("%d:%s:%d:%d", message.InboxID, message.MailboxKey, message.UIDValidity, message.UID)
	if message.UID == 0 {
		identity = fmt.Sprintf("%d:%s:%s", message.InboxID, message.EmailAlias, message.SourceID.String)
		if message.SourceID.String == "" {
			identity = uuid.NewString()
		}
	}
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(identity)))
	// Bound durable staging independently of IMAP volume. This short SQL-only
	// transaction serializes admission without holding a connection during I/O.
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended('mail-queue-admission',0))`); err != nil {
		return err
	}
	var exists bool
	if err = tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM incoming_mail_queue WHERE delivery_key=$1)`, key); err != nil {
		return err
	}
	if exists {
		return nil
	}
	limit := cap(m.incomingMessageQueue)
	if limit <= 0 {
		limit = 1000
	}
	var count int
	if err = tx.Get(&count, `SELECT count(*) FROM incoming_mail_queue WHERE completed_at IS NULL`); err != nil {
		return err
	}
	if count >= limit {
		return inbox.ErrIncomingQueueFull
	}
	if _, err = tx.Exec(`INSERT INTO incoming_mail_queue(inbox_id,delivery_key,payload) VALUES($1,$2,$3) ON CONFLICT(delivery_key) DO NOTHING`, message.InboxID, key, payload); err != nil {
		return err
	}
	return tx.Commit()
}

type incomingClaim struct {
	ID      int64  `db:"id"`
	Payload []byte `db:"payload"`
	Token   string `db:"claim_token"`
}

func (m *Manager) claimIncoming() (incomingClaim, error) {
	var row incomingClaim
	err := m.db.Get(&row, `WITH candidate AS (
 SELECT q.id FROM incoming_mail_queue q WHERE completed_at IS NULL AND next_attempt_at<=now() AND (lease_until IS NULL OR lease_until<now())
 AND NOT EXISTS (SELECT 1 FROM incoming_mail_queue older WHERE older.inbox_id=q.inbox_id AND older.id<q.id AND older.completed_at IS NULL AND (older.lease_until>now() OR older.next_attempt_at<=now()))
 ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED)
 UPDATE incoming_mail_queue q SET lease_until=now()+interval '10 minutes', claim_token=gen_random_uuid(),attempts=attempts+1
 FROM candidate WHERE q.id=candidate.id RETURNING q.id,q.payload,q.claim_token::text`)
	return row, err
}
func (m *Manager) processDurableIncoming(ctx context.Context) bool {
	row, err := m.claimIncoming()
	if err == sql.ErrNoRows {
		return false
	}
	if err != nil {
		m.lo.Error("claiming incoming mail", "error", err)
		return false
	}
	leaseCtx, cancelLease := context.WithCancel(ctx)
	defer cancelLease()
	go m.heartbeatIncoming(leaseCtx, row.ID, row.Token)
	var incoming models.IncomingMessage
	err = json.Unmarshal(row.Payload, &incoming)
	if err == nil {
		_, err = m.processIncomingMessage(ctx, incoming)
	}
	if err != nil {
		_, saveErr := m.db.Exec(`UPDATE incoming_mail_queue SET last_error=$3,lease_until=NULL,next_attempt_at=now()+LEAST(attempts*interval '30 seconds',interval '30 minutes') WHERE id=$1 AND claim_token=$2`, row.ID, row.Token, err.Error())
		m.lo.Error("incoming mail retained for retry", "error", err, "save_error", saveErr)
	} else {
		_, err = m.db.Exec(`UPDATE incoming_mail_queue SET completed_at=now(),payload=NULL,lease_until=NULL,last_error=NULL WHERE id=$1 AND claim_token=$2`, row.ID, row.Token)
		if err != nil {
			m.lo.Error("completing incoming mail", "error", err)
		}
	}
	return true
}

func (m *Manager) claimDelivery(messageID int) (string, error) {
	token := uuid.NewString()
	tx, err := m.db.Beginx()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var id int
	err = tx.Get(&id, `SELECT id FROM conversation_messages WHERE id=$1 AND status='pending' AND NOT EXISTS(SELECT 1 FROM mail_delivery_attempts WHERE message_id=$1 AND state IN ('sending','unknown','sent')) FOR UPDATE SKIP LOCKED`, messageID)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(`INSERT INTO mail_delivery_attempts(message_id,token,state,lease_until) VALUES($1,$2,'sending',now()+interval '10 minutes')`, id, token)
	if err != nil {
		return "", err
	}
	return token, tx.Commit()
}

func (m *Manager) finishDelivery(message models.Message, token, state string, deliveryErr error) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	reason := ""
	if deliveryErr != nil {
		reason = deliveryErr.Error()
	}
	result, err := tx.Exec(`UPDATE mail_delivery_attempts SET state=$2,finished_at=now(),error=NULLIF($3,'') WHERE token=$1 AND state='sending'`, token, state, reason)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return fmt.Errorf("delivery claim no longer active")
	}
	status := models.MessageStatusFailed
	if state == "sent" {
		status = models.MessageStatusSent
	}
	_, err = tx.Exec(`UPDATE conversation_messages SET status=$2,updated_at=now(),meta=COALESCE(meta,'{}') || jsonb_build_object('delivery_uncertain',$3::boolean,'delivery_error',$4::text) WHERE id=$1`, message.ID, status, state == "unknown", reason)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	m.BroadcastMessageUpdate(message.ConversationUUID, message.UUID, map[string]any{"status": status, "meta": m.deliveryMeta(message.UUID)})
	if updated, err := m.GetMessage(message.UUID); err == nil {
		m.webhookStore.TriggerEvent(wmodels.EventMessageUpdated, updated)
	}
	return nil
}
func (m *Manager) deliveryMeta(uuid string) json.RawMessage {
	var meta json.RawMessage
	_ = m.db.Get(&meta, `SELECT meta FROM conversation_messages WHERE uuid=$1`, uuid)
	return meta
}
func (m *Manager) recoverExpiredDeliveries() error {
	// Completed staging keeps only a delivery key; prune old tombstones in bounded
	// batches. The mailbox cursor and message-source constraint remain durable.
	if _, err := m.db.Exec(`DELETE FROM incoming_mail_queue WHERE id IN (SELECT id FROM incoming_mail_queue WHERE completed_at<now()-interval '30 days' ORDER BY completed_at,id LIMIT 1000)`); err != nil {
		return err
	}

	var expired []struct {
		UUID             string `db:"uuid"`
		ConversationUUID string `db:"conversation_uuid"`
	}
	err := m.db.Select(&expired, `WITH expired AS (UPDATE mail_delivery_attempts SET state='unknown',finished_at=now(),error='Delivery interrupted; verify with recipient before retrying' WHERE state='sending' AND lease_until<now() RETURNING message_id,error)
 UPDATE conversation_messages m SET status='failed',updated_at=now(),meta=COALESCE(meta,'{}') || jsonb_build_object('delivery_uncertain',true,'delivery_error',e.error) FROM expired e WHERE m.id=e.message_id RETURNING m.uuid,(SELECT uuid FROM conversations WHERE id=m.conversation_id) AS conversation_uuid`)
	if err != nil {
		return err
	}
	for _, msg := range expired {
		m.BroadcastMessageUpdate(msg.ConversationUUID, msg.UUID, map[string]any{"status": models.MessageStatusFailed, "meta": m.deliveryMeta(msg.UUID)})
	}
	return nil
}

func (m *Manager) heartbeatDelivery(ctx context.Context, token string) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := m.db.Exec(`UPDATE mail_delivery_attempts SET lease_until=now()+interval '10 minutes' WHERE token=$1 AND state='sending'`, token)
			if err != nil {
				m.lo.Error("renewing delivery lease", "error", err)
			}
		}
	}
}

// Recheck at dispatch: a reply may have been queued before an address was
// retired or its author lost access. Durable retry must not bypass that change.
func (m *Manager) ensureDeliveryAllowed(messageID int) error {
	var allowed bool
	err := m.db.Get(&allowed, `SELECT EXISTS(SELECT 1 FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id JOIN inboxes i ON i.id=c.inbox_id JOIN email_addresses a ON a.id=c.address_id WHERE m.id=$1 AND i.enabled AND i.deleted_at IS NULL AND a.enabled AND can_access_email_address(a.id,m.sender_id))`, messageID)
	if err != nil {
		return err
	}
	if !allowed {
		return fmt.Errorf("reply address is disabled or sender no longer has access")
	}
	return nil
}

func (m *Manager) heartbeatIncoming(ctx context.Context, id int64, token string) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := m.db.Exec(`UPDATE incoming_mail_queue SET lease_until=now()+interval '10 minutes' WHERE id=$1 AND claim_token=$2 AND completed_at IS NULL`, id, token); err != nil {
				m.lo.Error("renewing incoming lease", "error", err)
			}
		}
	}
}

// Reconciliation is independent of the low-latency dispatch poll. A database
// error waits for the next maintenance tick; it never creates a hot retry loop
// or prevents otherwise healthy pending messages from being dispatched.
func (m *Manager) runMailMaintenance(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if err := m.recoverExpiredDeliveries(); err != nil {
			m.lo.Error("reconciling mail delivery", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
