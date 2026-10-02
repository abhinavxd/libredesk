-- name: unsnooze-all
UPDATE conversations
SET snoozed_until = NULL, status_id = (SELECT id FROM conversation_statuses WHERE name = 'Open')
WHERE snoozed_until <= NOW()
  AND status_id = (SELECT id FROM conversation_statuses WHERE name = 'Snoozed');

-- name: insert-conversation
-- $11 = rate limit window start (timestamptz), $12 = max conversations (0 = unlimited)
-- $13 = subject reference marker template (placeholder: {ref})
-- $14 = user-facing email address ID (nullable for non-email channels)
WITH
status_id AS (
    SELECT id FROM conversation_statuses WHERE name = $2
),
reference_number AS (
    SELECT generate_reference_number($7) AS reference_number
)
INSERT INTO conversations
(contact_id, status_id, inbox_id, address_id, last_message, last_message_at, subject, reference_number, meta, custom_attributes)
SELECT
   $1,
   (SELECT id FROM status_id),
   $3,
   $14,
   $4,
   $5,
   CASE
      WHEN $8 = TRUE THEN CONCAT($6::text, ' - ', REPLACE($13::text, '{ref}', (SELECT reference_number FROM reference_number)))
      ELSE $6::text
   END,
   (SELECT reference_number FROM reference_number),
   $9,
   $10
WHERE $12::int = 0 OR (SELECT COUNT(*) FROM conversations WHERE contact_id = $1 AND created_at >= $11) < $12::int
RETURNING id, uuid;

-- name: get-conversations
-- $1 = viewing user ID for per-agent unread count
-- $2 = include mentioned message UUID (true for mentioned inbox, false otherwise)
SELECT
    COUNT(*) OVER() as total,
    conversations.id,
    conversations.inbox_id,
    conversations.address_id,
    conversations.created_at,
    conversations.updated_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.waiting_since,
    users.created_at as "contact.created_at",
    users.updated_at as "contact.updated_at",
    users.first_name as "contact.first_name",
    COALESCE(users.last_name, '') as "contact.last_name",
    users.email as "contact.email",
    users.avatar_url as "contact.avatar_url",
    inboxes.channel as inbox_channel,
    inboxes.name as inbox_name,
    conversations.first_reply_at,
    conversations.last_reply_at,
    conversations.resolved_at,
    conversations.subject,
    conversations.last_message,
    conversations.last_message_at,
    conversations.last_message_sender,
    conversations.last_interaction,
    conversations.last_interaction_at,
    conversations.last_interaction_sender,
    conversations.assigned_user_id,
    conversations.assigned_team_id,
    (
    SELECT CASE WHEN COUNT(*) > 9 THEN 10 ELSE COUNT(*) END
    FROM (
        SELECT 1 FROM conversation_messages
        WHERE conversation_id = conversations.id
        AND created_at > COALESCE(
            (SELECT last_seen_at FROM conversation_last_seen
             WHERE conversation_id = conversations.id AND user_id = $1),
            '1970-01-01'::TIMESTAMPTZ
        )
        AND (meta IS NULL OR NOT COALESCE((meta->>'continuity_email')::boolean, false))
        LIMIT 10
    ) t
    ) as unread_message_count,
    conversation_statuses.name as status,
    CASE WHEN $2 = true THEN (
        SELECT msg.uuid
        FROM conversation_mentions cm2
        JOIN conversation_messages msg ON msg.id = cm2.message_id
        WHERE cm2.conversation_id = conversations.id
          AND (cm2.mentioned_user_id = $1 OR EXISTS(
              SELECT 1 FROM team_members tm2
              WHERE tm2.team_id = cm2.mentioned_team_id AND tm2.user_id = $1
          ))
        ORDER BY cm2.created_at DESC
        LIMIT 1
    ) ELSE NULL END as mentioned_message_uuid
    FROM conversations
    JOIN users ON contact_id = users.id
    JOIN inboxes ON inbox_id = inboxes.id
    LEFT JOIN conversation_statuses ON status_id = conversation_statuses.id
WHERE inboxes.channel = 'email' %s

-- name: get-conversation-list-item
SELECT
    conversations.id,
    conversations.inbox_id,
    conversations.address_id,
    conversations.created_at,
    conversations.updated_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.waiting_since,
    users.created_at as "contact.created_at",
    users.updated_at as "contact.updated_at",
    users.first_name as "contact.first_name",
    COALESCE(users.last_name, '') as "contact.last_name",
    users.email as "contact.email",
    users.avatar_url as "contact.avatar_url",
    inboxes.channel as inbox_channel,
    inboxes.name as inbox_name,
    conversations.first_reply_at,
    conversations.last_reply_at,
    conversations.resolved_at,
    conversations.subject,
    conversations.last_message,
    conversations.last_message_at,
    conversations.last_message_sender,
    conversations.last_interaction,
    conversations.last_interaction_at,
    conversations.last_interaction_sender,
    conversations.assigned_user_id,
    conversations.assigned_team_id,
    conversation_statuses.name as status
FROM conversations
JOIN users ON contact_id = users.id
JOIN inboxes ON inbox_id = inboxes.id
LEFT JOIN conversation_statuses ON status_id = conversation_statuses.id
WHERE conversations.uuid = $1::uuid AND inboxes.channel = 'email';

-- name: get-conversation
SELECT
   c.id,
   c.created_at,
   c.updated_at,
   c.closed_at,
   c.resolved_at,
   c.contact_last_seen_at,
   c.inbox_id,
   c.address_id,
   inb.name as inbox_name,
   COALESCE(ea.address, c.meta->>'email_alias', inb.from, '') as inbox_mail,
   COALESCE(inb.config->>'reply_to', '') as inbox_reply_to,
   COALESCE(inb.channel::TEXT, '') as inbox_channel,
   c.status_id,
   s.name as status,
   s.category as status_category,
   c.uuid,
   c.reference_number,
   c.first_reply_at,
   c.last_reply_at,
   c.waiting_since,
   c.snoozed_until,
   c.assigned_user_id,
   c.assigned_team_id,
   c.subject,
   c.contact_id,
   c.meta,
   c.last_message_at,
   c.last_message_sender,
   c.last_message,
   c.last_interaction,
   c.last_interaction_at,
   c.last_interaction_sender,
   c.custom_attributes,
   COALESCE(latest_incoming.recipient, '') AS latest_incoming_recipient,
   ct.id as "contact.id",
   ct.first_name as "contact.first_name",
   COALESCE(ct.last_name, '') as "contact.last_name",
   ct.email as "contact.email",
   ct.type as "contact.type",
   ct.avatar_url as "contact.avatar_url",
   c.last_continuity_email_sent_at
FROM conversations c
JOIN users ct ON c.contact_id = ct.id
JOIN inboxes inb ON c.inbox_id = inb.id
LEFT JOIN email_addresses ea ON ea.id = c.address_id
LEFT JOIN LATERAL (
    SELECT lower(address.value) AS recipient
    FROM conversation_messages cm
    CROSS JOIN LATERAL jsonb_array_elements_text(COALESCE(cm.meta->'to', '[]'::jsonb)) WITH ORDINALITY AS address(value, position)
    WHERE cm.conversation_id = c.id
      AND cm.sender_type = 'contact'
      AND cm.type = 'incoming'
      AND cm.private = FALSE
    ORDER BY cm.created_at DESC, cm.id DESC, address.position
    LIMIT 1
) latest_incoming ON true
LEFT JOIN teams at ON at.id = c.assigned_team_id
LEFT JOIN conversation_statuses s ON c.status_id = s.id
WHERE inb.channel = 'email' AND (
  ($1 > 0 AND c.id = $1)
  OR
  (NULLIF($2, '')::uuid IS NOT NULL AND c.uuid = NULLIF($2, '')::uuid)
  OR
  ($3::TEXT != '' AND c.reference_number = $3::TEXT))

-- name: get-conversations-created-after
SELECT
    c.id,
    c.uuid
FROM conversations c
WHERE c.created_at > $1 AND c.id > $2
ORDER BY c.id
LIMIT $3;

-- name: get-conversation-uuid
SELECT uuid from conversations where id = $1;

-- name: update-conversation-contact-last-seen
UPDATE conversations
SET contact_last_seen_at = NOW(),
updated_at = NOW()
WHERE uuid = $1
RETURNING contact_last_seen_at;

-- name: update-conversation-status
WITH new_status AS (
    SELECT id, category FROM conversation_statuses WHERE name = $2
)
UPDATE conversations
SET status_id     = (SELECT id FROM new_status),
    resolved_at   = COALESCE(resolved_at, CASE WHEN (SELECT category FROM new_status) = 'resolved' THEN NOW() END),
    closed_at     = COALESCE(closed_at,   CASE WHEN $2 = 'Closed'                                  THEN NOW() END),
    snoozed_until = CASE WHEN $2 = 'Snoozed' THEN $3::timestamptz ELSE NULL END,
    updated_at    = NOW()
WHERE uuid = $1;

-- name: upsert-user-last-seen
INSERT INTO conversation_last_seen (user_id, conversation_id, last_seen_at)
VALUES ($1, (SELECT id FROM conversations WHERE uuid = $2), NOW())
ON CONFLICT (conversation_id, user_id)
DO UPDATE SET last_seen_at = NOW(), updated_at = NOW();

-- name: update-conversation-last-message
-- $1=id, $2=uuid, $3=content, $4=sender_type, $5=timestamp, $6=message_type, $7=private, $8=sender_id
UPDATE conversations SET
    last_message = $3,
    last_message_sender = $4,
    last_message_sender_id = $8,
    last_message_at = $5,
    last_interaction = CASE WHEN $6 != 'activity' AND $7 = false THEN $3 ELSE last_interaction END,
    last_interaction_sender = CASE WHEN $6 != 'activity' AND $7 = false THEN $4 ELSE last_interaction_sender END,
    last_interaction_sender_id = CASE WHEN $6 != 'activity' AND $7 = false THEN $8 ELSE last_interaction_sender_id END,
    last_interaction_at = CASE WHEN $6 != 'activity' AND $7 = false THEN $5 ELSE last_interaction_at END,
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
VALUES($1, (SELECT id FROM conversations WHERE uuid = $2))
ON CONFLICT (conversation_id, user_id) DO NOTHING;

-- name: get-conversation-uuid-from-message-uuid
SELECT c.uuid AS conversation_uuid
FROM conversation_messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE m.uuid = $1;

-- name: start-conversation-waiting-since
UPDATE conversations
SET waiting_since = $2,
    updated_at = NOW()
WHERE uuid = $1 AND waiting_since IS NULL;

-- name: update-conversation-reply-timestamps
WITH old AS (
    SELECT first_reply_at IS NULL AS is_first FROM conversations WHERE id = $1
)
UPDATE conversations SET
    first_reply_at = COALESCE(conversations.first_reply_at, $2),
    last_reply_at = $2,
    waiting_since = NULL,
    updated_at = NOW()
FROM old WHERE conversations.id = $1
RETURNING old.is_first AS is_first_reply;

-- name: re-open-conversation
-- Open the mail conversation if it is not already open.
UPDATE conversations
SET
  status_id = (SELECT id FROM conversation_statuses WHERE name = 'Open'),
  snoozed_until = NULL,
  updated_at = NOW()
WHERE
  uuid = $1
  AND status_id IN (
    SELECT id FROM conversation_statuses WHERE name NOT IN ('Open')
  )
RETURNING id;

-- name: get-conversation-by-message-id
SELECT
    c.id,
    c.uuid,
    c.assigned_team_id,
    c.assigned_user_id
FROM conversation_messages m
JOIN conversations c ON m.conversation_id = c.id
JOIN inboxes ON inboxes.id = c.inbox_id AND inboxes.channel = 'email'
WHERE m.id = $1;

-- name: delete-conversation
DELETE FROM conversations WHERE uuid = $1;

-- MESSAGE queries.
-- name: delete-private-message
-- $1 = message uuid, $2 = conversation uuid, $3 = deleted placeholder text, $4 = sender id, 0 to skip the sender check.
WITH deleted AS (
    UPDATE conversation_messages
    SET content = $3, text_content = $3, updated_at = NOW(),
        meta = COALESCE(meta, '{}'::jsonb) || jsonb_build_object('deleted_at', NOW())
    WHERE uuid = $1
      AND private = true
      AND meta->>'deleted_at' IS NULL
      AND ($4 = 0 OR sender_id = $4)
      AND conversation_id = (SELECT id FROM conversations WHERE uuid = $2)
    RETURNING id, conversation_id, created_at
),
media_unlink AS (
    UPDATE media SET model_id = 0
    FROM deleted d
    WHERE media.model_type = 'messages' AND media.model_id = d.id
),
preview AS (
    UPDATE conversations c
    SET last_message = $3, updated_at = NOW()
    FROM deleted d
    WHERE c.id = d.conversation_id
      AND NOT EXISTS (
          SELECT 1 FROM conversation_messages m2
          WHERE m2.conversation_id = d.conversation_id
            AND m2.created_at > d.created_at
      )
    RETURNING c.id
)
SELECT EXISTS (SELECT 1 FROM preview) AS preview_updated
FROM deleted d;

-- name: get-message-source-ids
SELECT
    source_id
FROM conversation_messages
WHERE conversation_id = $1
AND type in ('incoming', 'outgoing') and private = false AND status IN ('received','sent')
and source_id > ''
ORDER BY id DESC
LIMIT $2;

-- name: get-outgoing-pending-messages
SELECT
    m.id,
    m.created_at,
    m.updated_at,
    m.status,
    m.type,
    m.content,
    m.text_content,
    m.sender_type,
    m.content_type,
    m.conversation_id,
    m.uuid,
    m.private,
    m.sender_type,
    m.sender_id,
    m.meta,
    c.uuid as conversation_uuid,
    m.content_type,
    m.source_id,
    m.meta,
    ARRAY(SELECT jsonb_array_elements_text(m.meta->'cc')) AS cc,
    ARRAY(SELECT jsonb_array_elements_text(m.meta->'bcc')) AS bcc,
    ARRAY(SELECT jsonb_array_elements_text(m.meta->'to')) AS to,
    c.inbox_id,
    c.uuid as conversation_uuid,
    c.subject,
    c.contact_id as message_receiver_id,
    c.subject
FROM conversation_messages m
INNER JOIN conversations c ON c.id = m.conversation_id
JOIN inboxes ON inboxes.id = c.inbox_id AND inboxes.channel = 'email'
WHERE m.status = 'pending' AND m.type = 'outgoing' AND m.private = false
AND NOT(m.id = ANY($1::INT[]))
AND NOT EXISTS (SELECT 1 FROM mail_delivery_attempts a WHERE a.message_id=m.id AND a.state IN ('sending','unknown','sent'))
ORDER BY m.id LIMIT 100

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
    c.uuid as conversation_uuid,
    u.id AS "author.id",
    u.first_name AS "author.first_name",
    u.last_name AS "author.last_name",
    u.email AS "author.email",
    u.avatar_url AS "author.avatar_url",
    u.availability_status AS "author.availability_status",
    u.type AS "author.type",
    u.last_active_at AS "author.last_active_at",
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
    )::jsonb || COALESCE(m.meta->'unavailable_attachments', '[]'::jsonb) AS attachments
FROM conversation_messages m
INNER JOIN conversations c ON c.id = m.conversation_id
JOIN users u ON m.sender_id = u.id
LEFT JOIN media ON media.model_type = 'messages' AND media.model_id = m.id
WHERE m.uuid = $1
GROUP BY
    m.id, m.created_at, m.updated_at, m.status, m.type, m.content, m.uuid, m.private, m.sender_type, c.uuid,
    u.id, u.first_name, u.last_name, u.email, u.avatar_url, u.availability_status, u.type, u.last_active_at
ORDER BY m.created_at;

-- name: get-messages
SELECT
   COUNT(*) OVER() AS total,
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
   m.sender_id,
   m.sender_type,
   m.meta,
   $1::uuid AS conversation_uuid,
   u.id AS "author.id",
   u.first_name AS "author.first_name",
   u.last_name AS "author.last_name",
   u.email AS "author.email",
   u.avatar_url AS "author.avatar_url",
   u.availability_status AS "author.availability_status",
   u.type AS "author.type",
   u.last_active_at AS "author.last_active_at",
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
   '[]'::json)::jsonb || COALESCE(m.meta->'unavailable_attachments', '[]'::jsonb) AS attachments
FROM conversation_messages m
JOIN users u ON m.sender_id = u.id
WHERE m.conversation_id = (
   SELECT id FROM conversations WHERE uuid = $1 LIMIT 1
)
AND ($2::boolean IS NULL OR m.private = $2)
AND ($3::text[] IS NULL OR m.type::text = ANY($3))
AND (m.meta IS NULL OR NOT COALESCE((m.meta->>'continuity_email')::boolean, false))
ORDER BY m.created_at DESC %s

-- name: insert-message
WITH conversation_id AS (
   SELECT id
   FROM conversations
   WHERE CASE
       WHEN $3 > 0 THEN id = $3
       ELSE uuid = NULLIF($4,'')::uuid
   END
),
inserted_msg AS (
   INSERT INTO conversation_messages (
       "type", status, conversation_id, "content",
       text_content, sender_id, sender_type, private,
       content_type, source_id, meta, reply_to_source_id, created_at
   )
   VALUES (
       $1, $2, (SELECT id FROM conversation_id),
       $5, $6, $7, $8, $9, $10, $11, $12, $13, clock_timestamp()
   )
   RETURNING *
)
SELECT * FROM inserted_msg;

-- name: message-exists-by-source-id
SELECT m.conversation_id
FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id
WHERE m.source_id = ANY($1::text []) AND c.address_id=$2
ORDER BY array_position($1::text[],m.source_id),m.id DESC LIMIT 1;

-- name: update-message-status
update conversation_messages set status = $1, updated_at = NOW() where uuid = $2;

-- name: upsert-conversation-draft
INSERT INTO conversation_drafts (conversation_id, user_id, type, content, meta, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (conversation_id, user_id, type)
DO UPDATE SET content = EXCLUDED.content, meta = EXCLUDED.meta, updated_at = NOW()
RETURNING *;

-- name: get-all-user-drafts
SELECT cd.id, cd.conversation_id, cd.user_id, cd.type, cd.content, cd.meta, cd.created_at, cd.updated_at, c.uuid as conversation_uuid
FROM conversation_drafts cd
INNER JOIN conversations c ON cd.conversation_id = c.id
JOIN inboxes ON inboxes.id = c.inbox_id AND inboxes.channel = 'email'
WHERE cd.user_id = $1 AND can_access_email_address(c.address_id, $1)
ORDER BY cd.updated_at DESC;

-- name: delete-conversation-draft
DELETE FROM conversation_drafts
WHERE conversation_id IN (
  SELECT id FROM conversations
  WHERE ($1 > 0 AND id = $1) OR (NULLIF($2, '')::uuid IS NOT NULL AND uuid = NULLIF($2, '')::uuid)
) AND user_id = $3
AND ($4::text = '' OR type = $4::text);

-- name: delete-stale-drafts
DELETE FROM conversation_drafts
WHERE updated_at < $1;

-- name: insert-mention
INSERT INTO conversation_mentions (conversation_id, message_id, mentioned_user_id, mentioned_team_id, mentioned_by_user_id)
VALUES ($1, $2, $3, $4, $5);

-- name: mark-conversation-unread
WITH target AS (
    SELECT id FROM conversations WHERE uuid = $2
),
last_msg AS (
    SELECT created_at - INTERVAL '1 second' AS ts
    FROM conversation_messages
    WHERE conversation_id = (SELECT id FROM target)
      AND (meta IS NULL OR NOT COALESCE((meta->>'continuity_email')::boolean, false))
    ORDER BY created_at DESC LIMIT 1
)
INSERT INTO conversation_last_seen (user_id, conversation_id, last_seen_at)
SELECT $1, (SELECT id FROM target), ts FROM last_msg
ON CONFLICT (conversation_id, user_id)
DO UPDATE SET
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = NOW();

-- name: filter-authorized-list-uuids
-- $1: uuids (uuid[])
-- $2: user_id
-- $3: team_ids (int[])
-- $4: has 'conversations:read'
-- $5: has 'conversations:read_all'
-- $6: has 'conversations:read_assigned'
-- $7: has 'conversations:read_team_all'
-- $8: has 'conversations:read_team_inbox'
-- $9: has 'conversations:read_unassigned'
SELECT uuid::text
FROM conversations
WHERE uuid = ANY($1::uuid[])
  AND can_access_email_address(conversations.address_id, $2)
  AND $4
  AND (
       $5
    OR ($6 AND assigned_user_id = $2)
    OR ($7 AND assigned_team_id = ANY($3::int[]))
    OR ($8 AND assigned_team_id = ANY($3::int[]) AND assigned_user_id IS NULL)
    OR ($9 AND assigned_user_id IS NULL AND assigned_team_id IS NULL)
  );
