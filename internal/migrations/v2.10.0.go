package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS widget_campaign_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL,
    inbox_id INTEGER NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
    browser_key UUID NOT NULL,
    session_key UUID NOT NULL,
    contact_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    displayed BOOLEAN NOT NULL DEFAULT FALSE,
    opened BOOLEAN NOT NULL DEFAULT FALSE,
    dismissed BOOLEAN NOT NULL DEFAULT FALSE,
    replied BOOLEAN NOT NULL DEFAULT FALSE,
    conversation_uuid UUID REFERENCES conversations(uuid) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_widget_campaign_browser ON widget_campaign_deliveries(inbox_id, browser_key, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_widget_campaign_contact ON widget_campaign_deliveries(inbox_id, contact_id, created_at DESC) WHERE contact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_widget_campaign_stats ON widget_campaign_deliveries(inbox_id, campaign_id, created_at);
`)
	return err
}
