-- name: get-all-custom-attributes
SELECT
    id,
    created_at,
    updated_at,
    name,
    description,
    applies_to,
    key,
    values,
    data_type,
    regex,
    regex_hint
FROM
    custom_attribute_definitions
WHERE
    CASE WHEN $1 = '' THEN TRUE
         ELSE applies_to = $1
    END
ORDER BY
    updated_at DESC;

-- name: get-custom-attribute
SELECT
    id,
    created_at,
    updated_at,
    name,
    description,
    applies_to,
    key,
    values,
    data_type,
    regex,
    regex_hint
FROM
    custom_attribute_definitions
WHERE
    id = $1;

-- name: insert-custom-attribute
INSERT INTO
    custom_attribute_definitions (applies_to, name, description, key, values, data_type, regex, regex_hint)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *

-- name: delete-custom-attribute
DELETE FROM
    custom_attribute_definitions
WHERE
    id = $1;

-- name: update-custom-attribute
UPDATE
    custom_attribute_definitions
SET
    applies_to = $2,
    name = $3,
    description = $4,
    values = $5,
    regex = $6,
    regex_hint = $7,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;