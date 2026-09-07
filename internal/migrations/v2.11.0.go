package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_11_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user_two_factor (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    secret TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_step BIGINT NOT NULL DEFAULT -1,
    recovery_hashes TEXT[] NOT NULL DEFAULT '{}',
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    setup_expires_at TIMESTAMPTZ NOT NULL
);`)
	return err
}
