-- name: insert
INSERT INTO whatsapp_templates (
    inbox_id, meta_template_id, name, language, category, status,
    header_type, header_content, body_content, footer_content,
    buttons, sample_values, rejection_reason
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: update
UPDATE whatsapp_templates
SET name = $2,
    language = $3,
    category = $4,
    header_type = $5,
    header_content = $6,
    body_content = $7,
    footer_content = $8,
    buttons = $9,
    sample_values = $10,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: update-status
UPDATE whatsapp_templates
SET status = $2,
    rejection_reason = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: update-meta-id
UPDATE whatsapp_templates
SET meta_template_id = $2,
    status = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: delete
DELETE FROM whatsapp_templates WHERE id = $1;

-- name: get-by-id
SELECT * FROM whatsapp_templates WHERE id = $1;

-- name: get-by-inbox
SELECT * FROM whatsapp_templates WHERE inbox_id = $1 ORDER BY updated_at DESC;

-- name: get-by-name-language
SELECT * FROM whatsapp_templates WHERE inbox_id = $1 AND name = $2 AND language = $3;

-- name: upsert-from-meta
INSERT INTO whatsapp_templates (
    inbox_id, meta_template_id, name, language, category, status,
    header_type, header_content, body_content, footer_content,
    buttons, sample_values, rejection_reason
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (inbox_id, name, language) DO UPDATE SET
    meta_template_id = EXCLUDED.meta_template_id,
    category = EXCLUDED.category,
    status = EXCLUDED.status,
    header_type = EXCLUDED.header_type,
    header_content = EXCLUDED.header_content,
    body_content = EXCLUDED.body_content,
    footer_content = EXCLUDED.footer_content,
    buttons = EXCLUDED.buttons,
    rejection_reason = EXCLUDED.rejection_reason,
    updated_at = CASE WHEN (
        whatsapp_templates.meta_template_id IS DISTINCT FROM EXCLUDED.meta_template_id OR
        whatsapp_templates.category IS DISTINCT FROM EXCLUDED.category OR
        whatsapp_templates.status IS DISTINCT FROM EXCLUDED.status OR
        whatsapp_templates.header_type IS DISTINCT FROM EXCLUDED.header_type OR
        whatsapp_templates.header_content IS DISTINCT FROM EXCLUDED.header_content OR
        whatsapp_templates.body_content IS DISTINCT FROM EXCLUDED.body_content OR
        whatsapp_templates.footer_content IS DISTINCT FROM EXCLUDED.footer_content OR
        whatsapp_templates.buttons IS DISTINCT FROM EXCLUDED.buttons OR
        whatsapp_templates.rejection_reason IS DISTINCT FROM EXCLUDED.rejection_reason
    ) THEN NOW() ELSE whatsapp_templates.updated_at END
RETURNING *;

-- name: update-status-by-meta-name-language
UPDATE whatsapp_templates
SET status = $3,
    rejection_reason = $4,
    updated_at = NOW()
WHERE inbox_id = $1
  AND name = $2
  AND language = $5;
