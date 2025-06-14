-- name: get-all-webhooks
SELECT
    id,
    created_at,
    updated_at,
    name,
    url,
    events,
    secret,
    is_active
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
    is_active
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
    is_active
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
    is_active
FROM
    webhooks
WHERE
    is_active = true AND
    $1 = ANY(events);

-- name: insert-webhook
INSERT INTO
    webhooks (name, url, events, secret, is_active)
VALUES
    ($1, $2, $3, $4, $5)
RETURNING id;

-- name: update-webhook
UPDATE
    webhooks
SET
    name = $2,
    url = $3,
    events = $4,
    secret = $5,
    is_active = $6,
    updated_at = NOW()
WHERE
    id = $1;

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
    id = $1;
