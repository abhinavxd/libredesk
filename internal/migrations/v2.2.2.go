package migrations

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_2_2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
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
		_, err = db.Exec(fmt.Sprintf(`
			UPDATE settings SET value = '"%s"'::jsonb, updated_at = now()
			WHERE key = 'app.lang' AND value = '"%s"'::jsonb;
		`, localeRegion, localeCode))
		if err != nil {
			return err
		}
	}

	return nil
}
