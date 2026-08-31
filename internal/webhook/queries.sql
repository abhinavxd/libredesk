-- name: get-webhooks-compact
SELECT id, name FROM webhooks ORDER BY name;

-- name: get-all-webhooks
SELECT
    id,
    created_at,
    updated_at,
    name,
    url,
    events,
    secret,
    is_active,
    delivery
FROM
    webhooks
ORDER BY created_at DESC;

-- name: get-webhook
SELECT
    id,
    created_at,
    updated_at,
    name,
    url,
    events,
    secret,
    is_active,
    delivery
FROM
    webhooks
WHERE
    id = $1;

-- name: get-webhook-secret
SELECT
    secret
FROM
    webhooks
WHERE
    id = $1;

-- name: get-active-webhooks
SELECT
    id,
    created_at,
    updated_at,
    name,
    url,
    events,
    secret,
    is_active,
    delivery
FROM
    webhooks
WHERE
    is_active = true
ORDER BY created_at DESC;

-- name: get-webhooks-by-event
SELECT
    id,
    created_at,
    updated_at,
    name,
    url,
    events,
    secret,
    is_active,
    delivery
FROM
    webhooks
WHERE
    is_active = true AND
    $1 = ANY(events);

-- name: insert-webhook
INSERT INTO
    webhooks (name, url, events, secret, is_active, delivery)
VALUES
    ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: update-webhook
UPDATE
    webhooks
SET
    name = $2,
    url = $3,
    events = $4,
    secret = $5,
    is_active = $6,
    delivery = $7,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;

-- name: delete-webhook
DELETE FROM
    webhooks
WHERE
    id = $1;

-- name: toggle-webhook
UPDATE
    webhooks
SET
    is_active = NOT is_active,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;
