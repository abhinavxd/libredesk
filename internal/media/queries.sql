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
   ($2 != '' AND uuid = $2::uuid)

-- name: get-media-by-uuid
SELECT id, created_at, updated_at, "uuid", store, filename, content_type, content_id, model_id, model_type, disposition, "size", meta
FROM media
WHERE uuid = $1;

-- name: delete-media
DELETE FROM media
WHERE uuid = $1;

-- name: attach-to-model
UPDATE media
SET model_type = $2,
    model_id = $3
WHERE id = $1;

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
  AND created_at < NOW() - INTERVAL '1 day';

-- name: content-id-exists
SELECT uuid FROM media WHERE content_id = $1;