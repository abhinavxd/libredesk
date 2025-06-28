-- name: get-all-sections
select
    id,
    created_at,
    updated_at,
    name,
    category_id
from
    article_sections;

-- name: insert-section
INSERT into
    article_sections (name, category_id)
values
    ($1, $2);

-- name: delete-section
DELETE from
    article_sections
where
    id = $1;

-- name: update-section
UPDATE
    article_sections
set
    name = $2,
    category_id = $3,
    updated_at = now()
where
    id = $1;

-- name: get-sections-with-category
SELECT
    s.id,
    s.created_at,
    s.updated_at,
    s.name,
    s.category_id,
    c.name AS category_name
FROM
    article_sections s
    LEFT JOIN article_categories c ON s.category_id = c.id
ORDER BY
    s.updated_at DESC;

-- name: get-section-with-category
SELECT
    s.id,
    s.created_at,
    s.updated_at,
    s.name,
    s.category_id,
    c.name AS category_name
FROM
    article_sections s
    INNER JOIN article_categories c ON s.category_id = c.id
WHERE
    s.id = $1;
