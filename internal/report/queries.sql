-- name: get-overview-counts
WITH convs AS (
    SELECT
        COUNT(*) AS open,
        COUNT(*) FILTER (
            WHERE
                c.last_message_sender = 'contact'
        ) AS awaiting_response,
        COUNT(*) FILTER (
            WHERE
                c.assigned_user_id IS NULL
        ) AS unassigned,
        COUNT(*) FILTER (
            WHERE
                c.first_reply_at IS NULL
        ) AS pending
    FROM
        conversations c
        INNER JOIN conversation_statuses s ON c.status_id = s.id
    WHERE
        s.category != 'resolved'
),
agents AS (
    SELECT
        COUNT(*) FILTER (
            WHERE
                availability_status = 'online'
        ) AS agents_online,
        COUNT(*) FILTER (
            WHERE
                availability_status = 'away_manual'
        ) AS agents_away,
        COUNT(*) FILTER (
            WHERE
                availability_status = 'away_and_reassigning'
        ) AS agents_reassigning,
        COUNT(*) FILTER (
            WHERE
                availability_status IN ('offline', 'away')
        ) AS agents_offline
    FROM
        users
    WHERE
        type = 'agent'
        AND deleted_at IS NULL
)
SELECT
    json_build_object(
        'open', open,
        'awaiting_response', awaiting_response,
        'unassigned', unassigned,
        'pending', pending,
        'agents_online', agents_online,
        'agents_away', agents_away,
        'agents_reassigning', agents_reassigning,
        'agents_offline', agents_offline
    )
FROM
    convs,
    agents;

-- name: get-overview-sla-counts
-- Count only each conversation's latest applied SLA; superseded rows are kept as history and would double-count.
WITH latest_applied AS (
    SELECT DISTINCT ON (conversation_id)
        created_at, first_response_met_at, first_response_breached_at,
        resolution_met_at, resolution_breached_at
    FROM applied_slas
    WHERE created_at >= CASE
        WHEN %d = 0 THEN CURRENT_DATE
        ELSE NOW() - INTERVAL '%d days'
    END
    ORDER BY conversation_id, created_at DESC, id DESC
),
first_and_resolution AS (
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
        latest_applied
),
next_response AS (
    -- A reply after the deadline carries both met_at and breached_at, so counting the
    -- timestamps puts one event in both buckets. status holds a single terminal verdict.
    SELECT
        COUNT(*) FILTER (
            WHERE
                status = 'met'
        ) AS next_response_met_count,
        COUNT(*) FILTER (
            WHERE
                status = 'breached'
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
                    status = 'met'
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
    fas.avg_resolution_time_sec,
    CASE
        WHEN (fas.first_response_met_count + fas.first_response_breached_count) > 0
        THEN ROUND((fas.first_response_met_count::numeric / (fas.first_response_met_count + fas.first_response_breached_count)::numeric) * 100, 1)
        ELSE 0
    END AS first_response_compliance_percent,
    CASE
        WHEN (nr.next_response_met_count + nr.next_response_breached_count) > 0
        THEN ROUND((nr.next_response_met_count::numeric / (nr.next_response_met_count + nr.next_response_breached_count)::numeric) * 100, 1)
        ELSE 0
    END AS next_response_compliance_percent,
    CASE
        WHEN (fas.resolution_met_count + fas.resolution_breached_count) > 0
        THEN ROUND((fas.resolution_met_count::numeric / (fas.resolution_met_count + fas.resolution_breached_count)::numeric) * 100, 1)
        ELSE 0
    END AS resolution_compliance_percent
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
                c.resolved_at >= CASE
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

-- name: get-overview-csat
SELECT
    json_build_object(
        'average_rating',
        COALESCE(AVG(rating) FILTER (WHERE rating > 0), 0),
        'total_responses',
        COUNT(*) FILTER (WHERE rating > 0),
        'total_sent',
        COUNT(*),
        'response_rate',
        CASE
            WHEN COUNT(*) > 0
            THEN ROUND((COUNT(*) FILTER (WHERE rating > 0)::numeric / COUNT(*)::numeric) * 100, 1)
            ELSE 0
        END
    ) AS result
FROM
    csat_responses
WHERE
    created_at >= CASE
        WHEN %d = 0 THEN CURRENT_DATE
        ELSE NOW() - INTERVAL '%d days'
    END;

-- name: get-overview-message-volume
WITH per_conversation AS (
    SELECT
        conversation_id,
        COUNT(*) AS total,
        COUNT(*) FILTER (WHERE type = 'incoming') AS incoming,
        COUNT(*) FILTER (WHERE type = 'outgoing') AS outgoing
    FROM
        conversation_messages
    WHERE
        type IN ('incoming', 'outgoing')
        AND private = false
        AND (type = 'incoming' OR status = 'sent')
        AND created_at >= CASE
            WHEN %d = 0 THEN CURRENT_DATE
            ELSE NOW() - INTERVAL '%d days'
        END
    GROUP BY
        conversation_id
),
stats AS (
    SELECT
        COALESCE(SUM(total), 0) AS total,
        COALESCE(SUM(incoming), 0) AS incoming,
        COALESCE(SUM(outgoing), 0) AS outgoing,
        COUNT(*) AS convos
    FROM
        per_conversation
)
SELECT
    json_build_object(
        'total_messages', total,
        'incoming_messages', incoming,
        'outgoing_messages', outgoing,
        'messages_per_conversation',
        CASE
            WHEN convos > 0 THEN ROUND(total::numeric / convos::numeric, 1)
            ELSE 0
        END
    ) AS result
FROM
    stats;

-- name: get-overview-tag-distribution
WITH tag_counts AS (
    SELECT
        t.id AS tag_id,
        t.name AS tag_name,
        COUNT(c.id) AS count
    FROM
        tags t
        LEFT JOIN conversation_tags ct ON t.id = ct.tag_id
        LEFT JOIN conversations c ON ct.conversation_id = c.id
            AND c.created_at >= CASE
                WHEN %d = 0 THEN CURRENT_DATE
                ELSE NOW() - INTERVAL '%d days'
            END
    GROUP BY
        t.id, t.name
    ORDER BY
        count DESC, t.id
    LIMIT 10
),
tagging AS (
    SELECT
        COUNT(*) AS total,
        COUNT(*) FILTER (
            WHERE EXISTS (
                SELECT 1 FROM conversation_tags ct
                WHERE ct.conversation_id = c.id
            )
        ) AS tagged
    FROM
        conversations c
    WHERE
        c.created_at >= CASE
            WHEN %d = 0 THEN CURRENT_DATE
            ELSE NOW() - INTERVAL '%d days'
        END
)
SELECT
    json_build_object(
        'top_tags',
        COALESCE((SELECT json_agg(row_to_json(tc)) FROM tag_counts tc), '[]'::json),
        'tagged_conversations', tagged,
        'untagged_conversations', total - tagged,
        'tagged_percentage',
        CASE
            WHEN total > 0
            THEN ROUND((tagged::numeric / total::numeric) * 100, 1)
            ELSE 0
        END
    ) AS result
FROM
    tagging;