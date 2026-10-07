-- name: get-active-inboxes
SELECT inboxes.id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, reopen_window_hours, config, "from", from_name_template, linked_email_inbox_id,
COALESCE((SELECT jsonb_agg(jsonb_build_object('email', email, 'verification_status', verification_status, 'verified_at', verified_at) ORDER BY position, id) FROM inbox_email_addresses WHERE inbox_id = inboxes.id AND kind = 'alias'), '[]'::jsonb) AS aliases
FROM inboxes where enabled is TRUE and deleted_at is NULL;

-- name: get-all-inboxes
SELECT inboxes.id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, reopen_window_hours, config, "from", from_name_template, linked_email_inbox_id,
COALESCE((SELECT jsonb_agg(jsonb_build_object('email', email, 'verification_status', verification_status, 'verified_at', verified_at) ORDER BY position, id) FROM inbox_email_addresses WHERE inbox_id = inboxes.id AND kind = 'alias'), '[]'::jsonb) AS aliases
FROM inboxes where deleted_at is NULL;

-- name: insert-inbox
INSERT INTO inboxes
(channel, config, "name", "from", enabled, csat_enabled, prompt_tags_on_reply, reopen_window_hours, secret, linked_email_inbox_id, from_name_template)
VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *, '[]'::jsonb AS aliases

-- name: get-inbox
SELECT inboxes.id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, reopen_window_hours, config, "from", from_name_template, secret, linked_email_inbox_id,
COALESCE((SELECT jsonb_agg(jsonb_build_object('email', email, 'verification_status', verification_status, 'verified_at', verified_at) ORDER BY position, id) FROM inbox_email_addresses WHERE inbox_id = inboxes.id AND kind = 'alias'), '[]'::jsonb) AS aliases
FROM inboxes where inboxes.id = $1 and deleted_at is NULL;

-- name: get-inbox-by-uuid
SELECT inboxes.id, uuid, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, prompt_tags_on_reply, reopen_window_hours, config, "from", from_name_template, secret, linked_email_inbox_id,
COALESCE((SELECT jsonb_agg(jsonb_build_object('email', email, 'verification_status', verification_status, 'verified_at', verified_at) ORDER BY position, id) FROM inbox_email_addresses WHERE inbox_id = inboxes.id AND kind = 'alias'), '[]'::jsonb) AS aliases
FROM inboxes where uuid = $1 and deleted_at is NULL;

-- name: update
UPDATE inboxes
set channel = $2, config = $3, "name" = $4, "from" = $5, csat_enabled = $6, prompt_tags_on_reply = $7, reopen_window_hours = $8, enabled = $9, secret = $10, linked_email_inbox_id = $11, from_name_template = $12, updated_at = now()
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
DELETE FROM inbox_email_addresses
WHERE inbox_id = $1 AND (NOT (LOWER(email) = ANY($2)) OR (kind = 'primary' AND LOWER(email) != $3));

-- name: lock-inbox
SELECT id FROM inboxes WHERE id = $1 AND deleted_at IS NULL FOR UPDATE;

-- name: reset-alias-verification
UPDATE inbox_email_addresses
SET verification_status = 'not_verified', verification_token = NULL, verification_started_at = NULL, verified_at = NULL
WHERE inbox_id = $1 AND kind = 'alias';

-- name: insert-inbox-email-address
INSERT INTO inbox_email_addresses (inbox_id, email, kind, position, verification_status)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (inbox_id, LOWER(email)) DO UPDATE
SET kind = EXCLUDED.kind, position = EXCLUDED.position,
    verification_status = CASE WHEN EXCLUDED.kind = 'primary' THEN 'verified' ELSE inbox_email_addresses.verification_status END,
    verification_token = CASE WHEN EXCLUDED.kind = 'primary' THEN NULL ELSE inbox_email_addresses.verification_token END;

-- name: get-alias-verification-states
SELECT email, verification_status, verified_at
FROM inbox_email_addresses
WHERE inbox_id = $1 AND kind = 'alias'
FOR UPDATE;

-- name: start-alias-verification
UPDATE inbox_email_addresses
SET verification_status = $3,
    verification_token = $4,
    verification_started_at = NOW(),
    verified_at = CASE WHEN verification_status IN ('verified', 'pending') THEN verified_at ELSE NULL END
WHERE inbox_id = $1 AND LOWER(email) = LOWER($2) AND kind = 'alias';

-- name: fail-alias-verification
UPDATE inbox_email_addresses
SET verification_status = $3, verification_token = NULL, verified_at = NULL
WHERE inbox_id = $1 AND LOWER(email) = LOWER($2) AND kind = 'alias' AND verification_token = $4;

-- name: complete-alias-verification
UPDATE inbox_email_addresses
SET verification_status = $4, verification_token = NULL, verified_at = NOW()
WHERE inbox_id = $1 AND verification_token = $2 AND LOWER(email) = $3
  AND kind = 'alias';

-- name: fail-alias-verification-by-token
UPDATE inbox_email_addresses
SET verification_status = $3, verification_token = NULL, verified_at = NULL
WHERE inbox_id = $1 AND verification_token = $2 AND kind = 'alias' AND verification_status = $4;

-- name: expire-alias-verifications
UPDATE inbox_email_addresses
SET verification_status = $2, verification_token = NULL, verified_at = NULL
WHERE inbox_id = $1 AND kind = 'alias' AND verification_status = $3 AND verification_started_at < $4;
