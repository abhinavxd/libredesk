-- name: get-active-inboxes
SELECT id, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, config, "from", linked_email_inbox_id FROM inboxes where enabled is TRUE and deleted_at is NULL;

-- name: get-all-inboxes
SELECT id, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, config, "from", linked_email_inbox_id FROM inboxes where deleted_at is NULL;

-- name: insert-inbox
INSERT INTO inboxes
(channel, config, "name", "from", csat_enabled, secret, linked_email_inbox_id)
VALUES($1, $2, $3, $4, $5, $6, $7)
RETURNING *

-- name: get-inbox
SELECT id, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, config, "from", secret, linked_email_inbox_id FROM inboxes where id = $1 and deleted_at is NULL;

-- name: update
UPDATE inboxes
set channel = $2, config = $3, "name" = $4, "from" = $5, csat_enabled = $6, enabled = $7, secret = $8, linked_email_inbox_id = $9, updated_at = now()
where id = $1 and deleted_at is NULL
RETURNING *;

-- name: soft-delete
UPDATE inboxes set deleted_at = now(), updated_at = now(), config = '{}', enabled = false where id = $1 and deleted_at is NULL;

-- name: toggle
UPDATE inboxes
SET enabled = NOT enabled, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: update-config
UPDATE inboxes
SET config = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;