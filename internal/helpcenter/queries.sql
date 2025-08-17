-- Help Centers
-- name: get-all-help-centers
SELECT
    id,
    created_at,
    updated_at,
    name,
    slug,
    page_title,
    view_count,
    default_locale
FROM
    help_centers
ORDER BY created_at DESC;

-- name: get-help-center-by-id
SELECT
    id,
    created_at,
    updated_at,
    name,
    slug,
    page_title,
    view_count,
    default_locale
FROM
    help_centers
WHERE
    id = $1;

-- name: get-help-center-by-slug
SELECT
    id,
    created_at,
    updated_at,
    name,
    slug,
    page_title,
    view_count,
    default_locale
FROM
    help_centers
WHERE
    slug = $1;

-- name: insert-help-center
INSERT INTO
    help_centers (name, slug, page_title, default_locale)
VALUES
    ($1, $2, $3, $4)
RETURNING *;

-- name: update-help-center
UPDATE
    help_centers
SET
    name = $2,
    slug = $3,
    page_title = $4,
    default_locale = $5,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;

-- name: delete-help-center
DELETE FROM
    help_centers
WHERE
    id = $1;


-- Collections
-- name: get-collections-by-help-center
SELECT
    id,
    created_at,
    updated_at,
    help_center_id,
    slug,
    parent_id,
    locale,
    name,
    description,
    sort_order,
    is_published
FROM
    article_collections
WHERE
    help_center_id = $1
ORDER BY sort_order ASC, created_at DESC;

-- name: get-collections-by-help-center-and-locale
SELECT
    id,
    created_at,
    updated_at,
    help_center_id,
    slug,
    parent_id,
    locale,
    name,
    description,
    sort_order,
    is_published
FROM
    article_collections
WHERE
    help_center_id = $1 AND locale = $2
ORDER BY sort_order ASC, created_at DESC;



-- name: get-collection-by-id
SELECT
    id,
    created_at,
    updated_at,
    help_center_id,
    slug,
    parent_id,
    locale,
    name,
    description,
    sort_order,
    is_published
FROM
    article_collections
WHERE
    id = $1;


-- name: insert-collection
INSERT INTO
    article_collections (help_center_id, slug, parent_id, locale, name, description, sort_order, is_published)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: update-collection
UPDATE
    article_collections
SET
    slug = $2,
    parent_id = $3,
    locale = $4,
    name = $5,
    description = $6,
    sort_order = $7,
    is_published = $8,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;


-- name: move-collection
UPDATE
    article_collections
SET
    parent_id = $2,
    sort_order = $3,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;

-- name: toggle-collection-published
UPDATE
    article_collections
SET
    is_published = NOT is_published,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;

-- name: delete-collection
DELETE FROM
    article_collections
WHERE
    id = $1;

-- Articles
-- name: get-articles-by-collection
SELECT
    id,
    created_at,
    updated_at,
    collection_id,
    slug,
    locale,
    title,
    content,
    sort_order,
    status,
    view_count,
    ai_enabled
FROM
    help_articles
WHERE
    collection_id = $1
ORDER BY sort_order ASC, created_at DESC;

-- name: get-articles-by-collection-and-locale
SELECT
    id,
    created_at,
    updated_at,
    collection_id,
    slug,
    locale,
    title,
    content,
    sort_order,
    status,
    view_count,
    ai_enabled
FROM
    help_articles
WHERE
    collection_id = $1 AND locale = $2
ORDER BY sort_order ASC, created_at DESC;


-- name: get-article-by-id
SELECT
    id,
    created_at,
    updated_at,
    collection_id,
    slug,
    locale,
    title,
    content,
    sort_order,
    status,
    view_count,
    ai_enabled
FROM
    help_articles
WHERE
    id = $1;


-- name: insert-article
INSERT INTO
    help_articles (collection_id, slug, locale, title, content, sort_order, status, ai_enabled)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: update-article
UPDATE
    help_articles
SET
    slug = $2,
    locale = $3,
    title = $4,
    content = $5,
    sort_order = $6,
    status = $7,
    ai_enabled = $8,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;


-- name: move-article
UPDATE
    help_articles
SET
    collection_id = $2,
    sort_order = $3,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;

-- name: update-article-status
UPDATE
    help_articles
SET
    status = $2,
    updated_at = NOW()
WHERE
    id = $1
RETURNING *;

-- name: delete-article
DELETE FROM
    help_articles
WHERE
    id = $1;



-- Statistics and hierarchy queries


-- Get complete tree data for a help center
-- name: get-help-center-tree-data
SELECT 
    'collection' as type,
    c.id,
    c.created_at,
    c.updated_at,
    c.help_center_id,
    c.slug,
    c.parent_id,
    c.locale,
    c.name,
    c.description,
    c.sort_order,
    c.is_published,
    NULL::INTEGER as collection_id,
    NULL::TEXT as title,
    NULL::TEXT as content,
    NULL::TEXT as status,
    NULL::INTEGER as view_count,
    NULL::BOOLEAN as ai_enabled
FROM article_collections c
WHERE c.help_center_id = $1 
  AND ($2 = '' OR c.locale = $2)

UNION ALL

SELECT 
    'article' as type,
    a.id,
    a.created_at,
    a.updated_at,
    c.help_center_id,
    a.slug,
    NULL::INTEGER as parent_id,
    a.locale,
    a.title as name,
    NULL::TEXT as description,
    a.sort_order,
    NULL::BOOLEAN as is_published,
    a.collection_id,
    a.title,
    a.content,
    a.status,
    a.view_count,
    a.ai_enabled
FROM help_articles a
JOIN article_collections c ON a.collection_id = c.id
WHERE c.help_center_id = $1 
  AND ($2 = '' OR a.locale = $2)

ORDER BY help_center_id, type DESC, parent_id NULLS FIRST, sort_order, name;

-- AI and Embeddings queries
-- name: search-articles-by-vector
SELECT
    a.id,
    a.created_at,
    a.updated_at,
    a.collection_id,
    a.slug,
    a.locale,
    a.title,
    a.content,
    a.sort_order,
    a.status,
    a.view_count,
    a.ai_enabled,
    1 - (e.embedding <=> $2::vector) AS similarity
FROM help_articles a
JOIN article_collections c ON a.collection_id = c.id
JOIN embeddings e ON e.source_type = 'help_article' AND e.source_id = a.id
WHERE c.help_center_id = $1 
  AND a.status = 'published'
  AND a.ai_enabled = true
  AND e.embedding IS NOT NULL
ORDER BY 
    (CASE WHEN a.locale = $3 THEN 0 ELSE 1 END),
    e.embedding <=> $2::vector
LIMIT $4;

-- name: update-article-embedding
UPDATE embeddings 
SET embedding = $2::vector, updated_at = NOW()
WHERE source_type = 'help_article' AND source_id = $1;

-- name: search-knowledge-base
SELECT 
    'help_article' as source_type,
    a.id as source_id,
    a.title as title,
    e.chunk_text as content,
    c.help_center_id,
    1 - (e.embedding <=> $2) AS similarity
FROM help_articles a
JOIN article_collections c ON a.collection_id = c.id
JOIN embeddings e ON e.source_type = 'help_article' AND e.source_id = a.id
WHERE a.status = 'published'
  AND a.ai_enabled = true
  AND e.embedding IS NOT NULL
  AND c.help_center_id = $1
  AND a.locale = $3
  AND (1 - (e.embedding <=> $2)) >= $4
ORDER BY similarity DESC
LIMIT $5;

-- Embeddings management
-- name: insert-embedding
INSERT INTO embeddings (source_type, source_id, chunk_text, embedding, meta)
VALUES ($1, $2, $3, $4::vector, $5)
RETURNING *;


-- name: delete-embeddings-by-source
DELETE FROM embeddings 
WHERE source_type = $1 AND source_id = $2;

-- name: has-articles-in-language
SELECT COUNT(*) > 0
FROM help_articles a
JOIN article_collections c ON a.collection_id = c.id
WHERE c.help_center_id = $1 
  AND a.locale = $2
  AND a.status = 'published' 
  AND a.ai_enabled = true;


