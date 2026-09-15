-- name: get-notifications
SELECT
    n.id, n.created_at, n.updated_at, n.user_id, n.notification_type,
    n.title, n.body, n.is_read, n.conversation_id, n.message_id, n.actor_id, n.meta,
    u.first_name as actor_first_name, u.last_name as actor_last_name, u.avatar_url as actor_avatar_url,
    c.uuid as conversation_uuid, m.uuid as message_uuid
FROM user_notifications n
LEFT JOIN users u ON u.id = n.actor_id
LEFT JOIN conversations c ON c.id = n.conversation_id
LEFT JOIN conversation_messages m ON m.id = n.message_id
WHERE n.user_id = $1
ORDER BY n.created_at DESC
LIMIT $2 OFFSET $3;

-- name: get-notification-stats
SELECT
    COUNT(*) FILTER (WHERE is_read = false) as unread_count,
    COUNT(*) as total_count
FROM user_notifications
WHERE user_id = $1;

-- name: insert-notification
INSERT INTO user_notifications (user_id, notification_type, title, body, conversation_id, message_id, actor_id, meta)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at, updated_at, user_id, notification_type, title, body, is_read, conversation_id, message_id, actor_id, meta;

-- name: mark-as-read
UPDATE user_notifications SET is_read = true, updated_at = now() WHERE id = $1 AND user_id = $2 RETURNING id;

-- name: mark-assignment-as-read
UPDATE user_notifications
SET is_read = true, updated_at = now()
WHERE conversation_id = $1
  AND user_id = $2
  AND notification_type = 'assignment'
  AND is_read = false;

-- name: mark-all-as-read
UPDATE user_notifications SET is_read = true, updated_at = now() WHERE user_id = $1 AND is_read = false;

-- name: delete-notification
DELETE FROM user_notifications WHERE id = $1 AND user_id = $2;

-- name: delete-all-notifications
DELETE FROM user_notifications WHERE user_id = $1;

-- name: delete-old-notifications
DELETE FROM user_notifications WHERE created_at < NOW() - INTERVAL '30 days';

-- name: get-notification-preferences
SELECT notification_type, channel, enabled
FROM user_notification_preferences
WHERE user_id = $1;

-- name: get-notification-preferences-for-type
SELECT user_id, channel, enabled
FROM user_notification_preferences
WHERE user_id = ANY($1::bigint[]) AND notification_type = $2;

-- name: upsert-notification-preference
INSERT INTO user_notification_preferences (user_id, notification_type, channel, enabled)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, notification_type, channel)
DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = now();

-- name: get-push-subscriptions
SELECT id, endpoint, p256dh, auth
FROM notification_push_subscriptions
WHERE user_id = $1;

-- name: upsert-push-subscription
INSERT INTO notification_push_subscriptions (user_id, endpoint, p256dh, auth)
VALUES ($1, $2, $3, $4)
ON CONFLICT (endpoint) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    p256dh = EXCLUDED.p256dh,
    auth = EXCLUDED.auth,
    updated_at = now();

-- name: delete-push-subscription
DELETE FROM notification_push_subscriptions WHERE user_id = $1 AND endpoint = $2;

-- name: delete-push-subscription-by-id
DELETE FROM notification_push_subscriptions WHERE id = $1;

-- name: is-notification-seen
SELECT COALESCE((SELECT is_read FROM user_notifications WHERE id = $2 AND user_id = $1), false)
    OR EXISTS (
        SELECT 1 FROM conversation_last_seen
        WHERE conversation_id = $3 AND user_id = $1 AND last_seen_at >= $4
    );

-- name: enqueue-notification-email
INSERT INTO notification_email_queue
    (user_id, notification_id, notification_type, conversation_id, recipient_email, subject, content, send_at, message_created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (user_id, notification_type, conversation_id) DO UPDATE SET
    notification_id = EXCLUDED.notification_id,
    recipient_email = EXCLUDED.recipient_email,
    subject = EXCLUDED.subject,
    content = EXCLUDED.content,
    message_created_at = EXCLUDED.message_created_at,
    send_at = LEAST(notification_email_queue.send_at, EXCLUDED.send_at),
    attempts = 0,
    queued_at = now(),
    updated_at = now();

-- name: claim-due-notification-emails
WITH due AS (
    SELECT id
    FROM notification_email_queue
    WHERE send_at <= now()
    ORDER BY send_at
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
UPDATE notification_email_queue AS q
SET send_at = $2, updated_at = now()
FROM due
WHERE q.id = due.id
RETURNING q.id, q.updated_at, q.user_id, q.notification_id, q.notification_type, q.conversation_id, q.attempts,
    q.recipient_email, q.subject, q.content, COALESCE(q.message_created_at, q.queued_at) AS message_created_at;

-- name: delete-claimed-notification-email
DELETE FROM notification_email_queue WHERE id = $1 AND updated_at = $2;

-- name: retry-claimed-notification-email
UPDATE notification_email_queue
SET send_at = $2, attempts = attempts + 1, updated_at = now()
WHERE id = $1 AND updated_at = $3;
