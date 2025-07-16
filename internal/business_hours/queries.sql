-- name: get-business-hours
SELECT id,
    created_at,
    updated_at,
    "name",
    description,
    is_always_open,
    hours,
    holidays
FROM business_hours
WHERE id = $1;

-- name: get-all-business-hours
SELECT id,
    created_at,
    updated_at,
    "name",
    description,
    is_always_open,
    hours,
    holidays
FROM business_hours
ORDER BY updated_at DESC;

-- name: insert-business-hours
INSERT INTO business_hours (
        "name",
        description,
        is_always_open,
        hours,
        holidays
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: delete-business-hours
DELETE FROM business_hours
WHERE id = $1;

-- name: update-business-hours
UPDATE business_hours
SET "name" = $2,
    description = $3,
    is_always_open = $4,
    hours = $5,
    holidays = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;