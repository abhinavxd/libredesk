-- name: get-organization
SELECT
    id,
    created_at,
    updated_at,
    name,
    website,
    email_domain,
    phone
FROM
    organizations
WHERE
    id = $1;

-- name: insert-organization
INSERT INTO
    organizations (name, website, email_domain, phone)
VALUES
    ($1, $2, $3, $4)
RETURNING *;

-- name: update-organization
UPDATE
    organizations
SET
    name = $2,
    website = $3,
    email_domain = $4,
    phone = $5,
    updated_at = now()
WHERE
    id = $1
RETURNING *;

-- name: delete-organization
DELETE FROM
    organizations
WHERE
    id = $1;

-- name: get-organization-contacts-count
SELECT
    COUNT(*)
FROM
    users
WHERE
    organization_id = $1 AND type = 'contact' AND deleted_at IS NULL;
