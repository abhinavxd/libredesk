package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_4(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS resource_image_cache_pending (
 uuid UUID PRIMARY KEY,
 size BIGINT NOT NULL CHECK (size >= 0),
 state TEXT NOT NULL CHECK (state IN ('upload', 'delete')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS index_media_resource_cache_fifo
 ON media (created_at, id) INCLUDE (uuid, size)
 WHERE model_type IN ('resource_images', 'resource_avatars');`)
	return err
}
