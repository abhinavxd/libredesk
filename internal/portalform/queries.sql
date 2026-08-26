-- name: get-all
SELECT id, created_at, updated_at, "name", ask_subject, fields
FROM portal_forms
ORDER BY "name";

-- name: get
SELECT id, created_at, updated_at, "name", ask_subject, fields
FROM portal_forms
WHERE id = $1;

-- name: insert
INSERT INTO portal_forms ("name", ask_subject, fields)
VALUES ($1, $2, $3)
RETURNING id, created_at, updated_at, "name", ask_subject, fields;

-- name: update
UPDATE portal_forms
SET "name" = $2,
    ask_subject = $3,
    fields = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, "name", ask_subject, fields;

-- name: delete
DELETE FROM portal_forms WHERE id = $1;
