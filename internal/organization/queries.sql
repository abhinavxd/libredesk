-- name: get-all
SELECT id, created_at, updated_at, name, domains, notes, external_id, custom_attributes
FROM organizations
ORDER BY name ASC;

-- name: get
SELECT id, created_at, updated_at, name, domains, notes, external_id, custom_attributes
FROM organizations
WHERE id = $1;

-- name: insert
INSERT INTO organizations (name, domains, notes, external_id)
VALUES ($1, $2, $3, NULLIF($4, ''))
RETURNING id, created_at, updated_at, name, domains, notes, external_id, custom_attributes;

-- name: update
UPDATE organizations
SET name = $2, domains = $3, notes = $4, external_id = NULLIF($5, ''), updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, name, domains, notes, external_id, custom_attributes;

-- name: delete
DELETE FROM organizations WHERE id = $1;

-- name: find-by-domain
SELECT id, created_at, updated_at, name, domains, notes, external_id, custom_attributes
FROM organizations
WHERE lower($1) = ANY (SELECT lower(d) FROM unnest(domains) AS d)
ORDER BY id ASC
LIMIT 1;

-- name: assign-if-empty
UPDATE users
SET organization_id = $2, updated_at = NOW()
WHERE id = $1 AND type IN ('contact', 'visitor') AND organization_id IS NULL;

-- name: set-user-organization
UPDATE users
SET organization_id = $2, updated_at = NOW()
WHERE id = $1 AND type IN ('contact', 'visitor');
