-- name: get-teams
SELECT id, created_at, updated_at, name, emoji, conversation_assignment_type, max_auto_assigned_conversations, business_hours_id, sla_policy_id, timezone from teams order by updated_at desc;

-- name: get-teams-compact
SELECT id, name, emoji from teams order by name;

-- name: get-user-teams
SELECT id, created_at, updated_at, name, emoji, conversation_assignment_type, max_auto_assigned_conversations, business_hours_id, sla_policy_id, timezone from teams WHERE id IN (SELECT team_id FROM team_members WHERE user_id = $1) order by updated_at desc;

-- name: get-team
SELECT id, created_at, updated_at, name, emoji, conversation_assignment_type, max_auto_assigned_conversations, business_hours_id, sla_policy_id, timezone from teams where id = $1;

-- name: get-team-members
SELECT u.id, t.id as team_id, u.availability_status
FROM users u
JOIN team_members tm ON tm.user_id = u.id
JOIN teams t ON t.id = tm.team_id
WHERE t.id = $1 AND u.deleted_at IS NULL AND u.type = 'agent' AND u.enabled = true;

-- name: insert-team
INSERT INTO teams (name, timezone, conversation_assignment_type, business_hours_id, sla_policy_id, emoji, max_auto_assigned_conversations) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: update-team
UPDATE teams set name = $2, timezone = $3, conversation_assignment_type = $4, business_hours_id = $5, sla_policy_id = $6, emoji = $7, max_auto_assigned_conversations = $8, updated_at = now() where id = $1 RETURNING *;

-- name: upsert-user-teams
WITH delete_old_teams AS (
    DELETE FROM team_members 
    WHERE user_id = $1 
    AND team_id NOT IN (SELECT t.id FROM teams t WHERE t.name = ANY($2))
),
insert_new_teams AS (
    INSERT INTO team_members (user_id, team_id)
    SELECT $1, t.id 
    FROM teams t 
    WHERE t.name = ANY($2)
    ON CONFLICT DO NOTHING
)
SELECT 1;

-- name: delete-team
DELETE FROM teams where id = $1;

-- name: user-belongs-to-team
SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2);