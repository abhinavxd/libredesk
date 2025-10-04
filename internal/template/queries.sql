-- name: insert
INSERT INTO templates ("name", body, is_default, subject, type)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: update
WITH u AS (
    UPDATE templates
    SET 
        name = CASE WHEN $6::template_type = 'email_outgoing' THEN $2 ELSE name END,
        body = $3,
        is_default = $4,
        subject = $5,
        type = $6::template_type,
        updated_at = NOW()
    WHERE id = $1
    RETURNING *
)
SELECT * FROM u LIMIT 1;

-- name: get-default
SELECT id, created_at, updated_at, type, body, is_default, name, subject, is_builtin FROM templates WHERE is_default is TRUE;

-- name: get-all
SELECT id, created_at, updated_at, type, body, is_default, name, subject, is_builtin FROM templates WHERE type = $1 ORDER BY updated_at DESC;

-- name: get-template
SELECT id, created_at, updated_at, type, body, is_default, name, subject, is_builtin FROM templates WHERE id = $1;

-- name: delete
DELETE FROM templates WHERE id = $1;

-- name: get-by-name
SELECT id, created_at, updated_at, type, body, is_default, name, subject, is_builtin FROM templates WHERE name = $1;

-- name: is-builtin
SELECT EXISTS(SELECT 1 FROM templates WHERE id = $1 AND is_builtin is TRUE);