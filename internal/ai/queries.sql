-- name: get-default-provider
SELECT id, name, provider, config, is_default FROM ai_providers where is_default is true;

-- name: get-prompt
SELECT id, created_at, updated_at, key, title, content FROM ai_prompts where key = $1;

-- name: get-prompts
SELECT id, created_at, updated_at, key, title FROM ai_prompts order by title;

-- name: set-openai-key
UPDATE ai_providers 
SET config = jsonb_set(
    COALESCE(config, '{}'::jsonb),
    '{api_key}', 
    to_jsonb($1::text)
) 
WHERE provider = 'openai';
