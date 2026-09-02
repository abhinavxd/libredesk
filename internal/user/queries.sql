-- name: get-users-compact
SELECT COUNT(*) OVER() as total, users.id, users.avatar_url, users.type, users.created_at, users.updated_at, users.first_name, users.last_name, users.email, users.enabled, users.external_user_id, users.availability_status
FROM users
-- email != 'System' also drops NULL-email users (anonymous visitors); AI assistants have no email and must still be listed.
WHERE (users.email != 'System' OR users.type = 'ai_assistant') AND users.deleted_at IS NULL AND type = ANY($1)

-- name: get-agents-compact
SELECT users.id, users.avatar_url, users.type, users.created_at, users.updated_at, users.first_name, users.last_name, users.email, users.enabled, users.external_user_id, users.availability_status
FROM users
WHERE (users.email != 'System' OR users.type = 'ai_assistant') AND users.deleted_at IS NULL AND users.type = ANY($1)
    AND ($2 = '' OR CONCAT(users.first_name, ' ', COALESCE(users.last_name, '')) ILIKE '%' || $2 || '%' OR users.email ILIKE '%' || $2 || '%')
    AND ($3 = '' OR users.type::text = $3)
    AND (NOT $4 OR users.enabled)
ORDER BY users.first_name, users.last_name, users.id
LIMIT NULLIF($5, 0) OFFSET $6;

-- name: get-agents-compact-by-ids
SELECT users.id, users.avatar_url, users.type, users.created_at, users.updated_at, users.first_name, users.last_name, users.email, users.enabled, users.external_user_id, users.availability_status
FROM users
WHERE (users.email != 'System' OR users.type = 'ai_assistant') AND users.deleted_at IS NULL AND users.type = ANY($1) AND users.id = ANY($2)
ORDER BY users.first_name, users.last_name, users.id;

-- name: soft-delete-agent
WITH soft_delete AS (
    UPDATE users
    SET deleted_at = now(), updated_at = now()
    WHERE id = $1 AND type = 'agent'
    RETURNING id
),
-- Delete from user_roles and teams
delete_team_members AS (
    DELETE FROM team_members
    WHERE user_id IN (SELECT id FROM soft_delete)
    RETURNING 1
),
delete_user_roles AS (
    DELETE FROM user_roles
    WHERE user_id IN (SELECT id FROM soft_delete)
    RETURNING 1
)
SELECT count(*) FROM soft_delete;

-- name: get-user
SELECT
    u.id,
    u.created_at,
    u.updated_at,
    u.email,
    u.password,
    u.type,
    u.enabled,
    u.avatar_url,
    u.first_name,
    u.last_name,
    u.availability_status,
    u.last_active_at,
    u.last_login_at,
    u.phone_number_country_code,
    u.phone_number,
    u.country,
    u.api_key,
    u.api_key_last_used_at,
    u.external_user_id,
    u.api_secret,
    array_agg(DISTINCT r.name) FILTER (WHERE r.name IS NOT NULL) AS roles,
    COALESCE(
        (SELECT json_agg(json_build_object('id', t.id, 'name', t.name, 'emoji', t.emoji))
         FROM team_members tm
         JOIN teams t ON tm.team_id = t.id
         WHERE tm.user_id = u.id),
        '[]'
    ) AS teams,
    array_agg(DISTINCT p ORDER BY p) FILTER (WHERE p IS NOT NULL) AS permissions
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON r.id = ur.role_id
LEFT JOIN LATERAL unnest(r.permissions) AS p ON true
WHERE u.deleted_at IS NULL
    AND ($1 = 0 OR u.id = $1)
    AND ($2 = '' OR u.email = $2)
    AND (COALESCE(cardinality($3::text[]), 0) = 0 OR u.type::text = ANY($3::text[]))
GROUP BY u.id
ORDER BY u.id ASC
LIMIT 1;

-- name: set-user-password
UPDATE users
SET password = $1, updated_at = now()
WHERE id = $2;

-- name: update-agent
WITH not_removed_roles AS (
 SELECT r.id FROM unnest($5::text[]) role_name
 JOIN roles r ON r.name = role_name
),
old_roles AS (
 DELETE FROM user_roles 
 WHERE user_id = $1 
 AND role_id NOT IN (SELECT id FROM not_removed_roles)
),
new_roles AS (
 INSERT INTO user_roles (user_id, role_id)
 SELECT $1, r.id FROM not_removed_roles r
 ON CONFLICT (user_id, role_id) DO NOTHING
)
UPDATE users
SET first_name = COALESCE($2, first_name),
 last_name = COALESCE($3, last_name),
 email = COALESCE($4, email),
 avatar_url = COALESCE($6, avatar_url), 
 password = COALESCE($7, password),
 enabled = COALESCE($8, enabled),
 availability_status = COALESCE($9, availability_status),
 updated_at = now()
WHERE id = $1;

-- name: update-custom-attributes
UPDATE users
SET custom_attributes = $2,
updated_at = now()
WHERE id = $1;

-- name: upsert-custom-attributes
UPDATE users
SET custom_attributes = COALESCE(custom_attributes, '{}'::jsonb) || $2,
updated_at = now()
WHERE id = $1;

-- name: update-avatar
UPDATE users  
SET avatar_url = $2, updated_at = now()
WHERE id = $1;

-- name: update-availability
UPDATE users
SET availability_status = $2
WHERE id = $1;

-- name: update-last-active-at
WITH prev AS (
    SELECT availability_status AS old_status FROM users WHERE id = $1
)
UPDATE users
SET last_active_at = now(),
availability_status = CASE WHEN availability_status = 'offline' THEN 'online' ELSE availability_status END
FROM prev
WHERE users.id = $1
RETURNING (prev.old_status = 'offline')::boolean AS was_offline;

-- name: update-inactive-offline
UPDATE users
SET availability_status = 'offline'
WHERE
  type IN ('agent', 'contact', 'visitor')
  AND (last_active_at IS NULL OR last_active_at < NOW() - INTERVAL '5 minutes')
  AND availability_status NOT IN ('offline', 'away_and_reassigning', 'away_manual')
RETURNING id, type;

-- name: get-availability-status
SELECT availability_status FROM users WHERE id = $1;

-- name: set-reset-password-token
UPDATE users
SET reset_password_token = $2, reset_password_token_expiry = now() + interval '1 day'
WHERE id = $1 AND type = 'agent';

-- name: set-password
UPDATE users
SET password = $1, reset_password_token = NULL, reset_password_token_expiry = NULL
WHERE reset_password_token = $2 AND reset_password_token_expiry > now()
RETURNING id;

-- name: insert-agent
WITH inserted_user AS (
  INSERT INTO users (email, type, first_name, last_name, "password", avatar_url)
  VALUES ($1, 'agent', $2, $3, $4, $5)
  RETURNING id AS user_id
)
INSERT INTO user_roles (user_id, role_id)
SELECT inserted_user.user_id, r.id
FROM inserted_user, unnest($6::text[]) role_name
JOIN roles r ON r.name = role_name
RETURNING user_id;

-- name: insert-contact-with-external-id
INSERT INTO users (email, type, first_name, last_name, "password", avatar_url, external_user_id, custom_attributes, phone_number, phone_number_country_code)
VALUES ($1, 'contact', $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (external_user_id) WHERE type = 'contact' AND deleted_at IS NULL AND external_user_id IS NOT NULL
DO UPDATE SET email = COALESCE(NULLIF(EXCLUDED.email, ''), users.email),
              first_name = COALESCE(NULLIF(EXCLUDED.first_name, ''), users.first_name),
              last_name = COALESCE(NULLIF(EXCLUDED.last_name, ''), users.last_name),
              phone_number = COALESCE(NULLIF(EXCLUDED.phone_number, ''), users.phone_number),
              phone_number_country_code = COALESCE(NULLIF(EXCLUDED.phone_number_country_code, ''), users.phone_number_country_code),
              updated_at = now()
RETURNING id;

-- name: insert-contact-without-external-id
INSERT INTO users (email, type, first_name, last_name, "password", avatar_url, external_user_id)
VALUES ($1, 'contact', $2, $3, $4, $5, NULL)
ON CONFLICT (email) WHERE type = 'contact' AND deleted_at IS NULL AND external_user_id IS NULL
DO UPDATE SET first_name = COALESCE(NULLIF(EXCLUDED.first_name, ''), users.first_name),
              last_name = COALESCE(NULLIF(EXCLUDED.last_name, ''), users.last_name),
              updated_at = now()
RETURNING id;

-- name: insert-contact-if-absent
INSERT INTO users (email, type, first_name, last_name, "password", avatar_url, external_user_id, custom_attributes)
VALUES ($1, 'contact', $2, $3, $4, $5, $6, $7)
ON CONFLICT DO NOTHING
RETURNING id;

-- name: get-contact-by-email
SELECT id, external_user_id FROM users
WHERE email = $1 AND type = 'contact' AND deleted_at IS NULL
ORDER BY (external_user_id IS NOT NULL) DESC, id ASC LIMIT 1;

-- name: get-contact-by-email-without-ext-id
SELECT id FROM users
WHERE email = $1 AND type = 'contact' AND deleted_at IS NULL AND external_user_id IS NULL
LIMIT 1;

-- name: is-email-blocked
SELECT EXISTS(
    SELECT 1 FROM users
    WHERE email = $1 AND type IN ('contact', 'visitor') AND deleted_at IS NULL AND enabled = false
) AS is_blocked;

-- name: set-external-user-id
UPDATE users SET external_user_id = $2, updated_at = now()
WHERE id = $1 AND type = 'contact' AND deleted_at IS NULL;

-- name: set-contact-phone-if-missing
UPDATE users SET phone_number = $2, phone_number_country_code = NULLIF($3, ''), updated_at = now()
WHERE id = $1
  AND type IN ('contact', 'visitor')
  AND deleted_at IS NULL
  AND (phone_number IS NULL OR phone_number = '');

-- name: update-contact-name-if-default
UPDATE users SET first_name = $2, last_name = $3, updated_at = now()
WHERE id = $1
  AND type = 'contact'
  AND deleted_at IS NULL
  AND first_name = $4
  AND COALESCE(last_name, '') = '';

-- name: get-contact-id-by-channel-identity
SELECT contact_id FROM contact_channel_identities WHERE channel = $1::channels AND identifier = $2;

-- name: get-channel-identities-by-contact
SELECT channel, identifier FROM contact_channel_identities WHERE contact_id = $1 ORDER BY id;

-- name: get-channel-identity
SELECT identifier FROM contact_channel_identities WHERE contact_id = $1 AND channel = $2::channels ORDER BY id LIMIT 1;

-- name: insert-channel-identity
INSERT INTO contact_channel_identities (contact_id, channel, identifier)
VALUES ($1, $2::channels, $3)
ON CONFLICT (channel, identifier) DO UPDATE SET updated_at = now()
RETURNING contact_id;

-- name: update-channel-identity
-- $1=channel, $2=old identifier, $3=new identifier. No-op when the new identifier already belongs to a contact.
UPDATE contact_channel_identities SET identifier = $3, updated_at = now()
WHERE channel = $1::channels AND identifier = $2
AND NOT EXISTS (
    SELECT 1 FROM contact_channel_identities WHERE channel = $1::channels AND identifier = $3
)
RETURNING contact_id;

-- name: upsert-contact-with-channel-identity
-- Atomic: a contact with no email and no ext_id has no uniqueness key, so a separate insert + link would orphan user rows on retry.
-- $1=email, $2=first_name, $3=last_name, $4=password, $5=avatar_url, $6=channel, $7=identifier
WITH existing AS (
    SELECT contact_id FROM contact_channel_identities
    WHERE channel = $6::channels AND identifier = $7
),
new_contact AS (
    INSERT INTO users (email, type, first_name, last_name, "password", avatar_url, external_user_id)
    SELECT $1, 'contact', $2, $3, $4, $5, NULL
    WHERE NOT EXISTS (SELECT 1 FROM existing)
    RETURNING id
),
new_identity AS (
    INSERT INTO contact_channel_identities (contact_id, channel, identifier)
    SELECT id, $6::channels, $7 FROM new_contact
    ON CONFLICT (channel, identifier) DO UPDATE SET updated_at = now()
    RETURNING contact_id
)
SELECT COALESCE(
    (SELECT contact_id FROM new_identity),
    (SELECT contact_id FROM existing)
) AS id,
(SELECT id FROM new_contact) AS inserted_id;

-- name: delete-orphaned-contact
-- Cleans up the losing row of a concurrent identity upsert; the guard keeps it a no-op for any contact that gained an identity.
DELETE FROM users
WHERE id = $1 AND type = 'contact'
AND NOT EXISTS (SELECT 1 FROM contact_channel_identities WHERE contact_id = $1);

-- name: insert-visitor
INSERT INTO users (email, type, first_name, last_name, custom_attributes, phone_number, phone_number_country_code)
VALUES ($1, 'visitor', $2, $3, $4, $5, $6)
RETURNING *;

-- name: update-last-login-at
UPDATE users
SET last_login_at = now(),
updated_at = now()
WHERE id = $1;

-- name: toggle-enable
UPDATE users
SET enabled = $3, updated_at = NOW()
WHERE id = $1 AND type = $2;

-- name: update-contact
UPDATE users
SET first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    email = COALESCE($4, email),
    avatar_url = $5,
    phone_number = $6,
    phone_number_country_code = $7,
    country = $8,
    updated_at = now()
WHERE id = $1 and type in ('contact', 'visitor');

-- name: update-contact-basic-info
UPDATE users
SET first_name = COALESCE(NULLIF($2, ''), first_name),
    last_name = COALESCE(NULLIF($3, ''), last_name),
    email = COALESCE(NULLIF($4, ''), email),
    phone_number = COALESCE(NULLIF($5, ''), phone_number),
    phone_number_country_code = COALESCE(NULLIF($6, ''), phone_number_country_code),
    updated_at = now()
WHERE id = $1 AND type IN ('contact', 'visitor');

-- name: get-notes
SELECT 
    cn.id,
    cn.created_at,
    cn.updated_at,
    cn.contact_id,
    cn.note,
    cn.user_id,
    u.first_name,
    u.last_name,
    u.avatar_url
FROM contact_notes cn
INNER JOIN users u ON u.id = cn.user_id
WHERE cn.contact_id = $1
ORDER BY cn.created_at DESC;

-- name: insert-note
INSERT INTO contact_notes (contact_id, user_id, note)
VALUES ($1, $2, $3)
RETURNING *;

-- name: delete-note
DELETE FROM contact_notes
WHERE id = $1 AND contact_id = $2;

-- name: get-note
SELECT 
    cn.id,
    cn.created_at,
    cn.updated_at,
    cn.contact_id,
    cn.note,
    cn.user_id,
    u.first_name,
    u.last_name,
    u.avatar_url
FROM contact_notes cn
INNER JOIN users u ON u.id = cn.user_id
WHERE cn.id = $1;

-- name: get-user-by-api-key
SELECT
    u.id,
    u.created_at,
    u.updated_at,
    u.email,
    u.password,
    u.type,
    u.enabled,
    u.avatar_url,
    u.first_name,
    u.last_name,
    u.availability_status,
    u.last_active_at,
    u.last_login_at,
    u.phone_number_country_code,
    u.phone_number,
    u.country,
    u.api_key,
    u.api_key_last_used_at,
    u.api_secret,
    u.external_user_id,
    array_agg(DISTINCT r.name) FILTER (WHERE r.name IS NOT NULL) AS roles,
    COALESCE(
        (SELECT json_agg(json_build_object('id', t.id, 'name', t.name, 'emoji', t.emoji))
         FROM team_members tm
         JOIN teams t ON tm.team_id = t.id
         WHERE tm.user_id = u.id),
        '[]'
    ) AS teams,
    array_agg(DISTINCT p ORDER BY p) FILTER (WHERE p IS NOT NULL) AS permissions
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON r.id = ur.role_id
LEFT JOIN LATERAL unnest(r.permissions) AS p ON true
WHERE u.api_key = $1 AND u.enabled = true AND u.deleted_at IS NULL
GROUP BY u.id;

-- name: set-api-key
UPDATE users 
SET api_key = $2, api_secret = $3, api_key_last_used_at = NULL, updated_at = now()
WHERE id = $1;

-- name: revoke-api-key
UPDATE users 
SET api_key = NULL, api_secret = NULL, api_key_last_used_at = NULL, updated_at = now()
WHERE id = $1;

-- name: update-api-secret-hash
UPDATE users
SET api_secret = $3, updated_at = now()
WHERE id = $1 AND api_secret = $2;

-- name: update-api-key-last-used
UPDATE users
SET api_key_last_used_at = now()
WHERE id = $1;

-- name: get-user-by-external-id
SELECT
    u.id,
    u.created_at,
    u.updated_at,
    u.email,
    u.password,
    u.type,
    u.enabled,
    u.avatar_url,
    u.first_name,
    u.last_name,
    u.availability_status,
    u.last_active_at,
    u.last_login_at,
    u.phone_number_country_code,
    u.phone_number,
    u.country,
    u.external_user_id,
    u.custom_attributes,
    u.api_key,
    u.api_key_last_used_at,
    array_agg(DISTINCT r.name) FILTER (WHERE r.name IS NOT NULL) AS roles,
    COALESCE(
        (SELECT json_agg(json_build_object('id', t.id, 'name', t.name, 'emoji', t.emoji))
         FROM team_members tm
         JOIN teams t ON tm.team_id = t.id
         WHERE tm.user_id = u.id),
        '[]'
    ) AS teams,
    array_agg(DISTINCT p ORDER BY p) FILTER (WHERE p IS NOT NULL) AS permissions
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON r.id = ur.role_id
LEFT JOIN LATERAL unnest(r.permissions) AS p ON true
WHERE u.deleted_at IS NULL
    AND u.external_user_id = $1
    AND u.type = 'contact'
GROUP BY u.id;

-- name: get-visitor-by-email
SELECT id, email, external_user_id FROM users
WHERE email = $1 AND type = 'visitor' AND deleted_at IS NULL
LIMIT 1;

-- name: upgrade-visitor-to-contact
UPDATE users SET type = 'contact', updated_at = now()
WHERE id = $1 AND type = 'visitor';

-- name: merge-visitor-to-contact
WITH transfer_conversations AS (
    UPDATE conversations
    SET contact_id = $2, updated_at = now()
    WHERE contact_id = $1
    RETURNING id
),
transfer_messages AS (
    UPDATE conversation_messages
    SET sender_id = $2
    WHERE conversation_id IN (SELECT id FROM transfer_conversations) AND sender_id = $1
    RETURNING id
),
transfer_participants AS (
    UPDATE conversation_participants
    SET user_id = $2
    WHERE user_id = $1 AND NOT EXISTS (
        SELECT 1 FROM conversation_participants WHERE user_id = $2 AND conversation_id = conversation_participants.conversation_id
    )
    RETURNING id
),
delete_remaining_participants AS (
    DELETE FROM conversation_participants
    WHERE user_id = $1
    RETURNING id
),
transfer_notes AS (
    UPDATE contact_notes
    SET contact_id = $2
    WHERE contact_id = $1
    RETURNING id
),
delete_visitor AS (
    DELETE FROM users
    WHERE id = $1 AND type = 'visitor'
    RETURNING id
)
SELECT
    (SELECT COUNT(*) FROM transfer_conversations) as conversations_transferred,
    (SELECT COUNT(*) FROM transfer_messages) as messages_transferred,
    (SELECT COUNT(*) FROM delete_visitor) as visitor_deleted;

-- name: get-user-ids-by-role
SELECT user_id FROM user_roles WHERE role_id = $1;

-- name: delete-contact
DELETE FROM users
WHERE id = $1 AND type IN ('contact', 'visitor');

-- name: export-contact-data
SELECT jsonb_build_object(
    'contact', (
        SELECT jsonb_build_object(
            'id', id,
            'created_at', created_at,
            'first_name', first_name,
            'last_name', last_name,
            'email', email,
            'phone_number_country_code', phone_number_country_code,
            'phone_number', phone_number,
            'country', country,
            'avatar_url', avatar_url,
            'external_user_id', external_user_id,
            'custom_attributes', custom_attributes,
            'availability_status', availability_status,
            'last_active_at', last_active_at,
            'last_login_at', last_login_at,
            'enabled', enabled
        )
        FROM users
        WHERE id = $1 AND type IN ('contact', 'visitor')
    ),
    'conversations', (
        SELECT COALESCE(jsonb_agg(jsonb_build_object(
            'reference_number', c.reference_number,
            'created_at', c.created_at,
            'subject', c.subject,
            'status', cs.name,
            'custom_attributes', c.custom_attributes,
            'messages', (
                SELECT COALESCE(jsonb_agg(jsonb_build_object(
                    'created_at', m.created_at,
                    'type', m.type,
                    'sender_type', m.sender_type,
                    'content', m.text_content
                ) ORDER BY m.created_at), '[]'::jsonb)
                FROM conversation_messages m
                WHERE m.conversation_id = c.id AND m.private = false AND m.type IN ('incoming', 'outgoing')
            ),
            'csat_responses', (
                SELECT COALESCE(jsonb_agg(jsonb_build_object(
                    'created_at', cr.created_at,
                    'rating', cr.rating,
                    'feedback', cr.feedback,
                    'response_timestamp', cr.response_timestamp
                ) ORDER BY cr.created_at), '[]'::jsonb)
                FROM csat_responses cr
                WHERE cr.conversation_id = c.id AND cr.response_timestamp IS NOT NULL
            )
        ) ORDER BY c.created_at), '[]'::jsonb)
        FROM conversations c
        LEFT JOIN conversation_statuses cs ON cs.id = c.status_id
        WHERE c.contact_id = $1
    )
);
