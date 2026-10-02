-- Existing inboxes remain available until an administrator restricts them.
CREATE TABLE IF NOT EXISTS inbox_access (
    inbox_id INTEGER PRIMARY KEY REFERENCES inboxes(id) ON DELETE CASCADE,
    restricted BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE TABLE IF NOT EXISTS inbox_users (
    inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (inbox_id, user_id)
);
CREATE TABLE IF NOT EXISTS inbox_roles (
    inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (inbox_id, role_id)
);

-- One policy for HTTP reads, lists, search, counts and live notifications.
CREATE OR REPLACE FUNCTION can_access_inbox(target_inbox INTEGER, viewer BIGINT)
RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
    SELECT EXISTS (
        SELECT 1 FROM inboxes i
        JOIN users u ON u.id = viewer AND u.type = 'agent' AND u.enabled AND u.deleted_at IS NULL
        LEFT JOIN inbox_access a ON a.inbox_id = i.id
        WHERE i.id = target_inbox AND i.channel = 'email' AND i.deleted_at IS NULL
        AND (
            NOT COALESCE(a.restricted, FALSE)
            OR u.email = 'System'
            OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id = ur.role_id
                       WHERE ur.user_id = viewer AND r.name = 'Admin')
            OR EXISTS (SELECT 1 FROM inbox_users iu WHERE iu.inbox_id = i.id AND iu.user_id = viewer)
            OR EXISTS (SELECT 1 FROM inbox_roles ir JOIN user_roles ur ON ur.role_id = ir.role_id
                       WHERE ir.inbox_id = i.id AND ur.user_id = viewer)
        )
    );
$$;
