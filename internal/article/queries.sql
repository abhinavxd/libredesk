
-- name: insert-article
INSERT INTO
    articles (title, content, section_id)
VALUES
    ($1, $2, $3)
RETURNING id;

-- name: delete-article
DELETE FROM
    articles
WHERE
    id = $1;

-- name: update-article
UPDATE
    articles
SET
    title = $2,
    content = $3,
    section_id = $4,
    updated_at = now()
WHERE
    id = $1;

-- name: get-all-articles
SELECT
    a.id,
    a.created_at,
    a.updated_at,
    a.title,
    a.content,
    a.section_id,
    a.published_at,
    a.is_published,
    s.name AS section_name,
    s.category_id,
    c.name AS category_name
FROM
    articles a
    LEFT JOIN article_sections s ON a.section_id = s.id
    LEFT JOIN article_categories c ON s.category_id = c.id
ORDER BY
    a.updated_at DESC;


-- name: get-article-by-id
SELECT
    id,
    created_at,
    updated_at,
    title,
    content,
    section_id,
    published_at,
    is_published
FROM
    articles
WHERE
    id = $1;

-- name: get-articles-by-section-id
SELECT
    id,
    created_at,
    updated_at,
    title,
    content,
    section_id,
    published_at,
    is_published
FROM
    articles
WHERE
    section_id = $1
ORDER BY
    updated_at DESC;


-- name: publish-article
UPDATE
    articles
SET
    is_published = true,
    published_at = now(),
    updated_at = now()
WHERE
    id = $1;

-- name: unpublish-article
UPDATE
    articles
SET
    is_published = false,
    published_at = NULL,
    updated_at = now()
WHERE
    id = $1;