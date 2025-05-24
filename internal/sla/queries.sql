-- name: get-sla-policy
SELECT id, name, description, first_response_time, resolution_time, next_response_time, notifications, created_at, updated_at FROM sla_policies WHERE id = $1;

-- name: get-all-sla-policies
SELECT id, name, created_at, updated_at FROM sla_policies ORDER BY updated_at DESC;

-- name: insert-sla-policy
INSERT INTO sla_policies (
   name,
   description, 
   first_response_time,
   resolution_time,
   next_response_time,
   notifications
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: update-sla-policy
UPDATE sla_policies SET
   name = $2,
   description = $3,
   first_response_time = $4,
   resolution_time = $5,
   next_response_time = $6,
   notifications = $7,
   updated_at = NOW()
WHERE id = $1;

-- name: delete-sla-policy
DELETE FROM sla_policies WHERE id = $1;

-- name: apply-sla
WITH new_sla AS (
  INSERT INTO applied_slas (
    conversation_id,
    sla_policy_id,
    first_response_deadline_at,
    resolution_deadline_at
  ) VALUES ($1, $2, $3, $4)
  RETURNING conversation_id, id
)
-- update the conversation with the new SLA policy and next SLA deadline.
UPDATE conversations c
SET 
   sla_policy_id = $2,
   next_sla_deadline_at = LEAST($3, $4)
FROM new_sla ns
WHERE c.id = ns.conversation_id
RETURNING ns.id;

-- name: get-pending-slas
-- Get all the applied SLAs (applied to a conversation) that are pending
SELECT a.id, a.first_response_deadline_at, c.first_reply_at as conversation_first_response_at, a.sla_policy_id,
a.resolution_deadline_at, c.resolved_at as conversation_resolved_at, c.id as conversation_id, a.first_response_met_at, a.resolution_met_at, a.first_response_breached_at, a.resolution_breached_at
FROM applied_slas a 
JOIN conversations c ON a.conversation_id = c.id and c.sla_policy_id = a.sla_policy_id
WHERE a.status = 'pending'::applied_sla_status;

-- name: update-breach
UPDATE applied_slas SET
   first_response_breached_at = CASE WHEN $2 = 'first_response' THEN NOW() ELSE first_response_breached_at END,
   resolution_breached_at = CASE WHEN $2 = 'resolution' THEN NOW() ELSE resolution_breached_at END,
   updated_at = NOW()
WHERE id = $1;

-- name: update-met
UPDATE applied_slas SET
   first_response_met_at = CASE WHEN $2 = 'first_response' THEN NOW() ELSE first_response_met_at END,
   resolution_met_at = CASE WHEN $2 = 'resolution' THEN NOW() ELSE resolution_met_at END,
   updated_at = NOW()
WHERE id = $1;

-- name: set-conversation-sla-deadline
UPDATE conversations c
SET next_sla_deadline_at = CASE
    -- If resolved or closed, clear the deadline
    WHEN c.status_id IN (SELECT id FROM conversation_statuses WHERE name IN ('Resolved', 'Closed')) THEN NULL

    -- If an external timestamp ($3) is provided (e.g. next_response), use the earliest of $3.
    WHEN $3 IS NOT NULL THEN LEAST(
        $3,
        CASE
            WHEN c.first_reply_at IS NOT NULL AND c.resolved_at IS NULL AND a.resolution_deadline_at IS NOT NULL THEN a.resolution_deadline_at
            WHEN c.first_reply_at IS NULL AND c.resolved_at IS NULL AND a.first_response_deadline_at IS NOT NULL THEN a.first_response_deadline_at
            WHEN a.first_response_deadline_at IS NOT NULL AND a.resolution_deadline_at IS NOT NULL THEN LEAST(a.first_response_deadline_at, a.resolution_deadline_at)
            ELSE NULL
        END
    )

    -- No $3,
    ELSE CASE
        WHEN c.first_reply_at IS NOT NULL AND c.resolved_at IS NULL AND a.resolution_deadline_at IS NOT NULL THEN a.resolution_deadline_at
        WHEN c.first_reply_at IS NULL AND c.resolved_at IS NULL AND a.first_response_deadline_at IS NOT NULL THEN a.first_response_deadline_at
        WHEN a.first_response_deadline_at IS NOT NULL AND a.resolution_deadline_at IS NOT NULL THEN LEAST(a.first_response_deadline_at, a.resolution_deadline_at)
        ELSE NULL
    END
END
FROM applied_slas a
WHERE a.conversation_id = c.id
AND c.id = $1;

-- name: update-sla-status
UPDATE applied_slas
SET
  status = CASE 
     WHEN first_response_met_at IS NOT NULL AND resolution_met_at IS NOT NULL THEN 'met'::applied_sla_status
     WHEN first_response_breached_at IS NOT NULL AND resolution_breached_at IS NOT NULL THEN 'breached'::applied_sla_status
     WHEN (first_response_met_at IS NOT NULL OR first_response_breached_at IS NOT NULL) 
          AND (resolution_met_at IS NOT NULL OR resolution_breached_at IS NOT NULL) THEN 'partially_met'::applied_sla_status
     WHEN first_response_met_at IS NULL AND first_response_breached_at IS NULL THEN 'pending'::applied_sla_status
     ELSE 'pending'::applied_sla_status
  END,
  updated_at = NOW()
WHERE applied_slas.id = $1;

-- name: insert-scheduled-sla-notification
INSERT INTO scheduled_sla_notifications (
   applied_sla_id,
   sla_event_id,
   metric,
   notification_type,
   recipients,
   send_at
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: get-scheduled-sla-notifications
SELECT id, created_at, updated_at, applied_sla_id, sla_event_id, metric, notification_type, recipients, send_at, processed_at
FROM scheduled_sla_notifications
WHERE send_at <= NOW() AND processed_at IS NULL;

-- name: get-applied-sla
SELECT a.id,
   a.created_at,
   a.updated_at,
   a.conversation_id,
   a.sla_policy_id,
   a.first_response_deadline_at,
   a.resolution_deadline_at,
   a.first_response_met_at,
   a.resolution_met_at,
   a.first_response_breached_at,
   a.resolution_breached_at,
   a.status,
   c.first_reply_at as conversation_first_response_at,
   c.resolved_at as conversation_resolved_at,
   c.uuid as conversation_uuid,
   c.reference_number as conversation_reference_number,
   c.subject as conversation_subject,
   c.assigned_user_id as conversation_assigned_user_id,
   s.name as conversation_status
FROM applied_slas a INNER JOIN conversations c on a.conversation_id = c.id
LEFT JOIN conversation_statuses s ON c.status_id = s.id
WHERE a.id = $1;

-- name: get-latest-applied-sla-for-conversation
SELECT a.id,
   a.created_at,
   a.updated_at,
   a.conversation_id,
   a.sla_policy_id,
   a.first_response_deadline_at,
   a.resolution_deadline_at,
   a.first_response_met_at,
   a.resolution_met_at,
   a.first_response_breached_at,
   a.resolution_breached_at,
   a.status,
   c.first_reply_at as conversation_first_response_at,
   c.resolved_at as conversation_resolved_at,
   c.uuid as conversation_uuid,
   c.reference_number as conversation_reference_number,
   c.subject as conversation_subject,
   c.assigned_user_id as conversation_assigned_user_id,
   s.name as conversation_status
FROM applied_slas a
INNER JOIN conversations c ON a.conversation_id = c.id
LEFT JOIN conversation_statuses s ON c.status_id = s.id
WHERE a.conversation_id = $1
ORDER BY a.created_at DESC
LIMIT 1;

-- name: mark-notification-processed
UPDATE scheduled_sla_notifications
SET processed_at = NOW(),
      updated_at = NOW()
WHERE id = $1;

-- name: insert-next-response-sla-event
INSERT INTO sla_events (applied_sla_id, sla_policy_id, type, deadline_at)
SELECT $1, $2, 'next_response', $3
WHERE NOT EXISTS (
  SELECT 1 FROM sla_events 
  WHERE applied_sla_id = $1 AND type = 'next_response' AND met_at IS NULL
)
RETURNING id;

-- name: set-latest-sla-event-met-at
UPDATE sla_events
SET met_at = NOW(),
status = CASE WHEN NOW() > deadline_at THEN 'breached'::sla_event_status ELSE 'met'::sla_event_status END
WHERE id = (
  SELECT id FROM sla_events
  WHERE applied_sla_id = $1 AND type = $2 AND met_at IS NULL
  ORDER BY created_at DESC
  LIMIT 1
)
RETURNING met_at;

-- name: mark-sla-event-as-breached
UPDATE sla_events
SET breached_at = NOW(),
status = 'breached'
WHERE id = $1;

-- name: mark-sla-event-as-met
UPDATE sla_events
SET status = 'met'
WHERE id = $1;

-- name: get-sla-event
SELECT id, created_at, updated_at, applied_sla_id, sla_policy_id, type, deadline_at, met_at, breached_at
FROM sla_events
WHERE id = $1;

-- name: get-pending-sla-events
SELECT id
FROM sla_events
WHERE status = 'pending' and deadline_at IS NOT NULL;