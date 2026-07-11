-- name: get-assistants
SELECT a.id, a.created_at, a.updated_at, a.user_id, u.first_name AS name, u.avatar_url,
       a.description, a.instructions, a.guardrails, a.tone, a.response_length, a.max_turns, a.fallback_team_id, a.enabled
FROM ai_assistants a
JOIN users u ON u.id = a.user_id AND u.deleted_at IS NULL
ORDER BY a.updated_at DESC;

-- name: get-assistant
SELECT a.id, a.created_at, a.updated_at, a.user_id, u.first_name AS name, u.avatar_url,
       a.description, a.instructions, a.guardrails, a.tone, a.response_length, a.max_turns, a.fallback_team_id, a.enabled
FROM ai_assistants a
JOIN users u ON u.id = a.user_id AND u.deleted_at IS NULL
WHERE a.id = $1;

-- name: get-assistant-by-user-id
SELECT a.id, a.created_at, a.updated_at, a.user_id, u.first_name AS name, u.avatar_url,
       a.description, a.instructions, a.guardrails, a.tone, a.response_length, a.max_turns, a.fallback_team_id, a.enabled
FROM ai_assistants a
JOIN users u ON u.id = a.user_id AND u.deleted_at IS NULL
WHERE a.user_id = $1;

-- name: get-assistant-user-ids
SELECT a.user_id
FROM ai_assistants a
JOIN users u ON u.id = a.user_id AND u.deleted_at IS NULL;

-- name: insert-assistant-user
INSERT INTO users (type, first_name, last_name, enabled)
VALUES ('ai_assistant', $1, '', true)
RETURNING id;

-- name: insert-assistant
INSERT INTO ai_assistants (user_id, description, instructions, guardrails, tone, response_length, max_turns, fallback_team_id, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: update-assistant
UPDATE ai_assistants
SET description = $2, instructions = $3, guardrails = $4, tone = $5, response_length = $6, max_turns = $7, fallback_team_id = $8, enabled = $9, updated_at = now()
WHERE id = $1;

-- name: update-assistant-user
UPDATE users SET first_name = $2, updated_at = now()
WHERE id = $1;

-- name: soft-delete-assistant-user
UPDATE users SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND type = 'ai_assistant';

-- name: delete-assistant
DELETE FROM ai_assistants WHERE id = $1;

-- name: get-assistant-tools
SELECT tool_id FROM ai_assistant_tools WHERE assistant_id = $1 ORDER BY tool_id;

-- name: delete-assistant-tools
DELETE FROM ai_assistant_tools WHERE assistant_id = $1;

-- name: insert-assistant-tool
INSERT INTO ai_assistant_tools (assistant_id, tool_id)
VALUES ($1, $2)
ON CONFLICT (assistant_id, tool_id) DO NOTHING;

-- name: insert-ai-agent-event
INSERT INTO ai_agent_events (assistant_id, conversation_id, type) VALUES ($1, $2, $3);

-- name: get-assistant-window-stats
-- $1 = assistant user id (message sender), $2 = assistant id (events), $3 = window start, $4 = window end.
SELECT
  (SELECT count(DISTINCT conversation_id) FROM conversation_messages WHERE sender_id = $1 AND type = 'outgoing' AND private = false AND created_at >= $3 AND created_at < $4) AS conversations,
  (SELECT count(*) FROM conversation_messages WHERE sender_id = $1 AND type = 'outgoing' AND private = false AND created_at >= $3 AND created_at < $4) AS replies,
  (SELECT count(DISTINCT conversation_id) FROM ai_agent_events WHERE assistant_id = $2 AND type = 'handoff' AND created_at >= $3 AND created_at < $4) AS handoffs,
  (SELECT count(DISTINCT conversation_id) FROM ai_agent_events WHERE assistant_id = $2 AND type = 'resolve' AND created_at >= $3 AND created_at < $4) AS resolves,
  (SELECT count(DISTINCT e.conversation_id) FROM ai_agent_events e JOIN conversations c ON c.id = e.conversation_id JOIN conversation_statuses s ON s.id = c.status_id
     WHERE e.assistant_id = $2 AND e.type = 'resolve' AND e.created_at >= $3 AND e.created_at < $4 AND s.name <> 'Resolved') AS reopened,
  (SELECT count(*) FROM csat_responses cr WHERE cr.rating > 0 AND cr.created_at >= $3 AND cr.created_at < $4 AND EXISTS (
     SELECT 1 FROM conversation_messages m WHERE m.conversation_id = cr.conversation_id AND m.sender_id = $1 AND m.type = 'outgoing' AND m.private = false)) AS csat_count,
  COALESCE((SELECT round(avg(cr.rating)::numeric, 2) FROM csat_responses cr WHERE cr.rating > 0 AND cr.created_at >= $3 AND cr.created_at < $4 AND EXISTS (
     SELECT 1 FROM conversation_messages m WHERE m.conversation_id = cr.conversation_id AND m.sender_id = $1 AND m.type = 'outgoing' AND m.private = false)), 0)::float8 AS csat_avg,
  COALESCE((SELECT round((count(*) FILTER (WHERE cr.rating >= 4))::numeric / NULLIF(count(*), 0) * 100, 1) FROM csat_responses cr WHERE cr.rating > 0 AND cr.created_at >= $3 AND cr.created_at < $4 AND EXISTS (
     SELECT 1 FROM conversation_messages m WHERE m.conversation_id = cr.conversation_id AND m.sender_id = $1 AND m.type = 'outgoing' AND m.private = false)), 0)::float8 AS csat_positive;

-- name: count-ai-turns-since-assignment
-- Counts the assistant's public replies in the current engagement, i.e. since it was last
-- (re)assigned. Any assignment/status change writes an activity row, so the last activity marks
-- the start of this engagement and a fresh assignment resets the turn budget.
SELECT count(*) FROM conversation_messages
WHERE conversation_id = $1 AND sender_id = $2 AND type = 'outgoing' AND private = false
  AND created_at > COALESCE((
    SELECT max(created_at) FROM conversation_messages
    WHERE conversation_id = $1 AND type = 'activity'
  ), to_timestamp(0));

-- name: get-recent-contact-conversations
SELECT c.uuid, c.reference_number, c.created_at, COALESCE(c.subject, '') AS subject, s.name AS status
FROM conversations c
JOIN conversation_statuses s ON s.id = c.status_id
WHERE c.contact_id = $1 AND c.id != $2 AND c.created_at >= now() - make_interval(days => $3)
ORDER BY c.created_at DESC
LIMIT $4;

-- name: insert-faq-suggestion
INSERT INTO ai_faq_suggestions (conversation_id, question, answer) VALUES ($1, $2, $3);

-- name: count-faq-suggestions-by-conversation
SELECT count(*) FROM ai_faq_suggestions WHERE conversation_id = $1;

-- name: get-faq-suggestions
SELECT s.id, s.created_at, s.updated_at, s.conversation_id, s.question, s.answer, s.status, s.reviewed_by_id, s.reviewed_at,
       c.uuid AS conversation_uuid, c.reference_number
FROM ai_faq_suggestions s
JOIN conversations c ON c.id = s.conversation_id
WHERE ($1 = '' OR s.status = $1)
ORDER BY s.created_at DESC;

-- name: get-faq-suggestion
SELECT id, created_at, updated_at, conversation_id, question, answer, status, reviewed_by_id, reviewed_at
FROM ai_faq_suggestions WHERE id = $1;

-- name: update-faq-suggestion-status
UPDATE ai_faq_suggestions SET status = $2, reviewed_by_id = $3, reviewed_at = now(), updated_at = now() WHERE id = $1;
