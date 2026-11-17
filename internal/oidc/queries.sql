-- name: get-all-oidc
SELECT id, created_at, updated_at, name, provider_url, client_id, client_secret, enabled, provider, logo_url, verify_email FROM oidc ORDER BY updated_at DESC;

-- name: get-oidc
SELECT id, created_at, updated_at, name, provider_url, client_id, client_secret, enabled, provider, logo_url, verify_email FROM oidc WHERE id = $1;

-- name: insert-oidc
INSERT INTO oidc (name, provider, provider_url, client_id, client_secret, logo_url, verify_email)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, created_at, updated_at, name, provider_url, client_id, client_secret, enabled, provider, logo_url, verify_email;

-- name: update-oidc
UPDATE oidc
SET name = $2, provider = $3, provider_url = $4, client_id = $5, client_secret = $6, enabled = $7, logo_url = $8, verify_email = $9, updated_at = now()
WHERE id = $1
RETURNING id, created_at, updated_at, name, provider_url, client_id, client_secret, enabled, provider, logo_url, verify_email;

-- name: delete-oidc
DELETE FROM oidc WHERE id = $1;
