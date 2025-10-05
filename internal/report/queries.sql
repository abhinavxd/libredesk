-- name: get-overview-counts
SELECT
    json_build_object(
        'open',
        COUNT(*),
        'awaiting_response',
        COUNT(
            CASE
                WHEN COALESCE(c.meta->'last_message'->>'sender_type', c.last_message_sender) = 'contact' THEN 1
            END
        ),
        'unassigned',
        COUNT(
            CASE
                WHEN c.assigned_user_id IS NULL THEN 1
            END
        ),
        'pending',
        COUNT(
            CASE
                WHEN c.first_reply_at IS NULL THEN 1
            END
        ),
        'agents_online',
        (
            SELECT
                COUNT(*)
            FROM
                users
            WHERE
                availability_status = 'online'
                AND type = 'agent'
                AND deleted_at is null
        ),
        'agents_away',
        (
            SELECT
                COUNT(*)
            FROM
                users
            WHERE
                availability_status = 'away_manual'
                AND type = 'agent'
                AND deleted_at is null
        ),
        'agents_reassigning',
        (
            SELECT
                COUNT(*)
            FROM
                users
            WHERE
                availability_status = 'away_and_reassigning'
                AND type = 'agent'
                AND deleted_at is null
        ),
        'agents_offline',
        (
            SELECT
                COUNT(*)
            FROM
                users
            WHERE
                availability_status = 'offline'
                AND type = 'agent'
                AND deleted_at is null
        )
    )
FROM
    conversations c
    INNER JOIN conversation_statuses s ON c.status_id = s.id
WHERE
    s.name not in ('Resolved', 'Closed');

-- name: get-overview-sla-counts
WITH first_and_resolution AS (
    SELECT
        COUNT(*) FILTER (
            WHERE
                first_response_met_at IS NOT NULL
        ) AS first_response_met_count,
        COUNT(*) FILTER (
            WHERE
                first_response_breached_at IS NOT NULL
        ) AS first_response_breached_count,
        COUNT(*) FILTER (
            WHERE
                resolution_met_at IS NOT NULL
        ) AS resolution_met_count,
        COUNT(*) FILTER (
            WHERE
                resolution_breached_at IS NOT NULL
        ) AS resolution_breached_count,
        COALESCE(
            AVG(
                EXTRACT(
                    EPOCH
                    FROM
                        (first_response_met_at - created_at)
                )
            ) FILTER (
                WHERE
                    first_response_met_at IS NOT NULL
            ),
            0
        ) AS avg_first_response_time_sec,
        COALESCE(
            AVG(
                EXTRACT(
                    EPOCH
                    FROM
                        (resolution_met_at - created_at)
                )
            ) FILTER (
                WHERE
                    resolution_met_at IS NOT NULL
            ),
            0
        ) AS avg_resolution_time_sec
    FROM
        applied_slas
    WHERE
        created_at >= CASE
            WHEN %d = 0 THEN CURRENT_DATE
            ELSE NOW() - INTERVAL '%d days'
        END
),
next_response AS (
    SELECT
        COUNT(*) FILTER (
            WHERE
                met_at IS NOT NULL
        ) AS next_response_met_count,
        COUNT(*) FILTER (
            WHERE
                breached_at IS NOT NULL
        ) AS next_response_breached_count,
        COALESCE(
            AVG(
                EXTRACT(
                    EPOCH
                    FROM
                        (met_at - created_at)
                )
            ) FILTER (
                WHERE
                    met_at IS NOT NULL
            ),
            0
        ) AS avg_next_response_time_sec
    FROM
        sla_events
    WHERE
        created_at >= CASE
            WHEN %d = 0 THEN CURRENT_DATE
            ELSE NOW() - INTERVAL '%d days'
        END
        AND type = 'next_response'
)
SELECT
    fas.first_response_met_count,
    fas.first_response_breached_count,
    fas.avg_first_response_time_sec,
    nr.next_response_met_count,
    nr.next_response_breached_count,
    nr.avg_next_response_time_sec,
    fas.resolution_met_count,
    fas.resolution_breached_count,
    fas.avg_resolution_time_sec
FROM
    first_and_resolution fas,
    next_response nr;

-- name: get-overview-charts
WITH new_conversations AS (
    SELECT
        json_agg(row_to_json(agg)) AS data
    FROM
        (
            SELECT
                TO_CHAR(created_at :: date, 'YYYY-MM-DD') AS date,
                COUNT(*) AS count
            FROM
                conversations c
            WHERE
                c.created_at >= CASE
                    WHEN %d = 0 THEN CURRENT_DATE
                    ELSE NOW() - INTERVAL '%d days'
                END
            GROUP BY
                date
            ORDER BY
                date
        ) agg
),
resolved_conversations AS (
    SELECT
        json_agg(row_to_json(agg)) AS data
    FROM
        (
            SELECT
                TO_CHAR(resolved_at :: date, 'YYYY-MM-DD') AS date,
                COUNT(*) AS count
            FROM
                conversations c
            WHERE
                c.resolved_at IS NOT NULL
                AND c.created_at >= CASE
                    WHEN %d = 0 THEN CURRENT_DATE
                    ELSE NOW() - INTERVAL '%d days'
                END
            GROUP BY
                date
            ORDER BY
                date
        ) agg
)
SELECT
    json_build_object(
        'new_conversations',
        (
            SELECT
                data
            FROM
                new_conversations
        ),
        'resolved_conversations',
        (
            SELECT
                data
            FROM
                resolved_conversations
        )
    ) AS result;