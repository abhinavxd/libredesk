-- name: get-all-tags
select
    id,
    created_at,
    updated_at,
    name
from
    tags
order by
    name;

-- name: insert-tag
INSERT into
    tags (name)
values
    ($1)
RETURNING *;

-- name: delete-tag
DELETE from
    tags
where
    id = $1;

-- name: update-tag
UPDATE
    tags
set
    name = $2,
    updated_at = now()
where
    id = $1
RETURNING *;