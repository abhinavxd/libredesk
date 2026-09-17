-- name: search-conversations-by-reference-number
SELECT
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    cs.name AS status
FROM conversations
LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
WHERE reference_number::text = $1
  AND $3
  AND (
       $4
    OR ($5 AND conversations.assigned_user_id = $2)
    OR ($6 AND conversations.assigned_team_id = ANY($9::int[]))
    OR ($7 AND conversations.assigned_team_id = ANY($9::int[]) AND conversations.assigned_user_id IS NULL)
    OR ($8 AND conversations.assigned_user_id IS NULL AND conversations.assigned_team_id IS NULL)
  );

-- name: search-conversations-by-contact-email
SELECT
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    cs.name AS status
FROM conversations
JOIN users ON conversations.contact_id = users.id
LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
WHERE users.email ILIKE '%' || $1 || '%'
  AND $3
  AND (
       $4
    OR ($5 AND conversations.assigned_user_id = $2)
    OR ($6 AND conversations.assigned_team_id = ANY($9::int[]))
    OR ($7 AND conversations.assigned_team_id = ANY($9::int[]) AND conversations.assigned_user_id IS NULL)
    OR ($8 AND conversations.assigned_user_id IS NULL AND conversations.assigned_team_id IS NULL)
  )
ORDER BY conversations.created_at DESC
LIMIT $10;

-- name: search-messages
SELECT
    c.created_at as "conversation_created_at",
    c.reference_number as "conversation_reference_number",
    c.uuid as "conversation_uuid",
    LEFT(m.text_content, 200) AS text_content,
    cs.name as "conversation_status"
FROM conversation_messages m
    JOIN conversations c ON m.conversation_id = c.id
    LEFT JOIN conversation_statuses cs ON c.status_id = cs.id
WHERE m.type != 'activity' and m.text_content ILIKE '%' || $1 || '%'
  AND $3
  AND (
       $4
    OR ($5 AND c.assigned_user_id = $2)
    OR ($6 AND c.assigned_team_id = ANY($9::int[]))
    OR ($7 AND c.assigned_team_id = ANY($9::int[]) AND c.assigned_user_id IS NULL)
    OR ($8 AND c.assigned_user_id IS NULL AND c.assigned_team_id IS NULL)
  )
LIMIT $10;

-- name: search-contacts
SELECT
    id,
    created_at,
    first_name,
    last_name,
    email,
    phone_number,
    phone_number_country_code,
    external_user_id
FROM users
WHERE type = 'contact'
AND deleted_at IS NULL
AND (email ILIKE '%' || $1 || '%' OR phone_number ILIKE '%' || $1 || '%')
LIMIT $2;
