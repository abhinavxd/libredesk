-- name: get
SELECT
    id,
    created_at,
    updated_at,
    name,
    actions,
    visibility,
    visible_when,
    message_content,
    user_id,
    team_id,
    usage_count
FROM
    macros
WHERE
    id = $1;

-- name: get-all
SELECT
    id,
    created_at,
    updated_at,
    name,
    actions,
    visibility,
    visible_when,
    message_content,
    user_id,
    team_id,
    usage_count
FROM
    macros
ORDER BY
    updated_at DESC;

-- name: get-all-compact
SELECT
    COUNT(*) OVER() AS total,
    id,
    created_at,
    updated_at,
    name,
    actions,
    visibility,
    visible_when,
    message_content != '' AS has_message_content,
    user_id,
    team_id,
    usage_count
FROM
    macros
WHERE
    ($1 = '' OR name ILIKE '%' || $1 || '%')
ORDER BY
    updated_at DESC
LIMIT $2 OFFSET $3;

-- name: search-compact
SELECT
    id,
    created_at,
    updated_at,
    name,
    actions,
    visibility,
    visible_when,
    message_content != '' AS has_message_content,
    user_id,
    team_id,
    usage_count
FROM
    macros
WHERE
    ($1 = '' OR name ILIKE '%' || $1 || '%')
    AND ($2 = '' OR $2 = ANY(visible_when::text[]))
    AND (
        visibility = 'all'
        OR (visibility = 'user' AND user_id = $3)
        OR (visibility = 'team' AND team_id = ANY($4::bigint[]))
    )
ORDER BY
    usage_count DESC,
    updated_at DESC
LIMIT 30;

-- name: create
INSERT INTO
    macros (name, message_content, user_id, team_id, visibility, visible_when, actions)
VALUES
    ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: update
UPDATE
    macros
SET
    name = $2,
    message_content = $3,
    user_id = $4,
    team_id = $5,
    visibility = $6,
    visible_when = $7,
    actions = $8,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;

-- name: delete
DELETE FROM
    macros
WHERE
    id = $1;

-- name: increment-usage-count
UPDATE
    macros
SET
    usage_count = usage_count + 1
WHERE
    id = $1;