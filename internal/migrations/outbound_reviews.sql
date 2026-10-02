-- Contributor emails wait here until an Admin or Agent approves or denies them.
-- A submission is not a message: it is never dispatched, searched or counted
-- as unread. Approval queues a real message in the same transaction.
CREATE TABLE IF NOT EXISTS outbound_reviews (
    id BIGSERIAL PRIMARY KEY,
    "uuid" UUID NOT NULL DEFAULT gen_random_uuid() UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    kind TEXT NOT NULL CHECK (kind IN ('reply', 'new')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'withdrawn')),
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    address_id INTEGER NOT NULL REFERENCES email_addresses(id) ON DELETE CASCADE,
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE,
    subject TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    "to" TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    cc TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    bcc TEXT[] NOT NULL DEFAULT '{}'::TEXT[],
    reviewer_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    decision_note TEXT NOT NULL DEFAULT '',
    dismissed_at TIMESTAMPTZ,
    message_id BIGINT REFERENCES conversation_messages(id) ON DELETE SET NULL,
    CONSTRAINT constraint_outbound_reviews_reply_conversation CHECK (kind <> 'reply' OR conversation_id IS NOT NULL),
    CONSTRAINT constraint_outbound_reviews_subject CHECK (length(subject) <= 998),
    CONSTRAINT constraint_outbound_reviews_content CHECK (length(content) <= 1048576),
    CONSTRAINT constraint_outbound_reviews_note CHECK (length(decision_note) <= 2000)
);
CREATE INDEX IF NOT EXISTS index_outbound_reviews_on_status_and_address ON outbound_reviews(status, address_id);
CREATE INDEX IF NOT EXISTS index_outbound_reviews_on_author_and_status ON outbound_reviews(author_id, status);
CREATE INDEX IF NOT EXISTS index_outbound_reviews_on_conversation_id ON outbound_reviews(conversation_id);
-- One pending reply per contributor per conversation; the composer locks meanwhile.
CREATE UNIQUE INDEX IF NOT EXISTS index_unique_outbound_reviews_pending_reply
    ON outbound_reviews(author_id, conversation_id) WHERE status = 'pending' AND kind = 'reply';

-- Uploads referenced by a submission are protected from cleanup like draft uploads.
CREATE TABLE IF NOT EXISTS outbound_review_media (
    review_id BIGINT NOT NULL REFERENCES outbound_reviews(id) ON DELETE CASCADE,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    inline BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (review_id, media_id)
);
CREATE INDEX IF NOT EXISTS index_outbound_review_media_on_media_id ON outbound_review_media(media_id);
