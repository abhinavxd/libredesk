-- First-class, user-facing email addresses. A mailbox owns a transport inbox;
-- aliases are lightweight endpoints sharing that transport.
CREATE TABLE IF NOT EXISTS email_addresses (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    address TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT 'alias' CHECK (kind IN ('mailbox', 'alias')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    restricted BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT constraint_email_addresses_address CHECK (length(address) <= 320),
    CONSTRAINT constraint_email_addresses_display_name CHECK (length(display_name) <= 140)
);
CREATE UNIQUE INDEX IF NOT EXISTS index_email_addresses_on_normalized_address
    ON email_addresses (lower(address));
CREATE UNIQUE INDEX IF NOT EXISTS index_email_addresses_one_mailbox_per_inbox
    ON email_addresses (inbox_id) WHERE kind = 'mailbox';
CREATE INDEX IF NOT EXISTS index_email_addresses_on_inbox_id ON email_addresses(inbox_id);

CREATE TABLE IF NOT EXISTS email_address_users (
    address_id INTEGER NOT NULL REFERENCES email_addresses(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (address_id, user_id)
);
CREATE TABLE IF NOT EXISTS email_address_teams (
    address_id INTEGER NOT NULL REFERENCES email_addresses(id) ON DELETE CASCADE,
    team_id INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    PRIMARY KEY (address_id, team_id)
);
CREATE INDEX IF NOT EXISTS index_email_address_users_on_user_id ON email_address_users(user_id);
CREATE INDEX IF NOT EXISTS index_email_address_teams_on_team_id ON email_address_teams(team_id);

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS address_id INTEGER REFERENCES email_addresses(id) ON DELETE SET NULL ON UPDATE CASCADE;
CREATE INDEX IF NOT EXISTS index_conversations_on_address_id ON conversations(address_id);

-- Seed one mailbox endpoint for each existing email transport. It is initially
-- unrestricted to preserve legacy access until a saved address view supplies a
-- more specific user/team policy below.
INSERT INTO email_addresses (inbox_id, address, kind, enabled, restricted)
SELECT i.id,
       lower(btrim(COALESCE(NULLIF(i."from", ''), NULLIF(i.config->>'from', '')))),
       'mailbox',
       i.enabled,
       FALSE
FROM inboxes i
WHERE i.channel = 'email'
  AND i.deleted_at IS NULL
  AND btrim(COALESCE(NULLIF(i."from", ''), NULLIF(i.config->>'from', ''))) <> ''
ON CONFLICT (lower(address)) DO NOTHING;

-- Lift the previous configuration-only aliases into canonical address rows.
INSERT INTO email_addresses (inbox_id, address, display_name, kind, enabled, restricted)
SELECT i.id,
       lower(btrim(alias.value->>'address')),
       COALESCE(NULLIF(btrim(alias.value->>'display_name'), ''), NULLIF(btrim(alias.value->>'name'), ''), ''),
       CASE
           WHEN lower(btrim(alias.value->>'address')) = lower(btrim(COALESCE(NULLIF(i."from", ''), NULLIF(i.config->>'from', ''))))
               THEN 'mailbox'
           ELSE 'alias'
       END,
       COALESCE((alias.value->>'enabled')::boolean, TRUE),
       FALSE
FROM inboxes i
CROSS JOIN LATERAL jsonb_array_elements(COALESCE(i.config->'email_aliases', '[]'::jsonb)) AS alias(value)
WHERE i.channel = 'email'
  AND i.deleted_at IS NULL
  AND btrim(alias.value->>'address') <> ''
ON CONFLICT (lower(address)) DO NOTHING;

-- The runtime IMAP receiver remains intentionally simple: it reads this derived
-- delivery list, while email_addresses is the authoritative product model.
UPDATE inboxes i
SET config = jsonb_set(
        COALESCE(i.config, '{}'::jsonb),
        '{email_aliases}',
        COALESCE((
            SELECT jsonb_agg(jsonb_build_object(
                'address', a.address,
                'name', a.display_name,
                'display_name', a.display_name,
                'enabled', a.enabled,
                'default', a.kind = 'mailbox'
            ) ORDER BY CASE WHEN a.kind = 'mailbox' THEN 0 ELSE 1 END, lower(a.address))
            FROM email_addresses a
            WHERE a.inbox_id = i.id
        ), '[]'::jsonb),
        TRUE
    ),
    updated_at = NOW()
WHERE i.channel = 'email' AND i.deleted_at IS NULL;

-- Preserve the recipient chosen by the old alias implementation while making
-- the relationship queryable and enforceable.
UPDATE conversations c
SET address_id = COALESCE(
    (
        SELECT a.id
        FROM email_addresses a
        WHERE a.inbox_id = c.inbox_id
          AND lower(a.address) = lower(NULLIF(c.meta->>'email_alias', ''))
        LIMIT 1
    ),
    (
        SELECT a.id
        FROM email_addresses a
        WHERE a.inbox_id = c.inbox_id AND a.kind = 'mailbox'
        LIMIT 1
    )
)
WHERE c.address_id IS NULL
  AND EXISTS (SELECT 1 FROM inboxes i WHERE i.id = c.inbox_id AND i.channel = 'email');

CREATE OR REPLACE FUNCTION can_access_email_address(target_address INTEGER, viewer BIGINT)
RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
    SELECT EXISTS (
        SELECT 1
        FROM email_addresses a
        JOIN inboxes i ON i.id = a.inbox_id
            AND i.channel = 'email'
            AND i.deleted_at IS NULL
        JOIN users u ON u.id = viewer AND u.type = 'agent' AND u.enabled AND u.deleted_at IS NULL
        WHERE a.id = target_address
          AND (
              NOT a.restricted
              OR u.email = 'System'
              OR EXISTS (
                  SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
                  WHERE ur.user_id = viewer AND r.name = 'Admin'
              )
              OR EXISTS (
                  SELECT 1 FROM email_address_users eau
                  WHERE eau.address_id = a.id AND eau.user_id = viewer
              )
              OR EXISTS (
                  SELECT 1
                  FROM email_address_teams eat
                  JOIN team_members tm ON tm.team_id = eat.team_id
                  WHERE eat.address_id = a.id AND tm.user_id = viewer
              )
          )
    );
$$;
