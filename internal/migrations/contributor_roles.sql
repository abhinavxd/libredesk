-- Contributor role. Contributors read their assigned addresses; their replies
-- and new emails wait for an Admin or Agent to approve them.
INSERT INTO roles ("name", description, permissions)
VALUES (
    'Contributor',
    'Reads assigned addresses; replies and new emails are reviewed before sending.',
    '{conversations:read_all,conversations:read,conversations:create,messages:read,reviews:submit}'
)
ON CONFLICT ("name") DO NOTHING;

UPDATE roles SET permissions = array_append(permissions, 'conversations:create')
WHERE "name" IN ('Admin', 'Agent') AND NOT 'conversations:create' = ANY(permissions);
UPDATE roles SET permissions = array_append(permissions, 'reviews:manage')
WHERE "name" IN ('Admin', 'Agent') AND NOT 'reviews:manage' = ANY(permissions);

-- Open addresses are retired: Admins see every address, everyone else needs a
-- grant. Preserve today's access by granting each enabled non-admin agent the
-- addresses and transports that were open to them before the flag is removed.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema = current_schema() AND table_name = 'email_addresses' AND column_name = 'restricted') THEN
        INSERT INTO email_address_users (address_id, user_id)
        SELECT a.id, u.id
        FROM email_addresses a
        CROSS JOIN users u
        WHERE NOT a.restricted
          AND u.type = 'agent' AND u.enabled AND u.deleted_at IS NULL
          AND u.email IS DISTINCT FROM 'System'
          AND NOT EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
                          WHERE ur.user_id = u.id AND r.name = 'Admin')
        ON CONFLICT DO NOTHING;
        ALTER TABLE email_addresses DROP COLUMN restricted;
    END IF;

    INSERT INTO inbox_users (inbox_id, user_id)
    SELECT i.id, u.id
    FROM inboxes i
    LEFT JOIN inbox_access ia ON ia.inbox_id = i.id
    CROSS JOIN users u
    WHERE i.channel = 'email' AND i.deleted_at IS NULL
      AND NOT COALESCE(ia.restricted, FALSE)
      AND u.type = 'agent' AND u.enabled AND u.deleted_at IS NULL
      AND u.email IS DISTINCT FROM 'System'
      AND NOT EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
                      WHERE ur.user_id = u.id AND r.name = 'Admin')
    ON CONFLICT DO NOTHING;
END $$;

CREATE OR REPLACE FUNCTION can_access_inbox(target_inbox INTEGER, viewer BIGINT)
RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM inboxes i
        JOIN users u ON u.id = viewer AND u.type = 'agent' AND u.enabled AND u.deleted_at IS NULL
        WHERE i.id = target_inbox AND i.channel = 'email' AND i.deleted_at IS NULL
        AND (
            u.email = 'System'
            OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
                       WHERE ur.user_id = viewer AND r.name = 'Admin')
            OR EXISTS (SELECT 1 FROM inbox_users iu WHERE iu.inbox_id = i.id AND iu.user_id = viewer)
            OR EXISTS (SELECT 1 FROM inbox_roles ir JOIN user_roles ur ON ur.role_id = ir.role_id
                       WHERE ir.inbox_id = i.id AND ur.user_id = viewer)
        )
    );
$$;

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
              u.email = 'System'
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
