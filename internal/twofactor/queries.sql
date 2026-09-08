-- name: get-status
SELECT COALESCE((SELECT enabled FROM user_two_factor WHERE user_id = $1), FALSE) AS enabled,
    COALESCE((SELECT cardinality(recovery_hashes) FROM user_two_factor WHERE user_id = $1 AND enabled), 0) AS recovery_codes_remaining;

-- name: begin-setup
INSERT INTO user_two_factor (user_id, secret, setup_expires_at)
VALUES ($1, $2, NOW() + INTERVAL '10 minutes')
ON CONFLICT (user_id) DO UPDATE SET secret = EXCLUDED.secret, setup_expires_at = EXCLUDED.setup_expires_at
WHERE NOT user_two_factor.enabled;

-- name: lock-enrollment
SELECT secret, enabled, last_step, recovery_hashes, failed_attempts, locked_until, setup_expires_at
FROM user_two_factor WHERE user_id = $1 FOR UPDATE;

-- name: save-enrollment
UPDATE user_two_factor SET enabled = $2, last_step = $3, recovery_hashes = $4,
    failed_attempts = $5, locked_until = $6
WHERE user_id = $1;

-- name: delete-enrollment
DELETE FROM user_two_factor WHERE user_id = $1;
