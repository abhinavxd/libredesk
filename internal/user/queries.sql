-- name: get-users
SELECT u.id, u.updated_at, u.first_name, u.last_name, u.email, u.enabled
FROM users u 
WHERE u.email != 'System' AND u.deleted_at IS NULL AND u.type = 'agent'
ORDER BY u.updated_at DESC;

-- name: soft-delete-user
WITH soft_delete AS (
    UPDATE users
    SET deleted_at = now(), updated_at = now()
    WHERE id = $1 AND type = 'agent'
    RETURNING id
)
DELETE FROM team_members WHERE user_id IN (SELECT id FROM soft_delete);

-- name: get-users-compact
SELECT u.id, u.first_name, u.last_name, u.enabled, u.avatar_url
FROM users u
WHERE u.email != 'System' AND u.deleted_at IS NULL AND u.type = 'agent'
ORDER BY u.updated_at DESC;

-- name: get-user
SELECT
    u.id,
    u.email,
    u.password,
    u.created_at,
    u.updated_at,
    u.enabled,
    u.avatar_url,
    u.first_name,
    u.last_name,
    u.availability_status,
    array_agg(DISTINCT r.name) as roles,
    COALESCE(
         (SELECT json_agg(json_build_object('id', t.id, 'name', t.name, 'emoji', t.emoji))
          FROM team_members tm
          JOIN teams t ON tm.team_id = t.id
          WHERE tm.user_id = u.id),
         '[]'
    ) AS teams,
    array_agg(DISTINCT p) as permissions
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON r.id = ur.role_id,
     unnest(r.permissions) p
WHERE (u.id = $1 OR u.email = $2) AND u.type = $3 AND u.deleted_at IS NULL
GROUP BY u.id;

-- name: set-user-password
UPDATE users
SET password = $1, updated_at = now()
WHERE id = $2 AND type = 'agent';

-- name: update-user
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
 updated_at = now()
WHERE id = $1 AND type = 'agent';

-- name: update-avatar
UPDATE users  
SET avatar_url = $2, updated_at = now()
WHERE id = $1 AND type = 'agent';

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
AND availability_status != 'offline';

-- name: set-reset-password-token
UPDATE users
SET reset_password_token = $2, reset_password_token_expiry = now() + interval '1 day'
WHERE id = $1 AND type = 'agent';

-- name: reset-password
UPDATE users  
SET password = $1, reset_password_token = NULL, reset_password_token_expiry = NULL
WHERE reset_password_token = $2 AND reset_password_token_expiry > now() AND type = 'agent';

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