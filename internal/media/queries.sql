-- name: insert-media
INSERT INTO media (store, filename, content_type, size, meta, model_id, model_type, disposition, content_id, uuid)
VALUES(
  $1, 
  $2, 
  $3, 
  $4, 
  $5, 
  NULLIF($6, 0),
  NULLIF($7, ''),
  $8,
  $9,
  $10
)
RETURNING id;

-- name: get-media
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta
FROM media
WHERE
   ($1 > 0 AND id = $1)
   OR
   ($2 != '' AND uuid = NULLIF($2, '')::uuid)

-- name: get-media-by-uuid
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta
FROM media
WHERE uuid = $1;

-- name: delete-media
DELETE FROM media
WHERE uuid = $1;

-- name: link-message-media
UPDATE media
SET model_type = 'messages',
    model_id = $1,
    content_id = CASE
        WHEN uuid = ANY($3::uuid[]) THEN COALESCE(NULLIF(content_id, ''), 'ldsk-' || uuid::TEXT)
        ELSE content_id
    END
WHERE (id = ANY($2::INT[]) OR uuid = ANY($3::uuid[]))
  AND COALESCE(model_type, 'messages') = 'messages'
  AND COALESCE(model_id, 0) = 0;

-- name: get-model-media
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta
FROM media
WHERE model_type = $1
    AND model_id = $2;

-- name: get-unlinked-message-media
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta
FROM media
WHERE model_type = 'messages' 
  AND (model_id IS NULL OR model_id = 0) 
  AND created_at < NOW() - INTERVAL '7 days';

-- name: content-id-exists
SELECT m.uuid
FROM media m
INNER JOIN conversation_messages cm ON cm.id = m.model_id
WHERE m.model_type = 'messages'
  AND m.content_id = $1
  AND cm.conversation_id = (SELECT id FROM conversations WHERE uuid = $2::uuid LIMIT 1);

-- name: get-media-by-content-ids
SELECT m.id, m.created_at, m.updated_at, m."uuid", m.store, m.filename, m.content_type, m.content_id, m.model_id, m.model_type, m.disposition, m."size", m.meta
FROM media m
INNER JOIN conversation_messages cm ON cm.id = m.model_id
WHERE m.model_type = 'messages'
  AND m.content_id = ANY($1)
  AND cm.conversation_id = (SELECT id FROM conversations WHERE uuid = $2::uuid LIMIT 1);

-- name: get-draft-inline-media
SELECT m.id, m.created_at, m.updated_at, m."uuid", m.store, m.filename, m.content_type, m.content_id, m.model_id, m.model_type, m.disposition, m."size", m.meta
FROM media m
LEFT JOIN conversation_messages cm ON cm.id = m.model_id AND m.model_type = 'messages'
WHERE m.uuid = $1
  AND (COALESCE(m.model_id, 0) = 0 OR cm.conversation_id = $2);
