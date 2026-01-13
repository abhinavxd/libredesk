-- name: get-view
SELECT id, created_at, updated_at, name, filters, visibility, user_id, team_id
FROM views WHERE id = $1;

-- name: get-user-views
-- Returns personal views (visibility='user') for a specific user
SELECT id, created_at, updated_at, name, filters, visibility, user_id, team_id
FROM views WHERE user_id = $1 AND visibility = 'user'
ORDER BY name ASC;

-- name: get-shared-views-for-user
-- Returns shared views visible to a user (global + team views for user's teams)
SELECT id, created_at, updated_at, name, filters, visibility, user_id, team_id
FROM views
WHERE visibility = 'all'
   OR (visibility = 'team' AND team_id = ANY($1))
ORDER BY name ASC;

-- name: get-all-shared-views
SELECT id, created_at, updated_at, name, filters, visibility, user_id, team_id
FROM views
WHERE visibility != 'user'
ORDER BY updated_at DESC;

-- name: insert-view
INSERT INTO views (name, filters, visibility, user_id, team_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: delete-view
DELETE FROM views
WHERE id = $1;

-- name: update-view
UPDATE views
SET name = $2, filters = $3, visibility = $4, user_id = $5, team_id = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;
