-- name: unsnooze-all
UPDATE conversations
SET snoozed_until = NULL, status_id = (SELECT id FROM conversation_statuses WHERE name = 'Open')
WHERE snoozed_until <= NOW();

-- name: insert-conversation
WITH 
status_id AS (
    SELECT id FROM conversation_statuses WHERE name = $2
),
reference_number AS (
    SELECT generate_reference_number($7) AS reference_number
)
INSERT INTO conversations
(contact_id, status_id, inbox_id, last_message, last_message_at, subject, reference_number)
VALUES(
   $1, 
   (SELECT id FROM status_id), 
   $3, 
   $4, 
   $5, 
   CASE 
      WHEN $8 = TRUE THEN CONCAT($6::text, ' - #', (SELECT reference_number FROM reference_number), '')
      ELSE $6::text
   END, 
   (SELECT reference_number FROM reference_number)
)
RETURNING id, uuid;

-- name: get-conversations
SELECT
    COUNT(*) OVER() as total,
    conversations.id,
    conversations.created_at,
    conversations.updated_at,
    conversations.uuid,
    conversations.waiting_since,
    conversations.assignee_last_seen_at,
    users.created_at as "contact.created_at",
    users.updated_at as "contact.updated_at",
    users.first_name as "contact.first_name",
    users.last_name as "contact.last_name",
    users.avatar_url as "contact.avatar_url", 
    inboxes.channel as inbox_channel,
    inboxes.name as inbox_name,
    conversations.sla_policy_id,
    conversations.first_reply_at,
    conversations.last_reply_at,
    conversations.resolved_at,
    conversations.subject,
    COALESCE(
        conversations.meta->'last_message'->>'text_content',
        conversations.last_message
    ) as last_message,
    COALESCE(
        (conversations.meta->'last_message'->>'created_at')::timestamptz,
        conversations.last_message_at
    ) as last_message_at,
    COALESCE(
        conversations.meta->'last_message'->>'sender_type',
        conversations.last_message_sender::TEXT
    ) as last_message_sender,
    conversations.next_sla_deadline_at,
    conversations.priority_id,
    (
        SELECT CASE WHEN COUNT(*) > 9 THEN 10 ELSE COUNT(*) END
        FROM (
            SELECT 1 FROM conversation_messages 
            WHERE conversation_id = conversations.id 
            AND created_at > conversations.assignee_last_seen_at
            LIMIT 10
        ) t
    ) as unread_message_count,
    conversation_statuses.name as status,
    conversation_priorities.name as priority,
    as_latest.first_response_deadline_at,
    as_latest.resolution_deadline_at,
    as_latest.id as applied_sla_id,
    nxt_resp_event.deadline_at AS next_response_deadline_at,
    nxt_resp_event.met_at as next_response_met_at
    FROM conversations
    JOIN users ON contact_id = users.id
    JOIN inboxes ON inbox_id = inboxes.id  
    LEFT JOIN conversation_statuses ON status_id = conversation_statuses.id
    LEFT JOIN conversation_priorities ON priority_id = conversation_priorities.id
    LEFT JOIN LATERAL (
        SELECT id, first_response_deadline_at, resolution_deadline_at
        FROM applied_slas 
        WHERE conversation_id = conversations.id 
        ORDER BY created_at DESC LIMIT 1
    ) as_latest ON true
    LEFT JOIN LATERAL (
        SELECT se.deadline_at, se.met_at
        FROM sla_events se
        WHERE se.applied_sla_id = as_latest.id
        AND se.type = 'next_response'
        ORDER BY se.created_at DESC
        LIMIT 1
    ) nxt_resp_event ON true
WHERE 1=1 %s

-- name: get-conversation
SELECT
   c.id,
   c.created_at,
   c.updated_at,
   c.closed_at,
   c.resolved_at,
   c.inbox_id,
   COALESCE(inb.from, '') as inbox_mail,
   COALESCE(inb.channel::TEXT, '') as inbox_channel,
   c.status_id,
   c.priority_id,
   p.name as priority,
   s.name as status,
   c.uuid,
   c.reference_number,
   c.first_reply_at,
   c.last_reply_at,
   c.waiting_since,
   c.assigned_user_id,
   c.assigned_team_id,
   c.subject,
   c.contact_id,
   c.sla_policy_id,
   c.meta,
   sla.name as sla_policy_name,
   COALESCE(
       (c.meta->'last_message'->>'created_at')::TIMESTAMPTZ,
       c.last_message_at
   ) as last_message_at,
   COALESCE(
       c.meta->'last_message'->>'sender_type',
       c.last_message_sender::TEXT
   ) as last_message_sender,
   COALESCE(
       c.meta->'last_message'->>'text_content',
       c.last_message
   ) as last_message,
   c.custom_attributes,
   (SELECT COALESCE(
       (SELECT json_agg(t.name)
       FROM tags t
       INNER JOIN conversation_tags ct ON ct.tag_id = t.id
       WHERE ct.conversation_id = c.id),
       '[]'::json
   )) AS tags,
   ct.id as "contact.id",
   ct.created_at as "contact.created_at",
   ct.updated_at as "contact.updated_at",
   ct.first_name as "contact.first_name",
   ct.last_name as "contact.last_name", 
   ct.email as "contact.email",
   ct.type as "contact.type",
   ct.availability_status as "contact.availability_status",
   ct.avatar_url as "contact.avatar_url",
   ct.phone_number as "contact.phone_number",
   ct.phone_number_calling_code as "contact.phone_number_calling_code",
   ct.custom_attributes as "contact.custom_attributes",
   ct.avatar_url as "contact.avatar_url",
   ct.enabled as "contact.enabled",
   ct.last_active_at as "contact.last_active_at",
   ct.last_login_at as "contact.last_login_at",
   as_latest.first_response_deadline_at,
   as_latest.resolution_deadline_at,
   as_latest.id as applied_sla_id,
   nxt_resp_event.deadline_at AS next_response_deadline_at,
   nxt_resp_event.met_at as next_response_met_at
FROM conversations c
JOIN users ct ON c.contact_id = ct.id
JOIN inboxes inb ON c.inbox_id = inb.id
LEFT JOIN sla_policies sla ON c.sla_policy_id = sla.id
LEFT JOIN teams at ON at.id = c.assigned_team_id
LEFT JOIN conversation_statuses s ON c.status_id = s.id
LEFT JOIN conversation_priorities p ON c.priority_id = p.id
LEFT JOIN LATERAL (
    SELECT id, first_response_deadline_at, resolution_deadline_at
    FROM applied_slas
    WHERE conversation_id = c.id 
    ORDER BY created_at DESC LIMIT 1
) as_latest ON true
LEFT JOIN LATERAL (
  SELECT se.deadline_at, se.met_at
  FROM sla_events se
  WHERE se.applied_sla_id = as_latest.id
  AND se.type = 'next_response'
  ORDER BY se.created_at DESC
  LIMIT 1
) nxt_resp_event ON true
WHERE 
  ($1 > 0 AND c.id = $1)
  OR 
  ($2::uuid IS NOT NULL AND c.uuid = $2::uuid)


-- name: get-conversations-created-after
SELECT
    c.id,
    c.uuid
FROM conversations c
WHERE c.created_at > $1;

-- name: get-contact-conversations
SELECT
    c.uuid,
    u.first_name AS "contact.first_name",
    u.last_name AS "contact.last_name",
    u.avatar_url AS "contact.avatar_url",
    COALESCE(
        c.meta->'last_message'->>'text_content',
        c.last_message
    ) as last_message,
    COALESCE(
        (c.meta->'last_message'->>'created_at')::timestamptz,
        c.last_message_at
    ) as last_message_at
FROM users u
JOIN conversations c ON c.contact_id = u.id
WHERE c.contact_id = $1
ORDER BY c.created_at DESC
LIMIT 10;

-- name: get-contact-chat-conversations
SELECT
    c.uuid,
    COALESCE(c.meta->'last_chat_message'->>'text_content', '') as last_message,
    COALESCE((c.meta->'last_chat_message'->>'created_at')::timestamptz, NULL) as last_message_at,
    COALESCE(c.meta->'last_chat_message'->'sender'->>'first_name', '') AS last_message_sender_first_name,
    COALESCE(c.meta->'last_chat_message'->'sender'->>'last_name', '') AS last_message_sender_last_name,
    COALESCE(c.meta->'last_chat_message'->'sender'->>'avatar_url', '') AS last_message_sender_avatar_url,
    LEAST(10, COUNT(unread.id)) AS unread_message_count
FROM conversations c
LEFT JOIN conversation_messages unread ON unread.conversation_id = c.id 
    AND unread.created_at > c.contact_last_seen_at
    AND unread.type IN ('incoming', 'outgoing') AND unread.private = false
WHERE c.contact_id = $1
GROUP BY c.id, c.uuid, 
    c.meta->'last_chat_message'->>'text_content', 
    (c.meta->'last_chat_message'->>'created_at')::timestamptz,
    c.meta->'last_chat_message'->'sender'->>'first_name',
    c.meta->'last_chat_message'->'sender'->>'last_name',
    c.meta->'last_chat_message'->'sender'->>'avatar_url'
ORDER BY c.created_at DESC
LIMIT 100;

-- name: get-conversation-uuid
SELECT uuid from conversations where id = $1;

-- name: update-conversation-assigned-user
UPDATE conversations
SET assigned_user_id = $2,
-- Reset assignee_last_seen_at when assigned to a new user.
assignee_last_seen_at = NULL,
updated_at = NOW()
WHERE uuid = $1;

-- name: update-conversation-contact-last-seen
UPDATE conversations
SET contact_last_seen_at = NOW(),
updated_at = NOW()
WHERE uuid = $1;

-- name: update-conversation-assigned-team
UPDATE conversations
SET assigned_team_id = $2,
updated_at = NOW()
WHERE uuid = $1;

-- name: update-conversation-meta
UPDATE conversations
SET meta = COALESCE(meta, '{}'::jsonb) || $2,
    updated_at = NOW()
WHERE uuid = $1;

-- name: update-conversation-status
UPDATE conversations
SET status_id = (SELECT id FROM conversation_statuses WHERE name = $2),
    resolved_at = COALESCE(resolved_at, CASE WHEN $2 IN ('Resolved', 'Closed') THEN NOW() END),
    closed_at = COALESCE(closed_at, CASE WHEN $2 = 'Closed' THEN NOW() END),
    snoozed_until = CASE WHEN $2 = 'Snoozed' THEN $3::timestamptz ELSE snoozed_until END,
    updated_at = NOW()
WHERE uuid = $1;

-- name: get-user-active-conversations-count
SELECT COUNT(*) FROM conversations WHERE status_id IN (SELECT id FROM conversation_statuses WHERE name NOT IN ('Resolved', 'Closed')) and assigned_user_id = $1;

-- name: update-conversation-priority
UPDATE conversations 
SET priority_id = (SELECT id FROM conversation_priorities WHERE name = $2),
    updated_at = NOW()
WHERE uuid = $1;

-- name: update-conversation-assignee-last-seen
UPDATE conversations 
SET assignee_last_seen_at = NOW(),
    updated_at = NOW()
WHERE uuid = $1;

-- name: update-conversation-last-message
UPDATE conversations SET 
last_message = $3,
last_message_sender = $4,
last_message_at = $5,
last_interaction_at = COALESCE($6, last_interaction_at),
meta = COALESCE(meta, '{}'::jsonb) || $7,
updated_at = NOW() 
WHERE CASE 
    WHEN $1 > 0 THEN id = $1
    ELSE uuid = $2
END

-- name: get-conversation-participants
SELECT users.id as id, first_name, last_name, avatar_url 
FROM conversation_participants
INNER JOIN users ON users.id = conversation_participants.user_id
WHERE conversation_id =
(
    SELECT id FROM conversations WHERE uuid = $1
);

-- name: insert-conversation-participant
INSERT INTO conversation_participants
(user_id, conversation_id)
VALUES($1, (SELECT id FROM conversations WHERE uuid = $2));

-- name: get-unassigned-conversations
SELECT
    c.created_at,
    c.updated_at,
    c.uuid,
    c.assigned_team_id,
    inb.channel as inbox_channel,
    inb.name as inbox_name
FROM conversations c
    JOIN inboxes inb ON c.inbox_id = inb.id 
WHERE assigned_user_id IS NULL AND assigned_team_id IS NOT NULL;

-- name: update-conversation-first-reply-at
UPDATE conversations
SET first_reply_at = $2
WHERE first_reply_at IS NULL AND id = $1;

-- name: update-conversation-last-reply-at
UPDATE conversations
SET last_reply_at = $2
WHERE id = $1;

-- name: add-conversation-tags
-- Insert new tags
INSERT INTO conversation_tags (conversation_id, tag_id)
  SELECT c.id, t.id
  FROM conversations c, tags t
  WHERE t.name = ANY($2::text[]) AND c.uuid = $1
  ON CONFLICT (conversation_id, tag_id) DO UPDATE SET tag_id = EXCLUDED.tag_id;

-- name: set-conversation-tags
WITH conversation_id AS (
    SELECT id FROM conversations WHERE uuid = $1
),
-- Insert new tags
inserted AS (
    INSERT INTO conversation_tags (conversation_id, tag_id)
    SELECT conversation_id.id, t.id
    FROM conversation_id, tags t
    WHERE t.name = ANY($2::text[])
    ON CONFLICT (conversation_id, tag_id) DO UPDATE SET tag_id = EXCLUDED.tag_id
)
-- Delete tags that are not in the new list
DELETE FROM conversation_tags
WHERE conversation_id = (SELECT id FROM conversation_id) 
AND tag_id NOT IN (
    SELECT id FROM tags WHERE name = ANY($2::text[])
);

-- name: remove-conversation-tags
-- Delete tags that are not in the new list
DELETE FROM conversation_tags
WHERE conversation_id = (SELECT id FROM conversations WHERE uuid = $1)
AND tag_id IN (
    SELECT id FROM tags WHERE name = ANY($2::text[])
);

-- name: get-conversation-tags
SELECT t.name
FROM conversation_tags ct
JOIN tags t ON ct.tag_id = t.id
WHERE ct.conversation_id = (SELECT id FROM conversations WHERE uuid = $1);

-- name: get-conversation-uuid-from-message-uuid
SELECT c.uuid AS conversation_uuid
FROM conversation_messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE m.uuid = $1;

-- name: unassign-open-conversations
UPDATE conversations
SET assigned_user_id = NULL,
    updated_at = NOW()
WHERE assigned_user_id = $1 AND status_id in (SELECT id FROM conversation_statuses WHERE name NOT IN ('Resolved', 'Closed'));

-- name: update-conversation-custom-attributes
UPDATE conversations
SET custom_attributes = $2,
    updated_at = NOW()
WHERE uuid = $1;

-- name: update-conversation-waiting-since
UPDATE conversations
SET waiting_since = $2,
    updated_at = NOW()
WHERE uuid = $1;

-- name: remove-conversation-assignee
UPDATE conversations
SET 
    assigned_user_id = CASE WHEN $2 = 'user' THEN NULL ELSE assigned_user_id END,
    assigned_team_id = CASE WHEN $2 = 'team' THEN NULL ELSE assigned_team_id END,
    updated_at = NOW()
WHERE uuid = $1;

-- name: re-open-conversation
-- Open conversation if it is not already open and unset the assigned user if they are away and reassigning.
UPDATE conversations
SET 
  status_id = (SELECT id FROM conversation_statuses WHERE name = 'Open'),
  snoozed_until = NULL,
  updated_at = NOW(),
  assigned_user_id = CASE
    WHEN EXISTS (
      SELECT 1 FROM users 
      WHERE users.id = conversations.assigned_user_id 
        AND users.availability_status = 'away_and_reassigning'
    ) THEN NULL
    ELSE assigned_user_id
  END
WHERE 
  uuid = $1
  AND status_id IN (
    SELECT id FROM conversation_statuses WHERE name NOT IN ('Open')
  )

-- name: get-conversation-by-message-id
SELECT
    c.id,
    c.uuid,
    c.assigned_team_id,
    c.assigned_user_id
FROM conversation_messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE m.id = $1;

-- name: delete-conversation
DELETE FROM conversations WHERE uuid = $1;

-- MESSAGE queries.
-- name: get-message-source-ids
SELECT 
    source_id
FROM conversation_messages
WHERE conversation_id = $1
AND type in ('incoming', 'outgoing') and private = false
and source_id > ''
ORDER BY id DESC
LIMIT $2;

-- name: get-outgoing-pending-messages
SELECT
    m.created_at,
    m.id,
    m.uuid,
    m.sender_id,
    m.type,
    m.private,
    m.status,
    m.content,
    m.conversation_id,
    m.content_type,
    m.source_id,
    ARRAY(SELECT jsonb_array_elements_text(m.meta->'cc')) AS cc,
    ARRAY(SELECT jsonb_array_elements_text(m.meta->'bcc')) AS bcc,
    ARRAY(SELECT jsonb_array_elements_text(m.meta->'to')) AS to,
    c.inbox_id,
    c.uuid as conversation_uuid,
    c.subject
FROM conversation_messages m
INNER JOIN conversations c ON c.id = m.conversation_id
WHERE m.status = 'pending' AND m.type = 'outgoing' AND m.private = false
AND NOT(m.id = ANY($1::INT[]))

-- name: get-message
SELECT
    m.id,
    m.created_at,
    m.updated_at,
    m.status,
    m.type,
    m.content,
    m.text_content,
    m.content_type,
    m.conversation_id,
    m.uuid,
    m.private,
    m.sender_type,
    m.sender_id,
    m.meta,
    COALESCE(
        json_agg(
            json_build_object(
                'name', media.filename,
                'content_type', media.content_type,
                'uuid', media.uuid,
                'size', media.size,
                'content_id', media.content_id,
                'disposition', media.disposition
            ) ORDER BY media.filename
        ) FILTER (WHERE media.id IS NOT NULL),
        '[]'::json
    ) AS attachments
FROM conversation_messages m
LEFT JOIN media ON media.model_type = 'messages' AND media.model_id = m.id
WHERE m.uuid = $1
GROUP BY 
    m.id, m.created_at, m.updated_at, m.status, m.type, m.content, m.uuid, m.private, m.sender_type
ORDER BY m.created_at;

-- name: get-messages
SELECT
   COUNT(*) OVER() AS total,
   m.created_at,
   m.updated_at,
   m.status,
   m.type, 
   m.content,
   m.text_content,
   m.uuid,
   m.private,
   m.sender_id,
   m.sender_type,
   m.meta,
   COALESCE(
     (SELECT json_agg(
       json_build_object(
         'name', filename,
         'content_type', content_type, 
         'uuid', uuid,
         'size', size,
         'content_id', content_id,
         'disposition', disposition
       ) ORDER BY filename
     ) FROM media 
     WHERE model_type = 'messages' AND model_id = m.id),
   '[]'::json) AS attachments
FROM conversation_messages m
WHERE m.conversation_id = (
   SELECT id FROM conversations WHERE uuid = $1 LIMIT 1
)
AND m.type = ANY($2)
AND (
    $3::boolean IS NULL OR m.private = $3::boolean
)
ORDER BY m.created_at DESC %s

-- name: insert-message
WITH conversation_id AS (
   SELECT id 
   FROM conversations 
   WHERE CASE 
       WHEN $3 > 0 THEN id = $3 
       ELSE uuid = $4 
   END
),
inserted_msg AS (
   INSERT INTO conversation_messages (
       "type", status, conversation_id, "content", 
       text_content, sender_id, sender_type, private,
       content_type, source_id, meta
   )
   VALUES (
       $1, $2, (SELECT id FROM conversation_id),
       $5, $6, $7, $8, $9, $10, $11, $12
   )
   RETURNING id, uuid, created_at, conversation_id
)
SELECT id, uuid, created_at FROM inserted_msg;

-- name: message-exists-by-source-id
SELECT conversation_id
FROM conversation_messages
WHERE source_id = ANY($1::text []);

-- name: update-message-status
update conversation_messages set status = $1, updated_at = NOW() where uuid = $2;

-- name: get-latest-message
SELECT
    m.created_at,
    m.updated_at,
    m.status,
    m.type, 
    m.content,
    m.uuid,
    m.private,
    m.sender_id,
    m.sender_type,
    m.meta
FROM conversation_messages m
WHERE m.conversation_id = $1
AND m.type = ANY($2)
AND m.status = ANY($3)
AND m.private = NOT $4
ORDER BY m.created_at DESC
LIMIT 1;