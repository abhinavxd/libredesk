-- name: insert-media
INSERT INTO media (store, filename, content_type, size, meta, model_id, model_type, disposition, content_id, uuid, private, uploaded_by)
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
  $10,
  $11,
  NULLIF($12, 0)
)
RETURNING id;

-- name: get-media
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta, private, uploaded_by
FROM media
WHERE
   ($1 > 0 AND id = $1)
   OR
   ($2 != '' AND uuid = NULLIF($2, '')::uuid)

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
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta, private, uploaded_by
FROM media
WHERE model_type = $1
    AND model_id = $2;

-- name: get-unlinked-message-media
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta, private, uploaded_by
FROM media
WHERE (model_type = 'messages' OR model_type IS NULL)
  AND NOT EXISTS (SELECT 1 FROM conversation_draft_media dm WHERE dm.media_id = media.id)
  AND NOT EXISTS (SELECT 1 FROM outbound_review_media rm WHERE rm.media_id = media.id)
  AND (
    ((model_id IS NULL OR model_id = 0) AND created_at < NOW() - INTERVAL '7 days')
    OR (model_id > 0 AND created_at < NOW() - INTERVAL '24 hours' AND NOT EXISTS (SELECT 1 FROM conversation_messages cm WHERE cm.id = media.model_id))
  );

-- name: content-id-exists
SELECT m.uuid
FROM media m
INNER JOIN conversation_messages cm ON cm.id = m.model_id
WHERE m.model_type = 'messages'
  AND m.content_id = $1
  AND cm.conversation_id = (SELECT id FROM conversations WHERE uuid = $2::uuid LIMIT 1);

-- name: get-media-by-content-ids
-- UUID aliases let a quoted historic attachment retain its original MIME CID
-- while newer replies refer to the same bytes as cid:ldsk-<uuid>.
SELECT m.id, m.created_at, m.updated_at, m."uuid", m.store, m.filename, m.content_type,
       requested.content_id AS content_id, m.model_id, m.model_type, m.disposition,
       m."size", m.meta, m.private, m.uploaded_by
FROM media m
INNER JOIN conversation_messages cm ON cm.id = m.model_id
JOIN unnest($1::text[]) AS requested(content_id)
    ON m.content_id = requested.content_id OR 'ldsk-' || m.uuid::text = requested.content_id
WHERE m.model_type = 'messages'
  AND cm.conversation_id = (SELECT id FROM conversations WHERE uuid = $2::uuid LIMIT 1);

-- name: get-draft-inline-media
SELECT m.id, m.created_at, m.updated_at, m."uuid", m.store, m.filename, m.content_type, m.content_id, m.model_id, m.model_type, m.disposition, m."size", m.meta, m.private, m.uploaded_by
FROM media m
LEFT JOIN conversation_messages cm ON cm.id = m.model_id AND m.model_type = 'messages'
WHERE m.uuid = $1
  AND m.model_type = 'messages'
  AND ((COALESCE(m.model_id, 0) = 0 AND m.uploaded_by = $3) OR cm.conversation_id = $2);

-- name: get-unlinked-resource-images
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta, private, uploaded_by
FROM media
WHERE model_type = 'resource_images'
 AND NOT EXISTS (SELECT 1 FROM conversation_messages cm WHERE cm.id = media.model_id);

-- name: get-unlinked-resource-avatars
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta, private, uploaded_by
FROM media
WHERE model_type = 'resource_avatars'
 AND NOT EXISTS (SELECT 1 FROM users WHERE users.id = media.model_id);

-- name: get-unlinked-branding-media
-- Logos that were replaced, removed, or uploaded but never saved. The grace
-- period keeps an upload alive while the admin is still on the settings form.
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta, private, uploaded_by
FROM media
WHERE model_type = 'branding'
 AND created_at < NOW() - INTERVAL '24 hours'
 AND NOT EXISTS (
   SELECT 1 FROM settings s
   WHERE s.key = 'app.logo_url' AND (s.value #>> '{}') LIKE '%/uploads/' || media.uuid::text
 );
