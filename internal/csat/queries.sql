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

-- name: update
UPDATE csat_responses
SET rating = CASE WHEN response_timestamp IS NOT NULL THEN rating ELSE $2 END,
    feedback = $3,
    meta = (COALESCE($4::jsonb, '{}') - 'feedback_pending'),
    response_timestamp = COALESCE(response_timestamp, NOW()),
    updated_at = NOW()
WHERE uuid = $1 AND (response_timestamp IS NULL
    OR (meta->>'feedback_pending' = 'true' AND length(trim($3)) > 0));
