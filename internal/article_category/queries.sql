-- name: get-all-categories
select
    id,
    created_at,
    updated_at,
    name
from
    article_categories;

-- name: insert-category
INSERT into
    article_categories (name)
values
    ($1);

-- name: delete-category
DELETE from
    article_categories
where
    id = $1;

-- name: update-category
UPDATE
    article_categories
set
    name = $2,
    updated_at = now()
where
    id = $1;

-- name: get-all-categories-with-sections
SELECT
    c.id,
    c.created_at,
    c.updated_at,
    c.name,
    COALESCE(
        json_agg(
            json_build_object(
                'id', s.id,
                'created_at', s.created_at,
                'updated_at', s.updated_at,
                'name', s.name
            )
        ),
        '[]'::json
    ) AS sections
    FROM
    article_categories c
    LEFT JOIN
    article_sections s ON c.id = s.category_id
    GROUP BY
    c.id, c.created_at, c.updated_at, c.name
    ORDER BY
    c.updated_at DESC;
