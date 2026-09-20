package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_9_2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS message_image_permissions (
 user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
 message_id BIGINT REFERENCES conversation_messages(id) ON DELETE CASCADE,
 content_hash TEXT NOT NULL CHECK (length(content_hash) = 64),
 PRIMARY KEY (user_id, message_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS index_media_resource_image_source
 ON media (model_id, content_id) WHERE model_type = 'resource_images';`)
	return err
}
