-- name: get-enabled-rules
select
    type,
    events,
    rules,
    execution_mode
from automation_rules where enabled is TRUE ORDER BY weight ASC;

-- name: get-all
SELECT id, created_at, updated_at, "name", description, "type", rules, events, enabled, weight, execution_mode from automation_rules where type = $1 ORDER BY weight ASC;

-- name: get-rule
SELECT id, created_at, updated_at, "name", description, "type", rules, events, enabled, weight, execution_mode from automation_rules where id = $1;

-- name: update-rule
INSERT INTO automation_rules(id, name, description, type, events, rules, enabled)
VALUES($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id)
DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    type = EXCLUDED.type,
    events = EXCLUDED.events,
    rules = EXCLUDED.rules,
    enabled = EXCLUDED.enabled,
    updated_at = now()
WHERE $1 > 0
RETURNING *;

-- name: insert-rule
INSERT into automation_rules (name, description, type, events, rules) 
values ($1, $2, $3, $4, $5)
RETURNING *;

-- name: delete-rule
delete from automation_rules where id = $1;

-- name: toggle-rule
UPDATE automation_rules 
SET enabled = NOT enabled, updated_at = NOW() 
WHERE id = $1
RETURNING *;

-- name: update-rule-weight
UPDATE automation_rules
SET weight = $2, updated_at = NOW()
WHERE id = $1;

-- name: update-rule-execution-mode
UPDATE automation_rules
SET execution_mode = $2, updated_at = NOW()
WHERE type = $1;