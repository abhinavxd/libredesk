-- name: get-sla-policy
SELECT id, name, description, first_response_time, resolution_time, next_response_time, notifications, created_at, updated_at FROM sla_policies WHERE id = $1;

-- name: get-all-sla-policies
SELECT id, name, description, first_response_time, resolution_time, next_response_time, notifications, created_at, updated_at FROM sla_policies ORDER BY updated_at DESC;

-- name: insert-sla-policy
INSERT INTO sla_policies (
   name,
   description, 
   first_response_time,
   resolution_time,
   next_response_time,
   notifications
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: update-sla-policy
UPDATE sla_policies SET
   name = $2,
   description = $3,
   first_response_time = $4,
   resolution_time = $5,
   next_response_time = $6,
   notifications = $7,
   updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: delete-sla-policy
-- Clears the cached deadlines first: the delete cascades applied_slas away, so afterwards the
-- recompute has nothing left to read and cannot clear them itself.
WITH cleared AS (
  UPDATE conversations SET next_sla_deadline_at = NULL
  WHERE sla_policy_id = $1 AND next_sla_deadline_at IS NOT NULL
)
DELETE FROM sla_policies WHERE id = $1;

-- name: apply-sla
-- Swaps a conversation onto a policy in one statement, which is fiddly because the old
-- applied_slas row may already carry outcomes worth keeping.
--
-- scored decides whether the outgoing row earned a verdict. closed stamps a final status on
-- rows that earned one and keeps them as history; deleted drops the rest. superseded_events
-- and superseded_notifications clear out only what was still counting down.
--
-- The insert reads COUNT(*) from deleted purely to force those CTEs to run first, otherwise
-- the unique index on (conversation_id) WHERE status = 'pending' trips against the row being
-- retired. Finally the conversation gets the new policy id and a fresh cached deadline.
WITH conv_slas AS (
  SELECT id FROM applied_slas WHERE conversation_id = $1
),
-- Next-response events are recurring and reported per event; they decide the SLA's status only when fr/res produced no outcome.
scored AS (
  SELECT id,
    (first_response_met_at IS NOT NULL OR resolution_met_at IS NOT NULL) AS frres_met,
    (first_response_breached_at IS NOT NULL OR resolution_breached_at IS NOT NULL) AS frres_breached,
    EXISTS (SELECT 1 FROM sla_events e WHERE e.applied_sla_id = applied_slas.id
        AND (e.status = 'met' OR e.met_at <= e.deadline_at)) AS event_met,
    EXISTS (SELECT 1 FROM sla_events e WHERE e.applied_sla_id = applied_slas.id
        AND (e.status = 'breached' OR e.breached_at IS NOT NULL OR e.met_at > e.deadline_at
             OR (e.met_at IS NULL AND e.deadline_at <= NOW()))) AS event_breached
  FROM applied_slas
  WHERE conversation_id = $1 AND status = 'pending'::applied_sla_status
),
-- A pending SLA with a recorded outcome is kept as closed history; deleting it would cascade scored sla_events away.
closed AS (
  UPDATE applied_slas a
  SET
    status = CASE
       WHEN s.frres_met OR s.frres_breached THEN CASE
          WHEN s.frres_met AND s.frres_breached THEN 'partially_met'::applied_sla_status
          WHEN s.frres_breached THEN 'breached'::applied_sla_status
          ELSE 'met'::applied_sla_status
       END
       ELSE CASE
          WHEN s.event_met AND s.event_breached THEN 'partially_met'::applied_sla_status
          WHEN s.event_breached THEN 'breached'::applied_sla_status
          ELSE 'met'::applied_sla_status
       END
    END,
    updated_at = NOW()
  FROM scored s
  WHERE a.id = s.id AND (s.frres_met OR s.frres_breached OR s.event_met OR s.event_breached)
  RETURNING a.id
),
-- A stamped or overdue event is a recorded outcome awaiting the tick; only countdowns still inside their deadline die with the old policy.
superseded_events AS (
  DELETE FROM sla_events
  WHERE status = 'pending'
    AND met_at IS NULL AND breached_at IS NULL
    AND deadline_at > NOW()
    AND applied_sla_id IN (SELECT id FROM conv_slas)
),
-- Breach notifications record an outcome that already happened; only warnings for dead deadlines are cancelled.
superseded_notifications AS (
  DELETE FROM scheduled_sla_notifications
  WHERE processed_at IS NULL
    AND notification_type = 'warning'
    AND applied_sla_id IN (SELECT id FROM conv_slas)
),
deleted AS (
  DELETE FROM applied_slas
  WHERE conversation_id = $1 AND status = 'pending'::applied_sla_status
    AND id NOT IN (SELECT id FROM closed)
  RETURNING id
),
new_sla AS (
  INSERT INTO applied_slas (
    conversation_id,
    sla_policy_id,
    first_response_deadline_at,
    resolution_deadline_at
  )
  -- The COUNT ref forces closed and deleted to retire the old pending row before this insert's unique-index check.
  SELECT $1, $2, $3, $4
  WHERE (SELECT COUNT(*) FROM deleted) IS NOT NULL
  RETURNING conversation_id, id
)
UPDATE conversations c
SET
   sla_policy_id = $2,
   next_sla_deadline_at = CASE
      WHEN c.status_id IN (SELECT id FROM conversation_statuses WHERE category = 'resolved') THEN NULL
      ELSE LEAST(CASE WHEN c.first_reply_at IS NULL THEN $3::TIMESTAMPTZ END, $4::TIMESTAMPTZ)
   END
FROM new_sla ns
WHERE c.id = ns.conversation_id
RETURNING ns.id;

-- name: get-pending-applied-sla
-- Feeds the SLA sweep. Returns a pending row only when a configured metric is still
-- undecided AND something happened worth judging: the deadline passed, or the conversation
-- reached the state that satisfies the metric. Rows where nothing changed are skipped so
-- each tick does no writes for them.
SELECT a.id, a.first_response_deadline_at, c.first_reply_at as conversation_first_response_at, a.sla_policy_id,
a.resolution_deadline_at, c.resolved_at as conversation_resolved_at, c.id as conversation_id, a.first_response_met_at, a.resolution_met_at, a.first_response_breached_at, a.resolution_breached_at
FROM applied_slas a
JOIN conversations c ON a.conversation_id = c.id and c.sla_policy_id = a.sla_policy_id
WHERE a.status = 'pending'::applied_sla_status
  AND (
    (a.first_response_deadline_at IS NOT NULL
     AND a.first_response_met_at IS NULL AND a.first_response_breached_at IS NULL
     AND (a.first_response_deadline_at <= NOW() OR c.first_reply_at IS NOT NULL))
    OR
    (a.resolution_deadline_at IS NOT NULL
     AND a.resolution_met_at IS NULL AND a.resolution_breached_at IS NULL
     AND (a.resolution_deadline_at <= NOW() OR c.resolved_at IS NOT NULL))
  );

-- name: get-pending-applied-sla-by-conversation
SELECT a.id, a.first_response_deadline_at, c.first_reply_at as conversation_first_response_at, a.sla_policy_id,
a.resolution_deadline_at, c.resolved_at as conversation_resolved_at, c.id as conversation_id, a.first_response_met_at, a.resolution_met_at, a.first_response_breached_at, a.resolution_breached_at
FROM applied_slas a
JOIN conversations c ON a.conversation_id = c.id and c.sla_policy_id = a.sla_policy_id
WHERE a.status = 'pending'::applied_sla_status AND a.conversation_id = $1;

-- name: update-applied-sla-breached-at
UPDATE applied_slas SET
   first_response_breached_at = CASE WHEN $2 = 'first_response' THEN NOW() ELSE first_response_breached_at END,
   resolution_breached_at = CASE WHEN $2 = 'resolution' THEN NOW() ELSE resolution_breached_at END,
   updated_at = NOW()
WHERE id = $1;

-- name: update-applied-sla-met-at
UPDATE applied_slas SET
   first_response_met_at = CASE WHEN $2 = 'first_response' THEN NOW() ELSE first_response_met_at END,
   resolution_met_at = CASE WHEN $2 = 'resolution' THEN NOW() ELSE resolution_met_at END,
   updated_at = NOW()
WHERE id = $1;

-- name: lock-conversations
SELECT 1 FROM conversations WHERE id = ANY($1::INT[]) ORDER BY id FOR UPDATE;

-- name: update-conversation-sla-deadline
UPDATE conversations c
SET next_sla_deadline_at = CASE
    WHEN c.status_id IN (SELECT id FROM conversation_statuses WHERE category = 'resolved') THEN NULL

    -- A first reply discharges the first-response deadline before the tick stamps it; LEAST ignores NULLs.
    ELSE LEAST(
        CASE WHEN c.first_reply_at IS NULL AND a.first_response_met_at IS NULL AND a.first_response_breached_at IS NULL THEN a.first_response_deadline_at END,
        CASE WHEN a.resolution_met_at IS NULL AND a.resolution_breached_at IS NULL THEN a.resolution_deadline_at END,
        (SELECT MIN(e.deadline_at) FROM sla_events e
         WHERE e.applied_sla_id = a.id AND e.status = 'pending' AND e.met_at IS NULL AND e.breached_at IS NULL)
    )
END
-- History rows accumulate per conversation; only the latest SLA carries live deadlines.
FROM unnest($1::INT[]) AS target(id)
-- LEFT so a conversation whose applied_slas rows cascaded away on policy delete still clears.
LEFT JOIN LATERAL (
    SELECT id, first_response_deadline_at, resolution_deadline_at,
        first_response_met_at, first_response_breached_at, resolution_met_at, resolution_breached_at
    FROM applied_slas
    WHERE conversation_id = target.id
    ORDER BY created_at DESC, id DESC
    LIMIT 1
) a ON true
WHERE c.id = target.id;

-- name: close-settled-applied-slas
-- Retires pending rows whose configured column metrics have all landed, and derives the final
-- status from which ones were met versus breached. A NULL deadline counts as settled because
-- that metric was never configured.
UPDATE applied_slas
SET
  status = CASE
     WHEN first_response_breached_at IS NULL AND resolution_breached_at IS NULL THEN 'met'::applied_sla_status
     WHEN first_response_met_at IS NULL AND resolution_met_at IS NULL THEN 'breached'::applied_sla_status
     ELSE 'partially_met'::applied_sla_status
  END,
  updated_at = NOW()
WHERE status = 'pending'::applied_sla_status
  -- Next-response-only rows have both deadlines NULL and no fr/res outcome to judge; closing them would stamp a status they never earned.
  AND (first_response_deadline_at IS NOT NULL OR resolution_deadline_at IS NOT NULL)
  AND (first_response_deadline_at IS NULL OR first_response_met_at IS NOT NULL OR first_response_breached_at IS NOT NULL)
  AND (resolution_deadline_at IS NULL OR resolution_met_at IS NOT NULL OR resolution_breached_at IS NOT NULL)
RETURNING conversation_id;

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
WHERE send_at <= NOW() AND processed_at IS NULL
ORDER BY send_at;

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
   s.name as conversation_status,
   s.category as conversation_status_category
FROM applied_slas a INNER JOIN conversations c on a.conversation_id = c.id
LEFT JOIN conversation_statuses s ON c.status_id = s.id
WHERE a.id = $1;

-- name: update-notification-processed
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
WITH updated AS (
  UPDATE sla_events
  SET met_at = NOW()
  WHERE id = (
    SELECT id FROM sla_events
    WHERE applied_sla_id = $1 AND type = $2 AND met_at IS NULL
    ORDER BY created_at DESC
    LIMIT 1
  )
  RETURNING applied_sla_id, met_at
)
SELECT u.met_at, a.conversation_id
FROM updated u
JOIN applied_slas a ON a.id = u.applied_sla_id;

-- name: update-sla-event-as-breached
UPDATE sla_events
SET breached_at = NOW(),
    status = 'breached'
WHERE id = $1;

-- name: update-sla-event-as-met
UPDATE sla_events
SET status = 'met'
WHERE id = $1;

-- name: get-sla-event
SELECT id, created_at, updated_at, applied_sla_id, sla_policy_id, type, deadline_at, met_at, breached_at
FROM sla_events
WHERE id = $1;

-- name: get-pending-sla-events
-- Returns full event rows whose deadline has already passed (or that already have a met_at);
SELECT e.id, e.created_at, e.updated_at, e.applied_sla_id, e.sla_policy_id, e.type, e.deadline_at, e.met_at, e.breached_at, a.conversation_id
FROM sla_events e
JOIN applied_slas a ON a.id = e.applied_sla_id
WHERE e.status = 'pending'
  AND e.deadline_at IS NOT NULL
  AND (e.deadline_at <= NOW() OR e.met_at IS NOT NULL);
