package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0_RC11(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
ALTER TABLE users ADD COLUMN IF NOT EXISTS merged_into_id BIGINT REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS index_users_on_merged_into_id ON users(merged_into_id) WHERE merged_into_id IS NOT NULL;
ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'contact_merged';
UPDATE roles SET permissions = array_append(permissions, 'contacts:merge')
WHERE name = 'Admin' AND NOT ('contacts:merge' = ANY(permissions));

CREATE OR REPLACE FUNCTION resolve_merged_contact_id(contact_id BIGINT) RETURNS BIGINT AS $$
-- The lock keeps writes using an old contact ID from bypassing a concurrent merge.
DECLARE resolved BIGINT;
BEGIN
    SELECT COALESCE(merged_into_id, id) INTO resolved FROM users WHERE id = contact_id FOR KEY SHARE;
    RETURN COALESCE(resolved, contact_id);
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION redirect_merged_contact_reference() RETURNS TRIGGER AS $$
BEGIN
    IF TG_TABLE_NAME = 'conversation_messages' THEN
        NEW.sender_id := resolve_merged_contact_id(NEW.sender_id);
    ELSIF TG_TABLE_NAME IN ('conversation_participants', 'conversation_last_seen') THEN
        NEW.user_id := resolve_merged_contact_id(NEW.user_id);
    ELSIF TG_TABLE_NAME = 'media' THEN
        NEW.uploaded_by := resolve_merged_contact_id(NEW.uploaded_by);
    ELSE
        NEW.contact_id := resolve_merged_contact_id(NEW.contact_id);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER redirect_merged_contact BEFORE INSERT OR UPDATE OF contact_id ON conversations
FOR EACH ROW EXECUTE FUNCTION redirect_merged_contact_reference();
CREATE OR REPLACE TRIGGER redirect_merged_contact BEFORE INSERT OR UPDATE OF sender_id ON conversation_messages
FOR EACH ROW EXECUTE FUNCTION redirect_merged_contact_reference();
CREATE OR REPLACE TRIGGER redirect_merged_contact BEFORE INSERT OR UPDATE OF contact_id ON contact_notes
FOR EACH ROW EXECUTE FUNCTION redirect_merged_contact_reference();
CREATE OR REPLACE TRIGGER redirect_merged_contact BEFORE INSERT OR UPDATE OF contact_id ON contact_channel_identities
FOR EACH ROW EXECUTE FUNCTION redirect_merged_contact_reference();
CREATE OR REPLACE TRIGGER redirect_merged_contact BEFORE INSERT OR UPDATE OF user_id ON conversation_participants
FOR EACH ROW EXECUTE FUNCTION redirect_merged_contact_reference();
CREATE OR REPLACE TRIGGER redirect_merged_contact BEFORE INSERT OR UPDATE OF user_id ON conversation_last_seen
FOR EACH ROW EXECUTE FUNCTION redirect_merged_contact_reference();
CREATE OR REPLACE TRIGGER redirect_merged_contact BEFORE INSERT OR UPDATE OF uploaded_by ON media
FOR EACH ROW EXECUTE FUNCTION redirect_merged_contact_reference();
CREATE OR REPLACE TRIGGER redirect_merged_contact BEFORE INSERT OR UPDATE OF contact_id ON widget_campaign_deliveries
FOR EACH ROW EXECUTE FUNCTION redirect_merged_contact_reference();
`)
	return err
}
