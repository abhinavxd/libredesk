-- name: get-all-oidc
SELECT id, created_at, updated_at, name, provider_url, client_id, client_secret, enabled, provider FROM oidc order by updated_at desc;

-- name: get-oidc
SELECT id, created_at, updated_at, name, provider_url, client_id, client_secret, enabled, provider FROM oidc WHERE id = $1;

-- name: insert-oidc
INSERT INTO oidc (name, provider, provider_url, client_id, client_secret) 
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: update-oidc
UPDATE oidc 
SET name = $2, provider = $3, provider_url = $4, client_id = $5, client_secret = $6, enabled = $7, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: delete-oidc
DELETE FROM oidc WHERE id = $1;
