-- name: get-default-provider
SELECT id, name, provider, config, is_default FROM ai_providers where is_default is true;

-- name: get-prompt
SELECT id, created_at, updated_at, key, title, content FROM ai_prompts where key = $1;

-- name: get-prompts
SELECT id, created_at, updated_at, key, title FROM ai_prompts order by title;

-- name: update-provider-config
UPDATE ai_providers SET config = $1::jsonb, updated_at = now() WHERE is_default = true;
