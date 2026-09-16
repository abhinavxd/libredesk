-- name: search-conversations
SELECT
    COUNT(*) OVER() AS total,
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    conversations.last_message,
    conversations.last_message_at,
    conversations.assigned_user_id,
    conversations.assigned_team_id,
    assignee.first_name AS "assignee.first_name",
    assignee.last_name AS "assignee.last_name",
    assignee.avatar_url AS "assignee.avatar_url",
    teams.name AS team_name,
    cs.name AS status,
    cp.name AS priority,
    inboxes.name AS inbox_name,
    inboxes.channel AS inbox_channel,
    users.first_name AS "contact.first_name",
    users.last_name AS "contact.last_name",
    users.email AS "contact.email",
    users.avatar_url AS "contact.avatar_url",
    COALESCE(
        (SELECT ARRAY_AGG(tags.name ORDER BY tags.name) FROM conversation_tags JOIN tags ON tags.id = conversation_tags.tag_id WHERE conversation_tags.conversation_id = conversations.id),
        '{}'
    ) AS tags
FROM conversations
JOIN users ON conversations.contact_id = users.id
LEFT JOIN users assignee ON conversations.assigned_user_id = assignee.id
LEFT JOIN teams ON conversations.assigned_team_id = teams.id
LEFT JOIN inboxes ON conversations.inbox_id = inboxes.id
LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
LEFT JOIN conversation_priorities cp ON conversations.priority_id = cp.id
WHERE (
       conversations.reference_number::text = $1
    OR conversations.subject ILIKE '%' || $1 || '%'
    OR users.email ILIKE '%' || $1 || '%'
    OR CONCAT_WS(' ', users.first_name, users.last_name) ILIKE '%' || $1 || '%'
  )
  AND $3
  AND (
       $4
    OR ($5 AND conversations.assigned_user_id = $2)
    OR ($6 AND conversations.assigned_team_id = ANY($9::int[]))
    OR ($7 AND conversations.assigned_team_id = ANY($9::int[]) AND conversations.assigned_user_id IS NULL)
    OR ($8 AND conversations.assigned_user_id IS NULL AND conversations.assigned_team_id IS NULL)
  )

-- name: search-messages
SELECT
    COUNT(*) OVER() AS total,
    conversation_messages.uuid,
    conversation_messages.created_at,
    conversation_messages.type,
    conversation_messages.private,
    CASE WHEN POSITION(LOWER($1) IN LOWER(conversation_messages.text_content)) > 81 THEN '…' ELSE '' END
        || SUBSTRING(
            conversation_messages.text_content
            FROM GREATEST(POSITION(LOWER($1) IN LOWER(conversation_messages.text_content)) - 80, 1)
            FOR 300
        ) AS snippet,
    sender.first_name AS "sender.first_name",
    sender.last_name AS "sender.last_name",
    conversations.created_at AS conversation_created_at,
    conversations.uuid AS conversation_uuid,
    conversations.reference_number AS conversation_reference_number,
    conversations.subject AS conversation_subject,
    conversations.assigned_user_id,
    conversations.assigned_team_id,
    assignee.first_name AS "assignee.first_name",
    assignee.last_name AS "assignee.last_name",
    assignee.avatar_url AS "assignee.avatar_url",
    teams.name AS team_name,
    cs.name AS conversation_status,
    cp.name AS priority,
    inboxes.name AS inbox_name,
    inboxes.channel AS inbox_channel,
    users.first_name AS "contact.first_name",
    users.last_name AS "contact.last_name",
    users.email AS "contact.email",
    users.avatar_url AS "contact.avatar_url"
FROM conversation_messages
JOIN conversations ON conversation_messages.conversation_id = conversations.id
JOIN users ON conversations.contact_id = users.id
LEFT JOIN users sender ON conversation_messages.sender_id = sender.id
LEFT JOIN users assignee ON conversations.assigned_user_id = assignee.id
LEFT JOIN teams ON conversations.assigned_team_id = teams.id
LEFT JOIN inboxes ON conversations.inbox_id = inboxes.id
LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
LEFT JOIN conversation_priorities cp ON conversations.priority_id = cp.id
WHERE conversation_messages.type != 'activity'
  AND conversation_messages.text_content ILIKE '%' || $1 || '%'
  AND $3
  AND (
       $4
    OR ($5 AND conversations.assigned_user_id = $2)
    OR ($6 AND conversations.assigned_team_id = ANY($9::int[]))
    OR ($7 AND conversations.assigned_team_id = ANY($9::int[]) AND conversations.assigned_user_id IS NULL)
    OR ($8 AND conversations.assigned_user_id IS NULL AND conversations.assigned_team_id IS NULL)
  )

-- name: search-contacts
SELECT
    id,
    created_at,
    first_name,
    last_name,
    email,
    external_user_id
FROM users
WHERE type = 'contact'
AND deleted_at IS NULL
AND email ILIKE '%' || $1 || '%'
LIMIT $2;
