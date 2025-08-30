-- name: get-all
SELECT id, created_at, updated_at, name, description, permissions FROM roles;

-- name: get-role
SELECT id, created_at, updated_at, name, description, permissions FROM roles where id = $1;

-- name: delete-role
DELETE FROM roles where id = $1;

-- name: insert-role
INSERT INTO roles (name, description, permissions) VALUES ($1, $2, $3) RETURNING *;

-- name: update-role
UPDATE roles SET name = $2, description = $3, permissions = $4 WHERE id = $1 RETURNING *;