-- name: get-default-provider
SELECT id, name, provider, config, is_default FROM ai_providers where is_default is true;

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

-- name: get-ai-custom-answers
SELECT id, created_at, updated_at, question, answer, enabled
FROM ai_custom_answers
ORDER BY created_at DESC;

-- name: get-ai-custom-answer
SELECT id, created_at, updated_at, question, answer, enabled
FROM ai_custom_answers
WHERE id = $1;

-- name: insert-ai-custom-answer
INSERT INTO ai_custom_answers (question, answer, embedding, enabled)
VALUES ($1, $2, $3::vector, $4)
RETURNING id, created_at, updated_at, question, answer, enabled;

-- name: update-ai-custom-answer
UPDATE ai_custom_answers 
SET question = $2, answer = $3, embedding = $4::vector, enabled = $5, updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, question, answer, enabled;

-- name: delete-ai-custom-answer
DELETE FROM ai_custom_answers WHERE id = $1;

-- name: search-custom-answers
SELECT
    id,
    created_at,
    updated_at,
    question,
    answer,
    1 - (embedding <=> $1::vector) AS similarity
FROM ai_custom_answers
WHERE enabled = true
  AND embedding IS NOT NULL
  AND 1 - (embedding <=> $1::vector) >= $2  -- confidence threshold
ORDER BY embedding <=> $1::vector
LIMIT 1;
