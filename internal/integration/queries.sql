-- name: get-all-integrations
SELECT
    id,
    created_at,
    updated_at,
    provider,
    config,
    enabled
FROM
    integrations
ORDER BY created_at DESC;

-- name: get-integration
SELECT
    id,
    created_at,
    updated_at,
    provider,
    config,
    enabled
FROM
    integrations
WHERE
    provider = $1;

-- name: upsert-integration
INSERT INTO
    integrations (provider, config, enabled)
VALUES
    ($1, $2, $3)
ON CONFLICT (provider) DO UPDATE SET
    config = EXCLUDED.config,
    enabled = EXCLUDED.enabled,
    updated_at = NOW()
RETURNING *;

-- name: delete-integration
DELETE FROM
    integrations
WHERE
    provider = $1;

-- name: toggle-integration
UPDATE
    integrations
SET
    enabled = NOT enabled,
    updated_at = NOW()
WHERE
    provider = $1
RETURNING *;
