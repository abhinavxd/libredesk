-- Browser uploads have an owner before any message or draft refers to them.
-- Existing unlinked files have no trustworthy uploader provenance; do not infer
-- ownership from user-controlled legacy draft metadata. Reupload those files.
ALTER TABLE media ADD COLUMN IF NOT EXISTS uploaded_by BIGINT REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS index_media_on_uploaded_by ON media(uploaded_by) WHERE uploaded_by IS NOT NULL;

CREATE TABLE IF NOT EXISTS conversation_draft_media (
    draft_id BIGINT NOT NULL REFERENCES conversation_drafts(id) ON DELETE CASCADE,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    PRIMARY KEY (draft_id, media_id)
);
CREATE INDEX IF NOT EXISTS index_conversation_draft_media_on_media_id ON conversation_draft_media(media_id);

-- Protect existing draft references from collection without granting ownership.
-- Match strings instead of casting legacy metadata supplied by browsers.
INSERT INTO conversation_draft_media(draft_id, media_id)
SELECT DISTINCT d.id, m.id
FROM conversation_drafts d JOIN media m ON
    d.content LIKE '%cid:ldsk-' || m.uuid::text || '%'
    OR d.content LIKE '%/uploads/' || m.uuid::text || '%'
    OR EXISTS (
        SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(d.meta->'attachments')='array'
            THEN d.meta->'attachments' ELSE '[]'::jsonb END) attachment
        WHERE attachment->>'id'=m.id::text OR attachment->>'uuid'=m.uuid::text
    )
WHERE COALESCE(m.model_id,0)=0 AND (m.model_type='messages' OR m.model_type IS NULL)
ON CONFLICT DO NOTHING;
