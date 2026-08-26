-- name: insert
INSERT INTO csat_responses (conversation_id)
SELECT $1
WHERE NOT EXISTS (SELECT 1 FROM csat_responses WHERE conversation_id = $1)
RETURNING uuid;

-- name: get
SELECT id,
    uuid,
    created_at,
    updated_at,
    conversation_id,
    rating,
    feedback,
    meta,
    response_timestamp
FROM csat_responses
WHERE uuid = $1;

-- name: get-by-conversation
SELECT id,
    uuid,
    created_at,
    updated_at,
    conversation_id,
    rating,
    feedback,
    meta,
    response_timestamp
FROM csat_responses
WHERE conversation_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: update
UPDATE csat_responses
SET rating = $2,
    feedback = $3,
    meta = COALESCE($4::jsonb, '{}'),
    response_timestamp = NOW()
WHERE uuid = $1;
