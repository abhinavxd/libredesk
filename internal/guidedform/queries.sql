-- name: get-forms
SELECT id, created_at, updated_at, user_id, name, display_name, inbox_id, enabled, start_step_id, steps,
       on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human,
       abandoned_timeout_minutes
FROM guided_forms
ORDER BY updated_at DESC;

-- name: get-form
SELECT id, created_at, updated_at, user_id, name, display_name, inbox_id, enabled, start_step_id, steps,
       on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human,
       abandoned_timeout_minutes
FROM guided_forms WHERE id = $1;

-- name: get-form-by-user-id
SELECT id, created_at, updated_at, user_id, name, display_name, inbox_id, enabled, start_step_id, steps,
       on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human,
       abandoned_timeout_minutes
FROM guided_forms WHERE user_id = $1;

-- name: get-form-by-inbox-id
-- One enabled guided form per inbox is expected; the most recently updated one wins if more exist.
SELECT id, created_at, updated_at, user_id, name, display_name, inbox_id, enabled, start_step_id, steps,
       on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human,
       abandoned_timeout_minutes
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
INSERT INTO guided_forms (user_id, name, display_name, inbox_id, enabled, start_step_id, steps, on_complete_action, on_complete_assistant_id, on_complete_team_id, completion_message, allow_skip_to_human, abandoned_timeout_minutes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING id;

-- name: update-form
UPDATE guided_forms
SET name = $2, display_name = $3, inbox_id = $4, enabled = $5, start_step_id = $6, steps = $7, on_complete_action = $8,
    on_complete_assistant_id = $9, on_complete_team_id = $10, completion_message = $11, allow_skip_to_human = $12,
    abandoned_timeout_minutes = $13, updated_at = now()
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

-- name: get-abandoned-guided-form-conversations
-- A conversation still assigned to a guided-form bot whose own last message (the current
-- question) has sat unanswered past that form's configured timeout. last_message_sender is
-- NULL-safe compared since a brand new conversation can have no messages recorded yet.
SELECT c.id AS conversation_id, c.uuid AS conversation_uuid, gf.id AS form_id, gf.user_id AS bot_user_id,
       u.first_name AS bot_name
FROM conversations c
JOIN guided_forms gf ON gf.user_id = c.assigned_user_id
JOIN users u ON u.id = gf.user_id
JOIN conversation_statuses s ON s.id = c.status_id
WHERE gf.abandoned_timeout_minutes > 0
  AND s.category <> 'resolved'
  AND c.last_message_sender IS DISTINCT FROM 'contact'
  AND c.last_message_at IS NOT NULL
  AND c.last_message_at < now() - (gf.abandoned_timeout_minutes || ' minutes')::interval;
