package migrations

import (
	"testing"

	"github.com/jakedolan443/fernmail/internal/testutil"
)

func TestFernmailBrandingPreservesCustomSettings(t *testing.T) {
	db := testutil.NewDB(t, "fernmail_branding")
	restoreFaviconSetting(t, db, "/favicon.svg")
	for _, tc := range []struct{ name, icon, wantName, wantIcon string }{
		{"libredesk", "http://localhost:9000/favicon.ico", "Fernmail", "/favicon.svg"},
		{"LibreDesk", "/favicon.ico", "Fernmail", "/favicon.svg"},
		{"Studio mail", "https://example.com/icon.svg", "Studio mail", "https://example.com/icon.svg"},
	} {
		db.MustExec(`UPDATE settings SET value = to_jsonb($1::text) WHERE key = 'app.site_name'`, tc.name)
		db.MustExec(`UPDATE settings SET value = to_jsonb($1::text) WHERE key = 'app.favicon_url'`, tc.icon)
		for range 2 {
			if err := V3_3_0(db, nil, nil); err != nil {
				t.Fatal(err)
			}
		}
		for key, want := range map[string]string{"app.site_name": tc.wantName, "app.favicon_url": tc.wantIcon} {
			var got string
			if err := db.Get(&got, `SELECT value #>> '{}' FROM settings WHERE key = $1`, key); err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("%s: got %q, want %q", key, got, want)
			}
		}
	}
}
