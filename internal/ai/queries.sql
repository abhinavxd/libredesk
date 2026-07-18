-- name: get-provider-by-type
SELECT id, created_at, updated_at, name, provider, type, config, is_default FROM ai_providers WHERE type = $1;

-- name: update-provider-config
UPDATE ai_providers SET config = $2, updated_at = now() WHERE type = $1;

-- name: get-prompt
SELECT id, created_at, updated_at, key, title, content FROM ai_prompts WHERE key = $1;

-- name: get-prompts
SELECT id, created_at, updated_at, key, title FROM ai_prompts ORDER BY title;

-- name: get-knowledge-base-items
SELECT id, created_at, updated_at, type, title, content, enabled, source, source_url, embedded_fingerprint FROM ai_knowledge_base ORDER BY updated_at DESC;

-- name: get-knowledge-base-item
SELECT id, created_at, updated_at, type, title, content, enabled, source, source_url, embedded_fingerprint FROM ai_knowledge_base WHERE id = $1;

-- name: knowledge-base-item-exists
SELECT EXISTS(SELECT 1 FROM ai_knowledge_base WHERE id = $1);

-- name: insert-knowledge-base-item
INSERT INTO ai_knowledge_base (type, title, content, enabled, source, source_url) VALUES ($1, $2, $3, $4, $5, $6) RETURNING *;

-- name: update-knowledge-base-item
UPDATE ai_knowledge_base SET title = $2, content = $3, enabled = $4, updated_at = now() WHERE id = $1 RETURNING *;

-- name: delete-knowledge-base-item
DELETE FROM ai_knowledge_base WHERE id = $1;

-- name: set-knowledge-base-embedded-fingerprint
UPDATE ai_knowledge_base SET embedded_fingerprint = $2 WHERE id = $1;

-- name: insert-embedding
INSERT INTO embeddings (source_type, source_id, chunk_text, embedding, dimensions) VALUES ($1, $2, $3, $4, $5);

-- name: delete-embeddings-by-source
DELETE FROM embeddings WHERE source_type = $1 AND source_id = $2;

-- name: get-all-embeddings
SELECT id, source_type, source_id, chunk_text, embedding, dimensions FROM embeddings;

-- name: get-tools
SELECT id, created_at, updated_at, name, description, url, method, auth, parameters, enabled FROM ai_tools ORDER BY updated_at DESC;

-- name: get-enabled-tools
SELECT id, created_at, updated_at, name, description, url, method, auth, parameters, enabled FROM ai_tools WHERE enabled = true;

-- name: get-tool
SELECT id, created_at, updated_at, name, description, url, method, auth, parameters, enabled FROM ai_tools WHERE id = $1;

-- name: get-tool-auth
SELECT auth FROM ai_tools WHERE id = $1;

-- name: insert-tool
INSERT INTO ai_tools (name, description, url, method, auth, parameters, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: update-tool
UPDATE ai_tools SET name = $2, description = $3, url = $4, method = $5, auth = $6, parameters = $7, enabled = $8, updated_at = now()
WHERE id = $1 RETURNING *;

-- name: delete-tool
DELETE FROM ai_tools WHERE id = $1;

-- name: get-copilot-messages
SELECT role, content FROM copilot_messages WHERE conversation_id = $1 AND user_id = $2 ORDER BY id;

-- name: insert-copilot-message
INSERT INTO copilot_messages (conversation_id, user_id, role, content) VALUES ($1, $2, $3, $4);

-- name: delete-copilot-messages
DELETE FROM copilot_messages WHERE conversation_id = $1 AND user_id = $2;
