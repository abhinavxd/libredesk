-- Key Distribution: per-app pools of activation keys handed out from the
-- composer. A key leaves the redeemable pool only when an email carrying its
-- placeholder is queued, and it never returns.
CREATE TABLE IF NOT EXISTS activation_key_apps (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name TEXT NOT NULL,
    archived_at TIMESTAMPTZ,
    CONSTRAINT constraint_activation_key_apps_name CHECK (length(name) BETWEEN 1 AND 100)
);
CREATE UNIQUE INDEX IF NOT EXISTS index_unique_activation_key_apps_name ON activation_key_apps (lower(name));

CREATE TABLE IF NOT EXISTS activation_keys (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    app_id INTEGER NOT NULL REFERENCES activation_key_apps(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'redeemable' CHECK (status IN ('redeemable', 'activated', 'voided')),
    -- AES-GCM ciphertext. key_hash is an HMAC of the key, used to reject
    -- duplicates and to find a key without decrypting the pool.
    key_encrypted TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    added_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    activated_at TIMESTAMPTZ,
    message_id BIGINT REFERENCES conversation_messages(id) ON DELETE SET NULL,
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE SET NULL,
    placeholder_id TEXT,
    recipients TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    sent_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    voided_at TIMESTAMPTZ,
    voided_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT constraint_activation_keys_activated CHECK (status <> 'activated' OR activated_at IS NOT NULL)
);
CREATE UNIQUE INDEX IF NOT EXISTS index_unique_activation_keys_hash ON activation_keys (key_hash) WHERE status <> 'voided';
CREATE INDEX IF NOT EXISTS index_activation_keys_pool ON activation_keys (app_id, status, id);
CREATE INDEX IF NOT EXISTS index_activation_keys_message ON activation_keys (message_id) WHERE message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS index_activation_keys_conversation ON activation_keys (conversation_id) WHERE conversation_id IS NOT NULL;

-- Keys only move forward: redeemable to activated (by sending) or voided (by
-- an admin). Activated keys can't be edited, reset or deleted.
CREATE OR REPLACE FUNCTION guard_activation_key_status() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'activated' THEN
            RAISE EXCEPTION 'activated keys cannot be deleted';
        END IF;
        RETURN OLD;
    END IF;
    IF NEW.status IS DISTINCT FROM OLD.status AND OLD.status <> 'redeemable' THEN
        RAISE EXCEPTION 'activation key status % is final', OLD.status;
    END IF;
    IF OLD.status = 'activated' AND (NEW.key_encrypted IS DISTINCT FROM OLD.key_encrypted
        OR NEW.key_hash IS DISTINCT FROM OLD.key_hash OR NEW.app_id IS DISTINCT FROM OLD.app_id) THEN
        RAISE EXCEPTION 'activated keys cannot be changed';
    END IF;
    RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS guard_activation_key_status ON activation_keys;
CREATE TRIGGER guard_activation_key_status BEFORE UPDATE OR DELETE ON activation_keys
    FOR EACH ROW EXECUTE FUNCTION guard_activation_key_status();

-- The app each person last picked in the composer's key card.
CREATE TABLE IF NOT EXISTS activation_key_preferences (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    app_id INTEGER REFERENCES activation_key_apps(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO settings ("key", value)
VALUES ('key_distribution', '{"enabled":false,"max_keys_per_email":1,"low_stock_threshold":10}'::jsonb)
ON CONFLICT ("key") DO NOTHING;

UPDATE roles SET permissions = array_append(permissions, 'activation_keys:manage')
WHERE name = 'Admin' AND NOT ('activation_keys:manage' = ANY(permissions));
