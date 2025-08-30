-- name: get-active-inboxes
SELECT id, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, config, "from" FROM inboxes where enabled is TRUE and deleted_at is NULL;

-- name: get-all-inboxes
SELECT id, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, config, "from" FROM inboxes where deleted_at is NULL;

-- name: insert-inbox
INSERT INTO inboxes
(channel, config, "name", "from", csat_enabled)
VALUES($1, $2, $3, $4, $5)
RETURNING *

-- name: get-inbox
SELECT id, created_at, updated_at, "name", deleted_at, channel, enabled, csat_enabled, config, "from" FROM inboxes where id = $1 and deleted_at is NULL;

-- name: update
UPDATE inboxes
set channel = $2, config = $3, "name" = $4, "from" = $5, csat_enabled = $6, enabled = $7, updated_at = now()
where id = $1 and deleted_at is NULL
RETURNING *;

-- name: soft-delete
UPDATE inboxes set deleted_at = now(), updated_at = now(), config = '{}' where id = $1 and deleted_at is NULL;

-- name: toggle
UPDATE inboxes 
SET enabled = NOT enabled, updated_at = NOW() 
WHERE id = $1
RETURNING *;