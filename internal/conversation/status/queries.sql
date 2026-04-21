-- name: get-status
select id,
    created_at,
    name,
    category
from conversation_statuses
where id = $1;

-- name: get-all-statuses
select id,
    created_at,
    name,
    category
from conversation_statuses;

-- name: insert-status
INSERT into conversation_statuses(name, category) values ($1, $2) RETURNING *;

-- name: delete-status
DELETE from conversation_statuses where id = $1;

-- name: update-status
UPDATE conversation_statuses set name = $2, category = $3 where id = $1 RETURNING *;
