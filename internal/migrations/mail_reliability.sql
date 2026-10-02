-- Durable mailbox progress and retries; a new UIDVALIDITY gets an independent full scan.
CREATE TABLE IF NOT EXISTS mail_sync_cursors (
 inbox_id integer NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
 mailbox_key text NOT NULL, uid_validity bigint NOT NULL, last_uid bigint NOT NULL DEFAULT 0,
 PRIMARY KEY(inbox_id,mailbox_key,uid_validity)
);
CREATE TABLE IF NOT EXISTS mail_sync_failures (
 inbox_id integer NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
 mailbox_key text NOT NULL, uid_validity bigint NOT NULL, uid bigint NOT NULL,
 error text NOT NULL, updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(inbox_id,mailbox_key,uid_validity,uid)
);
CREATE TABLE IF NOT EXISTS incoming_mail_queue (
 id bigserial PRIMARY KEY, inbox_id integer NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
 delivery_key text NOT NULL UNIQUE, payload jsonb, attempts integer NOT NULL DEFAULT 0,
 next_attempt_at timestamptz NOT NULL DEFAULT now(), lease_until timestamptz,
 claim_token uuid, last_error text, completed_at timestamptz
);
CREATE INDEX IF NOT EXISTS incoming_mail_queue_pending ON incoming_mail_queue(next_attempt_at) WHERE completed_at IS NULL;
CREATE INDEX IF NOT EXISTS incoming_mail_queue_inbox_pending ON incoming_mail_queue(inbox_id,id) WHERE completed_at IS NULL;
CREATE INDEX IF NOT EXISTS incoming_mail_queue_completed ON incoming_mail_queue(completed_at,id) WHERE completed_at IS NOT NULL;
-- Protect address-local Message-ID identity without deleting historical duplicates.
CREATE TABLE IF NOT EXISTS received_mail_sources (
 address_id integer NOT NULL REFERENCES email_addresses(id) ON DELETE CASCADE,
 source_id text NOT NULL, message_id bigint NOT NULL REFERENCES conversation_messages(id) ON DELETE CASCADE,
 PRIMARY KEY(address_id,source_id)
);
INSERT INTO received_mail_sources(address_id,source_id,message_id)
 SELECT c.address_id,m.source_id,min(m.id) FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id
 WHERE c.address_id IS NOT NULL AND m.type='incoming' AND m.source_id IS NOT NULL AND m.source_id<>''
 GROUP BY c.address_id,m.source_id ON CONFLICT DO NOTHING;
CREATE OR REPLACE FUNCTION reserve_received_mail_source() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE endpoint integer;
BEGIN
 IF NEW.type='incoming' AND NEW.source_id IS NOT NULL AND NEW.source_id<>'' THEN
  SELECT address_id INTO endpoint FROM conversations WHERE id=NEW.conversation_id;
  IF endpoint IS NOT NULL THEN
   INSERT INTO received_mail_sources(address_id,source_id,message_id) VALUES(endpoint,NEW.source_id,NEW.id);
  END IF;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS reserve_received_mail_source ON conversation_messages;
CREATE TRIGGER reserve_received_mail_source AFTER INSERT ON conversation_messages FOR EACH ROW EXECUTE FUNCTION reserve_received_mail_source();
CREATE TABLE IF NOT EXISTS mail_delivery_attempts (
 id bigserial PRIMARY KEY, message_id bigint NOT NULL REFERENCES conversation_messages(id) ON DELETE CASCADE,
 token uuid NOT NULL UNIQUE, state text NOT NULL CHECK(state IN ('sending','sent','failed','unknown','retry_authorized')),
 created_at timestamptz NOT NULL DEFAULT now(), lease_until timestamptz NOT NULL,
 finished_at timestamptz, error text
);
CREATE UNIQUE INDEX IF NOT EXISTS mail_delivery_attempts_active ON mail_delivery_attempts(message_id) WHERE state IN ('sending','unknown');
ALTER TABLE conversation_messages ADD COLUMN IF NOT EXISTS reply_to_source_id text NOT NULL DEFAULT '';
