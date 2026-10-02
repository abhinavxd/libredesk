-- name: history
SELECT id, campaign_id, inbox_id, browser_key, session_key, COALESCE(contact_id, 0) AS contact_id,
       snapshot, created_at, displayed, opened, dismissed, replied, COALESCE(conversation_uuid::text, '') AS conversation_uuid
FROM widget_campaign_deliveries
WHERE inbox_id = $1 AND (browser_key = $2 OR (contact_id = $3 AND $3 > 0))
ORDER BY created_at DESC;

-- name: reserve
INSERT INTO widget_campaign_deliveries(campaign_id, inbox_id, browser_key, session_key, contact_id, snapshot)
VALUES ($1, $2, $3, $4, NULLIF($5, 0), $6)
RETURNING id;

-- name: get-delivery
SELECT id, campaign_id, inbox_id, browser_key, session_key, COALESCE(contact_id, 0) AS contact_id,
       snapshot, created_at, displayed, opened, dismissed, replied, COALESCE(conversation_uuid::text, '') AS conversation_uuid
FROM widget_campaign_deliveries
WHERE id = $1 AND inbox_id = $2
  AND ((contact_id = $4 AND $4 > 0) OR (browser_key = $3 AND
       (contact_id IS NULL OR ($4 = 0 AND EXISTS (SELECT 1 FROM users WHERE users.id = contact_id AND users.type = 'visitor')))));

-- name: record-event
UPDATE widget_campaign_deliveries
SET displayed = displayed OR $2 = 'displayed',
    opened = opened OR $2 = 'opened',
    dismissed = dismissed OR $2 = 'dismissed'
WHERE id = $1;

-- name: bind-contact
UPDATE widget_campaign_deliveries SET contact_id = $3
WHERE inbox_id = $1 AND browser_key = $2 AND contact_id IS NULL;

-- name: stats
SELECT campaign_id,
       COUNT(*) FILTER (WHERE displayed)::int AS displayed,
       COUNT(*) FILTER (WHERE opened)::int AS opened,
       COUNT(*) FILTER (WHERE dismissed)::int AS dismissed,
       COUNT(*) FILTER (WHERE replied)::int AS replied
FROM widget_campaign_deliveries
WHERE inbox_id = $1 AND created_at >= $2 AND created_at < $3
GROUP BY campaign_id;
