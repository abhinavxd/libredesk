-- name: get-all-tags
select
    id,
    created_at,
    updated_at,
    name
from
    tags
where
    ($1 = '' or name ilike '%' || $1 || '%')
order by
    name
limit NULLIF($2, 0) offset $3;

-- name: get-tags-by-ids
select
    id,
    created_at,
    updated_at,
    name
from
    tags
where
    id = ANY($1)
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