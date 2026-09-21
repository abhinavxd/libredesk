package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS user_image_permissions (
		user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
		senders TEXT[] NOT NULL DEFAULT '{}'::text[],
		CHECK (cardinality(senders) <= 100)
	)`)
	return err
}
