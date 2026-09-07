-- name: get-forms
SELECT id, created_at, updated_at, user_id, name, inbox_id, enabled, start_step_id, steps,
       on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human
FROM guided_forms
ORDER BY updated_at DESC;

-- name: get-form
SELECT id, created_at, updated_at, user_id, name, inbox_id, enabled, start_step_id, steps,
       on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human
FROM guided_forms WHERE id = $1;

-- name: get-form-by-user-id
SELECT id, created_at, updated_at, user_id, name, inbox_id, enabled, start_step_id, steps,
       on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human
FROM guided_forms WHERE user_id = $1;

-- name: get-form-by-inbox-id
-- One enabled guided form per inbox is expected; the most recently updated one wins if more exist.
SELECT id, created_at, updated_at, user_id, name, inbox_id, enabled, start_step_id, steps,
       on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human
FROM guided_forms WHERE inbox_id = $1 AND enabled = true
ORDER BY updated_at DESC LIMIT 1;

-- name: get-form-bot-user-ids
SELECT id FROM users WHERE type = 'guided_form_bot';

-- name: insert-form-user
INSERT INTO users (type, first_name, last_name, enabled)
VALUES ('guided_form_bot', $1, '', true)
RETURNING id;

-- name: update-form-user
UPDATE users SET first_name = $2, updated_at = now()
WHERE id = $1;

-- name: soft-delete-form-user
UPDATE users SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND type = 'guided_form_bot';

-- name: insert-form
INSERT INTO guided_forms (user_id, name, inbox_id, enabled, start_step_id, steps, on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id;

-- name: update-form
UPDATE guided_forms
SET name = $2, inbox_id = $3, enabled = $4, start_step_id = $5, steps = $6, on_complete_action = $7,
    on_complete_assistant_id = $8, on_complete_team_id = $9, completion_message = $10, allow_skip_to_human = $11, updated_at = now()
WHERE id = $1;

-- name: delete-form
DELETE FROM guided_forms WHERE id = $1;

-- name: disable-other-forms-on-inbox
-- Enforces one active form per inbox: run before enabling $2 on inbox $1 so the partial unique
-- index on (inbox_id) WHERE enabled never conflicts.
UPDATE guided_forms SET enabled = false, updated_at = now()
WHERE inbox_id = $1 AND enabled = true AND id != $2;

-- name: unassign-form-bot-conversations
UPDATE conversations
SET assigned_user_id = NULL, assigned_team_id = COALESCE($2, assigned_team_id), updated_at = now()
WHERE assigned_user_id = $1
  AND status_id IN (SELECT id FROM conversation_statuses WHERE category <> 'resolved');

-- name: insert-guided-form-event
INSERT INTO guided_form_events (form_id, conversation_id, type) VALUES ($1, $2, $3);
