-- name: get-users-compact
-- TODO: Remove hardcoded `type` of user in some queries in this file.
SELECT COUNT(*) OVER() as total, users.id, users.avatar_url, users.type, users.created_at, users.updated_at, users.first_name, users.last_name, users.email, users.enabled
FROM users
WHERE users.email != 'System' AND users.deleted_at IS NULL AND type = ANY($1)

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
SELECT 1;

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
    u.phone_number_calling_code,
    u.phone_number,
    u.api_key,
    u.api_key_last_used_at,
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
WHERE (u.id = $1 OR u.email = $2) AND u.type = $3 AND u.deleted_at IS NULL
GROUP BY u.id;

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

-- name: update-avatar
UPDATE users  
SET avatar_url = $2, updated_at = now()
WHERE id = $1;

-- name: update-availability
UPDATE users
SET availability_status = $2
WHERE id = $1;

-- name: update-last-active-at
UPDATE users
SET last_active_at = now(),
availability_status = CASE WHEN availability_status = 'offline' THEN 'online' ELSE availability_status END
WHERE id = $1;

-- name: update-inactive-offline
UPDATE users
SET availability_status = 'offline'
WHERE 
type = 'agent' 
AND (last_active_at IS NULL OR last_active_at < NOW() - INTERVAL '5 minutes')
AND availability_status NOT IN ('offline', 'away_and_reassigning', 'away_manual');

-- name: set-reset-password-token
UPDATE users
SET reset_password_token = $2, reset_password_token_expiry = now() + interval '1 day'
WHERE id = $1 AND type = 'agent';

-- name: set-password
UPDATE users  
SET password = $1, reset_password_token = NULL, reset_password_token_expiry = NULL
WHERE reset_password_token = $2 AND reset_password_token_expiry > now();

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

-- name: insert-contact
WITH contact AS (
   INSERT INTO users (email, type, first_name, last_name, "password", avatar_url)
   VALUES ($1, 'contact', $2, $3, $4, $5)
   ON CONFLICT (email, type) WHERE deleted_at IS NULL
   DO UPDATE SET updated_at = now()
   RETURNING id
)
INSERT INTO contact_channels (contact_id, inbox_id, identifier)
VALUES ((SELECT id FROM contact), $6, $7)
ON CONFLICT (contact_id, inbox_id) DO UPDATE SET updated_at = now()
RETURNING contact_id, id;

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
    phone_number_calling_code = $7,
    updated_at = now()
WHERE id = $1 and type = 'contact';

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
    u.phone_number_calling_code,
    u.phone_number,
    u.api_key,
    u.api_key_last_used_at,
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

-- name: update-api-key-last-used
UPDATE users 
SET api_key_last_used_at = now()
WHERE id = $1;