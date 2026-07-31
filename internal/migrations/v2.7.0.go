package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_device_tokens (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			user_id INT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			name TEXT NOT NULL,
			selector TEXT NOT NULL UNIQUE,
			verifier_hash BYTEA NOT NULL,
			last_used_at TIMESTAMPTZ NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			revoked_at TIMESTAMPTZ NULL,

			CONSTRAINT constraint_user_device_tokens_on_name CHECK (LENGTH(name) <= 140)
		);
	`); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS index_user_device_tokens_on_user_id ON user_device_tokens(user_id);`); err != nil {
		return err
	}
	return nil
}
