package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_3(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS index_media_resource_avatar_source
 ON media (model_id, content_id) WHERE model_type = 'resource_avatars';`)
	return err
}
