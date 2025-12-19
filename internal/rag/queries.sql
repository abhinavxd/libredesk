-- name: get-sources
SELECT * FROM rag_sources ORDER BY created_at DESC;

-- name: get-source
SELECT * FROM rag_sources WHERE id = $1;

-- name: get-enabled-sources
SELECT * FROM rag_sources WHERE enabled = true;

-- name: create-source
INSERT INTO rag_sources (name, source_type, config, enabled)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: update-source
UPDATE rag_sources
SET name = $2, config = $3, enabled = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: delete-source
DELETE FROM rag_sources WHERE id = $1;

-- name: update-source-synced
UPDATE rag_sources SET last_synced_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: get-documents-by-source
SELECT id, created_at, updated_at, source_id, source_ref, title, content, content_hash, metadata
FROM rag_documents WHERE source_id = $1;

-- name: get-document-by-source-ref
SELECT id, created_at, updated_at, source_id, source_ref, title, content, content_hash, metadata
FROM rag_documents WHERE source_id = $1 AND source_ref = $2;

-- name: delete-document
DELETE FROM rag_documents WHERE id = $1;

-- name: delete-documents-by-source
DELETE FROM rag_documents WHERE source_id = $1;
