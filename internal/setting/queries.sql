-- name: get-all
SELECT JSON_OBJECT_AGG(key, value) AS settings FROM (SELECT * FROM settings ORDER BY key) t;

-- name: update
UPDATE settings AS s
SET value = c.value,
    updated_at = now()
FROM (SELECT * FROM jsonb_each($1)) AS c(key, value)
WHERE s.key = c.key;

-- name: get-by-prefix
SELECT JSON_OBJECT_AGG(key, value) AS settings 
FROM settings 
WHERE key LIKE $1 || '%';

-- name: get
SELECT value FROM settings WHERE key = $1;

-- name: set-resource-policy
INSERT INTO settings (key, value) VALUES ('security.resource_policy', $1::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
