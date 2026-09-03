-- name: get-active-inboxes
SELECT inboxes.id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, config, "from", from_name_template, linked_email_inbox_id,
COALESCE((SELECT jsonb_agg(jsonb_build_object('email', email, 'verification_status', verification_status, 'verified_at', verified_at) ORDER BY position, id) FROM inbox_email_addresses WHERE inbox_id = inboxes.id AND kind = 'alias'), '[]'::jsonb) AS aliases
FROM inboxes where enabled is TRUE and deleted_at is NULL;

-- name: get-all-inboxes
SELECT inboxes.id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, config, "from", from_name_template, linked_email_inbox_id,
COALESCE((SELECT jsonb_agg(jsonb_build_object('email', email, 'verification_status', verification_status, 'verified_at', verified_at) ORDER BY position, id) FROM inbox_email_addresses WHERE inbox_id = inboxes.id AND kind = 'alias'), '[]'::jsonb) AS aliases
FROM inboxes where deleted_at is NULL;

-- name: insert-inbox
INSERT INTO inboxes
(channel, config, "name", "from", enabled, csat_enabled, prompt_tags_on_reply, secret, linked_email_inbox_id, from_name_template)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *, '[]'::jsonb AS aliases

-- name: get-inbox
SELECT inboxes.id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, config, "from", from_name_template, secret, linked_email_inbox_id,
COALESCE((SELECT jsonb_agg(jsonb_build_object('email', email, 'verification_status', verification_status, 'verified_at', verified_at) ORDER BY position, id) FROM inbox_email_addresses WHERE inbox_id = inboxes.id AND kind = 'alias'), '[]'::jsonb) AS aliases
FROM inboxes where inboxes.id = $1 and deleted_at is NULL;

-- name: get-inbox-by-uuid
SELECT inboxes.id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, config, "from", from_name_template, secret, linked_email_inbox_id,
COALESCE((SELECT jsonb_agg(jsonb_build_object('email', email, 'verification_status', verification_status, 'verified_at', verified_at) ORDER BY position, id) FROM inbox_email_addresses WHERE inbox_id = inboxes.id AND kind = 'alias'), '[]'::jsonb) AS aliases
FROM inboxes where uuid = $1 and deleted_at is NULL;

-- name: update
UPDATE inboxes
set channel = $2, config = $3, "name" = $4, "from" = $5, csat_enabled = $6, prompt_tags_on_reply = $7, enabled = $8, secret = $9, linked_email_inbox_id = $10, from_name_template = $11, updated_at = now()
where id = $1 and deleted_at is NULL
RETURNING *, '[]'::jsonb AS aliases;

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

-- name: delete-inbox-email-addresses
DELETE FROM inbox_email_addresses WHERE inbox_id = $1;

-- name: insert-inbox-email-address
INSERT INTO inbox_email_addresses (inbox_id, email, kind, position, verification_status, verification_token, verification_started_at, verified_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: get-email-address-owner
SELECT inbox_id FROM inbox_email_addresses WHERE LOWER(email) = LOWER($1);
