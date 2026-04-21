-- name: get-active-inboxes
SELECT id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, config, "from", linked_email_inbox_id FROM inboxes where enabled is TRUE and deleted_at is NULL;

-- name: get-all-inboxes
SELECT id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, config, "from", linked_email_inbox_id FROM inboxes where deleted_at is NULL;

-- name: insert-inbox
INSERT INTO inboxes
(channel, config, "name", "from", csat_enabled, prompt_tags_on_reply, secret, linked_email_inbox_id)
VALUES($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *

-- name: get-inbox
SELECT id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, config, "from", secret, linked_email_inbox_id FROM inboxes where id = $1 and deleted_at is NULL;

-- name: get-inbox-by-uuid
SELECT id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, config, "from", secret, linked_email_inbox_id FROM inboxes where uuid = $1 and deleted_at is NULL;

-- name: update
UPDATE inboxes
set channel = $2, config = $3, "name" = $4, "from" = $5, csat_enabled = $6, prompt_tags_on_reply = $7, enabled = $8, secret = $9, linked_email_inbox_id = $10, updated_at = now()
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