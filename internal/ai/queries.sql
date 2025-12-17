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
),
updated_at = NOW()
WHERE provider = 'openai';

-- name: get-providers
SELECT id, name, provider, config, is_default FROM ai_providers ORDER BY name;

-- name: set-default-provider
UPDATE ai_providers SET is_default = (provider = $1), updated_at = NOW();

-- name: upsert-openrouter
INSERT INTO ai_providers (name, provider, config, is_default)
VALUES ('OpenRouter', 'openrouter', '{"api_key": "", "model": "anthropic/claude-3-haiku"}'::jsonb, false)
ON CONFLICT (name) DO NOTHING;

-- name: set-openrouter-config
UPDATE ai_providers 
SET config = jsonb_build_object(
    'api_key', CASE WHEN $1::text = '' THEN config->>'api_key' ELSE $1::text END,
    'model', $2::text
),
    updated_at = NOW()
WHERE provider = 'openrouter';
