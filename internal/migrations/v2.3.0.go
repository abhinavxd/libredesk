package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	langMap := map[string]string{
		"da": "da-DK",
		"de": "de-DE",
		"en": "en-US",
		"es": "es-ES",
		"fa": "fa-IR",
		"fr": "fr-FR",
		"it": "it-IT",
		"ja": "ja-JP",
		"mr": "mr-IN",
	}

	for localeCode, localeRegion := range langMap {
		_, err := db.Exec(`
			UPDATE settings SET value = to_jsonb($1::text), updated_at = now()
			WHERE key = 'app.lang' AND value = to_jsonb($2::text);
		`, localeRegion, localeCode)
		if err != nil {
			return err
		}
	}

	return nil
}
