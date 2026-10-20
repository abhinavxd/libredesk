package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_0_RC11(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE EXTENSION IF NOT EXISTS pg_trgm;
		CREATE INDEX IF NOT EXISTS index_tgrm_users_on_full_name ON users USING GIN ((first_name || ' ' || COALESCE(last_name, '')) gin_trgm_ops);
	`)
	return err
}
