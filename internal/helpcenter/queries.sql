-- name: get-all-help-centers
SELECT id, created_at, updated_at, name, slug, page_title, meta_description, custom_css, custom_js, default_locale, allowed_locales, is_active, theme, custom_domain, template
FROM help_centers
ORDER BY created_at DESC;

-- name: get-active-help-centers
SELECT id, created_at, updated_at, name, slug, page_title, meta_description, custom_css, custom_js, default_locale, allowed_locales, is_active, theme, custom_domain, template
FROM help_centers
WHERE is_active = true
ORDER BY created_at DESC;

-- name: get-help-center-by-id
SELECT id, created_at, updated_at, name, slug, page_title, meta_description, custom_css, custom_js, default_locale, allowed_locales, is_active, theme, custom_domain, template
FROM help_centers
WHERE id = $1;

-- name: get-help-center-by-slug
SELECT id, created_at, updated_at, name, slug, page_title, meta_description, custom_css, custom_js, default_locale, allowed_locales, is_active, theme, custom_domain, template
FROM help_centers
WHERE slug = $1 AND is_active = true;

-- name: insert-help-center
INSERT INTO help_centers (name, slug, page_title, meta_description, custom_css, custom_js, default_locale, allowed_locales, theme, custom_domain, template)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: update-help-center
UPDATE help_centers
SET name = $2, slug = $3, page_title = $4, meta_description = $5, custom_css = $6, custom_js = $7, default_locale = $8, allowed_locales = $9, theme = $10, custom_domain = $11, template = $12, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: toggle-help-center-active
UPDATE help_centers
SET is_active = NOT is_active, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: delete-help-center
DELETE FROM help_centers
WHERE id = $1;

-- name: get-help-center-locales-in-use
SELECT DISTINCT locale FROM article_collections WHERE help_center_id = $1
UNION
SELECT DISTINCT a.locale
FROM help_articles a
JOIN article_collections c ON c.id = a.collection_id
WHERE c.help_center_id = $1;

-- name: get-collections-by-help-center
SELECT id, created_at, updated_at, help_center_id, slug, parent_id, locale, name, description, icon, sort_order, is_published
FROM article_collections
WHERE help_center_id = $1
ORDER BY sort_order ASC, created_at DESC;

-- name: get-collection-by-id
SELECT id, created_at, updated_at, help_center_id, slug, parent_id, locale, name, description, icon, sort_order, is_published
FROM article_collections
WHERE id = $1;

-- name: get-collection-subtree-depth
WITH RECURSIVE subtree AS (
    SELECT id, 1 AS depth FROM article_collections WHERE id = $1
    UNION ALL
    SELECT c.id, s.depth + 1 FROM article_collections c JOIN subtree s ON c.parent_id = s.id
)
SELECT COALESCE(MAX(depth), 1) FROM subtree;

-- name: lock-help-center
SELECT id FROM help_centers WHERE id = $1 FOR UPDATE;

-- name: lock-help-center-by-collection
SELECT hc.id FROM help_centers hc
JOIN article_collections c ON c.help_center_id = hc.id
WHERE c.id = $1
FOR UPDATE OF hc;

-- name: collection-has-content
SELECT EXISTS(SELECT 1 FROM article_collections WHERE parent_id = $1)
    OR EXISTS(SELECT 1 FROM help_articles WHERE collection_id = $1);

-- name: get-article-ids-in-collection-subtree
WITH RECURSIVE subtree AS (
    SELECT id FROM article_collections WHERE id = $1
    UNION ALL
    SELECT c.id FROM article_collections c JOIN subtree s ON c.parent_id = s.id
)
SELECT a.id FROM help_articles a WHERE a.collection_id IN (SELECT id FROM subtree);

-- name: get-article-ids-in-help-center
SELECT a.id FROM help_articles a
JOIN article_collections c ON c.id = a.collection_id
WHERE c.help_center_id = $1;

-- name: update-collection-sort-order
UPDATE article_collections
SET sort_order = $3, updated_at = NOW()
WHERE id = $1 AND help_center_id = $2;

-- name: get-collection-by-id-for-update
SELECT id, created_at, updated_at, help_center_id, slug, parent_id, locale, name, description, icon, sort_order, is_published
FROM article_collections
WHERE id = $1
FOR UPDATE;

-- name: collection-slug-exists-in-help-center
SELECT EXISTS(
    SELECT 1
    FROM article_collections
    WHERE help_center_id = $1 AND slug = $2 AND locale = $3
);

-- name: insert-collection
INSERT INTO article_collections (help_center_id, slug, parent_id, locale, name, description, icon, sort_order, is_published)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: update-collection
UPDATE article_collections
SET slug = $2, parent_id = $3, locale = $4, name = $5, description = $6, icon = $7, sort_order = $8, is_published = $9, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: toggle-collection-published
UPDATE article_collections
SET is_published = NOT is_published, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: delete-collection
DELETE FROM article_collections
WHERE id = $1;

-- name: get-article-by-id
SELECT a.id, a.created_at, a.updated_at, a.collection_id, a.author_id, a.created_by, a.slug, a.locale, a.title, a.content,
    a.excerpt, a.meta_title, a.meta_description, a.meta_image_url, a.sort_order, a.status, a.view_count, a.ai_enabled,
    TRIM(u.first_name || ' ' || COALESCE(u.last_name, '')) AS author_name,
    TRIM(cu.first_name || ' ' || COALESCE(cu.last_name, '')) AS created_by_name,
    (SELECT COUNT(*) FROM help_article_feedback f WHERE f.article_id = a.id AND f.is_helpful) AS helpful_count,
    (SELECT COUNT(*) FROM help_article_feedback f WHERE f.article_id = a.id AND NOT f.is_helpful) AS not_helpful_count
FROM help_articles a
LEFT JOIN users u ON u.id = a.author_id
LEFT JOIN users cu ON cu.id = a.created_by
WHERE a.id = $1;

-- name: insert-article
INSERT INTO help_articles (collection_id, author_id, created_by, slug, locale, title, content, excerpt, meta_title, meta_description, meta_image_url, sort_order, status, ai_enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: update-article
UPDATE help_articles
SET collection_id = COALESCE($9, collection_id), slug = $2, locale = $3, title = $4, content = $5, sort_order = $6, status = $7, ai_enabled = $8,
    excerpt = $10, meta_title = $11, meta_description = $12, meta_image_url = $13, author_id = $14, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: user-is-author-assignable
SELECT EXISTS(
    SELECT 1 FROM users
    WHERE id = $1 AND type IN ('agent', 'ai_assistant') AND deleted_at IS NULL
);

-- name: article-slug-exists-in-help-center
SELECT EXISTS(
    SELECT 1
    FROM help_articles a
    JOIN article_collections c ON c.id = a.collection_id
    WHERE c.help_center_id = (SELECT help_center_id FROM article_collections WHERE id = $1)
        AND a.slug = $2 AND a.locale = $3
);

-- name: other-article-slug-exists-in-help-center
SELECT EXISTS(
    SELECT 1
    FROM help_articles a
    JOIN article_collections c ON c.id = a.collection_id
    WHERE c.help_center_id = (SELECT help_center_id FROM article_collections WHERE id = $1)
        AND a.slug = $2 AND a.locale = $3 AND a.id <> $4
);

-- name: move-article-to-collection
UPDATE help_articles
SET collection_id = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: update-article-sort-order
UPDATE help_articles
SET sort_order = $3, updated_at = NOW()
WHERE id = $1 AND collection_id = $2;

-- name: update-article-status
UPDATE help_articles
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: delete-article
DELETE FROM help_articles
WHERE id = $1;

-- name: get-help-center-tree-data
SELECT
    'collection' AS type,
    c.id,
    c.created_at,
    c.updated_at,
    c.help_center_id,
    c.slug,
    c.parent_id,
    c.locale,
    c.name,
    c.description,
    c.icon,
    c.sort_order,
    c.is_published,
    NULL::INTEGER AS collection_id,
    NULL::TEXT AS title,
    NULL::TEXT AS content,
    NULL::TEXT AS status,
    NULL::INTEGER AS view_count,
    NULL::BOOLEAN AS ai_enabled,
    NULL::TEXT AS author_name,
    NULL::TEXT AS author_avatar
FROM article_collections c
WHERE c.help_center_id = $1 AND ($2 = '' OR c.locale = $2)

UNION ALL

SELECT
    'article' AS type,
    a.id,
    a.created_at,
    a.updated_at,
    c.help_center_id,
    a.slug,
    NULL::INTEGER AS parent_id,
    a.locale,
    a.title AS name,
    NULL::TEXT AS description,
    NULL::TEXT AS icon,
    a.sort_order,
    NULL::BOOLEAN AS is_published,
    a.collection_id,
    a.title,
    '' AS content,
    a.status,
    a.view_count,
    a.ai_enabled,
    NULL::TEXT AS author_name,
    NULL::TEXT AS author_avatar
FROM help_articles a
JOIN article_collections c ON a.collection_id = c.id
WHERE c.help_center_id = $1 AND ($2 = '' OR a.locale = $2)

ORDER BY type DESC, parent_id NULLS FIRST, sort_order, name;

-- name: get-public-tree-data
WITH RECURSIVE published_collections AS (
    SELECT id FROM article_collections
    WHERE help_center_id = $1 AND parent_id IS NULL AND is_published = true AND ($2 = '' OR locale = $2)
    UNION
    SELECT c.id FROM article_collections c
    JOIN published_collections p ON c.parent_id = p.id
    WHERE c.is_published = true AND ($2 = '' OR c.locale = $2)
)
SELECT
    'collection' AS type,
    c.id,
    c.created_at,
    c.updated_at,
    c.help_center_id,
    c.slug,
    c.parent_id,
    c.locale,
    c.name,
    c.description,
    c.icon,
    c.sort_order,
    c.is_published,
    NULL::INTEGER AS collection_id,
    NULL::TEXT AS title,
    NULL::TEXT AS content,
    NULL::TEXT AS status,
    NULL::INTEGER AS view_count,
    NULL::BOOLEAN AS ai_enabled,
    NULL::TEXT AS author_name,
    NULL::TEXT AS author_avatar
FROM article_collections c
WHERE c.id IN (SELECT id FROM published_collections)

UNION ALL

SELECT
    'article' AS type,
    a.id,
    a.created_at,
    a.updated_at,
    c.help_center_id,
    a.slug,
    NULL::INTEGER AS parent_id,
    a.locale,
    a.title AS name,
    NULL::TEXT AS description,
    NULL::TEXT AS icon,
    a.sort_order,
    NULL::BOOLEAN AS is_published,
    a.collection_id,
    a.title,
    '' AS content,
    a.status,
    a.view_count,
    a.ai_enabled,
    TRIM(u.first_name || ' ' || COALESCE(u.last_name, '')) AS author_name,
    u.avatar_url AS author_avatar
FROM help_articles a
JOIN article_collections c ON a.collection_id = c.id AND c.locale = a.locale
LEFT JOIN users u ON u.id = a.author_id
WHERE c.id IN (SELECT id FROM published_collections) AND a.status = 'published'
    AND ($2 = '' OR a.locale = $2)

ORDER BY type DESC, parent_id NULLS FIRST, sort_order, name;

-- name: get-published-article-by-slug
WITH RECURSIVE published_collections AS (
    SELECT c.id FROM article_collections c
    JOIN help_centers h ON h.id = c.help_center_id
    WHERE h.slug = $1 AND c.parent_id IS NULL AND c.is_published = true
    UNION
    SELECT c.id FROM article_collections c
    JOIN published_collections p ON c.parent_id = p.id
    WHERE c.is_published = true
)
SELECT a.id, a.created_at, a.updated_at, a.collection_id, a.author_id, a.slug, a.locale, a.title, a.content,
    a.excerpt, a.meta_title, a.meta_description, a.meta_image_url, a.sort_order, a.status, a.view_count, a.ai_enabled,
    TRIM(u.first_name || ' ' || COALESCE(u.last_name, '')) AS author_name,
    u.avatar_url AS author_avatar
FROM help_articles a
JOIN article_collections c ON c.id = a.collection_id AND c.locale = a.locale AND c.id IN (SELECT id FROM published_collections)
LEFT JOIN users u ON u.id = a.author_id
WHERE a.slug = $2 AND a.status = 'published' AND ($3 = '' OR a.locale = $3)
ORDER BY c.sort_order, a.sort_order, a.id
LIMIT 1;

-- name: get-published-article-locales
WITH RECURSIVE published_collections AS (
    SELECT c.id FROM article_collections c
    JOIN help_centers h ON h.id = c.help_center_id
    WHERE h.slug = $1 AND c.parent_id IS NULL AND c.is_published = true
    UNION
    SELECT c.id FROM article_collections c
    JOIN published_collections p ON c.parent_id = p.id
    WHERE c.is_published = true
)
SELECT DISTINCT a.locale
FROM help_articles a
JOIN article_collections c ON c.id = a.collection_id AND c.locale = a.locale AND c.id IN (SELECT id FROM published_collections)
WHERE a.slug = $2 AND a.status = 'published';

-- name: get-published-collection-locales
WITH RECURSIVE published_collections AS (
    SELECT c.id FROM article_collections c
    JOIN help_centers h ON h.id = c.help_center_id
    WHERE h.slug = $1 AND c.parent_id IS NULL AND c.is_published = true
    UNION
    SELECT c.id FROM article_collections c
    JOIN published_collections p ON c.parent_id = p.id
    WHERE c.is_published = true
)
SELECT DISTINCT c.locale
FROM article_collections c
WHERE c.slug = $2 AND c.id IN (SELECT id FROM published_collections);

-- name: get-published-articles
WITH RECURSIVE published_collections AS (
    SELECT c.id FROM article_collections c
    JOIN help_centers h ON h.id = c.help_center_id
    WHERE h.slug = $1 AND c.parent_id IS NULL AND c.is_published = true
    UNION
    SELECT c.id FROM article_collections c
    JOIN published_collections p ON c.parent_id = p.id
    WHERE c.is_published = true
)
SELECT a.id, a.created_at, a.updated_at, a.collection_id, a.slug, a.locale, a.title, a.excerpt, '' AS content, a.sort_order, a.status, a.view_count, a.ai_enabled
FROM help_articles a
JOIN article_collections c ON c.id = a.collection_id AND c.locale = a.locale AND c.id IN (SELECT id FROM published_collections)
WHERE a.status = 'published' AND ($2 = '' OR a.locale = $2)
ORDER BY a.view_count DESC, a.created_at DESC
LIMIT $3;

-- name: get-published-articles-by-collection
SELECT a.id, a.created_at, a.updated_at, a.collection_id, a.slug, a.locale, a.title, a.excerpt, '' AS content, a.sort_order, a.status, a.view_count, a.ai_enabled
FROM help_articles a
JOIN article_collections c ON c.id = a.collection_id AND c.locale = a.locale
WHERE a.collection_id = $1 AND a.id != $2 AND a.status = 'published' AND ($3 = '' OR a.locale = $3)
ORDER BY a.sort_order ASC, a.created_at DESC
LIMIT $4;

-- name: search-published-articles
WITH RECURSIVE published_collections AS (
    SELECT c.id FROM article_collections c
    JOIN help_centers h ON h.id = c.help_center_id
    WHERE h.slug = $1 AND c.parent_id IS NULL AND c.is_published = true
    UNION
    SELECT c.id FROM article_collections c
    JOIN published_collections p ON c.parent_id = p.id
    WHERE c.is_published = true
)
SELECT a.id, a.created_at, a.updated_at, a.collection_id, a.slug, a.locale, a.title,
    a.excerpt AS content,
    a.sort_order, a.status, a.view_count, a.ai_enabled
FROM help_articles a
JOIN article_collections c ON c.id = a.collection_id AND c.locale = a.locale AND c.id IN (SELECT id FROM published_collections)
WHERE a.status = 'published' AND a.locale = $4
    AND (a.search_tsv @@ to_tsquery(help_article_search_config($4), NULLIF($5, ''))
        OR a.title ILIKE '%' || $2 || '%' OR a.content ILIKE '%' || $2 || '%')
ORDER BY ts_rank(a.search_tsv, to_tsquery(help_article_search_config($4), NULLIF($5, '')), 2) DESC, a.view_count DESC, a.created_at DESC
LIMIT $3;

-- name: increment-article-view-count
UPDATE help_articles
SET view_count = view_count + 1
WHERE id = $1;

-- name: increment-published-article-view-count
WITH RECURSIVE published_collections AS (
    SELECT c.id FROM article_collections c
    JOIN help_centers h ON h.id = c.help_center_id
    WHERE h.slug = $1 AND c.parent_id IS NULL AND c.is_published = true
    UNION
    SELECT c.id FROM article_collections c
    JOIN published_collections p ON c.parent_id = p.id
    WHERE c.is_published = true
)
UPDATE help_articles
SET view_count = view_count + 1
WHERE id = (
    SELECT a.id
    FROM help_articles a
    JOIN article_collections c ON c.id = a.collection_id AND c.locale = a.locale AND c.id IN (SELECT id FROM published_collections)
    WHERE a.slug = $2 AND a.status = 'published' AND ($3 = '' OR a.locale = $3)
    ORDER BY c.sort_order, a.sort_order, a.id
    LIMIT 1
);

-- name: insert-article-feedback
INSERT INTO help_article_feedback (article_id, is_helpful)
SELECT $1, $2
WHERE EXISTS (SELECT 1 FROM help_articles WHERE id = $1 AND status = 'published');

-- name: insert-search-query
INSERT INTO help_search_queries (help_center_id, query, results_count)
VALUES ($1, $2, $3);

-- name: get-top-search-terms
SELECT LOWER(query) AS query, COUNT(*) AS count,
    COUNT(*) FILTER (WHERE results_count = 0) AS no_results,
    MAX(created_at)::text AS last_search
FROM help_search_queries
WHERE help_center_id = $1 AND created_at >= NOW() - ($3 * INTERVAL '1 day')
GROUP BY LOWER(query)
ORDER BY count DESC, last_search DESC
LIMIT $2;

-- name: get-no-result-search-terms
SELECT LOWER(query) AS query, COUNT(*) AS count,
    COUNT(*) AS no_results,
    MAX(created_at)::text AS last_search
FROM help_search_queries
WHERE help_center_id = $1 AND results_count = 0 AND created_at >= NOW() - ($3 * INTERVAL '1 day')
GROUP BY LOWER(query)
ORDER BY count DESC, last_search DESC
LIMIT $2;

-- name: delete-stale-search-queries
DELETE FROM help_search_queries
WHERE created_at < NOW() - ($1 * INTERVAL '1 day');
