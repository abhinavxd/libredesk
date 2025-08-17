-- name: get-prompt
SELECT id, key, title, content FROM ai_prompts where key = $1;

-- name: get-prompts
SELECT id, key, title FROM ai_prompts order by title;

-- name: set-openai-key
UPDATE ai_providers 
SET config = jsonb_set(
    COALESCE(config, '{}'::jsonb),
    '{api_key}', 
    to_jsonb($1::text)
) 
WHERE provider = 'openai';

-- name: get-knowledge-base-items
SELECT id, created_at, updated_at, type, content, enabled
FROM ai_knowledge_base
ORDER BY created_at DESC;

-- name: get-knowledge-base-item
SELECT id, created_at, updated_at, type, content, enabled
FROM ai_knowledge_base
WHERE id = $1;

-- name: insert-knowledge-base-item
INSERT INTO ai_knowledge_base (type, content, enabled)
VALUES ($1, $2, $3)
RETURNING id, created_at, updated_at, type, content, enabled;

-- name: update-knowledge-base-item
UPDATE ai_knowledge_base 
SET type = $2, content = $3, enabled = $4, updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, type, content, enabled;

-- name: delete-knowledge-base-item
DELETE FROM ai_knowledge_base WHERE id = $1;

-- name: insert-embedding
INSERT INTO embeddings (source_type, source_id, chunk_text, embedding, meta)
VALUES ($1, $2, $3, $4::vector, $5)
RETURNING id;

-- name: delete-embeddings-by-source
DELETE FROM embeddings 
WHERE source_type = $1 AND source_id = $2;

-- name: search-knowledge-base
WITH knowledge_results AS (
    SELECT
        kb.id,
        kb.created_at,
        kb.updated_at,
        kb.type,
        kb.content,
        (1 - (e.embedding <=> $1::vector)) as similarity
    FROM ai_knowledge_base kb
    JOIN embeddings e ON e.source_type = 'knowledge_base' AND e.source_id = kb.id
    WHERE kb.enabled = true
    AND (1 - (e.embedding <=> $1::vector)) >= $2
)
SELECT DISTINCT ON (id) 
    id,
    created_at,
    updated_at,
    type,
    content,
    similarity
FROM knowledge_results
ORDER BY id, similarity DESC
LIMIT $3;
